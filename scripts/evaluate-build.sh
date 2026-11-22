#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
go_bin=${GO_BIN:-go}
report_dir=${REPORT_DIR:-"$root/reports/build-evaluation"}
benchstat_bin=${BENCHSTAT:-benchstat}
mkdir -p "$report_dir"
cd "$root"

if ! command -v "$benchstat_bin" >/dev/null 2>&1; then
  GOBIN="$report_dir/bin" "$go_bin" install golang.org/x/perf/cmd/benchstat@latest
  benchstat_bin="$report_dir/bin/benchstat"
fi

for target in linux/amd64 linux/arm64 windows/amd64 windows/arm64; do
  goos=${target%/*}
  goarch=${target#*/}
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" "$go_bin" build -trimpath -ldflags='-s -w' -o "$report_dir/gtg-bench_${goos}_${goarch}" ./cmd/gtg-bench
done

run_pair() {
  left="$1"
  right="$2"
  key="$left-$right"
  for position in left-first right-first right-second left-second; do
    level="$left"
    case "$position" in right-*) level="$right" ;; esac
    GOAMD64="$level" CGO_ENABLED=0 "$go_bin" test ./internal/bench -run '^$' -bench 'BenchmarkIGE' -benchmem -benchtime=100ms -count 20 > "$report_dir/${key}-${position}.txt"
  done
  cat "$report_dir/${key}-left-first.txt" "$report_dir/${key}-left-second.txt" > "$report_dir/${key}-${left}.txt"
  cat "$report_dir/${key}-right-first.txt" "$report_dir/${key}-right-second.txt" > "$report_dir/${key}-${right}.txt"
  "$benchstat_bin" "$report_dir/${key}-${left}.txt" "$report_dir/${key}-${right}.txt" > "$report_dir/${key}-benchstat.txt"
}

run_pair v1 v2
run_pair v2 v3
run_pair v3 v4

if [ "${GTGBENCH_PGO:-0}" = 1 ]; then
  CGO_ENABLED=0 GOAMD64=v1 "$go_bin" test ./internal/bench -run '^$' -bench 'BenchmarkIGEEncrypt1MiB' -cpuprofile "$report_dir/default.pgo"
  CGO_ENABLED=0 GOAMD64=v1 "$go_bin" test ./internal/bench -run '^$' -bench 'BenchmarkIGE' -benchmem -count 20 > "$report_dir/pgo-baseline.txt"
  CGO_ENABLED=0 GOAMD64=v1 GOFLAGS="-pgo=$report_dir/default.pgo" "$go_bin" test ./internal/bench -run '^$' -bench 'BenchmarkIGE' -benchmem -count 20 > "$report_dir/pgo-candidate.txt"
  "$benchstat_bin" "$report_dir/pgo-baseline.txt" "$report_dir/pgo-candidate.txt" > "$report_dir/pgo-benchstat.txt"
fi

printf '%s\n' "Build evaluation written to $report_dir"
