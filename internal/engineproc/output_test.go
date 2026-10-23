package engineproc

import (
	"bytes"
	"testing"
)

func TestScriptValidationOutputBounded(t *testing.T) {
	output := newBoundedOutput(64 * 1024)
	text := bytes.Repeat([]byte("x"), 64*1024+1)
	if n, err := output.Write(text); err != nil || n != len(text) {
		t.Fatalf("write = %d, %v", n, err)
	}
	if _, err := output.Write(text); err != nil {
		t.Fatal(err)
	}
	if len(output.text) != 64*1024 || !output.truncated {
		t.Fatalf("length=%d truncated=%v", len(output.text), output.truncated)
	}
}
