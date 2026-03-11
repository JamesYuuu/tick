# Repository Guidelines

## Project Structure & Module Organization
`cmd/tick/main.go` is the CLI entry point. Core application logic lives under `internal/`: `app` coordinates workflows, `domain` defines task/day models, `store` contains persistence contracts plus the SQLite implementation, `timeutil` wraps time helpers, and `ui` contains the Bubble Tea TUI. Tests live beside the code they cover as `*_test.go`, for example `internal/ui/model_test.go`.

## Build, Test, and Development Commands
Use Go's standard toolchain from the repository root:

- `go run ./cmd/tick` launches the TUI locally.
- `go build ./...` checks that all packages compile.
- `go test ./...` runs the full test suite.
- `go test ./... -cover` reports package coverage during changes that affect behavior.
- `gofmt -w cmd internal` formats the tracked Go source before review.

The app stores local data in `~/.tick/todo.db`; remove that file only if you intentionally want a fresh local state.

## Coding Style & Naming Conventions
Follow standard Go formatting and keep files `gofmt`-clean. Use tabs for indentation, exported identifiers in `CamelCase`, unexported helpers in `camelCase`, and package names that are short and lowercase. Keep new code inside the existing package boundaries rather than reaching across `internal/` layers directly. Prefer table-free, focused tests unless a table materially improves coverage or readability.

## Testing Guidelines
Write tests next to the package under test and name them with `Test<Subject>_<Behavior>`, matching the current style such as `TestApp_Add_RejectsBlankTitle`. Favor deterministic tests with in-memory SQLite helpers and fake clocks where time matters. Run `go test ./...` before opening a PR, and use `go test ./... -run <Pattern>` for quick iteration on one area.

For TUI changes, prefer behavior-first tests and a small number of smoke-level rendering checks. Avoid brittle assertions on exact spacing, color codes, cursor styling, or other visual details unless they protect a known regression.

## Commit & Pull Request Guidelines
Recent history follows Conventional Commit style, including examples like `fix(ui): allow letter input in task modals` and `chore(release): dispatch homebrew tap update`. Keep commits small, imperative, and scoped when helpful (`feat(ui): ...`, `refactor(core): ...`). Pull requests should explain the user-visible change, note any storage or keybinding impacts, link related issues, and include screenshots or terminal captures for TUI updates.
