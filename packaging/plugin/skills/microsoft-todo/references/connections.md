# Microsoft To Do connection setup

## Purpose

Connect the selected Microsoft personal account and verify that its To Do lists can be read. The public application ID is included in the program; no client secret, environment variable or new app registration is required for ordinary users.

## Inspect current setup

Use `list_connections`. Reuse the intended Microsoft connection when present. Credential presence is not access verification: use `check_connection` when needed. Add a new connection only for a requested additional account.

## Connect

This version uses the bundled CLI for Microsoft connection creation and browser login. Task operations remain MCP calls.

Resolve `../../bin/todo-connect.exe` on Windows or `../../bin/todo-connect` on macOS/Linux relative to the directory containing this skill's `SKILL.md`. Use that absolute executable path as `PROGRAM`; do not assume it is on PATH. In PowerShell, invoke a quoted path with `&`.

1. Check `PROGRAM version`.
2. Choose a unique lowercase connection ID and a recognizable label.
3. Run `PROGRAM connections add --provider microsoft --id personal --label "Personal Microsoft"`, substituting that ID and label.
4. Run `PROGRAM login personal`. The program opens the system browser and receives the local OAuth callback.
5. Let the user complete account selection, password, MFA and permission consent in Microsoft's UI.

Do not capture credentials or one-time codes in chat. Device-code login is an explicit alternative, `PROGRAM login personal --device`, run in a terminal visible to the user; do not silently switch to it.

The built-in registration uses personal accounts, Tasks.ReadWrite and a desktop `http://localhost` redirect. An invalid redirect is an application-registration problem, not a reason to ask an ordinary user to register a new app. If the user explicitly supplies a compatible app registration, `connections add --client-id ID` selects it.

## Verify and resume

Call `check_connection` with the selected ID. A successful read, including an empty collection, establishes read access. Resume the user's requested task; do not create test tasks unless requested.

If authorization is revoked or expired, explain the error and use `PROGRAM login personal --reauthenticate` for a requested same-account sign-in. Use a separate connection for another account. A timeout or network failure alone does not justify repeating registration or login.

## Storage and disconnection

The CLI and MCP must use the same data-directory and key settings. Credentials are encrypted outside the plugin: Windows Local AppData, macOS Application Support or Linux XDG config, under `todo-connect`. `TODO_CONNECT_DATA_DIR` can explicitly select another absolute path.

Use `disconnect_connection` with `connection_id` and `confirm: true` for a requested disconnection. Local metadata and credentials are removed; provider tasks and consent remain. Do not change another connection.

For Docker/headless use, see the package's `RUNTIME.md` or [runtime guide](https://github.com/Kook-Dohyun/To-Do-Connect/blob/main/docs/runtime.md). Use the same persistent data/key volumes for setup and MCP.

[Desktop authentication](https://learn.microsoft.com/en-us/entra/identity-platform/scenario-desktop-app-overview)
