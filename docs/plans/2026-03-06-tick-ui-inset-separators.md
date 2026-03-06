# Tick UI Inset Separators Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan.

**Goal:** Make all three horizontal separator lines inset by 2 spaces on both left and right so they don't touch the frame edges or trigger kitty wrap artifacts.

**Architecture:** Define a single helper that renders an inset horizontal rule given a target width and inset. Use it for (1) the fullscreen separators above/below the sheet and (2) the History internal divider. Keep the sheet border unchanged.

**Tech Stack:** Go, Bubble Tea, Lipgloss.

---

### Task 1: Add an inset separator helper

**Files:**
- Modify: `internal/ui/styles.go`

**Step 1: Write the failing test**

Add tests that pin the intended behavior:

- `separatorLine(windowWidth)` should render a line that is shorter than `contentWidth(windowWidth)` by 4 characters (2 left + 2 right), and it should start with exactly 2 spaces.
- It should return "" when the available width is too small.

Example test shape (adapt to existing test file conventions):

```go
func TestSeparatorLine_InsetTwoSpaces(t *testing.T) {
    w := 80
    cw := contentWidth(w)
    got := separatorLine(w)
    if got == "" {
        t.Fatalf("expected non-empty")
    }
    if !strings.HasPrefix(got, "  ") {
        t.Fatalf("expected two-space prefix, got %q", got)
    }
    if ansi.StringWidth(got) != cw-2 { // 2 left + (cw-4 dashes) + 0/? (no forced right spaces)
        // Use exact expected width once implementation is decided.
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -count=1`

Expected: FAIL with current full-width separator behavior.

**Step 3: Implement the helper**

Implement a new helper (or adjust existing `separatorLine`) to produce:

- 2 leading spaces
- `contentWidth(windowWidth) - 4` dashes
- (optionally) 0 trailing spaces (right inset is achieved by being shorter)

Edge cases:

- If `contentWidth(windowWidth) <= 4`, return "".

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -count=1`

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/ui/styles.go internal/ui/model_history_test.go
git commit -m "ui: inset separator lines to avoid wrap"
```

---

### Task 2: Inset the History internal divider by 2 spaces

**Files:**
- Modify: `internal/ui/views.go`
- Test: `internal/ui/model_history_test.go`

**Step 1: Write the failing test**

Update/add a test to assert that the divider line in `renderHistoryBody`:

- starts with exactly two spaces
- is `innerW - 4` dashes long after the prefix

Pseudo-assertion:

```go
innerW := sheetInnerWidth(m.width)
want := "  " + strings.Repeat("-", innerW-4)
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -count=1`

Expected: FAIL with current divider length/prefix.

**Step 3: Implement minimal code**

In `renderHistoryBody`, render divider using the same inset approach:

- If `innerW <= 4`, divider becomes "" (or a single space line) to avoid negative repeats.
- Otherwise `divider := "  " + strings.Repeat("-", innerW-4)`

Ensure it visually aligns under the table and does not touch the sheet borders.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -count=1`

Expected: PASS.

**Step 5: Manual verification in kitty**

Run in a real TTY:

```bash
go run ./cmd/tick
```

Expected:

- The two global separators (above/below sheet) are inset by 2 spaces.
- The History internal divider is also inset by 2 spaces.
- No stray `-` artifacts appear below the divider.

**Step 6: Commit**

```bash
git add internal/ui/views.go internal/ui/model_history_test.go
git commit -m "ui: inset history divider by two spaces"
```

---

### Task 3: Keep invariants and run full suite

**Files:**
- Modify: tests as needed

**Step 1: Run full tests**

Run: `go test ./... -count=1`

Expected: PASS.

**Step 2: Sanity check narrow widths**

Manually resize terminal narrow to ensure separators don’t go negative/blank unexpectedly.

---

### Task 4: Update PR

**Step 1: Push branch**

Run:

```bash
git push
```

Expected: remote branch updates.
