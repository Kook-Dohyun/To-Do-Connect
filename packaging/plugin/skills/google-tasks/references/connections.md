# Google connection setup

## Purpose

Complete the user's Google Tasks connection from its **current state**, including Cloud Console preparation when needed. A missing Desktop client JSON or disabled Tasks API is unfinished onboarding, not a reason to send the user away to prepare it alone.

The user owns the Cloud project; the program runs locally. The MCP handles safe state inspection, local import, OAuth and Tasks checks. The agent uses its available browser tools for the Cloud Console steps. Tasks OAuth does not grant Cloud project administration; do not request Cloud-management scopes or imply these MCP tools create Cloud credentials themselves.

## Process

### 1. Inspect and reuse before creating anything

Call `list_connections`, then `get_google_setup` for the selected Google connection with `verify: true`. Ask which account only when selection is ambiguous. If none exists, call `get_google_setup` without an ID. Do not ask for credentials until existing preparation is checked.

| Reported state | Continue with |
| --- | --- |
| `select_connection` | Select the intended existing Google connection; adding another account must be explicit. |
| `client_json_required` | Inspect existing Console preparation or a user-designated downloaded file; continue steps 2–4. |
| `import_required` | Import the inspected local file in step 5. File validity does not prove API activation. |
| `login_required` | User sign-in and consent in step 5. |
| `verification_required` | Repeat inspection with `verify: true`; do not log in again. |
| `api_enable_required` | Enable Tasks API in the **owning project**, then repeat read verification using saved credentials. |
| `consent_required` | Explain missing Tasks permission; explicitly reauthorize the same account with Tasks access selected. |
| `ready` | Reuse the connection. No new project, client download, import or login. |
| `check_failed` | Diagnose the reported failure. A generic 403, timeout or rate limit is not evidence of a disabled API or a need for a new OAuth client. |

Inspection returns Console links and a next action. It does not modify Cloud configuration. It can refresh an expired access token when verifying an existing session. Do not equate local credential presence or successful sign-in with Tasks access.

### 2. Work in the user's Google Cloud Console

Explain once that you will help prepare a **user-owned** project for Google Tasks and connect it locally. Confirm the account and project before creating a project or changing its configuration. Reuse a suitable existing project/client; do not modify a work or unrelated project merely because Console opened there.

Use available browser tools to inspect the current page and perform the authorized setup. Ask the user to complete sign-in, MFA, CAPTCHA and any personal/legal consent themselves, then resume from the current page. Do not invent page state or selectors. If browser control is unavailable, guide the **next concrete screen/action**, ask for its result and continue; do not simply say “get a JSON first.”

For a new personal-use project, offer a descriptive name such as `To Do Connect Personal`; Google chooses a unique project ID. Record the actual selected ID, not its display name. Do not require billing, card entry, organization creation, a paid plan, unrelated APIs or permissions. If the observed environment demands one, explain the precise requirement and ask for direction before expanding scope.

### 3. Enable API and configure the user's OAuth app

In the selected project's API Library, open **Google Tasks API**. If disabled, enable it within the agreed setup scope; if already enabled, leave it unchanged. Verify its enabled/manage state.

Inspect **Google Auth Platform**:

- **Branding:** If unconfigured, help fill the app name, user-selected support email and developer contact. Let the user review and accept Google's terms. Preserve an existing suitable configuration.
- **Audience:** A personal Google account uses **External**. If the app is in **Testing**, add the intended account as a test user when absent.
- **Data Access:** This program requests `https://www.googleapis.com/auth/tasks` and `openid`. Do not add Gmail, Drive, Calendar or Cloud administration permissions.
- **Clients:** Reuse a suitable **Desktop app** OAuth client or create one in this same project. A web client, API key and service account JSON are not substitutes. Download its client JSON.

Testing with Tasks scope has a seven-day refresh-token lifetime. Explain this limitation for ongoing personal use; discuss production status separately. Do not publish the OAuth app or submit public verification without the user's decision. Publishing status and verification are different.

### 4. Handle the downloaded file without making it a prerequisite

Identify the actual browser download path. If the tools cannot report it, ask the user only for its local path or help locate the known download; never ask them to paste the JSON. Use a private local location outside Git, the plugin directory and synced folders. The repository's developer `config/` folder is **not a user installation requirement**.

