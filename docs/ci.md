# Verification

## Local checks

Run unit tests and vet for both Go modules as described in [development](development.md). Tests use temporary synthetic account data; they do not log in to personal accounts.

For a built candidate, set:

| Environment variable | Enables |
| --- | --- |
| `TODO_CONNECT_RELEASE_DIR` | Archive, executable and installer tests |
| `TODO_CONNECT_TEST_CODEX` | Isolated Codex host discovery and harmless tool calls |
| `TODO_CONNECT_PREVIOUS_RELEASE_DIR` | Old/new/old installation test with a synthetic connection |
| `TODO_CONNECT_TEST_IMAGE` | Tests against a prepared local Docker image |
| `TODO_CONNECT_TEST_OS_KEYRING=1` | Real OS key-store test with synthetic data |

Unset these overrides after testing. Never point isolated tests at the user's credential directory.

## Distribution workflow

The manually triggered **Verify distributable packages** workflow accepts a candidate version. It does not publish a GitHub release, push images or submit a public plugin.

- **Package:** builds native and plugin ZIPs for Windows, macOS and Linux, each on amd64 and arm64.
- **Native:** tests the matching executable on six runners, validates archives, installers, MCP discovery and an isolated pinned Codex host.
- **Docker:** runs Linux race checks, builds images on two architectures, tests persistence and MCP, exports/reloads images and repeats checks.
- **OS key stores:** test macOS Keychain and Linux Secret Service with temporary synthetic entries.

Node/npm install the test host in CI; they are not requirements for users of the native program.

Artifacts are temporary candidates, not published releases. Check the workflow's configured retention period before relying on a download.

## What each result proves

1. A Go test proves only the behavior exercised by that test.
2. A successful build and archive check prove packaging, not OAuth or host compatibility.
3. A stdio `tools/list` response proves what the server advertises.
4. Codex host discovery proves what that host registered.
5. A model-visible tool inventory and successful harmless call prove availability in that conversation.
6. Live provider checks and user-requested mutations test the account integration separately.

Do not infer missing-tool causes from an undocumented numerical limit. Compare the actual catalog with the host and conversation inventories.

For a fresh-user onboarding test, isolate account storage deliberately and leave the user's existing connections untouched. Verify both an initially empty state and reuse of a working connection. Report untested browser/OS paths explicitly.

Record public CI evidence against the exact source commit. Keep personal account details, downloaded client files and local diagnostic logs out of commits and release notes.
