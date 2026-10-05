package app

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func addTestConnection(t *testing.T, c *connections, id string) {
	t.Helper()
	if err := c.add(connection{ID: id, Provider: microsoftProvider, Label: "테스트 " + id, ClientID: clientID}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestConnectionLifecycle(t *testing.T) {
	c := &connections{dir: filepath.Join(t.TempDir(), "data")}
	initial, err := c.list()
	if err != nil || len(initial["connections"].([]object)) != 0 {
		t.Fatalf("initial state %v %v", initial, err)
	}
	if _, err := os.Stat(c.dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("listing connections must not create a data directory")
	}
	for _, id := range []string{"", "../outside", "a/b", `a\b`, "a:stream", "UPPER", strings.Repeat("a", 65)} {
		if err := c.add(connection{ID: id, Provider: microsoftProvider, Label: "test", ClientID: clientID}, nil); err == nil {
			t.Errorf("unsafe connection ID accepted: %q", id)
		}
	}
	addTestConnection(t, c, "personal")
	addTestConnection(t, c, "work")
	if err := c.add(connection{ID: "personal", Provider: microsoftProvider, Label: "replacement", ClientID: clientID}, nil); err == nil {
		t.Fatal("existing connection overwritten")
	}
	restarted := &connections{dir: c.dir}
	listed, err := restarted.list()
	if err != nil || len(listed["connections"].([]object)) != 2 {
		t.Fatalf("restart listing %v %v", listed, err)
	}
	for _, item := range listed["connections"].([]object) {
		if item["state"] != "not_connected" || item["provider"] != microsoftProvider {
			t.Fatalf("incorrect credential state: %v", item)
		}
	}
	if err := restarted.remove("personal", false); err == nil {
		t.Fatal("removal without confirmation")
	}
	if err := restarted.remove("personal", true); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.load("work"); err != nil {
		t.Fatal("removing one connection changed another", err)
	}
	if _, err := restarted.withMicrosoft(t.Context(), "personal", func(*auth, *graph) (object, error) {
		t.Fatal("removed connection still dispatched")
		return nil, nil
	}); err == nil {
		t.Fatal("removed connection accepted")
	}
	addTestConnection(t, restarted, "personal")
}

func TestConnectionLockAcrossProcesses(t *testing.T) {
	if path := os.Getenv("TODO_TEST_LOCK_PATH"); path != "" {
		unlock, err := lockConnection(path)
		if os.Getenv("TODO_TEST_LOCK_EXPECT_BUSY") == "1" {
			if !errors.Is(err, errConnectionBusy) {
				os.Exit(9)
			}
			os.Exit(0)
		}
		if err != nil {
			os.Exit(10)
		}
		_ = unlock
		// Simulate a process exit without graceful unlocking.
		os.Exit(0)
	}
	c := &connections{dir: t.TempDir()}
	addTestConnection(t, c, "personal")
	unlock, err := c.lock("personal")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if unlock != nil {
			unlock()
		}
	}()
	if err := c.remove("personal", true); !errors.Is(err, errConnectionBusy) {
		t.Fatal("active connection removed", err)
	}
	if _, err := c.withMicrosoft(t.Context(), "personal", func(*auth, *graph) (object, error) {
		t.Fatal("parallel request entered a locked connection")
		return nil, nil
	}); !errors.Is(err, errConnectionBusy) {
		t.Fatal("missing busy error", err)
	}
	other, err := c.lock("other")
	if err != nil {
		t.Fatal("unrelated connections should not block", err)
	}
	other()
	for _, busy := range []string{"1", "0"} {
		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestConnectionLockAcrossProcesses$")
		cmd.Env = append(os.Environ(), "TODO_TEST_LOCK_PATH="+c.path("personal", ".lock"), "TODO_TEST_LOCK_EXPECT_BUSY="+busy)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("cross-process lock: %v %s", err, out)
		}
		if unlock != nil {
			unlock()
			unlock = nil
		}
	}
	unlocked, err := c.lock("personal")
	if err != nil {
		t.Fatal("process exit retained an OS lock", err)
	}
	unlocked()
}

