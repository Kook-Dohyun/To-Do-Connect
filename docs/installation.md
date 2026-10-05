# Installation

[English overview](../README.md) · [한국어 소개](../README.ko.md)

To Do Connect is a local program. Install it on the machine where the MCP client runs. Native packages do not require Go, Node.js or Docker.

## Choose a package

Download from [GitHub Releases](https://github.com/Kook-Dohyun/To-Do-Connect/releases), or use a trusted development archive provided for local testing. Check the actual asset list: a source version is not proof that a package has been published.

| Package | Use |
| --- | --- |
| `todo-connect-plugin-VERSION-OS-ARCH.zip` | Desktop plugin archive upload |
| `todo-connect-VERSION-OS-ARCH.zip` | Native installation, local marketplace or other MCP clients |
| `todo-connect-VERSION-linux-ARCH.tar.gz`, when provided | Docker image archive |

OS values: `windows`, `darwin` (macOS), `linux`. Architecture: `amd64` (x64) or `arm64`. Compare the download's SHA-256 with the accompanying checksum file. Checksums detect corruption; they do not authenticate an untrusted publisher.

## Plugin archive

In a supported desktop client, open **Plugins → Add → Upload plugin archive**, select the matching plugin ZIP, and install the resulting entry. Start a new chat and select To Do Connect.

Ask “Connect my task service” or name the service directly. The bundled skill checks saved connections and guides unfinished setup. Log in and approve permissions in the provider's UI.

Archive upload, local installation and OpenAI public-directory publication are separate operations.

## AI-assisted GitHub installation

Use the installation prompt in the [English README](../README.md#install) or [한국어 README](../README.ko.md#설치). The assistant should select a real published asset, inspect the installer, preserve existing account data and verify discovery in your chosen host.

For manual use, download and inspect `install.ps1` or `install.sh`, then substitute an actual published version:

```powershell
./install.ps1 -Version VERSION -RegisterCodex
```

```sh
sh ./install.sh --version VERSION --register-codex
```

Codex registration requires an existing Codex CLI. Omit the registration option to install only the program. Windows needs PowerShell 5.1 or later; Unix needs curl, unzip and a SHA-256 utility.

For an unpublished/local package, supply its directory containing native ZIPs and `SHA256SUMS`:

```powershell
./install.ps1 -Version VERSION -ReleaseDirectory ABSOLUTE_RELEASE_FOLDER -RegisterCodex
```

```sh
sh ./install.sh --version VERSION --release-directory ABSOLUTE_RELEASE_FOLDER --register-codex
```

Default installer locations:

- Windows: `%LOCALAPPDATA%\Programs\ToDoConnect\VERSION`
- macOS: `~/Library/Application Support/ToDoConnect/VERSION`
- Linux: `${XDG_DATA_HOME:-$HOME/.local/share}/todo-connect/programs/VERSION`

Codex maintains its installed plugin copy in its own cache. These program locations are separate from [account data](runtime.md#account-data).

## Other MCP clients

Extract the native ZIP. Point the client's stdio server configuration to the executable. For clients using `mcpServers`:

```json
{
  "mcpServers": {
    "todo-connect": {
      "command": "C:/YOUR_INSTALL_DIRECTORY/plugins/todo-connect/bin/todo-connect.exe",
      "args": ["serve"]
    }
  }
}
```

On macOS/Linux, use the actual absolute path ending in `plugins/todo-connect/bin/todo-connect`. Preserve executable permissions. No public URL or listening port is required.

MCP-only registration does not automatically install skills in every client. Use the client's skill support or the package's `GETTING_STARTED.md` for account setup.

## Update or remove

Install a new version in a separate directory. Stop the old To Do Connect MCP process before replacing its plugin registration; keep account data and key settings unchanged.

For an installation registered as `todo-connect@todo-connect-local`, use the verified new package root containing `.agents` and `BUILD.json`:

```text
codex plugin remove todo-connect@todo-connect-local
codex plugin marketplace remove todo-connect-local
codex plugin marketplace add "NEW_PACKAGE_ROOT" --json
codex plugin add todo-connect@todo-connect-local --json
codex plugin list --marketplace todo-connect-local --json
```

Do not apply these commands to an unrelated marketplace or a separately uploaded account plugin. Restart/reload the client as required, then verify the installed version, tool list and a harmless connection-list call. Keep the prior package until that check succeeds.

Removing the plugin or program does not erase account data. Ask to disconnect a particular account to remove its local credentials. Remote tasks and service-side consent are separate.
