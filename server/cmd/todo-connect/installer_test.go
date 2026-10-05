package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWindowsInstaller(t *testing.T) {
	release := os.Getenv("TODO_CONNECT_RELEASE_DIR")
	if release == "" || runtime.GOOS != "windows" {
		t.Skip("Windows and TODO_CONNECT_RELEASE_DIR are required")
	}
	version := filepath.Base(release)
	installer, err := filepath.Abs(filepath.Join(release, "install.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "application with spaces")
	args := []string{"-NoProfile", "-NonInteractive", "-File", installer, "-ReleaseDirectory", release, "-Version", version, "-InstallDirectory", destination}
	// Use inbox Windows PowerShell, not a separately installed PowerShell runtime.
	cmd := exec.CommandContext(t.Context(), "powershell.exe", args...)
	cmd.Env = append(os.Environ(), "TODO_CONNECT_DATA_DIR="+t.TempDir())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native installer: %v %s", err, out)
	}
	program := filepath.Join(destination, "plugins", "todo-connect", "bin", "todo-connect.exe")
	if out, err := exec.CommandContext(t.Context(), program, "version").Output(); err != nil || strings.TrimSpace(string(out)) != version {
		t.Fatalf("installed program: %v %s", err, out)
	}
	// Re-running cannot replace a running program or wipe an existing install.
	if err := exec.CommandContext(t.Context(), "powershell.exe", args...).Run(); err == nil {
		t.Fatal("installer overwrote an existing destination")
	}
	// Integrity failure must be detected before creating the destination.
	badRelease := t.TempDir()
	archiveName := "todo-connect-" + version + "-windows-" + runtime.GOARCH + ".zip"
	if err := os.WriteFile(filepath.Join(badRelease, "SHA256SUMS"), []byte(strings.Repeat("0", 64)+"  "+archiveName+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badRelease, archiveName), []byte("corrupt archive"), 0600); err != nil {
		t.Fatal(err)
	}
	badDestination := filepath.Join(t.TempDir(), "must-not-exist")
	bad := exec.CommandContext(t.Context(), "powershell.exe", "-NoProfile", "-NonInteractive", "-File", installer, "-ReleaseDirectory", badRelease, "-Version", version, "-InstallDirectory", badDestination)
	if err := bad.Run(); err == nil {
		t.Fatal("installer accepted a corrupt archive")
	}
	if _, err := os.Stat(badDestination); !os.IsNotExist(err) {
		t.Fatal("integrity failure created an install destination")
	}
	if codex := os.Getenv("TODO_CONNECT_TEST_CODEX"); codex != "" {
		pluginDestination := filepath.Join(t.TempDir(), "registered application")
		isolatedHome, userRoot := t.TempDir(), t.TempDir()
		env := append(os.Environ(), "CODEX_HOME="+isolatedHome, "TODO_CONNECT_DATA_DIR=", "TODO_CONNECT_KEY_FILE=", "LOCALAPPDATA="+userRoot)
		register := exec.CommandContext(t.Context(), "powershell.exe", "-NoProfile", "-NonInteractive", "-File", installer, "-ReleaseDirectory", release, "-Version", version, "-InstallDirectory", pluginDestination, "-RegisterCodex")
		register.Env = append(env, "PATH="+filepath.Dir(codex)+string(os.PathListSeparator)+os.Getenv("PATH"))
		if out, err := register.CombinedOutput(); err != nil {
			t.Fatalf("installer plugin registration: %v %s", err, out)
		}
		installed := filepath.Join(pluginDestination, "plugins", "todo-connect", "bin", "todo-connect.exe")
		verifyCodexRuntime(t, codex, isolatedHome, pluginDestination, env, "")
		add := exec.CommandContext(t.Context(), installed, "connections", "add", "--provider", "microsoft", "--id", "release-test", "--label", "Installer test")
		add.Env = env
		if out, err := add.CombinedOutput(); err != nil {
			t.Fatalf("installer synthetic connection: %v %s", err, out)
		}
		verifyCodexRuntime(t, codex, isolatedHome, pluginDestination, env, "release-test")
	}
}

func TestWindowsInstallerDownload(t *testing.T) {
	release := os.Getenv("TODO_CONNECT_RELEASE_DIR")
	if release == "" || runtime.GOOS != "windows" {
		t.Skip("Windows and TODO_CONNECT_RELEASE_DIR are required")
	}
	version := filepath.Base(release)
	archive := "todo-connect-" + version + "-windows-" + runtime.GOARCH + ".zip"
	// Replace only the network command in the child shell. The shipped installer,
	// checksum verification, extraction and native executable are real.
	wrapper := filepath.Join(t.TempDir(), "download-test.ps1")
	script := `
$ErrorActionPreference = 'Stop'
function Invoke-WebRequest {
    param([switch]$UseBasicParsing, [string]$Uri, [string]$OutFile, [int]$TimeoutSec)
    $asset = [IO.Path]::GetFileName($OutFile)
    if ($asset -notin @('SHA256SUMS', $env:TEST_ARCHIVE)) { throw 'Unexpected asset' }
    $expected = "https://github.com/Kook-Dohyun/To-Do-Connect/releases/download/v$env:TEST_VERSION/$asset"
    if ($Uri -ne $expected -or -not $UseBasicParsing -or $TimeoutSec -ne 60) { throw 'Unexpected download arguments' }
    Add-Content -LiteralPath $env:TEST_DOWNLOAD_LOG -Value $asset
    if ($env:TEST_DOWNLOAD_FAILURE -eq $asset) { throw 'Simulated HTTP 404' }
    Copy-Item -LiteralPath (Join-Path $env:TEST_RELEASE $asset) -Destination $OutFile
}
& (Join-Path $env:TEST_RELEASE 'install.ps1') -Version $env:TEST_VERSION -InstallDirectory $env:TEST_DESTINATION
`
	if err := os.WriteFile(wrapper, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	for _, failure := range []string{"", "SHA256SUMS", archive} {
		t.Run("failure="+failure, func(t *testing.T) {
			root, temporary := t.TempDir(), t.TempDir()
			destination := filepath.Join(root, "application with spaces")
			logPath := filepath.Join(root, "downloads.log")
			cmd := exec.CommandContext(t.Context(), "powershell.exe", "-NoProfile", "-NonInteractive", "-File", wrapper)
			cmd.Env = append(os.Environ(), "TEST_RELEASE="+release, "TEST_VERSION="+version, "TEST_ARCHIVE="+archive,
				"TEST_DESTINATION="+destination, "TEST_DOWNLOAD_LOG="+logPath, "TEST_DOWNLOAD_FAILURE="+failure,
				"TEMP="+temporary, "TMP="+temporary)
			out, err := cmd.CombinedOutput()
			if failure == "" {
				if err != nil {
					t.Fatalf("download installation: %v %s", err, out)
				}
				program := filepath.Join(destination, "plugins", "todo-connect", "bin", "todo-connect.exe")
				if out, err := exec.CommandContext(t.Context(), program, "version").Output(); err != nil || strings.TrimSpace(string(out)) != version {
					t.Fatalf("downloaded program: %v %s", err, out)
				}
			} else {
				if err == nil || !strings.Contains(string(out), "Simulated HTTP 404") {
					t.Fatalf("expected download failure: %v %s", err, out)
				}
				if _, err := os.Stat(destination); !os.IsNotExist(err) {
					t.Fatal("download failure created an installation")
				}
			}
			logBytes, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatal(err)
			}
			want := "SHA256SUMS\n" + archive
			if failure == "SHA256SUMS" {
				want = "SHA256SUMS"
			}
			if strings.TrimSpace(strings.ReplaceAll(string(logBytes), "\r\n", "\n")) != want {
				t.Fatalf("unexpected download sequence: %s", logBytes)
			}
			entries, err := os.ReadDir(temporary)
			if err != nil || len(entries) != 0 {
				t.Fatalf("download temporary files remain: %v %v", entries, err)
			}
		})
	}
}
