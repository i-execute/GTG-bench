# GTG-bench

A local, cross-platform AES-256-IGE comparison lab for [GTG](https://github.com/i-execute/GTG), GoyGram, TgCrypto, and cryptg. It is an independent MIT-licensed repository. It does not send telemetry, upload results, use credentials, or modify GTG.

## Launch

Linux and macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/i-execute/GTG-bench/main/scripts/run.sh | bash
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/i-execute/GTG-bench/main/scripts/run.ps1 | iex
```

Both launchers preview every download and ask before downloading. Linux/macOS accepts `-y` with `curl -fsSL https://raw.githubusercontent.com/i-execute/GTG-bench/main/scripts/run.sh | bash -s -- -y`. Cleanup is `curl -fsSL https://raw.githubusercontent.com/i-execute/GTG-bench/main/scripts/run.sh | bash -s -- --clean`; on Windows, save the launcher and run `iwr https://raw.githubusercontent.com/i-execute/GTG-bench/main/scripts/run.ps1 -OutFile run.ps1; .\run.ps1 -clean`. The cache is user-local: `$XDG_CACHE_HOME/gtg-bench` or `~/.cache/gtg-bench` on Linux, `~/Library/Caches/gtg-bench` on macOS, and `%LOCALAPPDATA%\gtg-bench` on Windows.

Launchers obtain `SHA256SUMS`, verify the chosen archive before extracting it, and do not require administrator access. If a verified prebuilt binary cannot be obtained, they try the verified source bundles for this release and build locally with Go. A source fallback requires Go and the source bundles published next to `SHA256SUMS`.

## Local build

Until GTG publishes an immutable tag for commit `545d0b7f8b1ad0e5ccbc28e9f3d12208e833ba13`, this repository deliberately uses a local `replace` in `go.mod`.

```sh
git clone https://github.com/i-execute/GTG.git GTG
git -C GTG checkout 545d0b7f8b1ad0e5ccbc28e9f3d12208e833ba13
git clone https://github.com/i-execute/GTG-bench.git GTG-bench
cd GTG-bench
go test ./...
go run ./cmd/gtg-bench --fast
```

Useful invocations:

```sh
go run ./cmd/gtg-bench --fast
go run ./cmd/gtg-bench --full
go run ./cmd/gtg-bench --fast --no-python --plain
go run ./cmd/gtg-bench --env
go run ./cmd/gtg-bench --clean
```

`--fast` is a focused, hot-cache smoke comparison through 1 MiB. `--full` covers 16 B through 64 MiB with both hot and cold cache conditions. Every run writes `report.json`, `report.md`, and raw `samples.csv` to a timestamped `reports/` directory unless output paths are supplied.

## Text screenshot

```text
╭──────────────────────────────────────────────────────────────────────╮
│  GTG-bench  AES-256-IGE local comparison lab                         │
╰──────────────────────────────────────────────────────────────────────╯
  linux/amd64  ·  2 CPUs  ·  AES instructions reported

  1  Fast comparison     focused sizes, hot cache
  2  Full comparison     16 B to 64 MiB, hot and cold cache
  3  Environment         CPU, load, power, virtualization
  q  Quit
```

## Method

Before timing, GTG encrypts a deterministic AES-256-IGE vector with a 32-byte key and 32-byte IV. Each Python comparator must produce byte-identical ciphertext and must decrypt that ciphertext back to the original bytes. The check is repeated at every scheduled size before the first timed sample. A failing or unavailable comparator is marked non-comparable and is never ranked.

The compared public calls use `bytes` input and the public AES-256-IGE entry points documented in `requirements-manifest.json`:

- GTG: `ige.New`, then `Encrypt` or `Decrypt` into a caller-owned destination buffer.
- GoyGram 0.8.32: `goygram.ext.aes_ige_enc` and `aes_ige_dec`.
- TgCrypto 1.2.5: `tgcrypto.ige256_encrypt` and `ige256_decrypt`.
- cryptg 0.6.0: `cryptg.encrypt_ige` and `decrypt_ige`.

The Python harness is generated from a string embedded in the Go binary into a cache-local virtual environment. First use requires Python 3.9+ and creates an isolated virtual environment before installing the pinned packages. If Python is absent or setup fails, GTG-only measurements remain available and the report says why comparators are absent.

For every comparator, direction, cache mode, and size, paired rounds alternate `GTG, comparator, comparator, GTG` and `comparator, GTG, GTG, comparator`. All durations are preserved. The report forms one ratio per paired round as `GTG time / comparator time`, calculates a deterministic bootstrap 95% interval from the complete paired sample set, and does not discard observations. A row says **faster** only when the entire interval is below `0.97`, **slower** only when it is above `1.03`, and **parity** otherwise. Coefficient-of-variation and first-versus-last drift flags are descriptive; they never remove a sample or change that rule.

The terminal reports CPU, architecture, AES acceleration indication, virtualization, power state, and load. Those properties are recorded with results because they can materially alter local timing.

## Interpretation limits

A result is local to the machine, software versions, process state, and environment stored in that result file. It is not a general performance claim. The end-to-end public call cost intentionally includes each runtime's normal call and output behavior: GTG is timed from Go, while GoyGram, TgCrypto, and cryptg are timed inside CPython. Therefore tiny-message timing includes different Go and CPython call-boundary costs. For bulk analysis, compare the size-regression slope from raw observations rather than converting one short-call result directly into throughput.

The harness does not test protocol integration, networking, Telegram compatibility, memory safety beyond the libraries' public calls, or production readiness. AES-256-IGE is unauthenticated; this is a benchmark, not a recommendation to design new protocols with IGE.

## Releasing and removal

See `docs/RELEASE.md` for the exact publication procedure and `docs/BUILD.md` for build-configuration selection. To remove the tool, delete its user cache with the launcher `--clean` or remove the cache directory listed above. Delete generated `reports/` manually if desired. No service-side state exists because the tool has no telemetry.
