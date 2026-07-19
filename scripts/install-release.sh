#!/bin/bash

set -euo pipefail

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

bundle_dir="$(cd "$(dirname "$0")" && pwd)"
binary_path="${bundle_dir}/tabcli"
extension_archive="${bundle_dir}/tabcli-extension.zip"
version_file="${bundle_dir}/version.json"

for required_file in "$binary_path" "$extension_archive" "$version_file"; do
  [[ -f "$required_file" ]] || fail "required file is missing: ${required_file}"
done

/usr/bin/codesign --verify --strict --verbose=2 "$binary_path"

version="$(/usr/bin/sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$version_file")"
[[ "$version" =~ ^[0-9]+(\.[0-9]+){1,3}$ ]] || fail "invalid version in version.json: ${version:-empty}"

binary_version="$($binary_path --json version | /usr/bin/sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
[[ "$binary_version" == "$version" ]] || fail "binary version ${binary_version:-empty} does not match ${version}"

bin_dir="${TABCLI_BIN_DIR:-${HOME:?HOME is not set}/.local/bin}"
data_dir="${TABCLI_DATA_DIR:-${HOME:?HOME is not set}/.local/share/tabcli}"
release_dir="${data_dir}/releases/${version}"
extension_dir="${release_dir}/tabcli-extension-unpacked"

/bin/mkdir -p "$bin_dir" "$release_dir"

temp_binary=""
temp_extension=""
cleanup() {
  if [[ -n "$temp_binary" && -e "$temp_binary" ]]; then
    /bin/rm -f "$temp_binary"
  fi
  if [[ -n "$temp_extension" && -d "$temp_extension" ]]; then
    /bin/rm -rf "$temp_extension"
  fi
}
trap cleanup EXIT

temp_extension="$(/usr/bin/mktemp -d "${release_dir}/.extension.XXXXXX")"
/usr/bin/ditto -x -k "$extension_archive" "$temp_extension"

manifest_path="${temp_extension}/manifest.json"
[[ -f "$manifest_path" ]] || fail "extension archive does not contain manifest.json"
manifest_version="$(/usr/bin/sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$manifest_path")"
[[ "$manifest_version" == "$version" ]] || fail "extension version ${manifest_version:-empty} does not match ${version}"

if [[ -e "$extension_dir" ]]; then
  /usr/bin/diff -qr "$extension_dir" "$temp_extension" >/dev/null || \
    fail "extension destination already exists with different contents: ${extension_dir}"
  /bin/rm -rf "$temp_extension"
  temp_extension=""
else
  /bin/mv "$temp_extension" "$extension_dir"
  temp_extension=""
fi

temp_binary="$(/usr/bin/mktemp "${bin_dir}/.tabcli.new.XXXXXX")"
/usr/bin/install -m 0755 "$binary_path" "$temp_binary"
/usr/bin/codesign --verify --strict --verbose=2 "$temp_binary"
/bin/mv -f "$temp_binary" "${bin_dir}/tabcli"
temp_binary=""

if [[ "${TABCLI_SKIP_REGISTER:-0}" != "1" ]]; then
  "${bin_dir}/tabcli" install
fi

printf 'tabcli installed: %s\n' "${bin_dir}/tabcli"
printf 'Chrome extension directory: %s\n' "$extension_dir"
printf 'Open chrome://extensions, enable Developer mode, then choose Load unpacked.\n'
