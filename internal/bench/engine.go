package bench

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"sync"
	"time"

	"github.com/i-execute/GTG/ige"
	"github.com/i-execute/GTG/mtproto"
)

var libraries = []string{"GoyGram", "TgCrypto", "cryptg"}

func Run(options Options) (report Report, err error) {
	ctx := options.Context
	if ctx == nil {
		ctx = context.Background()
	}
	started := time.Now().UTC()
	resolution := timerResolution()
	mode := options.Mode
	if mode != Full {
		mode = Fast
	}
	workers := options.Workers
	if workers < 1 {
		workers = 1
	}
	report = Report{
		Schema:      "gtg-bench/v1",
		Started:     started,
		Mode:        mode,
		Workers:     workers,
		Environment: DetectEnvironment(),
		Methodology: "AES-256-IGE and MTProto Seal/Open. Each AES comparator is byte-qualified against GTG ciphertext and reverse-decryption before timing. Timed pairs alternate ABBA and BAAB only in single-worker mode. Small-call iterations are calibrated against the observed timer resolution. All samples are retained. Ratios are GTG elapsed time divided by comparator elapsed time.",
	}
	defer func() {
		report.Finished = time.Now().UTC()
	}()
	if workers > 1 {
		options.NoPython = true
		report.Warnings = append(report.Warnings, "parallel worker mode records GTG-only samples; cross-runtime comparator pairs are disabled")
	}
	if options.Resume != nil {
		if options.Resume.Schema != report.Schema || options.Resume.Mode != report.Mode || options.Resume.Workers != report.Workers {
			return report, fmt.Errorf("resume report schema, mode, or worker count does not match this run")
		}
		report.Started = options.Resume.Started
		report.Samples = append(report.Samples, options.Resume.Samples...)
		report.Resumed = true
		report.ResumeCount = options.Resume.ResumeCount + 1
		report.Warnings = append(report.Warnings, "resumed run retained prior raw samples; new samples use the current environment metadata")
	}
	if err := checkpoint(ctx, options, report); err != nil {
		return report, err
	}
	if options.Progress != nil {
		options.Progress("Qualifying GTG AES-256-IGE")
	}
	key := pattern(32, 11)
	iv := pattern(32, 17)
	plain := pattern(64, 31)
	instance, err := ige.New(key)
	if err != nil {
		return report, err
	}
	ciphertext := make([]byte, len(plain))
	opened := make([]byte, len(plain))
	err = instance.Encrypt(ciphertext, plain, iv)
	if err == nil {
		err = instance.Decrypt(opened, ciphertext, iv)
	}
	gtgQualified := err == nil && bytes.Equal(opened, plain)
	if !gtgQualified && err == nil {
		err = fmt.Errorf("GTG reverse decryption mismatch")
	}
	report.Qualifications = append(report.Qualifications, Qualification{Library: "GTG", Available: err == nil, Qualified: gtgQualified, Version: "local Go module", Reason: errorText(err)})
	if err != nil {
		return report, err
	}
	defer instance.Destroy()
	var harness *pythonHarness
	qualified := map[string]bool{}
	qualificationIndex := map[string]int{}
	if options.NoPython {
		report.Warnings = append(report.Warnings, "Python comparators were disabled; this is a GTG-only run")
	} else {
		harness, err = preparePython(options.Progress)
		if err != nil {
			report.Warnings = append(report.Warnings, "Python comparators unavailable: "+err.Error()+". Continuing with GTG-only measurements.")
		} else {
			defer harness.close()
			for _, library := range libraries {
				if options.Progress != nil {
					options.Progress("Qualifying " + library)
				}
				response, requestErr := harness.qualify(library)
				qualification := Qualification{Library: library, Available: response.Available, Qualified: false, Version: response.Version, Reason: response.Reason}
				if requestErr != nil {
					qualification.Reason = requestErr.Error()
				} else if response.OK {
					candidate, decodeErr := decodeEncoded(response.Ciphertext)
					openedCandidate, plainErr := decodeEncoded(response.Plaintext)
					if decodeErr != nil {
						qualification.Reason = decodeErr.Error()
					} else if plainErr != nil {
						qualification.Reason = plainErr.Error()
					} else if !bytes.Equal(candidate, ciphertext) {
						qualification.Reason = "ciphertext differs from GTG qualification vector"
					} else if !bytes.Equal(openedCandidate, plain) {
						qualification.Reason = "reverse decryption differs from qualification plaintext"
					} else {
						qualification.Qualified = true
						qualified[library] = true
					}
				}
				qualificationIndex[library] = len(report.Qualifications)
				report.Qualifications = append(report.Qualifications, qualification)
			}
		}
	}
	if harness != nil {
		for _, library := range libraries {
			if !qualified[library] {
				continue
			}
			for _, size := range sizesFor(report.Mode) {
				reason := qualifySize(instance, harness, library, key, iv, size)
				if reason == "" {
					continue
				}
				qualified[library] = false
				index := qualificationIndex[library]
				report.Qualifications[index].Qualified = false
				report.Qualifications[index].Reason = reason
				break
			}
		}
	}
	for _, warning := range report.Environment.Warnings {
		report.Warnings = append(report.Warnings, warning)
	}
	sizes := sizesFor(report.Mode)
	caches := []CacheMode{Hot}
	if report.Mode == Full {
		caches = append(caches, Cold)
	}
	rounds := 4
	if report.Mode == Full {
		rounds = 8
	}
	for _, size := range sizes {
		iterations := iterationsFor(size, report.Mode)
		if size <= 64 {
			iterations, err = calibratedIGEIterations(instance, key, iv, size, report.Mode, resolution, workers)
			if err != nil {
				return report, err
			}
		}
		for _, direction := range []Direction{Encrypt, Decrypt} {
			for _, cache := range caches {
				if options.Progress != nil {
					options.Progress(fmt.Sprintf("%s %s %s: %s", humanBytes(size), direction, cache, "GTG baseline"))
				}
				for round := 0; round < rounds; round++ {
					if hasSample(report.Samples, Sample{Library: "GTG", Operation: "AES-256-IGE", Direction: direction, Cache: cache, Size: size, Round: round, Position: 0, Order: "baseline"}) {
						continue
					}
					elapsed, checksum, runErr := measureGTG(instance, key, iv, size, direction, iterations, cache == Cold, round, workers)
					if runErr != nil {
						return report, runErr
					}
					sample := Sample{Library: "GTG", Operation: "AES-256-IGE", Direction: direction, Cache: cache, Size: size, Round: round, Position: 0, Order: "baseline", ElapsedNS: elapsed, Iterations: iterations * workers, Workers: workers, Checksum: checksum, At: time.Now().UTC()}
					if !hasSample(report.Samples, sample) {
						report.Samples = append(report.Samples, sample)
						if err := checkpoint(ctx, options, report); err != nil {
							return report, err
						}
					}
				}
				for _, library := range libraries {
					if !qualified[library] || harness == nil {
						continue
					}
					if options.Progress != nil {
						options.Progress(fmt.Sprintf("%s %s %s: paired GTG / %s", humanBytes(size), direction, cache, library))
					}
					for round := 0; round < rounds; round++ {
						order := []string{"GTG", library, library, "GTG"}
						orderName := "ABBA"
						if round%2 == 1 {
							order = []string{library, "GTG", "GTG", library}
							orderName = "BAAB"
						}
						for position, side := range order {
							if hasSample(report.Samples, Sample{Library: side, Operation: "AES-256-IGE", Direction: direction, Cache: cache, Size: size, Round: round, Position: position + 1, Order: orderName}) {
								continue
							}
							var elapsed int64
							var checksum uint64
							var runErr error
							if side == "GTG" {
								elapsed, checksum, runErr = measureGTG(instance, key, iv, size, direction, iterations, cache == Cold, round*4+position, workers)
							} else {
								response, requestErr := harness.measure(library, direction, size, iterations, cache == Cold)
								if requestErr != nil {
									runErr = requestErr
								} else if !response.OK {
									runErr = fmt.Errorf("%s measurement: %s", library, response.Reason)
								} else {
									elapsed, checksum = response.ElapsedNS, response.Checksum
								}
							}
							if runErr != nil {
								return report, runErr
							}
							report.Samples = append(report.Samples, Sample{Library: side, Operation: "AES-256-IGE", Direction: direction, Cache: cache, Size: size, Round: round, Position: position + 1, Order: orderName, ElapsedNS: elapsed, Iterations: iterations * workers, Workers: workers, Checksum: checksum, At: time.Now().UTC()})
							if err := checkpoint(ctx, options, report); err != nil {
								return report, err
							}
						}
					}
					report.Pairs = append(report.Pairs, analyzePair(report.Samples, library, direction, cache, size, rounds))
				}
			}
		}
	}
	if err := appendMTProtoSamples(ctx, &report, options, resolution); err != nil {
		return report, err
	}
	report.Finished = time.Now().UTC()
	return report, nil
}

