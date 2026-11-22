package bench

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/i-execute/GTG/ige"
)

func TestQualificationVector(t *testing.T) {
	key := pattern(32, 11)
	iv := pattern(32, 17)
	plain := pattern(64, 31)
	instance, err := ige.New(key)
	if err != nil {
		t.Fatal(err)
	}
	defer instance.Destroy()
	ciphertext := make([]byte, len(plain))
	opened := make([]byte, len(plain))
	if err := instance.Encrypt(ciphertext, plain, iv); err != nil {
		t.Fatal(err)
	}
	if err := instance.Decrypt(opened, ciphertext, iv); err != nil {
		t.Fatal(err)
	}
	for index := range plain {
		if opened[index] != plain[index] {
			t.Fatalf("mismatch at %d", index)
		}
	}
}

func TestParallelIGE(t *testing.T) {
	key := pattern(32, 11)
	iv := pattern(32, 17)
	instance, err := ige.New(key)
	if err != nil {
		t.Fatal(err)
	}
	defer instance.Destroy()
	elapsed, _, err := measureGTG(instance, key, iv, 64, Encrypt, 8, false, 0, 2)
	if err != nil || elapsed <= 0 {
		t.Fatalf("elapsed=%d err=%v", elapsed, err)
	}
}

func TestMTProtoSealOpen(t *testing.T) {
	key := pattern(256, 47)
	for _, direction := range []Direction{Encrypt, Decrypt} {
		elapsed, checksum, err := measureMTProto(key, 64, direction, 1)
		if err != nil || elapsed <= 0 || checksum == 0 {
			t.Fatalf("%s elapsed=%d checksum=%d err=%v", direction, elapsed, checksum, err)
		}
	}
}

func TestCalibration(t *testing.T) {
	calls := 0
	iterations, err := calibratedIterations(Fast, time.Nanosecond, func(value int) (int64, error) {
		calls++
		return int64(value) * int64(time.Microsecond), nil
	})
	if err != nil || iterations < 1000 || calls < 2 {
		t.Fatalf("iterations=%d calls=%d err=%v", iterations, calls, err)
	}
}

func TestRunResume(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	partial, err := Run(Options{Mode: Fast, NoPython: true, Context: ctx, Checkpoint: func(report Report) error {
		if len(report.Samples) > 0 {
			cancel()
		}
		return nil
	}})
	if !errors.Is(err, context.Canceled) || len(partial.Samples) == 0 {
		t.Fatalf("samples=%d err=%v", len(partial.Samples), err)
	}
	resumed, err := Run(Options{Mode: Fast, NoPython: true, Resume: &partial})
	if err != nil || !resumed.Resumed || len(resumed.Samples) <= len(partial.Samples) {
		t.Fatalf("partial=%d resumed=%d resumed-flag=%t err=%v", len(partial.Samples), len(resumed.Samples), resumed.Resumed, err)
	}
	for index, sample := range resumed.Samples {
		for prior := 0; prior < index; prior++ {
			other := resumed.Samples[prior]
			if sample.Library == other.Library && sample.Operation == other.Operation && sample.Direction == other.Direction && sample.Cache == other.Cache && sample.Size == other.Size && sample.Round == other.Round && sample.Position == other.Position && sample.Order == other.Order {
				t.Fatal("duplicate resumed sample")
			}
		}
	}
}

func TestVerdictBoundaries(t *testing.T) {
	if result := analyzePair(pairSamples("Other", []int64{80, 80, 80, 80}, []int64{100, 100, 100, 100}), "Other", Encrypt, Hot, 16, 4); result.Verdict != "faster" {
		t.Fatalf("got %s", result.Verdict)
	}
	if result := analyzePair(pairSamples("Other", []int64{120, 120, 120, 120}, []int64{100, 100, 100, 100}), "Other", Encrypt, Hot, 16, 4); result.Verdict != "slower" {
		t.Fatalf("got %s", result.Verdict)
	}
	if result := analyzePair(pairSamples("Other", []int64{100, 101, 99, 100}, []int64{100, 100, 100, 100}), "Other", Encrypt, Hot, 16, 4); result.Verdict != "parity" {
		t.Fatalf("got %s", result.Verdict)
	}
}

func pairSamples(library string, gtg []int64, other []int64) []Sample {
	output := make([]Sample, 0, len(gtg)*4)
	for round := range gtg {
		output = append(output, Sample{Library: "GTG", Direction: Encrypt, Cache: Hot, Size: 16, Round: round, Position: 1, Order: "ABBA", ElapsedNS: gtg[round]})
		output = append(output, Sample{Library: library, Direction: Encrypt, Cache: Hot, Size: 16, Round: round, Position: 2, Order: "ABBA", ElapsedNS: other[round]})
		output = append(output, Sample{Library: library, Direction: Encrypt, Cache: Hot, Size: 16, Round: round, Position: 3, Order: "ABBA", ElapsedNS: other[round]})
		output = append(output, Sample{Library: "GTG", Direction: Encrypt, Cache: Hot, Size: 16, Round: round, Position: 4, Order: "ABBA", ElapsedNS: gtg[round]})
	}
	return output
}

func TestSizes(t *testing.T) {
	full := sizesFor(Full)
	if full[0] != 16 || full[len(full)-1] != 64*1024*1024 {
		t.Fatal(full)
	}
}
