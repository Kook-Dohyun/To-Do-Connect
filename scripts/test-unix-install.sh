#!/bin/sh
# Developer regression test in an ephemeral container or private test workspace.
set -eu
release_directory=$1
version=$2
installer=$3
test_root=$4
[ ! -e "$test_root" ] || { printf 'Test root already exists\n' >&2; exit 1; }
mkdir -p "$test_root"
destination="$test_root/application with spaces"
sh "$installer" --release-directory "$release_directory" --version "$version" --install-directory "$destination"
program="$destination/plugins/todo-connect/bin/todo-connect"
[ "$("$program" version)" = "$version" ]
[ "$("$program" catalog | jq '.tools | length')" = 63 ]
if sh "$installer" --release-directory "$release_directory" --version "$version" --install-directory "$destination"; then
    printf 'Installer overwrote existing destination\n' >&2
    exit 1
fi
case "$(uname -m)" in x86_64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; *) exit 1 ;; esac
case "$(uname -s)" in Darwin) target_os=darwin ;; Linux) target_os=linux ;; *) exit 1 ;; esac
if [ "$target_os" = darwin ] && [ "$(sysctl -n hw.optional.arm64 2>/dev/null || true)" = 1 ]; then arch=arm64; fi
bad_release="$test_root/bad-release"
mkdir "$bad_release"
archive="todo-connect-$version-$target_os-$arch.zip"
printf '%064d  %s\n' 0 "$archive" > "$bad_release/SHA256SUMS"
printf 'corrupt archive' > "$bad_release/$archive"
if sh "$installer" --release-directory "$bad_release" --version "$version" --install-directory "$test_root/must-not-exist"; then
    printf 'Installer accepted corrupt release\n' >&2
    exit 1
fi
[ ! -e "$test_root/must-not-exist" ]
# Exercise the exact network command contract without publishing a GitHub release.
mkdir "$test_root/mock-bin" "$test_root/download-temp"
cp "$(dirname "$0")/test-download-curl.sh" "$test_root/mock-bin/curl"
chmod 700 "$test_root/mock-bin/curl"
export PATH="$test_root/mock-bin:$PATH" TMPDIR="$test_root/download-temp"
export TEST_VERSION="$version" TEST_OS="$target_os" TEST_ARCH="$arch" TEST_RELEASE="$release_directory" TEST_DOWNLOAD_LOG="$test_root/downloads.log"
sh "$installer" --version "$version" --install-directory "$test_root/from-download"
[ "$(wc -l < "$TEST_DOWNLOAD_LOG")" -eq 2 ]
[ -z "$(ls -A "$TMPDIR")" ]
export TEST_DOWNLOAD_FAILURE="$archive"
if sh "$installer" --version "$version" --install-directory "$test_root/failed-download"; then
    printf 'Installer accepted failed download\n' >&2; exit 1
fi
[ ! -e "$test_root/failed-download" ]
[ -z "$(ls -A "$TMPDIR")" ]
printf 'Unix installation checks passed. Container teardown removes test output.\n'
