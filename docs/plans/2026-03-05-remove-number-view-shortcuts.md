# Remove 1/2/3 View Shortcuts Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove direct view switching via `1/2/3` keys, keep only `Tab` cycling Today -> Upcoming -> History -> Today.

**Architecture:** Centralize view switching in the existing key handler and model update logic; eliminate the `1/2/3` key bindings and any update branches that reference them; keep `Tab` behavior unchanged. Update help text and tests to enter views via repeated `Tab`.

**Tech Stack:** Go, Bubble Tea TUI (internal `ui` package), Go tests.

---

### Task 1: Locate and remove 1/2/3 key bindings

**Files:**
- Modify: `internal/ui/keys.go`

**Step 1: Write/adjust a failing test (if any existing test covers 1/2/3)**

- Search tests for `keyRune('1')`, `keyRune('2')`, `keyRune('3')` and note which behavior they expect.

**Step 2: Run tests to see current baseline**

Run:
```bash
go test ./internal/ui -count=1
```
Expected: PASS (baseline before changes).

**Step 3: Remove bindings**

- In `internal/ui/keys.go`, delete key definitions / mapping entries for `1`, `2`, `3` that switch views.
- Ensure `Tab` mapping remains and still represents the cycle action.

**Step 4: Run unit tests**

Run:
```bash
go test ./internal/ui -count=1
```
Expected: FAIL due to tests or compilation if any references remain.

**Step 5: Commit (optional, only if you want a small intermediate commit)**

Skip unless user wants multiple commits; user requested a single new commit is acceptable.

---

### Task 2: Remove Update branches handling 1/2/3

**Files:**
- Modify: `internal/ui/model.go`

**Step 1: Write/adjust failing tests**

- Update any tests that reach views via `keyRune('2')` / `keyRune('3')` to use `Tab` cycling (see Task 4).

**Step 2: Implement minimal change**

- In `internal/ui/model.go`, remove `Update` cases/branches that match `1`, `2`, `3` (or corresponding actions) and set the current view directly.
- Keep `Tab` branch unchanged: it should cycle Today -> Upcoming -> History -> Today.
- Ensure any helper functions/constants referencing `1/2/3` are removed or updated.

**Step 3: Run targeted tests**

Run:
```bash
go test ./internal/ui -count=1
```
Expected: PASS (or remaining failures only in tests not yet updated).

---

### Task 3: Remove help text / styles mentioning 1/2/3

**Files:**
- Modify: `internal/ui/styles.go`
- Modify: `internal/ui/model.go` (if help text lives there)

**Step 1: Update help copy**

- Remove any help/legend text like "1/2/3 to jump".
- Keep (or add) copy that mentions `Tab` to cycle views.

**Step 2: Run tests**

Run:
```bash
go test ./internal/ui -count=1
```
Expected: PASS.

---

### Task 4: Update tests to use Tab cycling instead of 2/3

**Files:**
- Modify: `internal/ui/model_history_test.go`
- Modify: any other affected tests under `internal/ui/*_test.go`

**Step 1: Replace direct key runes**

- Replace sequences like:
  - `keyRune('2')` (jump to Upcoming)
  - `keyRune('3')` (jump to History)
- With `Tab` presses:
  - Today -> Upcoming: one `Tab`
  - Today -> History: two `Tab`
  - Upcoming -> History: one `Tab`
  - History -> Today: one `Tab`

**Step 2: Make helper for cycling (if repeated in many tests)**

- If tests repeat `keyTab()` many times, introduce a tiny test helper like:
```go
func pressTabN(m model, n int) model {
    for i := 0; i < n; i++ {
        m, _ = m.Update(keyTab())
    }
    return m
}
```

**Step 3: Run internal ui tests**

Run:
```bash
go test ./internal/ui -count=1
```
Expected: PASS.

**Step 4: Run full suite**

Run:
```bash
go test ./... -count=1
```
Expected: PASS.

---

### Task 5: Final commit (single commit, no amend)

**Files:**
- `internal/ui/keys.go`
- `internal/ui/model.go`
- `internal/ui/styles.go`
- `internal/ui/model_history_test.go` (and other modified tests)

**Step 1: Check git diff**

Run:
```bash
git diff
```

**Step 2: Create commit**

Run:
```bash
git add internal/ui/keys.go internal/ui/model.go internal/ui/styles.go internal/ui/*_test.go
git commit -m "ui: remove numeric view shortcuts; keep Tab cycling"
```

**Step 3: Verify clean status**

Run:
```bash
git status
```
Expected: clean (or only unrelated user changes).
