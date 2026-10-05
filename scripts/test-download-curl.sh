#!/bin/sh
# Test-only curl replacement. Never accesses the network or credentials.
set -eu
[ "$#" -eq 16 ]
[ "$1" = --proto ] && [ "$2" = '=https' ]
[ "$3" = --proto-redir ] && [ "$4" = '=https' ]
[ "$5" = --tlsv1.2 ] && [ "$6" = --fail ]
[ "$7" = --location ] && [ "$8" = --silent ] && [ "$9" = --show-error ]
shift 9
[ "$1" = --connect-timeout ] && [ "$2" = 15 ]
[ "$3" = --max-time ] && [ "$4" = 180 ] && [ "$5" = --output ]
destination=$6
url=$7
case "$url" in
    "https://github.com/Kook-Dohyun/To-Do-Connect/releases/download/v$TEST_VERSION/SHA256SUMS") asset=SHA256SUMS ;;
    "https://github.com/Kook-Dohyun/To-Do-Connect/releases/download/v$TEST_VERSION/todo-connect-$TEST_VERSION-$TEST_OS-$TEST_ARCH.zip") asset="todo-connect-$TEST_VERSION-$TEST_OS-$TEST_ARCH.zip" ;;
    *) printf 'Unexpected download URL\n' >&2; exit 1 ;;
esac
[ "$(basename "$destination")" = "$asset" ]
printf '%s\n' "$asset" >> "$TEST_DOWNLOAD_LOG"
if [ "${TEST_DOWNLOAD_FAILURE:-}" = "$asset" ]; then exit 22; fi
cp "$TEST_RELEASE/$asset" "$destination"
