//go:build linux || darwin

package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

const liveKeyringOptIn = "TODO_CONNECT_TEST_OS_KEYRING"
const liveKeyringStage = "TODO_CONNECT_KEYRING_TEST_STAGE"
const liveKeyringPayload = "synthetic-live-keyring-credential"
const liveKeyringFile = "synthetic.secret"

// Run this test alone: other unit tests deliberately replace the keyring backend
// with a fake. This opt-in test writes only an entry scoped to its own temp path.
func TestLiveOSKeyring(t *testing.T) {
	if os.Getenv(liveKeyringOptIn) != "1" {
		t.Skip("set TODO_CONNECT_TEST_OS_KEYRING=1 and run only TestLiveOSKeyring")
	}
	dir := t.TempDir()
	t.Setenv(dataDirEnv, dir)
	t.Setenv(keyFileEnv, "")
	digest := sha256.Sum256([]byte(dir))
	account := hex.EncodeToString(digest[:])
	t.Cleanup(func() {
		if err := keyring.Delete(keyringService, account); err != nil && !errors.Is(err, keyring.ErrNotFound) {
			t.Errorf("cleanup of test-owned keyring entry: %v", err)
		}
	})
	for _, stage := range []string{"write", "read"} {
		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestLiveOSKeyringWorker$", "-test.timeout=30s")
		cmd.Env = append(os.Environ(), liveKeyringStage+"="+stage)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("real keyring subprocess %s: %v %s", stage, err, out)
		}
	}
	encoded, err := keyring.Get(keyringService, account)
	if err != nil {
		t.Fatal("OS key was not retained", err)
	}
	key, err := hex.DecodeString(encoded)
	if err != nil || len(key) != 32 {
		t.Fatal("OS keyring did not retain a 256-bit key")
	}
	if err := keyring.Delete(keyringService, account); err != nil {
		t.Fatal("delete test-owned key", err)
	}
	if _, err := keyring.Get(keyringService, account); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatal("test-owned key still present or cleanup unverifiable", err)
	}
	t.Log("Real OS keyring encrypted data, a new process decrypted it, and the test-owned key was removed")
}

func TestLiveOSKeyringWorker(t *testing.T) {
	stage := os.Getenv(liveKeyringStage)
	if os.Getenv(liveKeyringOptIn) != "1" || stage == "" {
		t.Skip("subprocess helper for TestLiveOSKeyring")
	}
	path := filepath.Join(os.Getenv(dataDirEnv), liveKeyringFile)
	switch stage {
	case "write":
		data, err := protect([]byte(liveKeyringPayload))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(data, []byte(secretEnvelope)) || bytes.Contains(data, []byte(liveKeyringPayload)) {
			t.Fatal("credential was not encrypted")
		}
		if err := writePrivate(path, data); err != nil {
			t.Fatal(err)
		}
	case "read":
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		plain, err := unprotect(data)
		if err != nil || string(plain) != liveKeyringPayload {
			t.Fatal("new process could not decrypt using the persisted OS key", err)
		}
	default:
		t.Fatal("unknown keyring test stage")
	}
}
