# Development packaging and release

A source commit, a local package, an installed plugin, a GitHub Release and an OpenAI directory listing are different states. This project is in development; do not label a local build as published or stable.

## Build a candidate

From the repository root:

```powershell
./scripts/package-release.ps1 -Version '0.2.0-dev.18'
```

Choose a new version for changed package contents. Existing output directories are not overwritten. The version is embedded in the executable, plugin manifest and `BUILD.json`.

Under `dist/releases/VERSION/`:

| Output | Purpose |
| --- | --- |
| Six native ZIPs | Program, local marketplace, licenses and setup guide |
| `plugin-zips/` with six plugin ZIPs | Platform-specific plugin upload |
| `install.ps1`, `install.sh` | Native installers |
| Checksum files | Integrity checks for each archive group |

The native package contains `plugins/todo-connect/`, `.agents/plugins/marketplace.json`, `BUILD.json`, `GETTING_STARTED.md` and `RUNTIME.md`. The plugin ZIP has the plugin contents directly at its root. Both include an executable and two skills, not account data.

The current packaging script also creates intermediate platform binaries in `dist/VERSION/` and expanded bundles alongside native ZIPs. These are build intermediates, not extra versions or files to publish. Keep only verified final archives when preparing a distribution.

## Verify and install locally

Follow [verification](ci.md). Inspect file allowlists, checksums, versions, permissions, manifests, skills, icons and tool schemas. A package must contain no private connection data or development diary.

Use [installation](installation.md#update-or-remove) to update the local host. Recheck its installed version and all 27 tools. Preserve existing accounts and test first-run behavior with separate synthetic storage.

## Publish source or release assets

Review the exact staged files and commit history before pushing. Public source excludes build output, local configuration, credentials and personal plans. Ignoring a path does not remove an already tracked file or its history.

For an authorized GitHub Release, select the verified commit, create an immutable tag and upload the intended native/plugin/Docker assets. Merge their checksum entries into a single release `SHA256SUMS` when publishing a flat asset list. Mark development candidates as prereleases.

Download published assets afresh and verify their hashes and installation. Do not overwrite an existing version with different bytes.

## Public plugin directory

A local stdio MCP works on the user's machine. Account upload and public-directory acceptance follow separate OpenAI processes. Consult [official submission guidance](https://developers.openai.com/plugins/deploy/submission) for the chosen distribution path.

Do not introduce a hosted relay, change account scope or upload a new account-plugin version merely to publish source on GitHub.
