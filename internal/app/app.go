package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/i-execute/GTG-bench/internal/bench"
)

type Config struct {
	Mode         bench.Mode
	NoPython     bool
	JSONPath     string
	MarkdownPath string
	ReportDir    string
	Plain        bool
	Interactive  bool
}

func Run(config Config, input io.Reader, output io.Writer) (bench.Report, error) {
	reader := bufio.NewReader(input)
	if config.Interactive {
		choice, err := menu(reader, output, config.Plain)
		if err != nil {
			return bench.Report{}, err
		}
		if choice == "q" {
			return bench.Report{}, nil
		}
		if choice == "f" {
			config.Mode = bench.Full
		}
		if choice == "e" {
			showEnvironment(output, bench.DetectEnvironment(), config.Plain)
			choice, err = menu(reader, output, config.Plain)
			if err != nil {
				return bench.Report{}, err
			}
			if choice == "q" {
				return bench.Report{}, nil
			}
			if choice == "f" {
				config.Mode = bench.Full
			}
		}
	}
	if config.Mode == "" {
		config.Mode = bench.Fast
	}
	progress := func(message string) {
		if config.Plain {
			fmt.Fprintln(output, "[run]", message)
			return
		}
		fmt.Fprintf(output, "\r\033[2K\033[38;5;81m›\033[0m %s", message)
	}
	report, err := bench.Run(bench.Options{Mode: config.Mode, NoPython: config.NoPython, Progress: progress})
	if !config.Plain {
		fmt.Fprintln(output)
	}
	if err != nil {
		return report, err
	}
	directory := config.ReportDir
	if directory == "" {
		directory = filepath.Join("reports", report.Started.Format("20060102-150405Z"))
	}
	jsonPath := config.JSONPath
	if jsonPath == "" {
		jsonPath = filepath.Join(directory, "report.json")
	}
	markdownPath := config.MarkdownPath
	if markdownPath == "" {
		markdownPath = filepath.Join(directory, "report.md")
	}
	csvPath := filepath.Join(filepath.Dir(jsonPath), "samples.csv")
	if err := bench.WriteJSON(jsonPath, report); err != nil {
		return report, err
	}
	if err := bench.WriteMarkdown(markdownPath, report); err != nil {
		return report, err
	}
	if err := bench.WriteCSV(csvPath, report); err != nil {
		return report, err
	}
	showReport(output, report, jsonPath, markdownPath, csvPath, config.Plain)
	return report, nil
}

func menu(input *bufio.Reader, output io.Writer, plain bool) (string, error) {
	env := bench.DetectEnvironment()
	if plain {
		fmt.Fprintln(output, "GTG-bench")
		fmt.Fprintf(output, "System: %s/%s, %d logical CPUs, %s\n", env.OS, env.Arch, env.CPUs, env.Virtualization)
		fmt.Fprintln(output, "1) Fast AES-256-IGE comparison")
		fmt.Fprintln(output, "2) Full paired comparison")
		fmt.Fprintln(output, "3) Environment details")
		fmt.Fprintln(output, "q) Quit")
	} else {
		fmt.Fprint(output, "\033[2J\033[H")
		fmt.Fprintln(output, "\033[1;38;5;81m╭──────────────────────────────────────────────────────────────────────╮\033[0m")
		fmt.Fprintln(output, "\033[1;38;5;81m│  GTG-bench\033[0m  \033[38;5;245mAES-256-IGE local comparison lab                       \033[1;38;5;81m│\033[0m")
		fmt.Fprintln(output, "\033[1;38;5;81m╰──────────────────────────────────────────────────────────────────────╯\033[0m")
		fmt.Fprintf(output, "  \033[38;5;110m%s/%s\033[0m  ·  \033[38;5;110m%d CPUs\033[0m  ·  \033[38;5;110m%s\033[0m\n\n", env.OS, env.Arch, env.CPUs, env.Acceleration)
		fmt.Fprintln(output, "  \033[1;38;5;42m1\033[0m  Fast comparison     focused sizes, hot cache")
		fmt.Fprintln(output, "  \033[1;38;5;42m2\033[0m  Full comparison     16 B to 64 MiB, hot and cold cache")
		fmt.Fprintln(output, "  \033[1;38;5;42m3\033[0m  Environment         CPU, load, power, virtualization")
		fmt.Fprintln(output, "  \033[1;38;5;245mq\033[0m  Quit")
	}
	fmt.Fprint(output, "\n  Select: ")
	line, err := input.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	switch strings.TrimSpace(strings.ToLower(line)) {
	case "1", "", "fast":
		return "s", nil
	case "2", "full":
		return "f", nil
	case "3", "env", "environment":
		return "e", nil
	case "q", "quit", "exit":
		return "q", nil
	default:
		return "s", nil
	}
}

func showEnvironment(output io.Writer, env bench.Environment, plain bool) {
	if plain {
		fmt.Fprintln(output, "Environment")
	} else {
		fmt.Fprintln(output, "\n\033[1;38;5;81mEnvironment\033[0m")
	}
	fmt.Fprintf(output, "  OS / architecture: %s / %s\n", env.OS, env.Arch)
	fmt.Fprintf(output, "  Go runtime:        %s\n", env.GoVersion)
	fmt.Fprintf(output, "  CPU:               %s\n", env.CPUModel)
	fmt.Fprintf(output, "  Acceleration:      %s\n", env.Acceleration)
	fmt.Fprintf(output, "  Virtualization:    %s\n", env.Virtualization)
	fmt.Fprintf(output, "  Power:             %s\n", env.Power)
	fmt.Fprintf(output, "  Load:              %s\n\n", env.Load)
}

func showReport(output io.Writer, report bench.Report, jsonPath, markdownPath, csvPath string, plain bool) {
	if plain {
		fmt.Fprintln(output, "Results")
	} else {
		fmt.Fprintln(output, "\033[1;38;5;42m✓ Local run complete\033[0m")
	}
	for _, result := range report.Pairs {
		flags := strings.Trim(strings.Join([]string{result.Noise, result.Drift}, "; "), "; ")
		fmt.Fprintf(output, "  %-8s %-7s %7d B  ratio %.4f  CI [%.4f, %.4f]  %s", result.Library, result.Direction, result.Size, result.GeometricMean, result.CILow, result.CIHigh, result.Verdict)
		if flags != "" {
			fmt.Fprintf(output, "  [%s]", flags)
		}
		fmt.Fprintln(output)
	}
	if len(report.Pairs) == 0 {
		fmt.Fprintln(output, "  GTG-only samples were retained. No qualified Python comparator was available.")
	}
	fmt.Fprintln(output, "\n  Raw samples:", csvPath)
	fmt.Fprintln(output, "  JSON report:", jsonPath)
	fmt.Fprintln(output, "  Markdown:", markdownPath)
	fmt.Fprintln(output, "  Note: end-to-end public call costs cross different Go and CPython call boundaries; use size-regression slopes for bulk-throughput interpretation.")
}

func DefaultReportDir() string {
	return filepath.Join("reports", time.Now().UTC().Format("20060102-150405Z"))
}

func StandardInput() io.Reader {
	return os.Stdin
}
