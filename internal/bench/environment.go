package bench

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func DetectEnvironment() Environment {
	env := Environment{
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		GoVersion:      runtime.Version(),
		CPUs:           runtime.NumCPU(),
		Acceleration:   acceleration(),
		Virtualization: virtualization(),
		Power:          powerState(),
		Load:           loadState(),
	}
	env.CPUModel = cpuModel()
	if env.Virtualization != "none detected" {
		env.Warnings = append(env.Warnings, "virtualized execution detected; compare only runs made in a similar guest configuration")
	}
	if env.CPUs < 2 {
		env.Warnings = append(env.Warnings, "one logical CPU is visible; scheduler noise can dominate small-call measurements")
	}
	if env.Load != "unavailable" && env.Load != "low" {
		env.Warnings = append(env.Warnings, "host load is not low; no observations are removed, but confidence intervals can widen")
	}
	if env.Power != "AC or unavailable" {
		env.Warnings = append(env.Warnings, "battery power is reported; frequency policy can affect timing")
	}
	return env
}

func cpuModel() string {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "unavailable"
	}
	defer file.Close()
	fallback := ""
	s := bufio.NewScanner(file)
	for s.Scan() {
		parts := strings.SplitN(s.Text(), ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(parts[1])
		if name == "model name" || name == "hardware" {
			return value
		}
		if name == "processor" && fallback == "" {
			fallback = value
		}
	}
	if fallback != "" {
		return fallback
	}
	return "unavailable"
}

func acceleration() string {
	if runtime.GOARCH == "amd64" {
		data, err := os.ReadFile("/proc/cpuinfo")
		if err == nil {
			flags := strings.ToLower(string(data))
			if strings.Contains(flags, " aes ") || strings.Contains(flags, " aes\n") {
				return "AES instructions reported"
			}
		}
		return "AES instruction status unavailable"
	}
	if runtime.GOARCH == "arm64" {
		return "arm64 acceleration is platform-selected; inspect native run metadata"
	}
	return "architecture-specific acceleration not classified"
}

func virtualization() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err == nil && strings.Contains(strings.ToLower(string(data)), "hypervisor") {
		return "hypervisor CPU flag"
	}
	for _, path := range []string{"/sys/class/dmi/id/product_name", "/sys/class/dmi/id/sys_vendor"} {
		data, err := os.ReadFile(path)
		if err == nil {
			value := strings.ToLower(string(data))
			if strings.Contains(value, "kvm") || strings.Contains(value, "virtual") || strings.Contains(value, "vmware") || strings.Contains(value, "amazon") {
				return "DMI virtualization indicator"
			}
		}
	}
	return "none detected"
}

func powerState() string {
	entries, err := filepath.Glob("/sys/class/power_supply/*/status")
	if err != nil || len(entries) == 0 {
		return "AC or unavailable"
	}
	for _, entry := range entries {
		data, err := os.ReadFile(entry)
		if err == nil && strings.TrimSpace(strings.ToLower(string(data))) == "discharging" {
			return "battery discharging"
		}
	}
	return "AC or unavailable"
}

func loadState() string {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return "unavailable"
	}
	value := strings.Fields(string(data))
	if len(value) == 0 {
		return "unavailable"
	}
	load, err := strconv.ParseFloat(value[0], 64)
	if err != nil {
		return "unavailable"
	}
	if load <= float64(runtime.NumCPU())*0.35 {
		return "low"
	}
	if load <= float64(runtime.NumCPU()) {
		return "moderate"
	}
	return "high"
}
