//go:build linux || darwin

package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestOSKeyringContract(t *testing.T) {
	t.Setenv(keyFileEnv, "")
	t.Setenv(dataDirEnv, t.TempDir())
	keyring.MockInit()
	plain := []byte("synthetic-keyring-credential")
	data, err := protect(plain)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := unprotect(data)
	if err != nil || !bytes.Equal(plain, loaded) {
		t.Fatal("keyring round trip", err)
	}
	// A fresh keyring must not replace a missing key when the data still exists.
	if err := os.WriteFile(filepath.Join(os.Getenv(dataDirEnv), "sample.secret"), data, 0600); err != nil {
		t.Fatal(err)
	}
	keyring.MockInit()
	if _, err := unprotect(data); err == nil {
		t.Fatal("lost key should not recreate during read")
	}
	if _, err := protect(plain); err == nil {
		t.Fatal("lost key replaced with encrypted files present")
	}
	keyring.MockInitWithError(errors.New("test keyring locked"))
	if _, err := protect(plain); err == nil {
		t.Fatal("locked keyring silently fell back")
	}
}
