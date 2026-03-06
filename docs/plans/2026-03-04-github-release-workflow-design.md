# GitHub Release Workflow Design

**Goal:** When pushing a semver tag (e.g. `v0.0.1`), automatically build and package `tick` binaries for all supported OS/architectures and attach them to a GitHub Release.

## Triggers

- Trigger: `push` on tags matching `v*`.

## Target Platforms

- `linux/amd64`, `linux/arm64`
- `darwin/amd64`, `darwin/arm64`
- `windows/amd64`, `windows/arm64`

## Artifacts

- One archive per platform:
  - Linux/macOS: `.tar.gz`
  - Windows: `.zip`
- Naming: `tick-<tag>-<goos>-<goarch>.(tar.gz|zip)`
- Contents: the compiled executable (`tick` or `tick.exe`).

## Release Behavior

- Create (or update) a GitHub Release for the pushed tag.
- Upload all platform archives as Release assets.
- Generate release notes automatically.

## Build Behavior

- `go build -trimpath` from `./cmd/tuitodo` output binary named `tick`.
- Use `CGO_ENABLED=0` for predictable cross-platform builds.
- Run `go test ./... -count=1` once on `linux/amd64` to keep CI time reasonable.

## Permissions

- `contents: write` so Actions can create releases and upload assets.
