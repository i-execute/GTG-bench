package bench

import (
	"testing"

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
