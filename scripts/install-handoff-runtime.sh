#!/usr/bin/env bash
set -euo pipefail

if [[ $# -gt 1 ]]; then
    printf 'usage: %s [LOCAL_ARCHIVE]\n' "$0" >&2
    exit 2
fi

case "$(uname -s):$(uname -m)" in
    Linux:x86_64) platform=linux-amd64 ;;
    Darwin:arm64) platform=darwin-arm64 ;;
    *) printf 'No pinned runtime for %s %s.\n' "$(uname -s)" "$(uname -m)" >&2; exit 2 ;;
esac

if ! command -v yq >/dev/null; then
    printf 'Install Mike Farah yq v4 to read the runtime pin.\n' >&2
    exit 1
fi

repo=$(cd -- "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
sidecar=$repo/.punt-labs/ethos/checks/snapshots/crypt-go.yaml
pin='.checks["handoff-gate"].runtime.platforms[strenv(PLATFORM)]'
sha=$(PLATFORM=$platform yq -er "$pin.sha256" "$sidecar")
if [[ ! $sha =~ ^[0-9a-f]{64}$ ]]; then
    printf 'Invalid runtime SHA-256 for %s.\n' "$platform" >&2
    exit 1
fi

store=$HOME/.punt-labs/ethos/runtimes
mkdir -p "$store"
archive=$(mktemp "$store/.handoff-runtime.XXXXXX")
trap 'rm -f -- "$archive"' EXIT
if [[ $# -eq 1 ]]; then
    cp -- "$1" "$archive"
else
    url=$(PLATFORM=$platform yq -er "$pin.url" "$sidecar")
    curl --proto '=https' --proto-redir '=https' --fail --silent --show-error \
        --location --retry 2 --max-time 300 --output "$archive" "$url"
fi

sha_tool=(shasum -a 256)
if command -v sha256sum >/dev/null; then
    sha_tool=(sha256sum)
fi
printf '%s  %s\n' "$sha" "$archive" | "${sha_tool[@]}" -c - >&2
mv -f -- "$archive" "$store/$sha.tar.gz"
printf '%s\n' "$store/$sha.tar.gz"
