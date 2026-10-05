# Release packaging

Tags use `v<version>`. Executable, plugin manifest and `BUILD.json` versions must agree. Never overwrite a published version with different bytes.

## Build a candidate

From the repository root, with Go and PowerShell available:

```powershell
./scripts/package-release.ps1 -Version '0.2.0-beta.1'
```

This is a maintainer command, not an end-user installation step. An existing release directory is rejected. Packaging copies selected executables, plugin files, license notices and user documentation—not the source tree or user credential directories.

Output under `dist/releases/<version>/`:

| Artifact | Purpose |
| --- | --- |
| `todo-connect-<version>-<os>-<arch>.zip` × 6 | Native program, local marketplace and documentation |
| `plugin-zips/todo-connect-plugin-<version>-<os>-<arch>.zip` × 6 | Single-root ZIP for Codex plugin upload |
| `install.ps1`, `install.sh` | Optional native installers |
| `SHA256SUMS`, `plugin-zips/SHA256SUMS` | Checksums for the two groups |

The regular bundle contains `.agents/plugins/marketplace.json`, `plugins/todo-connect/`, `BUILD.json`, `GETTING_STARTED.md` and `RUNTIME.md`. The plugin ZIP contains `plugins/todo-connect/` contents directly at its root. Both contain a platform executable and two provider skills, but no account data.

Docker image archives are produced separately by CI. They contain the image filesystem, not account-data volumes. See [runtime instructions](runtime.md).

## Verify before publication

Run [the distribution workflow](ci.md) against the exact candidate source. Local unit and static checks:

```text
go -C server test ./... -count=1
go -C server vet ./...
go -C tools test ./... -count=1
go -C tools vet ./...
```

Set `TODO_CONNECT_RELEASE_DIR` to a candidate directory to enable archive tests. Set `TODO_CONNECT_TEST_CODEX` to an existing Codex CLI to include isolated host checks. Tests use temporary synthetic data, not the maintainer's connected accounts.

Check archive entries, platform metadata, executable permissions, versions, checksums and dependency notices. Inspect text and binary contents for unintended local paths or credentials. A cross-build alone does not establish platform-specific login behavior.

## Publish

1. Review public source and history. Exclude private plans, setup diaries, credentials and account caches. `.gitignore` does not remove previously tracked files or Git history.
2. Select a verified commit and immutable version. Mark an initial preview as a prerelease, not stable.
3. Assemble native, plugin and Docker assets. GitHub Release assets have a flat namespace: merge checksums into one `SHA256SUMS` and verify the full set. The installer expects that name.
4. Publish only after the contents are approved. An Actions artifact is not yet a public download.
5. Download published assets afresh, compare hashes and verify installation in an isolated environment. Never reset working user accounts for this check.

Do not commit `dist/`. Release notes describe functionality, installation, known limitations and that version's test evidence.

## Public plugin directory

GitHub distribution and Codex ZIP installation do not imply listing in OpenAI's public directory. Current [official packaging guidance](https://developers.openai.com/plugins/build/plugins) directs public MCP submissions to remote HTTPS and asks local MCP developers to contact OpenAI for support when that is not possible.

To Do Connect remains user-local; resolve directory eligibility without silently introducing a developer-operated relay.
