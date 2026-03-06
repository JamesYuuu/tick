# GitHub Release Workflow Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a GitHub Actions workflow named `release` that builds and packages `tick` for linux/macOS/windows on amd64+arm64 when a `v*` tag is pushed, and publishes the artifacts on a GitHub Release.

**Architecture:** Use a matrix build job to compile+package artifacts per target, upload them as Actions artifacts, then a final job creates the Release and uploads all assets.

**Tech Stack:** GitHub Actions (`actions/checkout`, `actions/setup-go`, `actions/upload-artifact`, `actions/download-artifact`, `softprops/action-gh-release`), Go toolchain.

---

### Task 1: Add release workflow

**Files:**
- Create: `.github/workflows/release.yml`

**Step 1: Add workflow skeleton**

- Define `on.push.tags: ["v*"]` and `permissions.contents: write`.

**Step 2: Add build matrix**

- Include `linux/darwin/windows` x `amd64/arm64`.
- Build: `go build -trimpath -o dist/tick(.exe) ./cmd/tuitodo`.

**Step 3: Package artifacts**

- Linux/macOS: tar.gz
- Windows: zip
- Name: `tick-${tag}-${goos}-${goarch}`.

**Step 4: Publish release assets**

- Download all artifacts in a `release` job.
- Use `softprops/action-gh-release@v2` with `files: artifacts/**/*.tar.gz` and `artifacts/**/*.zip`.

**Step 5: Local verification**

Run: `go test ./... -count=1`
Expected: all tests pass.

**Step 6: End-to-end verification (on GitHub)**

Run locally:

```bash
git tag -a v0.0.1 -m "tick v0.0.1"
git push origin v0.0.1
```

Expected on GitHub:
- Actions workflow `release` runs.
- A Release for `v0.0.1` exists.
- Release assets include archives for each target.
