package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNoticeCollection(t *testing.T) {
	name := "notices"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(t.TempDir(), name)
	if out, err := exec.CommandContext(t.Context(), "go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build fixture: %v %s", err, out)
	}
	got, err := collect(binary)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "LICENSE"))
	if err != nil || !bytes.Contains(got, want) {
		t.Fatal("Go license original bytes were not retained")
	}
	if bytes.Contains(got, []byte(binary)) {
		t.Fatal("private build path leaked into notices")
	}
	if _, err := collect(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing binary accepted")
	}
}

func TestLegalFilenames(t *testing.T) {
	for _, name := range []string{"LICENSE", "LICENSE-MIT", "LICENSE.txt", "LICENCE", "NOTICE", "NOTICE.md", "COPYING", "license"} {
		if !legalName(name) {
			t.Errorf("missed notice %s", name)
		}
	}
	for _, name := range []string{"main.go", "license_test.go", "README.md"} {
		if legalName(name) {
			t.Errorf("included unrelated file %s", name)
		}
	}
}
