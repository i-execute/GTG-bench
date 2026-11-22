package app

import (
	"bytes"
	"strings"
	"testing"
)

func TestMenuReadsSharedInput(t *testing.T) {
	input := strings.NewReader("3\nq\n")
	var output bytes.Buffer
	_, err := Run(Config{Plain: true, Interactive: true}, input, &output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Environment") {
		t.Fatal(output.String())
	}
}
