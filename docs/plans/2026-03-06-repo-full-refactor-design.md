## Background

The current branch has accumulated many fast UI iterations and related test additions. It works, but there is visible duplication across layers (`domain`, `store/sqlite`, `app`, `ui`) and across tests (repeated setup, repeated assertions, and overlapping behavior checks). The next step is a full-branch refactor focused on simplification, not framework changes.

---

## Goals

- Keep external behavior unchanged (CLI interactions, data semantics, command outputs, key workflows).
- Aggressively reduce code volume by removing duplication and low-value structure.
- Remove redundant or low-signal tests while preserving a compact, high-signal regression suite.
- Improve maintainability by consolidating repeated logic into clear shared helpers where helpful.

---

## Non-Goals

- No framework migration.
- No product-level behavior redesign.
- No schema or storage semantic changes.
- No broad style-only churn without simplification impact.

---

## Refactor Strategy (Recommended Approach 1)

Refactor in layer order to keep risk localized and debugging fast:

1. `domain`
2. `store/sqlite`
3. `app`
4. `ui`

For each layer:

- identify duplicate code paths and dead branches
- collapse repeated logic into minimal helper(s)
- delete thin wrappers that add no behavioral value
- simplify tests to behavior contracts, not implementation details
- run scoped tests, then full test suite

This sequence limits blast radius and makes regressions easier to attribute.

---

## Architecture and Code Simplification Plan

### 1) Domain layer

- Merge duplicated validation/parsing flows into one canonical path per concept (`Day`, `Task` invariants).
- Remove helpers that only forward arguments without adding invariants or readability.
- Keep constructors/validators that protect invariants; remove alternative entry points that duplicate checks.

### 2) Store / SQLite layer

- Consolidate repeated query boilerplate (prepare/scan/error mapping) into shared private helpers.
- Unify repeated day-range and status filtering fragments where currently reimplemented.
- Remove duplicated conversion glue between DB rows and domain objects.

### 3) App layer

- Normalize repeated “load -> map -> return” and “command -> persist -> report” pipelines.
- Reduce repeated day-boundary handling by using one clear boundary utility path.
- Keep public app methods stable while shrinking internal branching.

### 4) UI layer

- Extract common workspace geometry calculations used by Today/Upcoming/History.
- Consolidate repeated empty-state centering and rendering scaffolding.
- Keep current interaction model and visual output contracts, but remove helper sprawl and repeated small-format functions when they can be merged safely.

---

## Test Reduction Strategy (Aggressive, Behavior-Safe)

Guiding rule: keep tests that validate user-visible behavior and core business rules; remove tests that duplicate coverage or lock in private implementation shape.

### Keep

- Domain invariants and edge cases.
- Store correctness for CRUD, key queries, and migration behavior.
- App-level workflow correctness.
- UI-level key interactions and core render contracts (critical separators/layout invariants, key navigation semantics).

### Remove or Merge

- Duplicate tests asserting the same behavior with only cosmetic variation.
- Tests that assert intermediary private state when output behavior already covers it.
- Near-identical setup blocks repeated across files.
- Overly granular snapshot-like checks where one stronger behavioral assertion is enough.

### Shape

- Favor table-driven tests for same behavior across many inputs.
- Create shared test helpers for fake app/clock/data builders.
- Prefer fewer, stronger end-to-end behavior checks per layer.

---

## Safety and Verification

- After each layer refactor:
  - run layer-targeted tests
  - run full `go test ./... -count=1`
- For UI-sensitive changes:
  - preserve current separator and history interaction contracts through focused tests
- Before completion claim:
  - confirm full suite pass with fresh output evidence

---

## Deliverables

- Simplified production code across all internal layers with reduced duplication.
- Significantly reduced and higher-signal test suite.
- No external behavior changes.
- One or more clean commits with clear rationale in messages.

---

## Acceptance Criteria

- `go test ./... -count=1` passes.
- External behavior remains unchanged for CLI commands and UI interactions.
- Noticeable reduction in code/test volume and repeated logic.
- No orphaned dead code paths or unused helpers introduced by refactor.
