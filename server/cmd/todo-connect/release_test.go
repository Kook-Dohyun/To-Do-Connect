package main

import (
	"archive/zip"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Opt-in verification uses the actual distributable ZIPs, not the source binary.
func TestReleaseArchives(t *testing.T) {
	releaseDir := os.Getenv("TODO_CONNECT_RELEASE_DIR")
	if releaseDir == "" {
		t.Skip("set TODO_CONNECT_RELEASE_DIR to verify built release archives")
	}
	sums, err := os.ReadFile(filepath.Join(releaseDir, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(sums)), "\n")
	if len(lines) != 8 {
		t.Fatalf("expected six archive and two installer checksums, got %d", len(lines))
	}
	targets := map[string]bool{"windows/amd64": false, "windows/arm64": false, "linux/amd64": false, "linux/arm64": false, "darwin/amd64": false, "darwin/arm64": false}
	installers := map[string]bool{"install.ps1": false, "install.sh": false}
	for _, line := range lines {
		digest, name, ok := strings.Cut(strings.TrimSpace(line), "  ")
		if !ok || filepath.Base(name) != name {
			t.Fatal("invalid checksum entry")
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(releaseDir, name)
			contents, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			hash := sha256.Sum256(contents)
			if hex.EncodeToString(hash[:]) != digest {
				t.Fatal("archive checksum mismatch")
			}
			if seen, installer := installers[name]; installer {
				if seen || len(contents) == 0 {
					t.Fatal("duplicate or empty installer")
				}
				installers[name] = true
				return
			}
			archive, err := zip.OpenReader(path)
			if err != nil {
				t.Fatal(err)
			}
			defer archive.Close()
			dir := t.TempDir()
			for _, file := range archive.File {
				if !filepath.IsLocal(file.Name) {
					t.Fatal("archive path escapes package")
				}
				if !(file.Name == "BUILD.json" || file.Name == "GETTING_STARTED.md" || file.Name == "RUNTIME.md" || file.Name == ".agents/plugins/marketplace.json" || file.Name == "plugins/todo-connect/plugin.json" || file.Name == "plugins/todo-connect/mcp.json" || file.Name == "plugins/todo-connect/LICENSE" || file.Name == "plugins/todo-connect/THIRD_PARTY_NOTICES.txt" || file.Name == "plugins/todo-connect/assets/icon.png" || strings.HasPrefix(file.Name, "plugins/todo-connect/skills/") && strings.HasSuffix(file.Name, ".md") || file.Name == "plugins/todo-connect/bin/todo-connect" || file.Name == "plugins/todo-connect/bin/todo-connect.exe") {
					t.Fatalf("unexpected package file: %s", file.Name)
				}
				out := filepath.Join(dir, filepath.FromSlash(file.Name))
				if err := os.MkdirAll(filepath.Dir(out), 0700); err != nil {
					t.Fatal(err)
				}
				reader, err := file.Open()
				if err != nil {
					t.Fatal(err)
				}
				data, readErr := io.ReadAll(reader)
				reader.Close()
				if readErr != nil {
					t.Fatal(readErr)
				}
				if strings.HasPrefix(file.Name, "plugins/todo-connect/bin/") && file.Mode().Perm() != 0755 {
					t.Fatal("binary executable permission not preserved in ZIP")
				}
				if err := os.WriteFile(out, data, file.Mode().Perm()); err != nil {
					t.Fatal(err)
				}
			}
			var build struct{ Version, OS, Arch, Binary, SHA256 string }
			readReleaseJSON(t, filepath.Join(dir, "BUILD.json"), &build)
			target := build.OS + "/" + build.Arch
			if seen, exists := targets[target]; !exists || seen {
				t.Fatalf("unexpected or duplicate target: %s", target)
			}
			targets[target] = true
			info, err := buildinfo.ReadFile(filepath.Join(dir, build.Binary))
			if err != nil {
				t.Fatal(err)
			}
			settings := map[string]string{}
			for _, setting := range info.Settings {
				settings[setting.Key] = setting.Value
			}
			if settings["GOOS"] != build.OS || settings["GOARCH"] != build.Arch || settings["CGO_ENABLED"] != "0" {
				t.Fatal("bundled binary target differs from release metadata")
			}
			binary, err := os.ReadFile(filepath.Join(dir, build.Binary))
			if err != nil {
				t.Fatal(err)
			}
			binaryHash := sha256.Sum256(binary)
			if hex.EncodeToString(binaryHash[:]) != build.SHA256 {
				t.Fatal("binary checksum mismatch")
			}
			var manifest struct {
				Name, Version string
				Extensions    map[string]struct {
					Interface struct{ Logo, ComposerIcon string }
				}
			}
			pluginRoot := filepath.Join(dir, "plugins", "todo-connect")
			license, err := os.ReadFile(filepath.Join(pluginRoot, "LICENSE"))
			if err != nil || !strings.Contains(string(license), "MIT License") {
				t.Fatal("project MIT license not bundled")
			}
			notices, err := os.ReadFile(filepath.Join(pluginRoot, "THIRD_PARTY_NOTICES.txt"))
			if err != nil || !strings.Contains(string(notices), info.GoVersion) {
				t.Fatal("Go runtime license not bundled")
			}
			for _, module := range info.Deps {
				if !strings.Contains(string(notices), "Module: "+module.Path+"@"+module.Version+"\n") {
					t.Fatalf("bundled module has no license notice: %s", module.Path)
				}
			}
			readReleaseJSON(t, filepath.Join(pluginRoot, "plugin.json"), &manifest)
			if manifest.Name != "todo-connect" || manifest.Version != build.Version {
				t.Fatal("plugin identity/version mismatch")
			}
			listing := manifest.Extensions["com.openai"].Interface
			if listing.Logo != "./assets/icon.png" || listing.ComposerIcon != listing.Logo {
				t.Fatal("plugin must reference its bundled logo and composer icon")
			}
			icon, err := os.Open(filepath.Join(pluginRoot, "assets", "icon.png"))
			if err != nil {
				t.Fatal(err)
			}
			iconConfig, decodeErr := png.DecodeConfig(icon)
			icon.Close()
			if decodeErr != nil || iconConfig.Width != iconConfig.Height || iconConfig.Width < 256 {
				t.Fatal("bundled logo is not a square PNG of at least 256 pixels", decodeErr)
			}
			var config struct {
				MCPServers map[string]struct {
					Type, Command string
					Args          []string
				}
			}
			readReleaseJSON(t, filepath.Join(pluginRoot, "mcp.json"), &config)
			server := config.MCPServers["todo-connect"]
			expected := "./bin/" + filepath.Base(build.Binary)
			if server.Command != expected || server.Type != "stdio" || len(server.Args) != 1 || server.Args[0] != "serve" {
				t.Fatal("MCP does not point at the bundled executable")
			}
			if build.OS != runtime.GOOS || build.Arch != runtime.GOARCH {
				return // Archive integrity is verified, not execution on another OS.
			}
			exe := filepath.Join(pluginRoot, filepath.FromSlash(server.Command))
			// Exercise default per-user storage as installed hosts need not forward
			// arbitrary TODO_CONNECT_* environment variables to plugin processes.
			userRoot := t.TempDir()
			// Codex forwards HOME but does not forward XDG_CONFIG_HOME to plugin
			// MCPs by default. Both sides must use HOME/.config on Linux.
			env := append(os.Environ(), "PATH=", "TODO_CONNECT_DATA_DIR=", "TODO_CONNECT_KEY_FILE=", "LOCALAPPDATA="+userRoot, "XDG_CONFIG_HOME=", "HOME="+userRoot)
			version := exec.CommandContext(t.Context(), exe, "version")
			version.Env = env
			if out, err := version.Output(); err != nil || strings.TrimSpace(string(out)) != build.Version {
				t.Fatalf("packaged executable version differs from manifest: %s %v", out, err)
			}
			add := exec.CommandContext(t.Context(), exe, "connections", "add", "--provider", "microsoft", "--id", "release-test", "--label", "Release test")
			add.Env, add.Dir = env, t.TempDir()
			if out, err := add.CombinedOutput(); err != nil {
				t.Fatalf("extracted standalone program: %v %s", err, out)
			}
			cmd := exec.CommandContext(t.Context(), exe, server.Args...)
			cmd.Env, cmd.Dir = env, t.TempDir()
			client := mcp.NewClient(&mcp.Implementation{Name: "release-test", Version: "1"}, nil)
			session, err := client.Connect(t.Context(), &mcp.CommandTransport{Command: cmd}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			if session.InitializeResult().ServerInfo.Version != build.Version {
				t.Fatal("MCP version differs from release version")
			}
			tools, err := session.ListTools(t.Context(), nil)
			if err != nil || len(tools.Tools) != 27 {
				t.Fatalf("extracted MCP registration: %v", err)
			}
			result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "list_connections"})
			if err != nil || result.IsError {
				t.Fatalf("extracted MCP inventory: %v", err)
			}
			data, _ := json.Marshal(result)
			if !strings.Contains(string(data), "release-test") {
				t.Fatal("CLI and MCP did not reuse the same external data directory")
			}
			if codex := os.Getenv("TODO_CONNECT_TEST_CODEX"); codex != "" {
				// A separate child-process home keeps the real user's installed plugins unchanged.
				isolatedHome := t.TempDir()
				for _, args := range [][]string{
					{"plugin", "marketplace", "add", dir, "--json"},
					{"plugin", "add", "todo-connect@todo-connect-local", "--json"},
					{"plugin", "list", "--marketplace", "todo-connect-local", "--json"},
				} {
					cmd := exec.CommandContext(t.Context(), codex, args...)
					cmd.Env = append(os.Environ(), "CODEX_HOME="+isolatedHome)
					cmd.Dir = dir
					out, err := cmd.Output()
					if err != nil {
						t.Fatalf("isolated Codex %s: %v %s", args[1], err, out)
					}
					if !strings.Contains(string(out), "todo-connect") {
						t.Fatalf("isolated Codex did not report our package: %s", out)
					}
					if args[1] == "list" {
						var result struct {
							Installed []struct {
								PluginID, Version  string
								Installed, Enabled bool
							}
						}
						if json.Unmarshal(out, &result) != nil || len(result.Installed) != 1 {
							t.Fatalf("unexpected isolated plugin inventory: %s", out)
						}
						plugin := result.Installed[0]
						if plugin.PluginID != "todo-connect@todo-connect-local" || plugin.Version != build.Version || !plugin.Installed || !plugin.Enabled {
							t.Fatalf("plugin was not installed/enabled at the packaged version: %+v", plugin)
						}
					}
				}
				verifyCodexRuntime(t, codex, isolatedHome, dir, env, "release-test")
			}
		})
	}
	for target, seen := range targets {
		if !seen {
			t.Errorf("missing release target: %s", target)
		}
	}
	for name, seen := range installers {
		if !seen {
			t.Errorf("missing release installer: %s", name)
		}
	}
}

func readReleaseJSON(t *testing.T, path string, out any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatal(err)
	}
}
