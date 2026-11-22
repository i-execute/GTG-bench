#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
version=${VERSION:-0.1.0}
output=${OUTPUT_DIR:-"$root/dist"}
go_bin=${GO_BIN:-go}
rm -rf "$output"
mkdir -p "$output"

build_target() {
  goos="$1"
  goarch="$2"
  ext="$3"
  name="gtg-bench_${version}_${goos}_${goarch}"
  stage=$(mktemp -d)
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" "$go_bin" build -trimpath -ldflags='-s -w' -o "$stage/gtg-bench$ext" ./cmd/gtg-bench
  if [ "$goos" = windows ]; then
    (cd "$stage" && zip -q "$output/$name.zip" "gtg-bench$ext")
  else
    tar -C "$stage" -czf "$output/$name.tar.gz" "gtg-bench$ext"
  fi
  rm -rf "$stage"
}

source_bundle() {
  source_dir="$1"
  prefix="$2"
  destination="$3"
  stage=$(mktemp -d)
  mkdir -p "$stage/$prefix"
  tar --exclude=.git --exclude=dist --exclude=reports --exclude=.cache --exclude=node_modules -C "$source_dir" -cf - . | tar -C "$stage/$prefix" -xf -
  tar --sort=name --mtime='@0' --owner=0 --group=0 --numeric-owner -C "$stage" -czf "$destination" "$prefix"
  rm -rf "$stage"
}

cd "$root"
build_target linux amd64 ''
build_target linux arm64 ''
build_target windows amd64 '.exe'
build_target windows arm64 '.exe'
source_bundle "$root" "GTG-bench-${version}" "$output/gtg-bench_${version}_src.tar.gz"
source_bundle "$root/../GTG" 'GTG' "$output/gtg-545d0b7f8b1ad0e5ccbc28e9f3d12208e833ba13_src.tar.gz"
(cd "$output" && sha256sum * > SHA256SUMS)
printf '%s\n' "Release files written to $output"
