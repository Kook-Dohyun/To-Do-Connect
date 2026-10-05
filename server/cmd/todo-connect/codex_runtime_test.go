package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Verify the host resolves the installed plugin manifest and starts its MCP,
// without an LLM turn, a real account, or changes to the user's Codex home.
func verifyCodexRuntime(t *testing.T, codex, home, dir string, env []string, expectedConnection string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	// Compare the host with this package's actual catalog, including retained
	// older packages used by the upgrade/rollback test.
	var build struct{ Binary string }
	readReleaseJSON(t, filepath.Join(dir, "BUILD.json"), &build)
	if !filepath.IsLocal(build.Binary) {
		t.Fatal("invalid package binary path")
	}
	catalogCommand := exec.CommandContext(ctx, filepath.Join(dir, filepath.FromSlash(build.Binary)), "catalog")
	catalogCommand.Env, catalogCommand.Dir = env, dir
	catalogJSON, err := catalogCommand.Output()
	if err != nil {
		t.Fatal("package catalog", err)
	}
	var catalog struct{ Tools []struct{ Name string } }
	if err := json.Unmarshal(catalogJSON, &catalog); err != nil || len(catalog.Tools) == 0 {
		t.Fatal("empty or invalid package catalog", err)
	}
	cmd := exec.CommandContext(ctx, codex, "app-server", "--stdio")
	cmd.Env, cmd.Dir = append(env, "CODEX_HOME="+home), dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		in.Close()
		cancel()
		_ = cmd.Wait()
		if t.Failed() {
			t.Logf("isolated Codex diagnostics: %s", stderr.String())
		}
	}()
	encoder, decoder := json.NewEncoder(in), json.NewDecoder(out)
	nextID := 0
	rpc := func(method string, params any, result any) {
		t.Helper()
		nextID++
		if err := encoder.Encode(map[string]any{"id": nextID, "method": method, "params": params}); err != nil {
			t.Fatal(err)
		}
		for {
			var response struct {
				ID     int
				Method string
				Result json.RawMessage
				Error  json.RawMessage
			}
			if err := decoder.Decode(&response); err != nil {
				t.Fatalf("Codex %s response: %v", method, err)
			}
			if response.ID != nextID {
				continue // Initialization and thread startup also emit notifications.
			}
			if len(response.Error) > 0 && string(response.Error) != "null" {
				t.Fatalf("Codex %s: %s", method, response.Error)
			}
			if result != nil {
				if err := json.Unmarshal(response.Result, result); err != nil {
					t.Fatal(err)
				}
			}
			return
		}
	}
	rpc("initialize", map[string]any{"clientInfo": map[string]any{"name": "todo-connect-release-test", "version": "1"}, "capabilities": map[string]any{"experimentalApi": true}}, nil)
	if err := encoder.Encode(map[string]any{"method": "initialized", "params": map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	var started struct{ Thread struct{ ID string } }
	var plugin struct {
		Plugin struct {
			MCPServers []string
			Skills     []json.RawMessage
			Summary    struct {
				Interface struct{ Logo, ComposerIcon string }
			}
		}
	}
	rpc("plugin/read", map[string]any{"pluginName": "todo-connect", "marketplacePath": filepath.Join(dir, ".agents", "plugins", "marketplace.json")}, &plugin)
	t.Logf("Codex plugin detail: MCP=%v skills=%d", plugin.Plugin.MCPServers, len(plugin.Plugin.Skills))
	if len(plugin.Plugin.MCPServers) != 1 || plugin.Plugin.MCPServers[0] != "todo-connect" || len(plugin.Plugin.Skills) != 2 {
		t.Fatal("Codex did not discover the bundled MCP and both provider skills")
	}
	// Older release fixtures have no icon. If this package declares one, the
	// host must resolve both local icon fields to the actual bundled PNG bytes.
	icon, err := os.ReadFile(filepath.Join(dir, "plugins", "todo-connect", "assets", "icon.png"))
	if err == nil {
		for _, path := range []string{plugin.Plugin.Summary.Interface.Logo, plugin.Plugin.Summary.Interface.ComposerIcon} {
			if !filepath.IsAbs(path) {
				t.Fatalf("Codex did not resolve a local icon path: %q", path)
			}
			resolved, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(resolved, icon) {
				t.Fatalf("Codex icon differs from bundled asset: %s (%v)", path, err)
			}
		}
		t.Log("Codex resolved the bundled logo and composer icon to matching local PNG files")
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	rpc("thread/start", map[string]any{"cwd": dir, "ephemeral": true, "approvalPolicy": "never", "sandbox": "read-only"}, &started)
	if started.Thread.ID == "" {
		t.Fatal("Codex did not create an isolated ephemeral runtime")
	}
	var inventory struct {
		Data []struct {
			Name, PluginID, RuntimeStatus string
			Tools                         map[string]json.RawMessage
			ToolsError                    *string
		}
	}
	rpc("mcpServerStatus/list", map[string]any{"threadId": started.Thread.ID, "detail": "toolsAndAuthOnly"}, &inventory)
	serverName := ""
	for _, server := range inventory.Data {
		if strings.Contains(server.PluginID, "todo-connect") {
			if server.ToolsError != nil || len(server.Tools) != len(catalog.Tools) {
				t.Fatalf("Codex plugin discovery: status=%s tools=%d error=%v", server.RuntimeStatus, len(server.Tools), server.ToolsError)
			}
			for _, tool := range catalog.Tools {
				if _, ok := server.Tools[tool.Name]; !ok {
					t.Fatalf("host omitted packaged tool %s", tool.Name)
				}
			}
			serverName = server.Name
		}
	}
	if serverName == "" {
		t.Fatalf("Codex did not discover installed To Do Connect: %+v", inventory)
	}
	var called struct {
		IsError           bool
		Content           json.RawMessage
		StructuredContent json.RawMessage
	}
	rpc("mcpServer/tool/call", map[string]any{"threadId": started.Thread.ID, "server": serverName, "tool": "list_connections", "arguments": map[string]any{}}, &called)
	if called.IsError {
		t.Fatalf("Codex could not list connections: %s", called.Content)
	}
	if expectedConnection != "" {
		if !bytes.Contains(called.Content, []byte(expectedConnection)) {
			t.Fatalf("Codex could not read the synthetic connection through installed MCP: %s", called.Content)
		}
		return
	}
	var initial struct{ Connections []json.RawMessage }
	if err := json.Unmarshal(called.StructuredContent, &initial); err != nil || initial.Connections == nil || len(initial.Connections) != 0 {
		t.Fatalf("fresh installation did not start with zero connections: %s (%v)", called.Content, err)
	}
	rpc("mcpServer/tool/call", map[string]any{"threadId": started.Thread.ID, "server": serverName, "tool": "get_google_setup", "arguments": map[string]any{}}, &called)
	var setup struct {
		State      string
		NextAction string `json:"next_action"`
		Guide      string
	}
	if err := json.Unmarshal(called.StructuredContent, &setup); err != nil || called.IsError || setup.State != "client_json_required" || setup.NextAction == "" || setup.Guide != "google-tasks/references/connections.md" {
		t.Fatalf("fresh Google setup did not return its onboarding step: %s (%v)", called.Content, err)
	}
	t.Log("fresh installation: zero connections; Google setup directs to bundled onboarding guide without credentials")
}
