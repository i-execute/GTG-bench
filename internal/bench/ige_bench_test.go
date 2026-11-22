package bench

import (
	"testing"

	"github.com/i-execute/GTG/ige"
)

func BenchmarkIGEEncrypt1MiB(b *testing.B) {
	key := pattern(32, 11)
	iv := pattern(32, 17)
	input := pattern(1024*1024, 31)
	output := make([]byte, len(input))
	instance, err := ige.New(key)
	if err != nil {
		b.Fatal(err)
	}
	defer instance.Destroy()
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if err := instance.Encrypt(output, input, iv); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIGEDecrypt1MiB(b *testing.B) {
	key := pattern(32, 11)
	iv := pattern(32, 17)
	plain := pattern(1024*1024, 31)
	input := make([]byte, len(plain))
	output := make([]byte, len(plain))
	instance, err := ige.New(key)
	if err != nil {
		b.Fatal(err)
	}
	defer instance.Destroy()
	if err := instance.Encrypt(input, plain, iv); err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if err := instance.Decrypt(output, input, iv); err != nil {
			b.Fatal(err)
		}
	}
}
