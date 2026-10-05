package main

import (
	"crypto/rand"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestDockerRuntime(t *testing.T) {
	image := os.Getenv("TODO_CONNECT_TEST_IMAGE")
	if image == "" {
		t.Skip("set TODO_CONNECT_TEST_IMAGE to test an explicitly built local image")
	}
	runID := strings.ToLower(rand.Text())
	dataVolume := "todo-connect-test-data-" + runID
	keyVolume := "todo-connect-test-key-" + runID
	container := "todo-connect-test-mcp-" + runID
	docker := func(args ...string) []byte {
		t.Helper()
		out, err := exec.CommandContext(t.Context(), "docker", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("docker %s: %v %s", args[0], err, out)
		}
		return out
	}
	for _, volume := range []string{dataVolume, keyVolume} {
		docker("volume", "create", "--label", "todo-connect.test="+runID, volume)
		t.Cleanup(func() {
			// Only volumes created by this exact test run are eligible for cleanup.
			out, err := exec.Command("docker", "volume", "inspect", "--format", `{{index .Labels "todo-connect.test"}}`, volume).CombinedOutput()
			if err != nil || strings.TrimSpace(string(out)) != runID {
				t.Errorf("refusing unverified test-volume cleanup: %s %v", volume, err)
				return
			}
			if out, err := exec.Command("docker", "volume", "rm", volume).CombinedOutput(); err != nil {
				t.Errorf("test-volume cleanup: %v %s", err, out)
			}
		})
	}
	base := []string{"run", "--rm", "-v", dataVolume + ":/data", "-v", keyVolume + ":/keys"}
	run := func(args ...string) []byte {
		return docker(append(append(append([]string{}, base...), image), args...)...)
	}
	if user := strings.TrimSpace(string(docker("image", "inspect", "--format", "{{.Config.User}}", image))); user != "10001:10001" {
		t.Fatalf("runtime user: %s", user)
	}
	run("keygen", "--out", "/keys/todo-connect.key")
	run("connections", "add", "--provider", "microsoft", "--id", "ms-test", "--label", "Test Microsoft")
	args := append(append([]string{}, base...), "-i", image, "connections", "add", "--provider", "google", "--id", "google-test", "--label", "Test Google", "--credentials", "-")
	importClient := exec.CommandContext(t.Context(), "docker", args...)
	importClient.Stdin = strings.NewReader(`{"installed":{"client_id":"test-desktop-client","client_secret":"fake-client-secret"}}`)
	if out, err := importClient.CombinedOutput(); err != nil {
		t.Fatalf("Docker client import: %v %s", err, out)
	}
	var inventory struct{ Connections []map[string]any }
	if err := json.Unmarshal(run("connections", "list"), &inventory); err != nil || len(inventory.Connections) != 2 {
		t.Fatalf("container restart lost connections: %v", err)
	}
	// Read only synthetic encrypted bytes, not the encryption key or any real credentials.
	catArgs := append(append([]string{}, base...), "--entrypoint", "/bin/cat", image, "/data/google-test.client")
	encrypted := docker(catArgs...)
	if strings.Contains(string(encrypted), "fake-client-secret") || !strings.HasPrefix(string(encrypted), "TDC-AES256-GCM-1") {
		t.Fatal("Google client not encrypted")
	}
	cmdArgs := append(append([]string{}, base...), "--name", container, "--label", "todo-connect.test="+runID, "-i", image, "serve")
	cmd := exec.CommandContext(t.Context(), "docker", cmdArgs...)
	client := mcp.NewClient(&mcp.Implementation{Name: "docker-runtime-test", Version: "1"}, nil)
	t.Cleanup(func() {
		out, err := exec.Command("docker", "container", "inspect", "--format", `{{index .Config.Labels "todo-connect.test"}}`, container).CombinedOutput()
		if err != nil {
			return
		} // --rm normally already removed it when stdio closed.
		if strings.TrimSpace(string(out)) != runID {
			t.Error("refusing unverified container cleanup")
			return
		}
		if out, err := exec.Command("docker", "rm", "-f", container).CombinedOutput(); err != nil {
			t.Errorf("test-container cleanup: %v %s", err, out)
		}
	})
	cs, err := client.Connect(t.Context(), &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	tools, err := cs.ListTools(t.Context(), nil)
	if err != nil || len(tools.Tools) != 27 {
		t.Fatalf("Docker MCP tool count %v %v", tools, err)
	}
	r, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: "google_list_lists", Arguments: map[string]any{"connection_id": "google-test"}})
	if err != nil || !r.IsError {
		t.Fatalf("unsigned Google access: %v %v", r, err)
	}
	text, _ := json.Marshal(r)
	if !strings.Contains(string(text), "not signed in") {
		t.Fatalf("encrypted client could not be reused: %s", text)
	}
	r, err = cs.CallTool(t.Context(), &mcp.CallToolParams{Name: "disconnect_connection", Arguments: map[string]any{"connection_id": "google-test", "confirm": true}})
	if err != nil || r.IsError {
		t.Fatalf("Docker MCP disconnect: %v %v", r, err)
	}
	if err := json.Unmarshal(run("connections", "list"), &inventory); err != nil || len(inventory.Connections) != 1 || inventory.Connections[0]["id"] != "ms-test" {
		t.Fatalf("Docker connection removal isolation: %v %v", inventory, err)
	}
}
