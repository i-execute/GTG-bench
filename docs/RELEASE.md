# Release publication procedure

This repository is prepared locally only. It has not been pushed by the build process.

## Preconditions

1. Create an immutable GTG release tag at commit `545d0b7f8b1ad0e5ccbc28e9f3d12208e833ba13` before changing this repository from its local `replace` dependency. The required GTG tag for the first public GTG-bench release is `v0.1.0`.
2. Update `go.mod` to require that GTG tag and remove the local `replace` only after the tag is visible to the Go module proxy.
3. Run all validation commands in the release checklist below on a clean checkout.
4. Review `git log`, `git status --ignored`, and the prohibited-material scan before publication.

## Local packaging

With the GTG checkout as a sibling directory:

```sh
VERSION=0.1.0 OUTPUT_DIR=dist scripts/build-release.sh
cd dist
sha256sum -c SHA256SUMS
```

The publishable set is:

```text
gtg-bench_0.1.0_linux_amd64.tar.gz
gtg-bench_0.1.0_linux_arm64.tar.gz
gtg-bench_0.1.0_windows_amd64.zip
gtg-bench_0.1.0_windows_arm64.zip
gtg-bench_0.1.0_src.tar.gz
gtg-545d0b7f8b1ad0e5ccbc28e9f3d12208e833ba13_src.tar.gz
SHA256SUMS
```

The source pair is required by the launchers' verified fallback path. Keep the names exactly aligned with `scripts/run.sh` and `scripts/run.ps1`.

## GitHub publication

1. Push `main` only after the preconditions are met.
2. Create signed or verified tag `v0.1.0` from the reviewed main commit.
3. Create a GitHub release from that tag. The included Actions workflow builds the target matrix, creates source bundles, generates `SHA256SUMS`, and uploads all assets.
4. Download each asset and verify `sha256sum -c SHA256SUMS` from a clean directory.
5. Test the Linux launcher with a non-privileged account. Test the PowerShell launcher on native Windows. Test macOS through its verified source fallback unless a Darwin binary release target is added.
6. Do not add a token or secret to the workflow. It uses only the default repository `GITHUB_TOKEN` exposed as `github.token`.

## Release checklist

```sh
go test ./...
go vet ./...
GOGC=20 go test -race -p 1 ./...
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/gtg-bench
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build ./cmd/gtg-bench
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/gtg-bench
GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build ./cmd/gtg-bench
bash -n scripts/run.sh scripts/build-release.sh scripts/evaluate-build.sh
python3 -c 'import pathlib; p=pathlib.Path("scripts/run.ps1").read_text(); assert p.count("{")==p.count("}")'
```

Run the terminal interface in a pseudoterminal, then run `--fast` with available comparators and `--fast --no-python`. Preserve only reports that are intentionally being published; reports identify the local environment and are not universal claims.
