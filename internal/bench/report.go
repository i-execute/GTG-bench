package bench

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func WriteJSON(path string, report Report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0600)
}

func WriteMarkdown(path string, report Report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	var builder strings.Builder
	builder.WriteString("# GTG-bench local report\n\n")
	builder.WriteString("This report contains only values measured by this invocation.\n\n")
	builder.WriteString("## Environment\n\n")
	builder.WriteString("| Field | Value |\n|---|---|\n")
	for _, item := range [][2]string{{"OS", report.Environment.OS}, {"Architecture", report.Environment.Arch}, {"Go", report.Environment.GoVersion}, {"Logical CPUs", strconv.Itoa(report.Environment.CPUs)}, {"CPU", report.Environment.CPUModel}, {"Acceleration", report.Environment.Acceleration}, {"Virtualization", report.Environment.Virtualization}, {"Power", report.Environment.Power}, {"Load", report.Environment.Load}} {
		builder.WriteString("| " + escapeCell(item[0]) + " | " + escapeCell(item[1]) + " |\n")
	}
	builder.WriteString("\n## Qualification\n\n| Library | Available | Qualified | Version | Reason |\n|---|---:|---:|---|---|\n")
	for _, qualification := range report.Qualifications {
		builder.WriteString(fmt.Sprintf("| %s | %t | %t | %s | %s |\n", escapeCell(qualification.Library), qualification.Available, qualification.Qualified, escapeCell(qualification.Version), escapeCell(qualification.Reason)))
	}
	builder.WriteString("\n## Paired results\n\nThe ratio is GTG time divided by comparator time. Faster requires the complete 95% paired-ratio interval below 0.97. Slower requires it above 1.03. All other rows are parity.\n\n")
	builder.WriteString("| Comparator | Direction | Cache | Size | Ratio | 95% CI | Verdict | Flags |\n|---|---|---|---:|---:|---:|---|---|\n")
	for _, result := range report.Pairs {
		flags := strings.Trim(strings.Join([]string{result.Noise, result.Drift}, "; "), "; ")
		builder.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %.4f | [%.4f, %.4f] | %s | %s |\n", escapeCell(result.Library), result.Direction, result.Cache, humanBytes(result.Size), result.GeometricMean, result.CILow, result.CIHigh, result.Verdict, escapeCell(flags)))
	}
	builder.WriteString("\n## Warnings\n\n")
	if len(report.Warnings) == 0 {
		builder.WriteString("None reported.\n")
	} else {
		for _, warning := range report.Warnings {
			builder.WriteString("- " + warning + "\n")
		}
	}
	builder.WriteString("\n## Interpretation\n\n")
	builder.WriteString("The timed operation is public AES-256-IGE encryption or decryption with bytes inputs and output returned by each library. GTG runs inside this Go process; other libraries run under CPython. This measures each library's end-to-end public call cost in its natural runtime, not a language-neutral instruction count. Size-regression slopes can describe bulk throughput more safely than tiny-input call costs. Raw, undiscarded observations are retained in the adjacent JSON and CSV files.\n")
	return os.WriteFile(path, []byte(builder.String()), 0600)
}

func WriteCSV(path string, report Report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	if err := writer.Write([]string{"library", "direction", "cache", "size", "round", "position", "order", "elapsed_ns", "iterations", "checksum", "at"}); err != nil {
		return err
	}
	for _, sample := range report.Samples {
		if err := writer.Write([]string{sample.Library, string(sample.Direction), string(sample.Cache), strconv.Itoa(sample.Size), strconv.Itoa(sample.Round), strconv.Itoa(sample.Position), sample.Order, strconv.FormatInt(sample.ElapsedNS, 10), strconv.Itoa(sample.Iterations), strconv.FormatUint(sample.Checksum, 10), sample.At.Format("2006-01-02T15:04:05.999999999Z")}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func escapeCell(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "|", "\\|"), "\n", " ")
}
