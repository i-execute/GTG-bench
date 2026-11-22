# Build selection

## Selected release configuration

Release binaries use the portable baseline:

```sh
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' ./cmd/gtg-bench
```

The release workflow produces static Linux `amd64` and `arm64` binaries plus Windows `amd64` and `arm64` binaries. `amd64` release binaries deliberately use the Go baseline ISA level, equivalent to `GOAMD64=v1`, so one artifact does not silently exclude otherwise supported x86-64 machines. The program has no cgo dependency.

`-trimpath` removes local source paths from build metadata. `-ldflags='-s -w'` strips symbol and DWARF tables from release binaries. These are packaging choices, not runtime-speed claims.

## Measured GOAMD64 selection evidence

The following local measurement was run on 2026-11-22 with Go 1.27.4, Linux amd64, two visible logical CPUs, an Intel Xeon Processor at 2.60 GHz, and a reported hypervisor CPU flag. It is valid only for that guest and run state.

The command sequence was the repository's `scripts/evaluate-build.sh`: consecutive level pairs were run as A/B/B/A, each invocation used `go test ./internal/bench -run '^$' -bench BenchmarkIGE -benchmem -benchtime=100ms -count 20`, and `benchstat` compared the 40 retained observations per level. The two benchmarks encrypt and decrypt a caller-owned 1 MiB AES-256-IGE buffer. No observation was omitted.

| Pair | Encrypt result | Decrypt result | Geometric summary | Selection conclusion |
|---|---|---|---|---|
| v1 → v2 | 858.0 µs → 861.7 µs, no significant change (`p=0.360`) | 967.7 µs → 966.0 µs, no significant change (`p=1.000`) | +0.12% time | Do not specialize. |
| v2 → v3 | 861.9 µs → 856.2 µs, no significant change (`p=0.099`) | 965.2 µs → 974.6 µs, no significant change (`p=0.424`) | +0.15% time | Do not specialize. |
| v3 → v4 | 874.0 µs → 862.9 µs, encryption -1.27% (`p=0.008`) | 976.3 µs → 1002.6 µs, no significant change (`p=0.441`) | +0.69% time | Do not specialize. |

The measured outcome does not justify a `GOAMD64=v2`, `v3`, or `v4` release variant. The portable v1 artifact is retained. These figures are not benchmark-comparator claims and must not be extrapolated to another host.

## PGO

No PGO artifact is retained. PGO is intentionally opt-in in `scripts/evaluate-build.sh` through `GTGBENCH_PGO=1`; it writes a profile, runs a 20-count baseline/candidate comparison, and leaves its raw reports in the selected report directory. A PGO binary may be retained only if that local paired evaluation shows a material, statistically credible benefit for both encrypt and decrypt without a regression. No such retained result exists in this release.

## Arm64 and Windows boundaries

All four target binaries were cross-built successfully during the build-configuration check. Cross-compilation proves buildability, not runtime behavior or speed. No native arm64 execution was available for this evaluation, so this project makes no arm64 speed claim. Windows artifacts were cross-built but not executed in this Linux environment; Windows launcher and runtime behavior require native Windows validation before a public release.

## Reproduce

```sh
GO_BIN=go REPORT_DIR=reports/build-evaluation scripts/evaluate-build.sh
GO_BIN=go REPORT_DIR=reports/build-pgo GTGBENCH_PGO=1 scripts/evaluate-build.sh
VERSION=0.1.0 OUTPUT_DIR=dist scripts/build-release.sh
```
