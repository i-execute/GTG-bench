package bench

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func ReadJSON(path string) (Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Report{}, err
	}
	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return Report{}, err
	}
	if report.Schema != "gtg-bench/v1" {
		return Report{}, fmt.Errorf("unsupported report schema %q", report.Schema)
	}
	if report.Mode != Fast && report.Mode != Full {
		return Report{}, fmt.Errorf("unsupported report mode %q", report.Mode)
	}
	if report.Workers == 0 {
		report.Workers = 1
		for index := range report.Samples {
			if report.Samples[index].Workers == 0 {
				report.Samples[index].Workers = 1
			}
		}
	}
	return report, nil
}

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
	for _, item := range [][2]string{{"OS", report.Environment.OS}, {"Architecture", report.Environment.Arch}, {"Go", report.Environment.GoVersion}, {"Workers", strconv.Itoa(report.Workers)}, {"Logical CPUs", strconv.Itoa(report.Environment.CPUs)}, {"CPU", report.Environment.CPUModel}, {"CPU frequency (Hz)", strconv.FormatInt(report.Environment.CPUFrequencyHz, 10)}, {"Memory (bytes)", strconv.FormatUint(report.Environment.MemoryBytes, 10)}, {"Timer resolution (ns)", strconv.FormatInt(report.Environment.TimerResolutionNS, 10)}, {"Acceleration", report.Environment.Acceleration}, {"Virtualization", report.Environment.Virtualization}, {"Power", report.Environment.Power}, {"Load", report.Environment.Load}} {
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
	builder.WriteString("\n## GTG operation samples\n\n| Operation | Direction | Cache | Size | Median ns/op | Iterations | Workers | Samples |\n|---|---|---|---:|---:|---:|---:|---:|\n")
	for _, row := range operationRows(report.Samples) {
		builder.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %.1f | %d | %d | %d |\n", escapeCell(row.Operation), row.Direction, row.Cache, humanBytes(row.Size), row.NSPerOperation, row.Iterations, row.Workers, row.Count))
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
	if err := writer.Write([]string{"library", "operation", "direction", "cache", "size", "round", "position", "order", "elapsed_ns", "iterations", "workers", "checksum", "at"}); err != nil {
		return err
	}
	for _, sample := range report.Samples {
		if err := writer.Write([]string{sample.Library, sample.Operation, string(sample.Direction), string(sample.Cache), strconv.Itoa(sample.Size), strconv.Itoa(sample.Round), strconv.Itoa(sample.Position), sample.Order, strconv.FormatInt(sample.ElapsedNS, 10), strconv.Itoa(sample.Iterations), strconv.Itoa(sample.Workers), strconv.FormatUint(sample.Checksum, 10), sample.At.Format("2006-01-02T15:04:05.999999999Z")}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

type operationRow struct {
	Operation      string
	Direction      Direction
	Cache          CacheMode
	Size           int
	NSPerOperation float64
	Iterations     int
	Workers        int
	Count          int
}

func operationRows(samples []Sample) []operationRow {
	type key struct {
		operation string
		direction Direction
		cache     CacheMode
		size      int
		workers   int
	}
	groups := map[key][]float64{}
	iterations := map[key]int{}
	for _, sample := range samples {
		if sample.Library != "GTG" || sample.Order == "baseline" || sample.Operation == "AES-256-IGE" || sample.Iterations <= 0 {
			continue
		}
		item := key{operation: sample.Operation, direction: sample.Direction, cache: sample.Cache, size: sample.Size, workers: sample.Workers}
		groups[item] = append(groups[item], float64(sample.ElapsedNS)/float64(sample.Iterations))
		iterations[item] = sample.Iterations
	}
	rows := make([]operationRow, 0, len(groups))
	for item, values := range groups {
		sort.Float64s(values)
		middle := len(values) / 2
		median := values[middle]
		if len(values)%2 == 0 {
			median = (values[middle-1] + values[middle]) / 2
		}
		rows = append(rows, operationRow{Operation: item.operation, Direction: item.direction, Cache: item.cache, Size: item.size, NSPerOperation: median, Iterations: iterations[item], Workers: item.workers, Count: len(values)})
	}
	sort.Slice(rows, func(left, right int) bool {
		if rows[left].Operation != rows[right].Operation {
			return rows[left].Operation < rows[right].Operation
		}
		if rows[left].Size != rows[right].Size {
			return rows[left].Size < rows[right].Size
		}
		return rows[left].Workers < rows[right].Workers
	})
	return rows
}

func escapeCell(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "|", "\\|"), "\n", " ")
}
