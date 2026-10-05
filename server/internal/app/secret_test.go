package app

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Synthetic per-test key; never uses the user's OS keychain or real credential data.
func useTestKey(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.key")
	if err := generateKeyFile(path); err != nil {
		t.Fatal(err)
	}
	t.Setenv(keyFileEnv, path)
	return path
}

func TestPortableEncryption(t *testing.T) {
	path := useTestKey(t)
	plain := []byte("synthetic-credential-한글")
	one, err := protect(plain)
	if err != nil {
		t.Fatal(err)
	}
	two, err := protect(plain)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(one, two) || bytes.Contains(one, plain) {
		t.Fatal("ciphertext must use fresh random nonces and hide plaintext")
	}
	got, err := unprotect(one)
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatal("decryption failed", err)
	}
	for _, damaged := range [][]byte{one[:len(secretEnvelope)], append(append([]byte(nil), one[:len(one)-1]...), one[len(one)-1]^1)} {
		if _, err := unprotect(damaged); err == nil {
			t.Fatal("truncated or modified ciphertext accepted")
		}
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := generateKeyFile(path); err == nil {
		t.Fatal("keygen overwrote an existing key")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("key changed")
	}
	other := filepath.Join(t.TempDir(), "other.key")
	if err := generateKeyFile(other); err != nil {
		t.Fatal(err)
	}
	t.Setenv(keyFileEnv, other)
	if _, err := unprotect(one); err == nil {
		t.Fatal("wrong key accepted")
	}
	t.Setenv(keyFileEnv, filepath.Join(t.TempDir(), "missing.key"))
	if _, err := protect(plain); err == nil {
		t.Fatal("missing explicit key silently fell back")
	}
	if _, err := unprotect(one); err == nil {
		t.Fatal("missing key accepted")
	}
}

func TestKeyFileValidation(t *testing.T) {
	path := useTestKey(t)
	if _, err := readKeyFile("relative.key"); err == nil {
		t.Fatal("relative key accepted")
	}
	if _, err := readKeyFile(filepath.Dir(path)); err == nil {
		t.Fatal("directory accepted as key")
	}
	if err := os.WriteFile(path, []byte("bad"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readKeyFile(path); err == nil {
		t.Fatal("short key accepted")
	}
	if runtime.GOOS != "windows" {
		if err := os.WriteFile(path, make([]byte, 32), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := readKeyFile(path); err == nil {
			t.Fatal("world-readable key accepted")
		}
	}
}