func TestExplicitConnectionRouting(t *testing.T) {
	c := &connections{dir: t.TempDir()}
	addTestConnection(t, c, "personal")
	for _, id := range []string{"", "missing", "../personal"} {
		if _, err := c.callMicrosoft(t.Context(), "list_lists", connectedInput{ConnectionID: id}); err == nil {
			t.Errorf("invalid connection dispatched: %q", id)
		}
	}
	if _, err := c.callMicrosoft(t.Context(), "list_lists", connectedInput{ConnectionID: "personal"}); err == nil {
		t.Fatal("unconnected account has read access")
	}
	// A metadata file with a different provider must never use Microsoft credentials or APIs.
	b := []byte(`{"id":"other","provider":"other-provider","label":"Other","client_id":"test"}`)
	if err := os.WriteFile(c.path("other", ".json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := c.withMicrosoft(t.Context(), "other", func(*auth, *graph) (object, error) {
		t.Fatal("wrong provider dispatched")
		return nil, nil
	}); err == nil {
		t.Fatal("provider mismatch accepted")
	}
}

func TestConnectionMCP(t *testing.T) {
	c := &connections{dir: t.TempDir()}
	addTestConnection(t, c, "one")
	addTestConnection(t, c, "two")
	st, ct := mcp.NewInMemoryTransports()
	ss, err := server(c).Connect(t.Context(), st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	cs, err := client.Connect(t.Context(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	listed, err := cs.ListTools(t.Context(), nil)
	if err != nil || len(listed.Tools) != 63 {
		t.Fatalf("catalog: %v %v", listed, err)
	}
	for _, tool := range listed.Tools {
		if tool.Name == "list_connections" || tool.Name == googleSetupTool {
			continue
		}
		schema := tool.InputSchema.(map[string]any)
		if strings.HasPrefix(tool.Name, microsoftToolPrefix) {
			properties := schema["properties"].(map[string]any)
			for _, field := range []string{"connection_id", "list_id", "task_id", "body", "confirm"} {
				if properties[field] == nil {
					t.Errorf("%s schema lost %s", tool.Name, field)
				}
			}
		}
		required, _ := schema["required"].([]any)
		found := false
		for _, field := range required {
			found = found || field == "connection_id"
		}
		if !found {
			t.Errorf("%s does not require connection_id", tool.Name)
		}
	}
	for _, tc := range []struct {
		name string
		args object
		fail bool
	}{
		{"list_connections", object{}, false},
		{microsoftToolPrefix + "list_lists", object{}, true},
		{microsoftToolPrefix + "list_lists", object{"connection_id": "missing"}, true},
		{"check_connection", object{"connection_id": "one"}, true},
		{"disconnect_connection", object{"connection_id": "one"}, true},
		{"disconnect_connection", object{"connection_id": "one", "confirm": true}, false},
	} {
		r, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: tc.name, Arguments: tc.args})
		if err != nil || r.IsError != tc.fail {
			t.Fatalf("%s %v: %v %v", tc.name, tc.args, r, err)
		}
	}
	if _, err := c.load("two"); err != nil {
		t.Fatal("MCP disconnected the wrong connection", err)
	}
}

func TestDataDirectoryOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(dataDirEnv, dir)
	c, err := openConnections()
	if err != nil || c.dir != dir {
		t.Fatalf("override %v %v", c, err)
	}
	t.Setenv(dataDirEnv, "relative-state")
	if _, err := openConnections(); err == nil {
		t.Fatal("relative state could put credentials in a source repository")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := c.withMicrosoft(ctx, "any", func(*auth, *graph) (object, error) { return nil, nil }); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled operation started", err)
	}
}
