# Heading Colors & Gradients — Design

## Problem

Today `color`/`headerFont` in frontmatter style a slide uniformly — there is
no way to give `h1`, `h2`, `h3`, `h4` different colors, and no way to apply a
CSS gradient to heading text at all. `h3`/`h4` additionally have zero CSS
rules today (`web/static/css/styles.css` has no `h3`/`h4` selector), so they
render with plain browser defaults regardless of any slide styling.

## Goal

Let users set a solid color or CSS gradient per heading level (h1–h4), via
frontmatter, with a deck-wide default and per-slide override.

## Frontmatter shape

New key `headingColors`, a map keyed by heading level:

```yaml
# Global default — slide 0's frontmatter block
---
headingColors:
  h1: "linear-gradient(90deg, #f472b6, #60a5fa)"
  h2: "#22d3ee"
---

# Local override — only overrides the levels it lists
---
headingColors:
  h1: "#facc15"   # h2 still falls back to the global value above
---
```

- Recognized keys: `h1`, `h2`, `h3`, `h4`. Unknown keys are ignored (no
  validation error — consistent with how unknown frontmatter fields behave
  elsewhere in this parser).
- Merge is per-key: for each level, use the local slide's value if present,
  else fall back to the global (slide 0) value, else no override (heading
  keeps whatever `color`/browser default it would have had).
- No CSS validation on the value — raw passthrough, same trust model as the
  existing `color`/`background` keys (this codebase already renders with
  `html.WithUnsafe()` and treats frontmatter as author-trusted content).

## Gradient detection

A value is treated as a gradient if it contains the substring `gradient(`
(case-insensitive) — matches `linear-gradient`, `radial-gradient`,
`conic-gradient`, and their `repeating-*` variants. Anything else is treated
as a solid color.

- Solid: `color: <value>;`
- Gradient: `background: <value>; -webkit-background-clip: text; background-clip: text; color: transparent; -webkit-text-fill-color: transparent;`

## Rendering integration

Solid-only color could ride the existing CSS-custom-property mechanism (like
`headerFont`), but gradient text needs five properties toggled together per
level, which a single CSS var can't express, and `h3`/`h4` have no existing
selector to hook a var into anyway. So:

- After merging global+local `headingColors` for a slide, compute the full
  CSS text in Go and store it on the slide as `HeadingColorCSS
  template.CSS`, e.g.:
  ```css
  #slide-3 h1 { background: linear-gradient(90deg, #f472b6, #60a5fa); -webkit-background-clip: text; background-clip: text; color: transparent; -webkit-text-fill-color: transparent; }
  #slide-3 h2 { color: #22d3ee; }
  ```
  Scoped with the slide's existing DOM id (`#slide-{{ .Index }}`, already
  emitted by `_slide.html`), so no new id/class scheme is needed.
- Emit `{{ if .HeadingColorCSS }}<style>{{ .HeadingColorCSS }}</style>{{ end }}`
  immediately before the slide's `<div id="slide-N" ...>` in
  `web/templates/_slide.html`. This is the single shared partial used by
  `presentation.html`, `export.html`, and `presenter.html`, so all three
  renderers (serve/export/html) get this for free with no duplication.
- Add base `.slide h3` and `.slide h4` selectors to
  `web/static/css/styles.css` (font-size etc., consistent with the existing
  `h1`/`h2` scale) since none exist today — required for `headingColors.h3`/
  `.h4` to have any visible default to override.

## Data flow / structs touched

- `internal/parser/parser.go`:
  - `Presentation.HeadingColors map[string]string \`yaml:"headingColors"\`` (global)
  - `Slide.HeadingColors map[string]string \`yaml:"headingColors"\`` (local)
  - `Slide.HeadingColorCSS template.CSS` (computed, not user-set)
  - add `"headingColors"` to `coreFrontmatterKeys` allowlist
  - new function (e.g. `buildHeadingColorCSS(slideIndex int, merged map[string]string) template.CSS`) called after per-slide frontmatter is parsed, once global+local are merged
- `web/templates/_slide.html`: emit the `<style>` block
- `web/static/css/styles.css`: add `.slide h3`, `.slide h4` base rules

No changes needed to `internal/server/server.go`, `internal/exporter/html.go`,
or `internal/exporter/exporter.go` template-func registration — this feature
doesn't need a new template func, since the CSS is fully precomputed in Go
and just interpolated as a value.

## Testing

- Parser unit tests: global-only, local-only, local-overrides-one-level,
  gradient detection (`gradient(` substring match, case variants), no
  `headingColors` set (field stays empty, no `<style>` emitted).
- Render test: confirm `<style>` block appears before the slide div and
  contains the expected scoped selectors for a slide with mixed
  solid/gradient levels.
- Manual: `gophern serve` a deck using slide 16-18 area of `USAGE.md` as a
  base, add a `headingColors` example, confirm h1 gradient renders in
  browser and h3/h4 pick up their new base sizing.
