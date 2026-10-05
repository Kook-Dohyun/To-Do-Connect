# Distribution verification

The **Verify distributable packages** workflow is manually triggered with a version without the `v` prefix. It builds and tests a candidate; it does not publish a release, push a container image, authenticate a real account, or modify user tasks. Repository access is read-only.

## Jobs

1. **Package:** build Windows, macOS and Linux archives for amd64 and arm64, including separate single-root plugin ZIPs.
2. **Native (six hosts):** run Go tests and vet, inspect archives, start the matching executable, exercise stdio MCP and an isolated Codex test host, and test installers with synthetic connections.
3. **Docker (two Linux hosts):** run race checks, build runtime images, test encrypted persistence and MCP, export and reload each image, and retest it.

The host test installs a pinned Codex CLI into a temporary directory. Node/npm are CI dependencies for that test host, not distributed-program requirements. No LLM turn or provider login is started.

Separate opt-in tests exercise macOS Keychain and Linux Secret Service using synthetic credentials and temporary unlocked key stores. They remove their test keys, but do not test interactive desktop unlock prompts or account recovery.

## Candidate files

`release-candidate` contains six native bundles, six plugin ZIPs, two installers and their checksum lists. Each `docker-candidate-linux-<arch>` artifact contains a tested compressed image and its own checksum list. Artifacts are retained for one day and are **not GitHub Releases**.

Keep checksum lists separate when downloading Actions artifacts. Before publishing flat GitHub Release assets, combine entries into one `SHA256SUMS` covering the selected files. Do not upload different files with the same asset name. Recheck all hashes.

## Evidence boundaries

A passing run does not establish real OAuth consent, provider reads/writes, independent first-time onboarding, interactive key-store behavior, or public-directory acceptance. Simulated installer downloads do not verify downloads from a published release. The stopped-host upgrade test is opt-in and requires two candidates.

Use the workflow run associated with the exact release commit as evidence. Do not transfer older results to a new build. Release notes should contain public build evidence, not personal account details or local development-session logs.