func calibratedIGEIterations(instance *ige.IGE, key, iv []byte, size int, mode Mode, resolution time.Duration, workers int) (int, error) {
	return calibratedIterations(mode, resolution, func(iterations int) (int64, error) {
		elapsed, _, err := measureGTG(instance, key, iv, size, Encrypt, iterations, false, 0, workers)
		return elapsed, err
	})
}

func appendMTProtoSamples(ctx context.Context, report *Report, options Options, resolution time.Duration) error {
	if report.Workers > 1 {
		report.Warnings = append(report.Warnings, "parallel worker mode currently measures AES-IGE only; MTProto Seal/Open rows require single-worker mode")
		return nil
	}
	key := pattern(256, 47)
	for _, size := range sizesFor(report.Mode) {
		if options.Progress != nil {
			options.Progress(fmt.Sprintf("%s MTProto Seal/Open", humanBytes(size)))
		}
		iterations := iterationsFor(size, report.Mode)
		if size <= 64 {
			var err error
			iterations, err = calibratedMTProtoIterations(key, size, report.Mode, resolution)
			if err != nil {
				return err
			}
		}
		rounds := 4
		if report.Mode == Full {
			rounds = 8
		}
		for _, direction := range []Direction{Encrypt, Decrypt} {
			for round := 0; round < rounds; round++ {
				operation := "MTProto Seal"
				if direction == Decrypt {
					operation = "MTProto Open"
				}
				if hasSample(report.Samples, Sample{Library: "GTG", Operation: operation, Direction: direction, Cache: Hot, Size: size, Round: round, Position: 0, Order: "mtproto"}) {
					continue
				}
				elapsed, checksum, err := measureMTProto(key, size, direction, iterations)
				if err != nil {
					return err
				}
				report.Samples = append(report.Samples, Sample{Library: "GTG", Operation: operation, Direction: direction, Cache: Hot, Size: size, Round: round, Position: 0, Order: "mtproto", ElapsedNS: elapsed, Iterations: iterations, Workers: report.Workers, Checksum: checksum, At: time.Now().UTC()})
				if err := checkpoint(ctx, options, *report); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func calibratedMTProtoIterations(key []byte, size int, mode Mode, resolution time.Duration) (int, error) {
	return calibratedIterations(mode, resolution, func(iterations int) (int64, error) {
		elapsed, _, err := measureMTProto(key, size, Encrypt, iterations)
		return elapsed, err
	})
}

func measureMTProto(key []byte, size int, direction Direction, iterations int) (int64, uint64, error) {
	box, err := mtproto.New(key)
	if err != nil {
		return 0, 0, err
	}
	defer box.Destroy()
	plaintext := pattern(size, 73)
	if remainder := len(plaintext) & 15; remainder != 0 {
		plaintext = append(plaintext, make([]byte, 16-remainder)...)
	}
	ciphertext := make([]byte, len(plaintext))
	messageKey, ciphertext, err := box.Seal(ciphertext[:0], plaintext, mtproto.ClientToServer)
	if err != nil {
		return 0, 0, err
	}
	start := time.Now()
	var checksum uint64
	for index := 0; index < iterations; index++ {
		if direction == Encrypt {
			messageKey, ciphertext, err = box.Seal(ciphertext[:0], plaintext, mtproto.ClientToServer)
			if err == nil {
				checksum ^= uint64(ciphertext[len(ciphertext)-1]) | uint64(messageKey[0])<<8
			}
		} else {
			opened, openErr := box.Open(plaintext[:0], messageKey[:], ciphertext, mtproto.ClientToServer)
			err = openErr
			if err == nil {
				checksum ^= uint64(opened[0]) | uint64(opened[len(opened)-1])<<8
			}
		}
		if err != nil {
			return 0, 0, err
		}
	}
	return time.Since(start).Nanoseconds(), checksum, nil
}

func qualifySize(instance *ige.IGE, harness *pythonHarness, library string, key, iv []byte, size int) string {
	plain := pattern(size, 31)
	expected := make([]byte, size)
	opened := make([]byte, size)
	if err := instance.Encrypt(expected, plain, iv); err != nil {
		return err.Error()
	}
	if err := instance.Decrypt(opened, expected, iv); err != nil {
		return err.Error()
	}
	if !bytes.Equal(opened, plain) {
		return "GTG reverse qualification mismatch"
	}
	response, err := harness.qualifySize(library, size)
	if err != nil {
		return err.Error()
	}
	if !response.OK {
		return response.Reason
	}
	candidate, err := decodeEncoded(response.Ciphertext)
	if err != nil {
		return err.Error()
	}
	candidatePlain, err := decodeEncoded(response.Plaintext)
	if err != nil {
		return err.Error()
	}
	if !bytes.Equal(candidate, expected) {
		return fmt.Sprintf("ciphertext differs from GTG at %d bytes", size)
	}
	if !bytes.Equal(candidatePlain, plain) {
		return fmt.Sprintf("reverse decryption differs from GTG at %d bytes", size)
	}
	return ""
}

func measureGTG(instance *ige.IGE, key, iv []byte, size int, direction Direction, iterations int, cold bool, slot, workers int) (int64, uint64, error) {
	if workers <= 1 {
		return measureGTGOne(instance, key, iv, size, direction, iterations, cold, slot)
	}
	return measureGTGParallel(key, iv, size, direction, iterations, cold, slot, workers)
}

func measureGTGOne(instance *ige.IGE, key, iv []byte, size int, direction Direction, iterations int, cold bool, slot int) (int64, uint64, error) {
	plain := pattern(size, 31)
	input := plain
	if direction == Decrypt {
		input = make([]byte, size)
		if err := instance.Encrypt(input, plain, iv); err != nil {
			return 0, 0, err
		}
	}
	if cold {
		evict(size, slot)
	}
	output := make([]byte, size)
	start := time.Now()
	var checksum uint64
	for index := 0; index < iterations; index++ {
		var err error
		if direction == Encrypt {
			err = instance.Encrypt(output, input, iv)
		} else {
			err = instance.Decrypt(output, input, iv)
		}
		if err != nil {
			return 0, 0, err
		}
		checksum ^= uint64(output[0])<<8 | uint64(output[len(output)-1])
	}
	return time.Since(start).Nanoseconds(), checksum, nil
}

func measureGTGParallel(key, iv []byte, size int, direction Direction, iterations int, cold bool, slot, workers int) (int64, uint64, error) {
	plain := pattern(size, 31)
	input := plain
	if direction == Decrypt {
		seed, err := ige.New(key)
		if err != nil {
			return 0, 0, err
		}
		input = make([]byte, size)
		err = seed.Encrypt(input, plain, iv)
		seed.Destroy()
		if err != nil {
			return 0, 0, err
		}
	}
	if cold {
		evict(size, slot)
	}
	instances := make([]*ige.IGE, workers)
	outputs := make([][]byte, workers)
	for index := range instances {
		instance, err := ige.New(key)
		if err != nil {
			for _, opened := range instances {
				if opened != nil {
					opened.Destroy()
				}
			}
			return 0, 0, err
		}
		instances[index] = instance
		outputs[index] = make([]byte, size)
	}
	defer func() {
		for _, instance := range instances {
			instance.Destroy()
		}
	}()
	start := make(chan struct{})
	errs := make(chan error, workers)
	checksums := make(chan uint64, workers)
	var group sync.WaitGroup
	group.Add(workers)
	for worker := 0; worker < workers; worker++ {
		worker := worker
		go func() {
			defer group.Done()
			<-start
			var checksum uint64
			for index := 0; index < iterations; index++ {
				var err error
				if direction == Encrypt {
					err = instances[worker].Encrypt(outputs[worker], input, iv)
				} else {
					err = instances[worker].Decrypt(outputs[worker], input, iv)
				}
				if err != nil {
					errs <- err
					return
				}
				checksum ^= uint64(outputs[worker][0])<<8 | uint64(outputs[worker][len(outputs[worker])-1])
			}
			checksums <- checksum
		}()
	}
	started := time.Now()
	close(start)
	group.Wait()
	elapsed := time.Since(started).Nanoseconds()
	close(errs)
	for err := range errs {
		if err != nil {
			return 0, 0, err
		}
	}
	close(checksums)
	var checksum uint64
	for value := range checksums {
		checksum ^= value
	}
	return elapsed, checksum, nil
}

func evict(size, slot int) uint64 {
	length := 8 * 1024 * 1024
	if size > length/2 {
		length = size * 2
	}
	if length > 128*1024*1024 {
		length = 128 * 1024 * 1024
	}
	data := make([]byte, length)
	var value uint64
	for index := slot % 64; index < len(data); index += 64 {
		data[index] = byte(index*17 + 3)
		value ^= uint64(data[index]) + uint64(index)
	}
	return value
}

func pattern(length, salt int) []byte {
	data := make([]byte, length)
	for index := range data {
		data[index] = byte(index*131 + salt)
	}
	return data
}

func sizesFor(mode Mode) []int {
	if mode == Full {
		return []int{16, 32, 64, 128, 256, 512, 1024, 4 * 1024, 16 * 1024, 64 * 1024, 256 * 1024, 1024 * 1024, 4 * 1024 * 1024, 16 * 1024 * 1024, 64 * 1024 * 1024}
	}
	return []int{16, 64, 1024, 16 * 1024, 256 * 1024, 1024 * 1024}
}

func iterationsFor(size int, mode Mode) int {
	target := 2 * 1024 * 1024
	if mode == Full {
		target = 16 * 1024 * 1024
	}
	iterations := target / size
	if iterations < 1 {
		iterations = 1
	}
	limit := 1024
	if mode == Full {
		limit = 16384
	}
	if iterations > limit {
		iterations = limit
	}
	return iterations
}

func analyzePair(samples []Sample, library string, direction Direction, cache CacheMode, size, rounds int) PairResult {
	result := PairResult{Library: library, Direction: direction, Cache: cache, Size: size}
	for round := 0; round < rounds; round++ {
		var gtgTotal int64
		var otherTotal int64
		var gtgCount int
		var otherCount int
		for _, sample := range samples {
			if sample.Direction != direction || sample.Cache != cache || sample.Size != size || sample.Round != round || sample.Order == "baseline" {
				continue
			}
			if sample.Library == "GTG" {
				gtgTotal += sample.ElapsedNS
				gtgCount++
			}
			if sample.Library == library {
				otherTotal += sample.ElapsedNS
				otherCount++
			}
		}
		if gtgCount > 0 && otherCount > 0 && otherTotal > 0 {
			result.Ratios = append(result.Ratios, float64(gtgTotal)/float64(gtgCount)/(float64(otherTotal)/float64(otherCount)))
		}
	}
	result.GeometricMean = geometricMean(result.Ratios)
	result.CILow, result.CIHigh = bootstrapCI(result.Ratios)
	if result.CIHigh < 0.97 {
		result.Verdict = "faster"
	} else if result.CILow > 1.03 {
		result.Verdict = "slower"
	} else {
		result.Verdict = "parity"
	}
	result.Noise = noiseFlag(result.Ratios)
	result.Drift = driftFlag(result.Ratios)
	return result
}

func geometricMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var total float64
	for _, value := range values {
		if value <= 0 {
			return 0
		}
		total += math.Log(value)
	}
	return math.Exp(total / float64(len(values)))
}

func bootstrapCI(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	random := rand.New(rand.NewPCG(0x474754, uint64(len(values))*0x9e3779b97f4a7c15))
	distribution := make([]float64, 4000)
	for index := range distribution {
		draw := make([]float64, len(values))
		for sample := range draw {
			draw[sample] = values[random.IntN(len(values))]
		}
		distribution[index] = geometricMean(draw)
	}
	sort.Float64s(distribution)
	return distribution[99], distribution[3899]
}

func noiseFlag(values []float64) string {
	if len(values) < 2 {
		return "insufficient paired rounds"
	}
	mean := 0.0
	for _, value := range values {
		mean += value
	}
	mean /= float64(len(values))
	variance := 0.0
	for _, value := range values {
		variance += (value - mean) * (value - mean)
	}
	stddev := math.Sqrt(variance / float64(len(values)-1))
	if mean != 0 && stddev/math.Abs(mean) > 0.15 {
		return "paired ratio coefficient of variation exceeds 15%"
	}
	return ""
}

func driftFlag(values []float64) string {
	if len(values) < 4 {
		return ""
	}
	half := len(values) / 2
	first := geometricMean(values[:half])
	last := geometricMean(values[half:])
	if first > 0 && math.Abs(last/first-1) > 0.10 {
		return "first-versus-last paired ratio changed by more than 10%"
	}
	return ""
}

func hasSample(samples []Sample, candidate Sample) bool {
	for _, sample := range samples {
		if sample.Library == candidate.Library && sample.Operation == candidate.Operation && sample.Direction == candidate.Direction && sample.Cache == candidate.Cache && sample.Size == candidate.Size && sample.Round == candidate.Round && sample.Position == candidate.Position && sample.Order == candidate.Order {
			return true
		}
	}
	return false
}

func checkpoint(ctx context.Context, options Options, report Report) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if options.Checkpoint != nil {
		return options.Checkpoint(report)
	}
	return nil
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func humanBytes(value int) string {
	if value < 1024 {
		return fmt.Sprintf("%d B", value)
	}
	units := []string{"KiB", "MiB", "GiB"}
	v := float64(value)
	for _, unit := range units {
		v /= 1024
		if v < 1024 || unit == "GiB" {
			return fmt.Sprintf("%.0f %s", v, unit)
		}
	}
	return fmt.Sprintf("%d B", value)
}
