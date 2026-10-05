# Development

## Repository layout

```text
server/                    Go runtime module
  cmd/todo-connect/        CLI/MCP entrypoint and executable tests
  internal/app/            Providers, authentication, storage and MCP tools
tools/                     Standard-library-only packaging utilities
packaging/
  plugin/                  Manifest, logo, MCP configuration and two skills
  docker/                  Container build
scripts/                   Build, packaging and installation commands
docs/                      Public product and contributor documentation
dist/                      Ignored generated builds and packages
README.md                  English introduction
README.ko.md               Korean introduction
```

The repository is the product; `server/` is its Go module. One executable provides CLI setup and the stdio MCP server for both providers. `tools/` is a separate module for archive creation and dependency-license collection.

There is no repository credential directory. User data belongs outside source and packages; see [runtime](runtime.md).

## Build and test

Use the Go version in `server/go.mod` (currently 1.25.13 or later) and PowerShell for the scripts. From the repository root:

```powershell
./scripts/build.ps1
./dist/dev/windows-amd64/todo-connect.exe catalog
go -C server test ./... -count=1
go -C server vet ./...
go -C tools test ./... -count=1
go -C tools vet ./...
```

The executable path above is Windows x64. `build.ps1` chooses `dist/dev/<os>-<arch>/` from Go's target; Unix executable names have no `.exe`. `build-targets.ps1` builds all six OS/CPU combinations. Root `bin/` is not used.

Build directly if needed:

```text
go -C server build -o ../dist/dev/windows-amd64/todo-connect.exe ./cmd/todo-connect
```

Use the appropriate target path and environment for cross-compilation. Cross-building is not runtime verification.

## Runtime entrypoints

`server/cmd/todo-connect/main.go` calls `app.Run`. Useful commands:

| Command | Purpose |
| --- | --- |
| `version` | Report executable build version |
| `catalog` | Print the registered tool schemas without provider access |
| `serve` | Run stdio MCP |
| `connections list` | Inspect local connection metadata |
| `connections add` | Add a provider connection |
| `login ID` | Browser authentication |
| `check ID` | Read-only provider access check |
| `connections remove --id ID --confirm` | Remove one local connection |
| `keygen --out ABSOLUTE_PATH` | Create a key for explicitly selected key-file storage |

Run the program without arguments for complete CLI usage. Do not run provider operations against a user's account as a substitute for synthetic tests.

## MCP contract

`server/internal/app/tools.go` registers the server and Microsoft tools. `google_tasks.go` registers Google task operations; `google_setup.go` registers onboarding.

The SDK generates the JSON-RPC `tools/list` response from these registrations. It is not a hand-maintained JSON file. The current contract is 27 tools: 12 Microsoft, 9 Google task operations, 3 connection-management and 3 Google onboarding tools.

Use operation-specific schemas and retain only inputs the operation consumes. Tests compare the catalog, actual stdio discovery and Codex host discovery. The model's available tool set must be checked separately from server registration.

## Skills and documentation

Both provider skills use the same structure: purpose, capabilities, connection selection, task execution and verification. Each has `references/connections.md` and `references/API.md`. Keep provider-specific instructions self-contained and refer to actual tool names.

Maintain both README languages together. Public docs describe the product, not a maintainer's personal accounts, workstation, OAuth IDs or troubleshooting diary. The built-in public Microsoft application ID is part of the software, not a private token.

## Packaging

See [packaging](releases.md) and [verification](ci.md). The source manifest is in `packaging/plugin/plugin.json`; packaging sets the executable's relative MCP path and bundles the skills and logo. Do not edit an installed plugin cache as source.
