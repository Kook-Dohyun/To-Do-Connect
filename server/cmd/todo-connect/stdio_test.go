package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestExecutableStdio(t *testing.T) {
	dir := t.TempDir()
	name := "todo-connect"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	exe := filepath.Join(dir, name)
	build := exec.CommandContext(t.Context(), "go", "build", "-o", exe, ".")
	if out, e := build.CombinedOutput(); e != nil {
		t.Fatalf("build: %v %s", e, out)
	}
	// The installed program must not require Go or a project working directory.
	env := append(os.Environ(), "TODO_CONNECT_DATA_DIR="+filepath.Join(dir, "data"), "PATH=")
	for _, id := range []string{"personal", "work"} {
		add := exec.CommandContext(t.Context(), exe, "connections", "add", "--provider", "microsoft", "--id", id, "--label", "Test "+id)
		add.Env = env
		add.Dir = dir
		if out, err := add.CombinedOutput(); err != nil {
			t.Fatalf("standalone add: %v %s", err, out)
		}
	}
	list := exec.CommandContext(t.Context(), exe, "connections", "list")
	list.Env = env
	list.Dir = dir
	out, err := list.Output()
	if err != nil {
		t.Fatal(err)
	}
	var inventory struct{ Connections []map[string]any }
	if err := json.Unmarshal(out, &inventory); err != nil || len(inventory.Connections) != 2 {
		t.Fatalf("CLI restart did not retain connections: %s %v", out, err)
	}
	cmd := exec.CommandContext(t.Context(), exe, "serve")
	// No application configuration is required; isolate the real user's login cache.
	cmd.Env = env
	cmd.Dir = dir
	client := mcp.NewClient(&mcp.Implementation{Name: "stdio-test", Version: "1"}, nil)
	session, e := client.Connect(t.Context(), &mcp.CommandTransport{Command: cmd}, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer session.Close()
	if session.InitializeResult().ServerInfo.Version != "dev" {
		t.Fatal("unstamped source build must report dev")
	}
	tools, e := session.ListTools(t.Context(), nil)
	if e != nil || len(tools.Tools) != 63 {
		t.Fatalf("tools=%v error=%v", tools, e)
	}
	// With no login cache a real binary must reject data access, without making a Graph call.
	r, e := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "microsoft_list_lists", Arguments: map[string]any{}})
	if e != nil || !r.IsError {
		t.Fatalf("unauthenticated call: %v %v", r, e)
	}
	r, e = session.CallTool(t.Context(), &mcp.CallToolParams{Name: "microsoft_list_lists", Arguments: map[string]any{"connection_id": "personal"}})
	if e != nil || !r.IsError {
		t.Fatalf("configured but unsigned connection: %v %v", r, e)
	}
	r, e = session.CallTool(t.Context(), &mcp.CallToolParams{Name: "disconnect_connection", Arguments: map[string]any{"connection_id": "personal", "confirm": true}})
	if e != nil || r.IsError {
		t.Fatalf("disconnect through MCP: %v %v", r, e)
	}
	list = exec.CommandContext(t.Context(), exe, "connections", "list")
	list.Env = env
	list.Dir = dir
	out, err = list.Output()
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out, &inventory); err != nil || len(inventory.Connections) != 1 || inventory.Connections[0]["id"] != "work" {
		t.Fatalf("MCP and CLI state diverged: %s %v", out, err)
	}
}
