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

## Completed on next

- Added observed timer-resolution reporting and calibration. The 16-byte and 64-byte AES-IGE and MTProto cases now increase their retained iteration count until their timing interval is meaningful relative to the clock; no observation is discarded.
- Extended environment reporting with available CPU frequency, total memory, timer resolution, AES indication, virtualization, power, and load. Missing platform data is recorded as zero or unavailable rather than inferred.
- Added GTG-only bounded MTProto Seal/Open operation rows for every benchmark body size. These rows use the public MTProto core API, are local samples only, and are never cross-library rankings.
- Added Markdown operation-row summaries and CSV/JSON operation labels, plus unit coverage for timer calibration and Seal/Open execution.
- Added context cancellation, atomic partial JSON checkpoints, report-schema/mode validation, sample-key de-duplication, and `--resume` in-place continuation. The completion test proves canceled samples resume without duplicate keys.
- Added `--workers N` for GTG-only AES-IGE wall-clock worker groups. Each worker owns its own cipher state; total operations and worker count are retained in raw and Markdown output. Comparator pairs and MTProto operation rows deliberately remain single-worker-only.
- Linux amd64 KVM smoke runs with Python disabled completed for the normal and two-worker modes and emitted JSON, Markdown, and CSV reports. They are not comparator, Windows, native arm64, or Ryzen results.

## Current work order

1. Add an arrow-key menu while preserving a no-dependency terminal fallback; the current menu remains line-oriented and transparent about that limitation.
2. Extend the system report and TUI interaction without making non-local performance claims.
3. Keep raw observations, source/build metadata, directional ratio labels, and warnings in every report.
4. Validate Linux behavior locally; document Windows and native Ryzen validation as pending unless actually executed there.
5. Confirm exact launcher-consumed assets against a public release before claiming the public launcher functional.

## Resume and cancellation

- `--resume path/to/report.json` resumes a schema- and mode-compatible partial report in place. Existing raw sample keys are retained and skipped; new output is checkpointed after each retained sample.
- Ctrl-C is routed through the Go context. The active operation is allowed to return, after which JSON, Markdown, and CSV partial reports are written and the terminal prints the exact resume command.
- Unit coverage cancels a run after the first retained sample and verifies that a resumed run completes without duplicate sample keys. A local CLI resume of a completed GTG-only report was also exercised; it retained its samples and recorded the resume count.


## Required verification before delivery

```text
go test ./...
go vet ./...
GOGC=20 go test -race -p 1 ./...
bash -n scripts/run.sh scripts/build-release.sh scripts/evaluate-build.sh
```

Cross-build, TUI pseudoterminal, Python harness, launcher, static scan, and release-manifest checks are updated as scope changes.
