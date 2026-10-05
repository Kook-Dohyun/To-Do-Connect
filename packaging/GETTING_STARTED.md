# Start using To Do Connect

This native package contains the program, a local plugin marketplace, two provider skills and license notices. Go, Node.js and Docker are not required. This is a development preview, not a stable release.

## Install the package

Select the OS and CPU matching the execution host. `BUILD.json` identifies the executable and version. Compare the ZIP hash with its checksum file before extracting.

For a local package set, run the inspected installer with its version and directory:

```powershell
./install.ps1 -Version VERSION -ReleaseDirectory ABSOLUTE_RELEASE_FOLDER -RegisterCodex
```

```sh
sh ./install.sh --version VERSION --release-directory ABSOLUTE_RELEASE_FOLDER --register-codex
```

Omit the registration option to install without changing Codex. For a published release, omitting the release-directory option downloads the matching package from GitHub. Never assume an unpublished development version has a download URL.

Alternatively, extract this native bundle and register its root as a local marketplace. For the desktop **Upload plugin archive** screen, use the separate `todo-connect-plugin-...zip`, not this native bundle.

## Connect with AI

Select To Do Connect and ask to connect your task service. The skills check existing connections first.

- Microsoft To Do: the built-in app opens browser sign-in for a personal account.
- Google Tasks: the skill helps prepare your own Cloud project, enable Tasks API and obtain a Desktop OAuth client, then imports and authorizes it.

Complete passwords, MFA and consent in the provider UI. Share a local client-file path when needed, not its contents. Working connections are reused.

## Manual setup

Replace `PROGRAM` below with the absolute path to `plugins/todo-connect/bin/todo-connect.exe` on Windows or `plugins/todo-connect/bin/todo-connect` on Unix. In PowerShell invoke a quoted path with `&`.

```text
PROGRAM version
PROGRAM connections list
PROGRAM connections add --provider microsoft --id personal --label "Personal Microsoft"
PROGRAM login personal
PROGRAM check personal
```

For a prepared Google Desktop OAuth client:

```text
PROGRAM connections add --provider google --id google-personal --label "Personal Google" --credentials ABSOLUTE_PRIVATE_JSON_PATH
PROGRAM login google-personal
PROGRAM check google-personal
```

A successful check proves read access, not every write operation. Do not recreate an existing working connection.

## Use another MCP client

Set the client's stdio command to the absolute executable path and arguments to `["serve"]`. There is no public URL or port. Clients with no skill support can use the manual setup above.

## Update, remove and protect data

Stop the old MCP process before replacing its registration. Install the new program separately and retain existing data/key settings. Check the installed version and tool discovery before removing the previous package.

For Codex marketplace replacement, follow [installation](https://github.com/Kook-Dohyun/To-Do-Connect/blob/main/docs/installation.md#update-or-remove).

Credentials are encrypted outside the package; see `RUNTIME.md`. Plugin removal does not erase them. `PROGRAM connections remove --id ID --confirm` disconnects one local account, without deleting remote tasks or revoking provider consent.

Project license: `plugins/todo-connect/LICENSE`. Dependency notices: `plugins/todo-connect/THIRD_PARTY_NOTICES.txt`.
