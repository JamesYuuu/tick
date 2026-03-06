# Tick UI Refresh Design (Calm Paper)

**Goal:** Improve the TUI visual design to feel less barebones while preserving all existing interactions and behavior.

## Principles

- Calm, paper-like palette: warm background, ink-like text, restrained accent.
- Strong information hierarchy via spacing + typography (bold/underline) rather than noisy colors.
- Better scanability: clear header, clear body container, stable footer layout.
- Preserve behavior: keys, views, list content, and delayed logic remain unchanged.

## Layout

- **Header:** `tick` brand + tab bar with clearer active state.
- **Body:** render inside a padded "sheet" (rounded border), with view title where appropriate.
- **Footer:** split into two lines when needed:
  - Status line (errors / history ratios)
  - Help line (keys)

## Components

- **Tabs:** inactive muted, active bold with subtle underline/bracket.
- **Lists (Today/Upcoming):**
  - Selected row has a paper-highlight effect (background tint or inverse).
  - Delayed items use restrained red + optional symbol.
  - Empty state copy is friendly and short.
- **History:** improve columns alignment and headings (Done/Abandoned), with consistent spacing.

## Responsive Rules

- Small terminals: reduce padding/borders first; never render negative widths/heights.
- Avoid truncation artifacts: keep brand/tabs stable; let body scroll/list handle overflow.

## Testing

- Add/extend UI snapshot-style string tests to assert:
  - header contains `tick`
  - footer shows status and help in stable positions
  - selected item styling changes output
  - empty states are deterministic
