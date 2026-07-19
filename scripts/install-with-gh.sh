#!/bin/bash

set -euo pipefail

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

command -v gh >/dev/null 2>&1 || fail "GitHub CLI (gh) is required"
[[ "$(uname -s)" == "Darwin" ]] || fail "only macOS is supported"

case "$(uname -m)" in
  arm64) architecture="arm64" ;;
  x86_64) architecture="amd64" ;;
  *) fail "unsupported architecture: $(uname -m)" ;;
esac

repository="${TABCLI_REPOSITORY:-masahide/tabcli}"
tag="$(gh release view --repo "$repository" --json tagName --jq .tagName)"
[[ "$tag" =~ ^v[0-9]+(\.[0-9]+){1,3}$ ]] || fail "invalid latest release tag: ${tag:-empty}"

asset="tabcli-${tag#v}-darwin-${architecture}.zip"
download_dir="$(mktemp -d)"
cleanup() {
  /bin/rm -rf "$download_dir"
}
trap cleanup EXIT

gh release download "$tag" \
  --repo "$repository" \
  --pattern "$asset" \
  --pattern SHA256SUMS \
  --dir "$download_dir"

(
  cd "$download_dir"
  /usr/bin/awk -v file="$asset" \
    '$2 == file { print; found=1 } END { if (!found) exit 1 }' SHA256SUMS | \
    /usr/bin/shasum -a 256 -c -
) || fail "checksum verification failed for ${asset}"

/bin/mkdir "$download_dir/bundle"
/usr/bin/ditto -x -k "$download_dir/$asset" "$download_dir/bundle"
/bin/bash "$download_dir/bundle/install.sh"
