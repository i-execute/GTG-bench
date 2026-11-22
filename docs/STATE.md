# Project state

## Next client-layer benchmark program

- Active branch: `next`, based on public GTG-bench `main` at `e5f0a294b71086472591d98cd5f81b9c9d70f15a`.
- GTG dependency baseline: public `github.com/i-execute/GTG v0.1.0`.
- Current execution host: Linux amd64, Intel Xeon Processor at 2.60 GHz, two visible logical CPUs, KVM virtualization, Go 1.27.4 and Python 3.13.18. It is not the requested Windows Ryzen target. No Windows or Zen 4 measurement claim may be made from this host.
- This repository may name public comparison libraries. The GTG repository must not contain comparison-library names or materials.
- Credentials remain environment-only. They must not be written into sources, reports, raw samples, cache markers, commits, release assets, or CI configuration.

## Baseline present before next work

- The terminal application supports GTG and optional Python comparator qualification, paired AES-IGE runs, raw CSV/JSON/Markdown export, local cache setup, verified release launchers, and release packaging scripts.
- The release protocol is versioned for `v0.1.0`; public release assets must be checked separately from source work.
- The current UI is menu-based but lacks the requested arrow-key selection, detailed system screen, resumable full execution, timer-resolution calibration, operation matrix, and explicit multi-process comparator controls.

## Current work order

1. Establish baseline validation and create timer-resolution and duration-calibration tests before changing score calculations.
2. Extend the system report and TUI interaction without making non-local performance claims.
3. Add MTProto Seal/Open benchmarks only after the GTG public APIs expose a stable bounded operation.
4. Keep raw observations, source/build metadata, directional ratio labels, and warnings in every report.
5. Validate Linux behavior locally; document Windows and native Ryzen validation as pending unless actually executed there.

## Required verification before delivery

```text
go test ./...
go vet ./...
GOGC=20 go test -race -p 1 ./...
bash -n scripts/run.sh scripts/build-release.sh scripts/evaluate-build.sh
```

Cross-build, TUI pseudoterminal, Python harness, launcher, static scan, and release-manifest checks are updated as scope changes.
