package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// This exercises a stopped host moving between two real distribution packages.
// It does not authenticate, modify user settings, or claim live hot-reloading.
func TestCodexPluginUpgrade(t *testing.T) {
	previous := os.Getenv("TODO_CONNECT_PREVIOUS_RELEASE_DIR")
	current := os.Getenv("TODO_CONNECT_RELEASE_DIR")
	codex := os.Getenv("TODO_CONNECT_TEST_CODEX")
	if previous == "" || current == "" || codex == "" {
		t.Skip("set TODO_CONNECT_PREVIOUS_RELEASE_DIR, TODO_CONNECT_RELEASE_DIR and TODO_CONNECT_TEST_CODEX")
	}
	if !filepath.IsAbs(previous) || !filepath.IsAbs(current) || previous == current {
		t.Fatal("two distinct absolute release directories are required")
	}
	home, userRoot := t.TempDir(), t.TempDir()
	env := append(os.Environ(), "CODEX_HOME="+home, "TODO_CONNECT_DATA_DIR=", "TODO_CONNECT_KEY_FILE=", "LOCALAPPDATA="+userRoot, "XDG_CONFIG_HOME=", "HOME="+userRoot)
	run := func(exe, dir string, args ...string) []byte {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), exe, args...)
		cmd.Env, cmd.Dir = env, dir
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("%s: %v %s %s", args[0], err, out, stderr.String())
		}
		return out
	}
	type installedPackage struct{ dir, version, exe string }
	packages := make([]installedPackage, 0, 2)
	for _, release := range []string{previous, current} {
		version := filepath.Base(release)
		archivePath := filepath.Join(release, "todo-connect-"+version+"-"+runtime.GOOS+"-"+runtime.GOARCH+".zip")
		archive, err := zip.OpenReader(archivePath)
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		err = os.CopyFS(dir, archive)
		archive.Close()
		if err != nil {
			t.Fatal(err)
		}
		var build struct{ Binary, Version string }
		readReleaseJSON(t, filepath.Join(dir, "BUILD.json"), &build)
		if build.Version != version || !filepath.IsLocal(build.Binary) {
			t.Fatal("unexpected package metadata")
		}
		packages = append(packages, installedPackage{dir, version, filepath.Join(dir, build.Binary)})
	}
	run(packages[0].exe, packages[0].dir, "connections", "add", "--provider", "microsoft", "--id", "release-test", "--label", "Plugin upgrade test")
	before := run(packages[0].exe, packages[0].dir, "connections", "list")
	// Codex rejects moving an existing marketplace name to a different source.
	// Remove its plugin registration and marketplace first, not account data.
	// Each host runtime is stopped before switching; rollback follows the same flow.
	for index, pkg := range []installedPackage{packages[0], packages[1], packages[0]} {
		if index > 0 {
			run(codex, pkg.dir, "plugin", "remove", "todo-connect@todo-connect-local")
			run(codex, pkg.dir, "plugin", "marketplace", "remove", "todo-connect-local")
		}
		run(codex, pkg.dir, "plugin", "marketplace", "add", pkg.dir, "--json")
		run(codex, pkg.dir, "plugin", "add", "todo-connect@todo-connect-local", "--json")
		out := run(codex, pkg.dir, "plugin", "list", "--marketplace", "todo-connect-local", "--json")
		var inventory struct {
			Installed []struct {
				PluginID, Version  string
				Installed, Enabled bool
			}
		}
		if err := json.Unmarshal(out, &inventory); err != nil || len(inventory.Installed) != 1 {
			t.Fatalf("unexpected plugin inventory: %s", out)
		}
		plugin := inventory.Installed[0]
		if plugin.PluginID != "todo-connect@todo-connect-local" || plugin.Version != pkg.version || !plugin.Installed || !plugin.Enabled {
			t.Fatalf("host did not switch to %s: %s", pkg.version, out)
		}
		verifyCodexRuntime(t, codex, home, pkg.dir, env, "release-test")
		if after := run(pkg.exe, pkg.dir, "connections", "list"); !bytes.Equal(before, after) {
			t.Fatal("plugin replacement changed the saved connection inventory")
		}
		t.Logf("Installed host version %s and read the existing connection", pkg.version)
	}
}
