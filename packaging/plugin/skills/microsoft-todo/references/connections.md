# Local Microsoft connections — development build

## Purpose

Connect an existing Microsoft personal account to the program on the user's PC. Each connection has a stable local ID and its own encrypted cache. No developer-operated relay server is involved.

## Locate the bundled CLI when setup is needed

Routine task operations use the MCP tools and do not need a CLI lookup. For adding a connection or interactive login, resolve `../../bin/todo-connect.exe` on Windows or `../../bin/todo-connect` on macOS/Linux **from the directory containing this skill's `SKILL.md`**, not the current working directory or this reference file. Use its absolute path in place of `todo-connect` below. The installer does not modify PATH; do not install Go/Node or search for an unrelated global executable to compensate. In PowerShell invoke a quoted executable path with `&`. Check `version` before setup. If the bundled executable is absent, report that the installed package is incomplete rather than silently building source or downloading an arbitrary binary.

## Process

1. Inspect `list_connections` through MCP, or `todo-connect connections list`. If the desired connection exists, use it. A `credentials_stored_not_verified` state only means a cache file exists. `check_connection` / `todo-connect check ID` tests read access. A transient network failure is not proof that a new app or login is needed.
2. Only when the user wants a new connection, choose a unique lowercase ID and a meaningful label. Run `todo-connect connections add --provider microsoft --id personal --label "Personal Microsoft"`. The public Microsoft client ID is built in; an optional `--client-id` selects the user's own compatible personal-account registration. No client secret is required.
3. Run `todo-connect login personal` in a user-visible local terminal. The program uses MSAL's system-browser authorization-code flow with PKCE. The app registration must have a **Mobile and desktop applications** redirect URI `http://localhost`. This setting has not yet been live-verified for the bundled registration. Do not promise a working browser login until it succeeds. Password, MFA and consent stay in Microsoft's UI.
4. `todo-connect login personal --device` explicitly selects the previously used device-code flow. It requires the app's public-client flow setting. The short code is shown in the user's terminal only, never returned through MCP or copied into chat. Do not silently fall back to this flow or run login in captured automation logs.
5. Existing credentials are not replaced by ordinary login. For a confirmed expired/revoked connection, `todo-connect login personal --reauthenticate` explicitly requests reauthentication; select the same Microsoft account. Add another connection to use a different account.

Add another connection with another ID to use accounts together. Do not disconnect the first to connect the second. Microsoft task tools have a `microsoft_` prefix and require `connection_id` on every call.

Prototype migration, if the user requests retaining an earlier login: add a destination with the same public client ID, then run `todo-connect connections import-microsoft --id personal --from "ABSOLUTE_ENCRYPTED_CACHE_PATH"`. This locally validates one cached account and copies encrypted bytes; it does not read tasks or remove the source. Do not guess that an AppData path is visible from every execution host: Windows packaged apps may redirect it. Never display the cache contents.

## Verification and disconnection

Run `todo-connect check personal`, or `check_connection` with that ID. Success means a read-only To Do list request completed. It does not verify task writes, all API methods, another account, or provider consent revocation.

When explicitly asked to disconnect, use `disconnect_connection` with the selected ID and `confirm: true`, or `todo-connect connections remove --id personal --confirm`. Only local metadata and encrypted credentials are removed. A small lock file remains to coordinate processes. Provider tasks and consent are unchanged.

Default data lives outside the project under the OS user's `todo-connect` application-data directory (Windows: Local AppData, possibly redirected by a packaged host). An absolute `TODO_CONNECT_DATA_DIR` can make an explicitly chosen data directory consistent across execution hosts. Do not place it in a repository, plugin package or synchronized Vault. Use the same data directory when checking a connection from the CLI and from MCP.

Windows uses DPAPI by default. macOS/Linux use an OS keyring-held AES-GCM key (macOS Keychain / Linux Secret Service); these desktop backends still need live validation. Containers use an explicitly selected 32-byte key file through `TODO_CONNECT_KEY_FILE`, generated once with `todo-connect keygen --out ABSOLUTE_PRIVATE_PATH`. Keep the key and encrypted data in persistent private storage outside the package. Use the same storage selection for CLI and MCP. Do not generate a replacement key over an existing one, change storage modes on an existing connection, or fall back to plaintext. Windows DPAPI files are not portable to a container.

For a local Docker container, persist `/data` and `/keys` across commands and use `login ID --device` in the user's terminal; no callback port is needed for Microsoft. MCP uses `docker run --rm -i` with those same volumes and `serve`, without a TTY or published port. Container startup and encrypted persistence passed synthetic tests; real sign-in and final installation UX are still pending. Google connections coexist with Microsoft connections and have separate setup instructions in the Google Tasks skill. Report these separately from passing local tests.
