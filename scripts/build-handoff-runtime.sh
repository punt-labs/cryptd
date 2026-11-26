#!/usr/bin/env bash
set -euo pipefail
umask 022
export LC_ALL=C
unset TAR_OPTIONS GZIP

if [[ $# -ne 2 || ! -d $2 ]]; then
    printf 'usage: %s PLATFORM OUTPUT_DIRECTORY\n' "$0" >&2
    exit 2
fi

platform=$1
case "$platform" in
    linux-amd64)
        node_platform=linux-x64
        go_sha=ccfe7ae822d2c2f41f4fe7c956199967a4dac22d0ee0d2b2bfdb440aab3b20c5
        node_sha=061ec3e3062115705b56b51fa9cc330ab50d34b6969387d4e4b1a50c6809acc2
        ;;
    darwin-arm64)
        node_platform=darwin-arm64
        go_sha=0b5183492292ada170f9b561a834863808dd02edba599e4ad0e89899330020b3
        node_sha=ff6e17d4874b76d272212b908cc020d6e815fba5a6c7eaaa06add725840307fd
        ;;
    *) printf 'unsupported handoff platform: %s\n' "$platform" >&2; exit 2 ;;
esac

go_version=1.26.10
node_version=24.22.3
staticcheck_version=v0.8.1

repo=$(cd -- "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
output=$(cd -- "$2" && pwd -P)
tar_tool=$(command -v gtar || command -v tar)
if [[ $("$tar_tool" --version) != *'GNU tar'* ]]; then
    printf 'GNU tar is required (gtar on macOS).\n' >&2
    exit 1
fi
gzip_tool=$(command -v gzip || true)
if [[ -z $gzip_tool || $("$gzip_tool" --version 2>/dev/null) != *'Free Software Foundation'* ]]; then
    printf 'GNU gzip is required; put it first on PATH or build on Linux as CI does.\n' >&2
    exit 1
fi
sha_tool=(shasum -a 256)
if command -v sha256sum >/dev/null; then
    sha_tool=(sha256sum)
fi

mkdir -p "$repo/.tmp"
work=$(mktemp -d "$repo/.tmp/cryptd-runtime.XXXXXX")
trap 'chmod -R u+w "$work" && rm -rf -- "$work"' EXIT
runtime=$work/runtime
mkdir -p "$runtime/bin" "$runtime/node" "$runtime/cache/go-mod" "$runtime/licenses" "$work/home" "$work/npm"

fetch() {
    local url=$1 sha=$2 archive=$3
    curl --proto '=https' --proto-redir '=https' --fail --silent --show-error \
        --location --retry 2 --max-time 300 --output "$archive" "$url"
    printf '%s  %s\n' "$sha" "$archive" | "${sha_tool[@]}" -c - >&2
}

fetch "https://go.dev/dl/go$go_version.$platform.tar.gz" "$go_sha" "$work/go.tar.gz"
"$tar_tool" -xzf "$work/go.tar.gz" -C "$runtime"
fetch "https://nodejs.org/dist/v$node_version/node-v$node_version-$node_platform.tar.gz" "$node_sha" "$work/node.tar.gz"
"$tar_tool" -xzf "$work/node.tar.gz" --strip-components=1 -C "$runtime/node"

host_go=$(command -v go)
if [[ $(GOTOOLCHAIN=local "$host_go" env GOVERSION) != "go$go_version" ]]; then
    printf 'Use Go %s to build the handoff runtime.\n' "$go_version" >&2
    exit 1
fi
go_env=(env GOTOOLCHAIN=local GOENV=off GOWORK=off GOFLAGS= GOEXPERIMENT=)
"${go_env[@]}" "GOMODCACHE=$runtime/cache/go-mod" "$host_go" -C "$repo" mod download
rm -rf -- "$runtime/cache/go-mod/cache/download/sumdb"
find "$runtime/cache/go-mod/cache/download" -name '*.lock' -type f -delete
host_platform=$("${go_env[@]}" "$host_go" env GOHOSTOS GOHOSTARCH | paste -sd -)
"${go_env[@]}" GOOS="${platform%-*}" GOARCH="${platform#*-}" GOAMD64=v1 GOARM64=v8.0 CGO_ENABLED=0 \
    GOBIN= "GOPATH=$work/gopath" "GOMODCACHE=$work/gopath/pkg/mod" \
    "$host_go" install -trimpath "honnef.co/go/tools/cmd/staticcheck@$staticcheck_version"
staticcheck_bin=$work/gopath/bin
if [[ $platform != "$host_platform" ]]; then
    staticcheck_bin+=/${platform/-/_}
fi
install -m 0755 "$staticcheck_bin/staticcheck" "$runtime/bin/staticcheck"
install -m 0644 "$work/gopath/pkg/mod/honnef.co/go/tools@$staticcheck_version/LICENSE" "$runtime/licenses/staticcheck.txt"

cp "$repo/scripts/handoff-runtime/"package{,-lock}.json "$work/npm/"
env -i PATH="$PATH" "HOME=$work/home" LC_ALL=C npm --prefix "$work/npm" ci \
    --ignore-scripts --no-audit --no-fund >&2
lint_dir=$runtime/node/lib/node_modules/markdownlint-cli2
mv "$work/npm/node_modules/markdownlint-cli2" "$lint_dir"
rm -rf -- "$work/npm/node_modules/.bin" "$work/npm/node_modules/.package-lock.json"
mv "$work/npm/node_modules" "$lint_dir/node_modules"
ln -s ../lib/node_modules/markdownlint-cli2/markdownlint-cli2-bin.mjs "$runtime/node/bin/markdownlint-cli2"

archive=$output/cryptd-handoff-$platform.tar.gz
"$tar_tool" --format=gnu --sort=name --mtime=@0 --owner=0 --group=0 --numeric-owner --hard-dereference \
    -C "$runtime" -cf - bin cache go licenses node | "$gzip_tool" -n -6 > "$work/runtime.tar.gz"
"${sha_tool[@]}" "$work/runtime.tar.gz" | awk '{print "SHA-256: " $1}' >&2
mv -- "$work/runtime.tar.gz" "$archive"
printf '%s\n' "$archive"
