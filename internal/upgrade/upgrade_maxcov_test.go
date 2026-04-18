package upgrade

import (
	"strings"
	"testing"
)

func TestTrvlDir(t *testing.T) {
	dir, err := trvlDir()
	if err != nil {
		t.Fatalf("trvlDir() error: %v", err)
	}
	if dir == "" {
		t.Fatal("trvlDir() should not be empty")
	}
	if !strings.HasSuffix(dir, ".trvl") {
		t.Errorf("trvlDir() = %q, should end with .trvl", dir)
	}
}
