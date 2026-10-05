# Google Tasks connection setup

## Purpose

Connect the selected Google account using a user-owned Cloud project and Desktop OAuth client, then verify Tasks access. Help complete unfinished setup rather than requiring the user to arrive with credentials already prepared.

## Inspect current setup

Use `list_connections`, then `get_google_setup` with the selected ID and `verify: true`. If no Google connection exists, omit the ID. Reuse existing projects, clients and working connections.

| State | Next action |
| --- | --- |
| `select_connection` | Choose the intended existing connection |
| `client_json_required` | Continue the Cloud Console steps below |
| `import_required` | Import the inspected local file |
| `login_required` | Open browser authorization |
| `verification_required` | Call again with `verify: true` |
| `api_enable_required` | Enable Tasks API in the client's project; keep saved tokens |
| `consent_required` | Request same-account reauthorization with Tasks permission |
| `ready` | Resume the task without setup |
| `check_failed` | Diagnose the returned error without recreating the connection |

The tool returns project-specific Console links and safe metadata. It does not administer Google Cloud projects.

## Connect

### Prepare the user's Cloud project

Use available browser controls to guide the following steps. Confirm the intended account and project before creating or changing a project. Reuse suitable existing configuration. If browser controls are unavailable, guide the next screen and continue after the user's result.

1. Open the [Google Cloud Console](https://console.cloud.google.com/). Let the user complete login/MFA. Select an existing project or create a project for this connection.
2. Open the selected project's **Google Tasks API** page and enable the API if disabled.
3. In **Google Auth Platform**, configure **Branding** with the user's chosen app name, support email and contact details if not already configured.
4. Configure **Audience** for the account type. Personal Google accounts use External; in Testing, add the intended account as a test user.
5. Configure the required access: `https://www.googleapis.com/auth/tasks` and `openid`.
6. Under **Clients**, reuse or create a **Desktop app** OAuth client in that project and download its JSON. An API key, web client or service-account key is not a substitute.

The user handles passwords, MFA and legal/permission consent. This workflow does not require unrelated APIs or a paid project; if the actual Console requests billing or broader organizational changes, explain the requirement before proceeding.

For External apps in Testing, refresh tokens with Tasks access generally expire after seven days. Explain this when relevant to recurring use; changing publishing status is a separate user decision, not a silent setup step.

### Import and authorize

1. Locate the downloaded JSON outside the repository/plugin. Ask for its local path if needed, never its contents.
2. Call `get_google_setup` with the absolute `credentials_path`. Match its `project_id` to the project whose API was enabled.
3. Call `import_google_connection` with a unique `connection_id`, recognizable `label`, `credentials_path` and `confirm: true` for the requested connection. Import encrypts client settings; it is not login.
4. Call `login_google_connection` with that ID and `confirm: true`. It opens the system browser, waits for consent and verifies Tasks access.
5. If Google displays an unverified-app warning, confirm it is the user's own OAuth app and review requested access before discussing continuation. Do not dismiss warnings for an unknown app.

The import leaves the original download unchanged. Do not print or upload client secrets, authorization codes or tokens.

## Verify and resume

Success is `state: ready` / `read_access: verified` for the chosen connection. A downloaded file or successful login page alone is not sufficient.

If the API is disabled, enable it in the owning project and call `get_google_setup` with `verify: true` again; reuse the saved authorization. If a login call times out, inspect state before restarting it. Reauthorization uses `reauthenticate: true` only for a requested same-account retry after a demonstrated consent/authentication problem.

Resume the user's task after verification. An empty list is a valid read result. Do not create test tasks unless requested.

## Storage and disconnection

Credentials are encrypted outside the plugin in the user's `todo-connect` application-data directory. An explicit `TODO_CONNECT_DATA_DIR` must match across CLI and MCP.

Use `disconnect_connection` with `connection_id` and `confirm: true` for a requested disconnection. Local metadata and credentials are removed; provider tasks, consent and the original JSON download remain. Do not change another connection.

For Docker/headless use, see the package's `RUNTIME.md` or [runtime guide](https://github.com/Kook-Dohyun/To-Do-Connect/blob/main/docs/runtime.md). Desktop setup normally uses the MCP tools above.

[Consent configuration](https://developers.google.com/workspace/guides/configure-oauth-consent) · [Desktop credentials](https://developers.google.com/workspace/guides/create-credentials) · [OAuth](https://developers.google.com/identity/protocols/oauth2/native-app)
