# To Do Connect — development package

Connect Microsoft To Do and Google Tasks to AI using a program running on your own computer. Both providers and multiple connections can coexist. No developer-hosted relay is used. This is an independent project, not an official Microsoft or Google product.

## Start the program

The release provides `install.ps1` for Windows and `install.sh` for macOS/Linux, alongside the ZIPs and `SHA256SUMS`. After downloading and inspecting the installer from the trusted release, run `./install.ps1 -Version VERSION` on Windows or `sh ./install.sh --version VERSION` on Unix. It downloads only the checksums and your OS/CPU archive from `Kook-Dohyun/To-Do-Connect`, tag `vVERSION`. The release must already be published; a local development version has no download URL yet. Downloads are checksum-verified before extraction and temporary downloads are removed on success or failure. No login is performed automatically.

For offline installation, add `-ReleaseDirectory ABSOLUTE_RELEASE_FOLDER` on Windows or `--release-directory ABSOLUTE_RELEASE_FOLDER` on Unix. Unix uses the OS `unzip` and SHA-256 utilities, plus `curl` for downloads, and reports missing tools without installing them. Add `-RegisterCodex` or `--register-codex` only if you want to modify your Codex plugin registrations. Checksums detect corruption; they do not replace trusting the release publisher or authenticate a modified installer.

This archive contains a prebuilt program: **Go, Node and Docker are not required**. `BUILD.json` identifies the OS, CPU architecture and executable path. Use the archive for your execution host, not a remote machine running a different OS. `darwin` means macOS; `amd64` means x64; `arm64` means ARM64.

Extract the archive into a private application folder outside a Git repository or synchronized Vault. Keep its directory structure intact. The executable is `plugins/todo-connect/bin/todo-connect.exe` on Windows and `plugins/todo-connect/bin/todo-connect` on macOS/Linux. On Unix, ensure its executable permission is retained. The development binaries are not code-signed or notarized.

In the examples below, replace `PROGRAM` with that executable's absolute path. You do not need to change your PATH. In PowerShell, invoke a quoted path with `&`.

```text
PROGRAM connections list
PROGRAM connections add --provider microsoft --id personal --label "Personal Microsoft"
PROGRAM login personal
PROGRAM check personal
```

For Google, ask the AI to connect Google Tasks. The bundled Google skill guides it through checking existing setup, preparing your own Cloud project and enabling Tasks API, downloading/importing a Desktop OAuth client, browser consent and read verification. You do not need to prepare a repository `config/` folder. Complete sign-in, MFA and consent yourself when prompted; the AI resumes the remaining setup. This full fresh-user browser flow still needs independent live validation.

If you already have a downloaded **Desktop app OAuth JSON**, the manual CLI equivalent is:

```text
PROGRAM connections add --provider google --id google-personal --label "Personal Google" --credentials ABSOLUTE_PRIVATE_JSON_PATH
PROGRAM login google-personal
PROGRAM check google-personal
```

Complete passwords, MFA and consent in your provider's browser UI. Google requires your own OAuth project setup; see the bundled Google skill's `references/connections.md`. Do not repeat setup if the connection already works. A successful `check` is a read-only list request, not a task-write test.

## Connect MCP or a plugin host

For a local stdio MCP host, set `command` to the executable's absolute path and `args` to `["serve"]`. No TCP port, public URL or developer server is needed.

The archive also contains an Agent Plugins package in `plugins/todo-connect/`, with a package-relative `mcp.json` and two provider skills. `.agents/plugins/marketplace.json` is a local Codex marketplace catalog. For Codex's ZIP upload screen, use the separate `todo-connect-plugin-...zip` asset, not this marketplace bundle. Follow the release's installation notes; a cross-build alone does not verify a particular host's behavior.

## Storage, updates and removal

Credentials are stored in the OS user's separate application-data folder, not in this archive. Windows uses DPAPI; macOS/Linux use an OS keyring. Optional key-file storage and Docker commands are described in `RUNTIME.md`. Never include credentials or encryption keys in an archive, Git repository or synced Vault.

For an update, stop the MCP process, extract the new version to a separate application folder, and point the host to it. Keep the same data-directory/key settings. Do not copy credentials into the new program folder or recreate existing connections. Verify with `connections list` and a read-only `check`. Keep the previous program folder until verification succeeds; there is no automatic updater or storage migration in this development package.

For the bundled Codex marketplace, install the new version without the registration option first. After stopping the old host, run `codex plugin remove todo-connect@todo-connect-local`, then `codex plugin marketplace remove todo-connect-local`. Register the new package root with `codex plugin marketplace add "NEW_PACKAGE_ROOT" --json` followed by `codex plugin add todo-connect@todo-connect-local --json`. Check with `codex plugin list --marketplace todo-connect-local --json` before restarting the host. This replaces this marketplace's registration/cache, not account data. It does not apply to GUI-uploaded plugins or other marketplaces. Rollback uses the retained old package root.

Uninstalling the program is different from disconnecting an account. `connections remove --id ID --confirm` removes only that local connection and its credentials. Remote tasks and provider consent remain. Deleting the program folder alone does not erase credentials from the separate data directory.

Project code is MIT-licensed; see `plugins/todo-connect/LICENSE`. Go and dependency notices are preserved in `plugins/todo-connect/THIRD_PARTY_NOTICES.txt`. Signing, branding and publication are not finalized. These are local development artifacts, not a published stable release.
