# Tick UI Refresh Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refresh Tick's TUI styling ("calm paper") to improve hierarchy and scanability, without changing any app behavior or keybindings.

**Architecture:** Keep Bubble Tea model logic as-is; concentrate changes in `internal/ui/styles.go` and `internal/ui/views.go`, plus delegates in `internal/ui/model.go`. Add deterministic string-based tests for key UI outputs.

**Tech Stack:** Go, Bubble Tea, Bubbles list/textinput, Lip Gloss.

---

### Task 1: Introduce a "sheet" frame and stable footer

**Files:**
- Modify: `internal/ui/styles.go`
- Test: `internal/ui/branding_test.go`

**Step 1: Write failing test for stable footer layout**

- Add a test that asserts `View()` renders help line even when `statusMsg` is set, and that status appears above help.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -count=1`
Expected: FAIL (status/help ordering not as expected).

**Step 3: Implement sheet frame + footer ordering**

- Update `Model.frame(...)` to render a bordered/padded body container.
- Ensure footer order:
  - status line if present else history footer (ratios)
  - help line always

**Step 4: Run UI tests**

Run: `go test ./internal/ui -count=1`
Expected: PASS.

**Step 5: Commit**

Run:

```bash
git add internal/ui/styles.go internal/ui/branding_test.go
git commit -m "ui: add sheet frame and stable footer"
```

### Task 2: Improve tabs, brand, and spacing hierarchy

**Files:**
- Modify: `internal/ui/styles.go`
- Test: `internal/ui/branding_test.go`

**Step 1: Write failing test for tab styling markers**

- Add a test that asserts the active tab is rendered with a distinct marker (e.g. brackets/underline) and inactive tabs are not.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -count=1`
Expected: FAIL.

**Step 3: Implement tab style tweaks**

- Adjust `styles.Tab` / `styles.TabOn` and `Model.tab(...)` output.

**Step 4: Run UI tests**

Run: `go test ./internal/ui -count=1`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/ui/styles.go internal/ui/branding_test.go
git commit -m "ui: refine header tabs and spacing"
```

### Task 3: Selected row styling + delayed styling refinement

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/styles.go`
- Test: `internal/ui/model_test.go`

**Step 1: Write failing test for selected row highlighting**

- Add a test that builds a model with 2 tasks and asserts the selected row is visually distinct in `View()` output.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -count=1`
Expected: FAIL.

**Step 3: Implement selected styling**

- Update list item delegates to render selected row with a calm highlight (background tint / inverse).
- Keep delayed styling (red) but make it restrained and still legible.

**Step 4: Run UI tests**

Run: `go test ./internal/ui -count=1`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/ui/model.go internal/ui/styles.go internal/ui/model_test.go
git commit -m "ui: highlight selection and refine delayed styling"
```

### Task 4: Better empty states and History layout polish

**Files:**
- Modify: `internal/ui/views.go`
- Test: `internal/ui/model_history_test.go`

**Step 1: Write failing tests for empty states**

- Today/Upcoming: when no items, show a short calm message.
- History: when no done/abandoned for selected day, show empty copy under headings.

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui -count=1`
Expected: FAIL.

**Step 3: Implement empty state text + column spacing**

- Update `renderToday`, `renderUpcoming`, and `renderHistoryBody` to render consistent headings and empty copy.
- Keep data content unchanged.

**Step 4: Run UI tests**

Run: `go test ./internal/ui -count=1`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/ui/views.go internal/ui/model_history_test.go
git commit -m "ui: improve empty states and history layout"
```

### Task 5: Full suite verification

**Step 1: Run full test suite**

Run: `go test ./... -count=1`
Expected: PASS.
