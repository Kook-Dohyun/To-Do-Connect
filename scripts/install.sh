#!/bin/sh
# Install from GitHub Releases or a trusted local release, without Go, Node or Docker.
set -eu

fail() { printf '%s\n' "$*" >&2; exit 1; }
release_directory=
version=
install_directory=
register_codex=false
while [ "$#" -gt 0 ]; do
    case "$1" in
        --release-directory|--version|--install-directory)
            [ "$#" -ge 2 ] || fail "Missing value for $1"
            case "$1" in
                --release-directory) release_directory=$2 ;;
                --version) version=$2 ;;
                --install-directory) install_directory=$2 ;;
            esac
            shift 2 ;;
        --register-codex) register_codex=true; shift ;;
        *) fail "Unknown argument: $1" ;;
    esac
done
[ -n "$version" ] || fail 'Required: --version VERSION (optional: --release-directory PATH for offline installation)'
printf '%s\n' "$version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$' || fail 'Invalid release version'
case "$(uname -s)" in
    Linux) target_os=linux ;;
    Darwin) target_os=darwin ;;
    *) fail 'Supported systems: macOS and Linux. Use install.ps1 on Windows.' ;;
esac
native_arch=$(uname -m)
if [ "$target_os" = darwin ] && [ "$(sysctl -n hw.optional.arm64 2>/dev/null || true)" = 1 ]; then
    native_arch=arm64
fi
case "$native_arch" in
    x86_64|amd64) target_arch=amd64 ;;
    aarch64|arm64) target_arch=arm64 ;;
    *) fail "Unsupported architecture: $native_arch" ;;
esac
command -v unzip >/dev/null 2>&1 || fail 'The OS unzip utility is required to extract this release; nothing was installed.'
if [ "$target_os" = darwin ]; then
    command -v shasum >/dev/null 2>&1 || fail 'The OS shasum utility is required.'
    hash_file() { shasum -a 256 "$1" | awk '{print $1}'; }
else
    command -v sha256sum >/dev/null 2>&1 || fail 'The OS sha256sum utility is required.'
    hash_file() { sha256sum "$1" | awk '{print $1}'; }
fi
if [ "$register_codex" = true ]; then
    command -v codex >/dev/null 2>&1 || fail 'Codex CLI was not found; nothing was installed.'
fi
archive_name="todo-connect-$version-$target_os-$target_arch.zip"
download_directory=
cleanup_download() {
    if [ -n "$download_directory" ]; then
        rm -f -- "$download_directory/SHA256SUMS" "$download_directory/$archive_name"
        rmdir -- "$download_directory"
    fi
}
if [ -z "$release_directory" ]; then
    command -v curl >/dev/null 2>&1 || fail 'The OS curl utility is required for GitHub downloads. Use --release-directory for offline installation.'
    download_directory=$(mktemp -d "${TMPDIR:-/tmp}/todo-connect-download.XXXXXXXX")
    trap cleanup_download EXIT
    trap 'exit 1' HUP INT TERM
    release_url="https://github.com/Kook-Dohyun/To-Do-Connect/releases/download/v$version"
    for asset in SHA256SUMS "$archive_name"; do
        curl --proto '=https' --proto-redir '=https' --tlsv1.2 --fail --location --silent --show-error --connect-timeout 15 --max-time 180 --output "$download_directory/$asset" "$release_url/$asset" || fail "Release download failed: $asset; nothing was installed."
    done
    release_directory=$download_directory
fi
release_directory=$(cd "$release_directory" && pwd -P)
archive="$release_directory/$archive_name"
expected=$(awk -v name="$archive_name" '{sub(/\r$/, "")} $2 == name {print $1}' "$release_directory/SHA256SUMS")
[ "${#expected}" -eq 64 ] || fail 'Release checksum entry is missing or duplicated.'
case "$expected" in *[!0-9a-fA-F]*) fail 'Invalid checksum entry' ;; esac
[ "$(hash_file "$archive")" = "$expected" ] || fail 'Release checksum mismatch; nothing was installed.'
if [ -z "$install_directory" ]; then
    if [ "$target_os" = darwin ]; then
        install_directory="$HOME/Library/Application Support/ToDoConnect/$version"
    else
        install_directory="${XDG_DATA_HOME:-$HOME/.local/share}/todo-connect/programs/$version"
    fi
fi
case "$install_directory" in /*) ;; *) fail 'Install directory must be absolute' ;; esac
[ ! -e "$install_directory" ] && [ ! -L "$install_directory" ] || fail "Install destination already exists and was not modified: $install_directory"

# Only contained paths are accepted before extraction. No existing installation
# is deleted; incomplete output remains for inspection if extraction fails.
entries=$(unzip -Z1 "$archive") || fail 'Cannot inspect release archive'
printf '%s\n' "$entries" | awk '
    /^\// || /\\/ || /(^|\/)\.\.(\/|$)/ {exit 1}
' || fail 'Release archive contains an unsafe path'
umask 077
mkdir -p "$(dirname "$install_directory")"
mkdir "$install_directory"
unzip -q "$archive" -d "$install_directory"
program="$install_directory/plugins/todo-connect/bin/todo-connect"
[ -f "$program" ] && [ ! -L "$program" ] && [ -x "$program" ] || fail 'Extracted program is missing or has no executable permission'
[ "$("$program" version)" = "$version" ] || fail 'Installed executable did not report the expected version'
if [ "$register_codex" = true ]; then
    codex plugin marketplace add "$install_directory" --json || fail "Program installed, but marketplace registration failed: $install_directory"
    codex plugin add todo-connect@todo-connect-local --json || fail "Marketplace registered, but plugin installation failed: $install_directory"
fi
printf 'Version: %s\nTarget: %s/%s\nProgram: %s\nMarketplace directory: %s\n' "$version" "$target_os" "$target_arch" "$program" "$install_directory"
