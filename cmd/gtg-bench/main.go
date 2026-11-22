package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"

	"github.com/i-execute/GTG-bench/internal/app"
	"github.com/i-execute/GTG-bench/internal/bench"
)

var version = "0.1.0-dev"

func main() {
	fast := flag.Bool("fast", false, "run focused hot-cache comparison")
	full := flag.Bool("full", false, "run 16 B to 64 MiB hot/cold comparison")
	noPython := flag.Bool("no-python", false, "do not initialize Python comparators")
	workers := flag.Int("workers", 1, "GTG AES-IGE workers; values above one run GTG-only parallel samples")
	jsonPath := flag.String("json", "", "write JSON report to path")
	markdownPath := flag.String("markdown", "", "write Markdown report to path")
	reportDir := flag.String("report-dir", "", "directory for default reports")
	resumePath := flag.String("resume", "", "resume a partial JSON report in place")
	plain := flag.Bool("plain", false, "disable terminal styling")
	environment := flag.Bool("env", false, "print environment and exit")
	clean := flag.Bool("clean", false, "remove GTG-bench cache and exit")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}
	if *workers < 1 {
		fail(fmt.Errorf("workers must be positive"))
	}
	if *clean {
		root := cacheRoot()
		if err := os.RemoveAll(root); err != nil {
			fail(err)
		}
		fmt.Println("Removed", root)
		return
	}
	if *environment {
		env := bench.DetectEnvironment()
		fmt.Printf("OS: %s\nArchitecture: %s\nGo: %s\nCPUs: %d\nCPU: %s\nCPU frequency (Hz): %d\nMemory (bytes): %d\nTimer resolution (ns): %d\nAcceleration: %s\nVirtualization: %s\nPower: %s\nLoad: %s\n", env.OS, env.Arch, env.GoVersion, env.CPUs, env.CPUModel, env.CPUFrequencyHz, env.MemoryBytes, env.TimerResolutionNS, env.Acceleration, env.Virtualization, env.Power, env.Load)
		for _, warning := range env.Warnings {
			fmt.Println("Warning:", warning)
		}
		return
	}
	mode := bench.Mode("")
	interactive := true
	if *fast {
		mode = bench.Fast
		interactive = false
	}
	if *full {
		mode = bench.Full
		interactive = false
	}
	if *resumePath != "" {
		interactive = false
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	_, err := app.Run(app.Config{Mode: mode, NoPython: *noPython, Workers: *workers, JSONPath: *jsonPath, MarkdownPath: *markdownPath, ReportDir: *reportDir, Plain: *plain, Interactive: interactive, Context: ctx, ResumePath: *resumePath}, os.Stdin, os.Stdout)
	if err != nil {
		fail(err)
	}
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
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "gtg-bench")
	}
	return filepath.Join(home, ".cache", "gtg-bench")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "gtg-bench:", err)
	os.Exit(1)
}
