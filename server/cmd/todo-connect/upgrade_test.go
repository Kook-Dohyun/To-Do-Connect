package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Explicit local binaries only. Synthetic configuration is shared across two
// installed versions; no real credentials or provider requests are involved.
func TestBinaryUpgradePreservesConnections(t *testing.T) {
	previous, current := os.Getenv("TODO_CONNECT_PREVIOUS_EXE"), os.Getenv("TODO_CONNECT_CURRENT_EXE")
	if previous == "" || current == "" {
		t.Skip("set TODO_CONNECT_PREVIOUS_EXE and TODO_CONNECT_CURRENT_EXE to test an upgrade")
	}
	if !filepath.IsAbs(previous) || !filepath.IsAbs(current) || previous == current {
		t.Fatal("upgrade requires two distinct absolute executable paths")
	}
	dataDir := t.TempDir()
	keyPath := filepath.Join(t.TempDir(), "storage.key")
	env := append(os.Environ(), "PATH=", "TODO_CONNECT_DATA_DIR="+dataDir, "TODO_CONNECT_KEY_FILE="+keyPath)
	run := func(exe string, args ...string) []byte {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), exe, args...)
		cmd.Env, cmd.Dir = env, t.TempDir()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("standalone upgrade command %s: %v %s", args[0], err, out)
		}
		return out
	}
	run(previous, "keygen", "--out", keyPath)
	clientPath := filepath.Join(t.TempDir(), "synthetic-client.json")
	if err := os.WriteFile(clientPath, []byte(`{"installed":{"client_id":"upgrade-fake","client_secret":"upgrade-fake-secret"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	run(previous, "connections", "add", "--provider", "microsoft", "--id", "ms-upgrade", "--label", "Synthetic Microsoft")
	run(previous, "connections", "add", "--provider", "google", "--id", "google-upgrade", "--label", "Synthetic Google", "--credentials", clientPath)
	before := run(previous, "connections", "list")
	files := map[string][]byte{}
	for _, name := range []string{"ms-upgrade.json", "google-upgrade.json", "google-upgrade.client"} {
		data, err := os.ReadFile(filepath.Join(dataDir, name))
		if err != nil {
			t.Fatal(err)
		}
		files[name] = data
	}
	if bytes.Contains(files["google-upgrade.client"], []byte("upgrade-fake-secret")) {
		t.Fatal("old version did not encrypt client configuration")
	}
	if after := run(current, "connections", "list"); !bytes.Equal(before, after) {
		t.Fatal("new version changed connection inventory")
	}
	cmd := exec.CommandContext(t.Context(), current, "serve")
	cmd.Env, cmd.Dir = env, t.TempDir()
	client := mcp.NewClient(&mcp.Implementation{Name: "upgrade-test", Version: "1"}, nil)
	session, err := client.Connect(t.Context(), &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	r, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "list_connections"})
	if err != nil || r.IsError {
		t.Fatalf("upgraded MCP inventory: %v", err)
	}
	inventory, _ := json.Marshal(r)
	for _, id := range []string{"ms-upgrade", "google-upgrade"} {
		if !bytes.Contains(inventory, []byte(id)) {
			t.Fatal("upgraded MCP did not reuse both providers")
		}
	}
	// This must decrypt the old client config successfully before discovering
	// that there is no login token. No request is sent to Google.
	r, err = session.CallTool(t.Context(), &mcp.CallToolParams{Name: "check_connection", Arguments: map[string]any{"connection_id": "google-upgrade"}})
	if err != nil || !r.IsError {
		t.Fatalf("synthetic unsigned connection was accepted: %v", err)
	}
	result, _ := json.Marshal(r)
	if !strings.Contains(string(result), "Google connection is not signed in") {
		t.Fatalf("upgraded binary could not read old encrypted client: %s", result)
	}
	for name, before := range files {
		after, err := os.ReadFile(filepath.Join(dataDir, name))
		if err != nil || !bytes.Equal(before, after) {
			t.Fatalf("upgrade altered persisted %s", name)
		}
	}
	if rollback := run(previous, "connections", "list"); !bytes.Equal(before, rollback) {
		t.Fatal("previous program could not reuse unchanged connection data")
	}
}
