package bench

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type pythonHarness struct {
	command *exec.Cmd
	input   io.WriteCloser
	output  *bufio.Reader
	lock    sync.Mutex
	root    string
}

type harnessRequest struct {
	Action     string `json:"action"`
	Library    string `json:"library,omitempty"`
	Direction  string `json:"direction,omitempty"`
	Size       int    `json:"size,omitempty"`
	Iterations int    `json:"iterations,omitempty"`
	Cold       bool   `json:"cold,omitempty"`
}

type harnessResponse struct {
	OK         bool   `json:"ok"`
	Available  bool   `json:"available"`
	Version    string `json:"version,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Ciphertext string `json:"ciphertext,omitempty"`
	Plaintext  string `json:"plaintext,omitempty"`
	ElapsedNS  int64  `json:"elapsed_ns,omitempty"`
	Checksum   uint64 `json:"checksum,omitempty"`
}

func cacheRoot() string {
	if value := os.Getenv("GTGBENCH_CACHE"); value != "" {
		return value
	}
	if runtime.GOOS == "darwin" {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, "Library", "Caches", "gtg-bench")
		}
	}
	if runtime.GOOS == "windows" {
		if value := os.Getenv("LOCALAPPDATA"); value != "" {
			return filepath.Join(value, "gtg-bench")
		}
	}
	if value := os.Getenv("XDG_CACHE_HOME"); value != "" {
		return filepath.Join(value, "gtg-bench")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".cache", "gtg-bench")
	}
	return filepath.Join(os.TempDir(), "gtg-bench")
}

func findPython() string {
	for _, candidate := range []string{"python3", "python"} {
		path, err := exec.LookPath(candidate)
		if err != nil {
			continue
		}
		out, err := exec.Command(path, "-c", "import sys; print('%d.%d' % sys.version_info[:2])").Output()
		if err != nil {
			continue
		}
		parts := strings.Split(strings.TrimSpace(string(out)), ".")
		major, majorErr := strconv.Atoi(parts[0])
		minor := 0
		var minorErr error
		if len(parts) == 2 {
			minor, minorErr = strconv.Atoi(parts[1])
		}
		if len(parts) == 2 && majorErr == nil && minorErr == nil && (major > 3 || major == 3 && minor >= 9) {
			return path
		}
	}
	return ""
}

func preparePython(progress func(string)) (*pythonHarness, error) {
	python := findPython()
	if python == "" {
		return nil, errors.New("Python 3.9 or newer was not found")
	}
	root := cacheRoot()
	venv := filepath.Join(root, "python")
	if runtime.GOOS == "windows" {
		venv = filepath.Join(root, "python")
	}
	interpreter := filepath.Join(venv, "bin", "python")
	if runtime.GOOS == "windows" {
		interpreter = filepath.Join(venv, "Scripts", "python.exe")
	}
	if _, err := os.Stat(interpreter); err != nil {
		if progress != nil {
			progress("Creating cache-local Python environment")
		}
		if err := os.MkdirAll(root, 0700); err != nil {
			return nil, err
		}
		if output, err := exec.Command(python, "-m", "venv", venv).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("create Python environment: %w: %s", err, strings.TrimSpace(string(output)))
		}
	}
	marker := filepath.Join(venv, ".gtg-bench-requirements-v1")
	if _, err := os.Stat(marker); err != nil {
		if progress != nil {
			progress("Installing pinned Python comparators in the cache-local environment")
		}
		args := []string{"-m", "pip", "install", "--disable-pip-version-check", "--upgrade", "GoyGram==0.8.32", "TgCrypto==1.2.5", "cryptg==0.6.0"}
		if output, err := exec.Command(interpreter, args...).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("install Python comparators: %w: %s", err, strings.TrimSpace(string(output)))
		}
		if err := os.WriteFile(marker, []byte("GoyGram==0.8.32\nTgCrypto==1.2.5\ncryptg==0.6.0\n"), 0600); err != nil {
			return nil, err
		}
	}
	harnessPath := filepath.Join(root, "harness.py")
	if err := os.WriteFile(harnessPath, []byte(pythonProgram), 0600); err != nil {
		return nil, err
	}
	command := exec.Command(interpreter, "-u", harnessPath)
	input, err := command.StdinPipe()
	if err != nil {
		return nil, err
	}
	output, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return nil, err
	}
	return &pythonHarness{command: command, input: input, output: bufio.NewReader(output), root: root}, nil
}

func (h *pythonHarness) request(request harnessRequest) (harnessResponse, error) {
	h.lock.Lock()
	defer h.lock.Unlock()
	var response harnessResponse
	data, err := json.Marshal(request)
	if err != nil {
		return response, err
	}
	if _, err := h.input.Write(append(data, '\n')); err != nil {
		return response, err
	}
	line, err := h.output.ReadBytes('\n')
	if err != nil {
		return response, err
	}
	if err := json.Unmarshal(line, &response); err != nil {
		return response, fmt.Errorf("decode Python response %q: %w", strings.TrimSpace(string(line)), err)
	}
	return response, nil
}

func (h *pythonHarness) close() {
	if h == nil {
		return
	}
	_ = h.input.Close()
	_ = h.command.Wait()
}

func (h *pythonHarness) qualify(library string) (harnessResponse, error) {
	return h.qualifySize(library, 64)
}

func (h *pythonHarness) qualifySize(library string, size int) (harnessResponse, error) {
	return h.request(harnessRequest{Action: "qualify", Library: library, Size: size})
}

func (h *pythonHarness) measure(library string, direction Direction, size, iterations int, cold bool) (harnessResponse, error) {
	return h.request(harnessRequest{Action: "measure", Library: library, Direction: string(direction), Size: size, Iterations: iterations, Cold: cold})
}

func decodeEncoded(value string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(value)
}
