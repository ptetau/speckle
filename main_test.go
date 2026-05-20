package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// testBinary is the speckle binary built once for all acceptance tests.
// CLI acceptance tests exec it; HTTP acceptance tests start it as a
// subprocess and drive it over HTTP via the lockfile-discovered port.
var testBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "speckle-bin-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "TestMain: mktemp:", err)
		os.Exit(2)
	}
	bin := filepath.Join(dir, "speckle")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintln(os.Stderr, "TestMain: go build failed:\n", string(out))
		os.RemoveAll(dir)
		os.Exit(2)
	}
	testBinary = bin
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
