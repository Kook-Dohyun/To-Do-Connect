package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeTestAccount(t *testing.T, path, homeID string) {
	t.Helper()
	// Synthetic MSAL cache: account metadata only, no real token and no network login.
	b, err := json.Marshal(object{"Account": object{homeID: object{"home_account_id": homeID, "environment": "login.microsoftonline.com", "realm": "consumers", "username": homeID + "@example.invalid"}}})
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := protect(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := writePrivate(path, encrypted); err != nil {
		t.Fatal(err)
	}
	disk, err := os.ReadFile(path)
	if err != nil || bytes.Contains(disk, []byte(homeID)) {
		t.Fatal("account cache not encrypted", err)
	}
}

func TestConnectionsEncryptedIsolationAndImport(t *testing.T) {
	useTestKey(t)
	c := &connections{dir: t.TempDir()}
	for _, id := range []string{"one", "two", "imported"} {
		addTestConnection(t, c, id)
	}
	writeTestAccount(t, c.path("one", ".secret"), "account-one")
	writeTestAccount(t, c.path("two", ".secret"), "account-two")
	for range 2 { // New store and MSAL instances model restart/reconnection.
		reopened := &connections{dir: c.dir}
		for _, id := range []string{"one", "two"} {
			_, err := reopened.withMicrosoft(t.Context(), id, func(a *auth, _ *graph) (object, error) {
				accounts, err := a.client.Accounts(t.Context())
				if err != nil || len(accounts) != 1 || accounts[0].HomeAccountID != "account-"+id {
					t.Fatalf("wrong account routed for %s: %v", id, err)
				}
				// Existing credentials must not automatically open a browser again.
				if err := a.login(t.Context(), false, false); err == nil {
					t.Fatal("routine login should not reauthorize an existing connection")
				}
				return nil, nil
			})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	before, err := os.ReadFile(c.path("one", ".secret"))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.importMicrosoft(t.Context(), "imported", c.path("one", ".secret")); err != nil {
		t.Fatal(err)
	}
	if err := c.importMicrosoft(t.Context(), "imported", c.path("two", ".secret")); err == nil {
		t.Fatal("import overwrote a connected account")
	}
	if err := c.remove("imported", true); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(c.path("one", ".secret"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("import/removal changed the source cache", err)
	}
	if _, err := os.Stat(c.path("imported", ".secret")); !os.IsNotExist(err) {
		t.Fatal("disconnected credentials retained", err)
	}
	if matches, err := filepath.Glob(filepath.Join(c.dir, ".auth-*")); err != nil || len(matches) != 0 {
		t.Fatal("temporary files remain", matches, err)
	}
}
