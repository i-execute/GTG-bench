#!/usr/bin/env bash
set -euo pipefail

VERSION="0.1.0"
REPOSITORY="i-execute/GTG-bench"
RELEASE_BASE="https://github.com/${REPOSITORY}/releases/download/v${VERSION}"
if [ -n "${GTGBENCH_CACHE:-}" ]; then
  CACHE_ROOT="$GTGBENCH_CACHE"
elif [ "$(uname -s)" = Darwin ]; then
  CACHE_ROOT="$HOME/Library/Caches/gtg-bench"
else
  CACHE_ROOT="${XDG_CACHE_HOME:-$HOME/.cache}/gtg-bench"
fi
ASSUME_YES=0
CLEAN=0

usage() {
  printf '%s\n' "Usage: run.sh [-y|--yes] [--clean]"
}

for argument in "$@"; do
  case "$argument" in
    -y|--yes) ASSUME_YES=1 ;;
    --clean) CLEAN=1 ;;
    -h|--help) usage; exit 0 ;;
    *) printf '%s\n' "Unknown option: $argument" >&2; usage >&2; exit 2 ;;
  esac
done

if [ "$CLEAN" -eq 1 ]; then
  rm -rf "$CACHE_ROOT"
  printf '%s\n' "Removed $CACHE_ROOT"
  exit 0
fi

case "$(uname -s)" in
  Linux) platform=linux ;;
  Darwin) platform=darwin ;;
  *) printf '%s\n' "Unsupported platform: $(uname -s)" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) printf '%s\n' "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

asset="gtg-bench_${VERSION}_${platform}_${arch}.tar.gz"
asset_url="${RELEASE_BASE}/${asset}"
sums_url="${RELEASE_BASE}/SHA256SUMS"
bin_dir="${CACHE_ROOT}/bin/${VERSION}/${platform}_${arch}"
binary="${bin_dir}/gtg-bench"

preview() {
  url="$1"
  size="unknown"
  if headers=$(curl --fail --location --silent --show-error --head "$url" 2>/dev/null); then
    size=$(printf '%s\n' "$headers" | awk 'BEGIN{IGNORECASE=1} /^content-length:/ {print $2}' | tr -d '\r' | tail -n 1)
    [ -n "$size" ] || size="unknown"
  fi
  printf '%s\n' "Download preview: $url"
  printf '%s\n' "Declared size: $size bytes"
}

confirm() {
  [ "$ASSUME_YES" -eq 1 ] && return 0
  printf '%s' "Continue? [y/N] "
  read -r answer
  case "$answer" in y|Y|yes|YES) return 0 ;; esac
  printf '%s\n' "Cancelled."
  exit 0
}

verify() {
  file="$1"
  name="$2"
  expected=$(awk -v name="$name" '$2 == name || $2 == "*" name {print $1}' "$3" | head -n 1)
  [ -n "$expected" ] || return 1
  if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$file" | awk '{print $1}')
  else
    actual=$(shasum -a 256 "$file" | awk '{print $1}')
  fi
  [ "$actual" = "$expected" ]
}

download() {
  curl --fail --location --silent --show-error --retry 2 "$1" -o "$2"
}

mkdir -p "$bin_dir"
if [ -x "$binary" ]; then
  exec "$binary"
fi

preview "$asset_url"
confirm
work=$(mktemp -d "${TMPDIR:-/tmp}/gtg-bench.XXXXXX")
trap 'rm -rf "$work"' EXIT
if download "$sums_url" "$work/SHA256SUMS" && download "$asset_url" "$work/$asset" && verify "$work/$asset" "$asset" "$work/SHA256SUMS"; then
  tar -xzf "$work/$asset" -C "$bin_dir"
  chmod 700 "$binary"
  exec "$binary"
fi

printf '%s\n' "Verified binary asset unavailable. Trying pinned source build."
if ! command -v go >/dev/null 2>&1; then
  printf '%s\n' "Go is required for the pinned source fallback and was not found." >&2
  exit 1
fi
source_asset="gtg-bench_${VERSION}_src.tar.gz"
gtg_asset="gtg-545d0b7f8b1ad0e5ccbc28e9f3d12208e833ba13_src.tar.gz"
preview "${RELEASE_BASE}/${source_asset}"
preview "${RELEASE_BASE}/${gtg_asset}"
if ! download "$sums_url" "$work/SHA256SUMS" || ! download "${RELEASE_BASE}/${source_asset}" "$work/$source_asset" || ! download "${RELEASE_BASE}/${gtg_asset}" "$work/$gtg_asset" || ! verify "$work/$source_asset" "$source_asset" "$work/SHA256SUMS" || ! verify "$work/$gtg_asset" "$gtg_asset" "$work/SHA256SUMS"; then
  printf '%s\n' "Pinned source fallback could not be verified." >&2
  exit 1
fi
tar -xzf "$work/$source_asset" -C "$work"
tar -xzf "$work/$gtg_asset" -C "$work"
source_dir=$(find "$work" -mindepth 1 -maxdepth 1 -type d -name 'GTG-bench-*' | head -n 1)
gtg_dir=$(find "$work" -mindepth 1 -maxdepth 1 -type d -name 'GTG' | head -n 1)
[ -n "$source_dir" ] && [ -n "$gtg_dir" ] || { printf '%s\n' "Invalid verified source layout." >&2; exit 1; }
(cd "$source_dir" && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o "$binary" ./cmd/gtg-bench)
chmod 700 "$binary"
exec "$binary"