Call `get_google_setup` with `credentials_path` to validate Desktop-client format and expose only nonsecret setup metadata. Confirm the returned `project_id` matches the project whose API was enabled. If an older imported connection has no project ID, inspect its matching original JSON with both `connection_id` and `credentials_path`; a client mismatch is rejected. Without that file, identify the owning project in Console rather than assuming the current project or making the user redownload a working client unnecessarily.

Do not print file contents, client secrets, login URLs/codes or tokens into chat/logs. Do not include them in the repository, plugin bundle or container image. Leave the source download unchanged; deleting it is a separate user decision.

### 5. Import, sign in, verify and resume

1. Call `import_google_connection` with a new `connection_id`, user-recognizable `label`, absolute `credentials_path` and `confirm: true` only for the user-requested connection. Existing Microsoft and Google connections remain intact. Import encrypts the client settings; it does not authenticate.
2. Tell the user a Google browser sign-in will open. Call `login_google_connection` with the chosen ID and `confirm: true`. The user completes password, MFA, account selection and permission consent; the program receives a loopback callback using PKCE/state. The tool waits for up to ten minutes and then checks Tasks access.
3. For an unverified-app warning, first verify the app/developer belong to the user's own project and review requested permissions. Only then explain the personal-use continuation option. Never coach bypass of unknown-app warnings. If continuation is unavailable, diagnose the actual block. On granular consent, Google Tasks access must be selected; sign-in alone is not authorization.
4. If `api_enable_required` is returned, continue step 3 for that client’s project, **preserving the connection/token**. After enablement, use `get_google_setup` with the ID and `verify: true`; do not reimport or reauthenticate. Allow for propagation if the same recognized disabled-API response remains, and recheck after a bounded wait rather than looping blindly.
5. If the user cancels or the host times out, inspect current connection state before retrying. Reuse a token if the callback already completed; wait if the connection remains busy. For an actual revoked/expired authorization or missing Tasks permission, explain the reason and use `reauthenticate: true` only for a user-approved retry. Different accounts need distinct connections.

Do not claim the Console steps or live authorization succeeded merely because the tool schemas or synthetic tests passed.

## CLI fallback and Docker

Prefer MCP tools for normal desktop setup. When the host cannot keep an interactive login call alive, resolve `../../bin/todo-connect.exe` (Windows) or `../../bin/todo-connect` (macOS/Linux) **relative to this skill's SKILL.md directory**, not this reference or the working directory. Invoke the absolute path; the installer does not require PATH, Go or Node. Check `version`. A missing bundled binary is an installation issue, not a reason to download an arbitrary executable.

Run `PROGRAM login ID` in a user-visible terminal without copying sensitive output into chat, then `get_google_setup` with `verify: true`. Explicit same-account reauthorization uses `PROGRAM login ID --reauthenticate`. Do not launch a second login while the first holds the connection lock.

For Docker, browser callbacks and filesystem paths belong to the container host, not necessarily the agent's computer. Follow the bundled RUNTIME.md: persist /data and /keys, import client JSON through stdin with `connections add ... --credentials -`, and use `login ID --headless --container-port 8765` with host publication `127.0.0.1:8765:8765`. The user opens the URL locally on that host. Do not expose the callback publicly, make the JSON world-readable, or report a native path as container-readable. Normal stdio MCP has no published port.

## Verification

Completion requires `state: ready` / `read_access: verified` from the selected connection, not a downloaded file, successful login page or Console screenshot alone. An empty list is a valid successful read. Report the selected connection and verified capability without dumping the user's tasks. Test writes only when requested.

On subsequent calls, use the same connection; reconnect only for a demonstrated need. Local disconnection removes local credentials, not remote tasks or Google consent. Do not change encryption mode or regenerate a lost storage key to work around a failure.

Official references: [Tasks setup](https://developers.google.com/workspace/tasks/quickstart/go), [consent and scopes](https://developers.google.com/workspace/guides/configure-oauth-consent), [Desktop credentials](https://developers.google.com/workspace/guides/create-credentials), [OAuth/PKCE](https://developers.google.com/identity/protocols/oauth2/native-app), [testing audience](https://support.google.com/cloud/answer/15549945).
