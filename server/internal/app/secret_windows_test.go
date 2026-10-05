package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
)

type cacheBytes struct{ b []byte }

func (c *cacheBytes) Marshal() ([]byte, error) { return c.b, nil }
func (c *cacheBytes) Unmarshal(b []byte) error { c.b = b; return nil }
func TestEncryptedCacheRestart(t *testing.T) {
	t.Setenv(keyFileEnv, "") // This regression specifically verifies native Windows DPAPI.
	path := filepath.Join(t.TempDir(), "cache.dpapi")
	d := diskCache{path}
	source := &cacheBytes{[]byte("test-only-secret-not-a-real-token")}
	if e := d.Export(t.Context(), source, cache.ExportHints{}); e != nil {
		t.Fatal(e)
	}
	data, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Contains(data, source.b) {
		t.Fatal("plaintext credential cache")
	}
	loaded := &cacheBytes{}
	if e = (diskCache{path}).Replace(t.Context(), loaded, cache.ReplaceHints{}); e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(loaded.b, source.b) {
		t.Fatal("restart cache mismatch")
	}
	if e = d.Export(t.Context(), source, cache.ExportHints{}); e != nil {
		t.Fatal("cache replacement", e)
	}
	files, e := os.ReadDir(filepath.Dir(path))
	if e != nil || len(files) != 1 {
		t.Fatalf("temporary cache file leaked: %v %v", files, e)
	}
}
