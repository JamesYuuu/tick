# Tick UI Fullscreen 3-Zone Layout Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the tick TUI render full-screen with clear header/workspace/footer zones, centered as a 96-column max-width layout, and change the title to the `[tick]` text logo.

**Architecture:** Keep view-specific renderers focused on producing workspace content. Move the overall screen layout (centering, separators, fixed footer height, full-height padding/truncation) into a single top-level renderer called by `Model.View()`.

**Tech Stack:** Go, Bubble Tea, Bubbles list/textinput, Lip Gloss.

---

### Task 1: Add layout helpers for centering + line sizing

**Files:**
- Modify: `internal/ui/styles.go`

**Step 1: Write the failing test**

In `internal/ui/model_test.go`, add a new test that:

- Initializes model with a fixed day/app.
- Sends `tea.WindowSizeMsg{Width: 120, Height: 20}`.
- Asserts `View()` renders exactly 20 lines.
- Asserts at least one line begins with spaces (centering padding) when width > 96.

Example snippet:

```go
func TestModel_View_FillsHeight_AndCentersWhenWide(t *testing.T) {
    orig := tickEvery
    tickEvery = 0
    t.Cleanup(func() { tickEvery = orig })

    day := domain.MustParseDay("2026-03-04")
    a := newFakeApp(day, []domain.Task{{ID: 1, Title: "one", Status: domain.StatusActive, CreatedDay: day, DueDay: day}})

    m := NewWithDeps(a, fakeClock{now: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}, time.UTC)
    um, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 20})
    m = um.(Model)
    m = applyCmd(m, m.Init())

    out := m.View()
    lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
    if len(lines) != 20 {
        t.Fatalf("expected 20 lines, got %d", len(lines))
    }
    padded := false
    for _, ln := range lines {
        if strings.HasPrefix(ln, " ") {
            padded = true
            break
        }
    }
    if !padded {
        t.Fatalf("expected centered output to include left padding")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./... -count=1`

Expected: FAIL (height not filled and/or no centering).

**Step 3: Write minimal implementation**

In `internal/ui/styles.go` (or a nearby UI rendering file), implement helpers:

- `contentWidth(windowWidth int) int` (cap at 96)
- `padLeft(windowWidth, contentWidth int) int`
- `padLinesLeft(s string, pad int) string` (prefix each line with pad spaces)
- `clipLine(s string, width int) string` (ASCII-safe clipping)
- `fitHeight(s string, height int) string` (pad with blank lines or truncate to height)

Keep them unexported.

**Step 4: Run tests to verify they pass**

Run: `go test ./... -count=1`

Expected: PASS.

**Step 5: Commit**

Run:

```bash
git add internal/ui/styles.go internal/ui/model_test.go
git commit -m "ui: add fullscreen centering layout helpers"
```

### Task 2: Refactor View() to render 3 zones and fill height

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/styles.go`

**Step 1: Write the failing test**

Add a test in `internal/ui/model_test.go` that:

- Sends `tea.WindowSizeMsg{Width: 80, Height: 24}`.
- Asserts output contains `[tick]` on the first line.
- Asserts there are exactly 2 separator lines made of `-` with length `min(80,96)=80` (after trimming left padding).
- Asserts footer is exactly 2 lines (status + help) at the bottom by checking the last line contains `q:Quit`.

**Step 2: Run test to verify it fails**

Run: `go test ./... -count=1`

Expected: FAIL.

**Step 3: Implement minimal code**

- Change header title from `tick` to `[tick]`.
- Introduce a top-level renderer used by `Model.View()`:
  - Header (1 line)
  - Separator
  - Workspace (variable height)
  - Separator
  - Footer (2 lines)
- Ensure `View()` returns exactly `m.height` lines using `fitHeight`.
- Apply left padding to all lines based on centering rules.

**Step 4: Run tests to verify they pass**

Run: `go test ./... -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/ui/model.go internal/ui/styles.go internal/ui/model_test.go
git commit -m "ui: render fullscreen 3-zone layout with centered column"
```

### Task 3: Resize list/input using centered workspace inner size

**Files:**
- Modify: `internal/ui/model.go`

**Step 1: Write the failing test**

Add a test that sets a narrow width (e.g. `Width: 50, Height: 20`) and asserts:

- `View()` does not exceed the content width per line after trimming left padding.

Keep the assertion simple by scanning lines and ensuring visible rune count (ASCII) is <= 50.

**Step 2: Run test to verify it fails**

Run: `go test ./... -count=1`

Expected: FAIL if sheet/list overflows.

**Step 3: Implement minimal code**

On `tea.WindowSizeMsg`:

- Compute `cw := min(msg.Width, 96)`.
- Compute `workspaceH := max(msg.Height - (1 + 2*1 + 2), 0)`.
- Compute the inner width/height available for list/input by subtracting sheet border/padding.
- Call `SetSize(innerW, innerH)` on lists (and adjust input width if needed).

**Step 4: Run tests to verify they pass**

Run: `go test ./... -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/ui/model.go internal/ui/model_test.go
git commit -m "ui: size workspace content to centered sheet"
```

### Task 4: Update/extend rendering tests for footer invariants

**Files:**
- Modify: `internal/ui/model_test.go`

**Step 1: Add tests**

- Test when `statusMsg` is empty: footer still outputs a blank status line + help line.
- Test in history view: when history footer is shown, it occupies the status line (still 2-line footer total).

**Step 2: Run tests**

Run: `go test ./... -count=1`
Expected: PASS.

**Step 3: Commit**

```bash
git add internal/ui/model_test.go
git commit -m "test: assert fullscreen footer and separator invariants"
```

### Task 5: Final verification and PR update

**Files:**
- Modify: (as needed)

**Step 1: Run full test suite**

Run: `go test ./... -count=1`
Expected: PASS.

**Step 2: Push branch to update PR**

Run: `git push`
