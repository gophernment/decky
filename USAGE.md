---
title: "Gophern Usage Guide"
author: "Gophernment"
theme: "slate"
aspectRatio: "16:9"
fonts:
  sans: "Poppins, 'Noto Sans Thai'"
showControls: true
showSlideNumber: true
layout: "cover"
background: "linear-gradient(135deg, #0f172a 0%, #1e293b 100%)"
color: "#ffffff"
---

# Gophern Usage Guide 🐹
### A complete, runnable tour of every feature

This deck is a real `.md` file — run it with `gophern serve USAGE.md`
and navigate with **→**, **Space**, or **Page Down**. A fullscreen
button sits in the bottom-right corner; the prev/next buttons and slide
number are off by default (turned on for this deck — see slide 19).

<!--
This is USAGE.md: a hands-on manual. Every syntax shown on a slide is the
exact syntax that produced that very slide — open the raw file side by
side with the rendered output to see how each piece works.
-->

---
layout: "default"
background: "#0f172a"
color: "#f8fafc"
---

# 1. Installing & Running

### Install
```bash
go install github.com/gophernment/gophern@latest
# or clone the repo and `go build .`
```

### Three commands, that's it
```bash
gophern serve [-port 8080] USAGE.md        # live server + hot reload
gophern export [-o output.pdf] USAGE.md    # single PDF, one page per slide
gophern html [-o output.html] USAGE.md     # single self-contained HTML file
```

`serve` starts a local HTTP server with a `/presenter` console and
auto-reloads whenever you save the `.md` file. `export` drives a local
headless Chrome to capture each slide and assemble a PDF — the thing to
hand out or attach to an email. `html` bundles the CSS, JS, and every
slide into one self-contained HTML file with no server required to view
it — the thing to host as a static page (e.g. GitHub Pages) so a live
gophern process isn't required just to view the deck online.

<!--
Mention that `html` never needs network access at view time — everything
is inlined into the single output file, including the syntax-highlighted
code blocks. `export`'s PDF, by contrast, requires a local Chrome/Chromium
install at build time (not at view time — the PDF itself is static).
-->

---
layout: "default"
background: "#1e293b"
color: "#f8fafc"
---

# 2. Slide Delimiter

Slides are separated by a line containing exactly three hyphens, `---`,
on its own line.

```markdown
# Slide One
Content here.

---

# Slide Two
More content.
```

A `---` **inside a fenced code block** (between ` ``` ` or `~~~`) is never
treated as a delimiter — it's safe to show shell output or YAML examples
containing `---` inside your slides.

---
layout: "default"
background: "#0f172a"
color: "#f8fafc"
---

# 3. Frontmatter: Global vs. Local

Every slide can start with a YAML frontmatter block (`---` ... `---`)
setting properties for that slide.

- **Global**: put it on **Slide 0** (the very first block in the file).
  It sets defaults — `title`, `author`, `theme`, `aspectRatio`, `fonts` —
  that apply to the whole deck.
- **Local**: any later slide's own frontmatter block overrides `layout`,
  `background`, `color`, `headerFont`, etc. for **that slide only**.

```markdown
---
title: "My Deck"
fonts:
  sans: 'Space Grotesk'
---
# Slide 0 inherits both title and font

---
background: "#1e293b"
---
# Slide 1 only overrides background
```

<!--
Global frontmatter must be the very first block in the file (before any
slide content). If the file has no frontmatter at all, gophern falls back
to sensible defaults: title "Presentation", aspectRatio "16:9".
-->

---
layout: "two-cols"
background: "#0f172a"
color: "#f8fafc"
---

# 4. Per-Slide Config Reference

## Local frontmatter keys
- `layout` — one of `default`, `cover`, `two-cols`, `split-h`, `split-v`, `split-3`, `grid-4`
- `background` — any CSS `background` value (color, gradient, image URL)
- `color` — the slide's base text color
- `headerFont` — overrides just this slide's `<h1>` font
- `fragments` — step-reveal this slide's list items one at a time (see slide 20)

## Split-layout-only keys
- `ratio` — e.g. `"60/40"` for `split-h`/`split-v`, `"30/40/30"` for `split-3`
- `cols` / `rows` — independent ratios, `grid-4` only

Every key above goes in a slide's **own** frontmatter block and affects
**that slide only** — none of them are inherited by later slides.

---
layout: "default"
background: "#1e293b"
color: "#f8fafc"
---

# 5. Layout: `default`

The normal vertical flow layout — headings, paragraphs, lists, and code
blocks stack top to bottom. This is what you get when a slide has no
`layout:` field at all, or `layout: "default"` explicitly.

- Bulleted lists work as expected
- So do **bold**, *italic*, and `inline code`
- Fenced code blocks get server-side syntax highlighting (see slide 9)

```markdown
---
layout: "default"
---
# Any Title
Regular markdown content goes here.
```

---
layout: "cover"
background: "linear-gradient(135deg, #1e3a8a 0%, #0f172a 100%)"
color: "#ffffff"
---

# 6. Layout: `cover`
### Centered, large title — built for opening/closing slides

```markdown
---
layout: "cover"
background: "linear-gradient(135deg, #1e3a8a 0%, #0f172a 100%)"
color: "#ffffff"
---
# Title
### Subtitle
```

<!--
cover centers everything both horizontally and vertically and applies a
larger, gradient-accented heading style — this very slide is the example.
-->

---
layout: "two-cols"
background: "#0f172a"
color: "#f8fafc"
---

# 7. Layout: `two-cols`

## Left-ish content
A CSS-grid two-column layout. Headings and paragraphs auto-flow into the
grid — good for a quick side-by-side without needing `::left::`/`::right::`
markers.

## Right-ish content
For precise control over what goes in which column, prefer the newer
`split-h` layout (slide 12) instead — it lets you address each side
explicitly by name.

```markdown
---
layout: "two-cols"
---
# Title
## Section A
...
## Section B
...
```

---
layout: "default"
background: "#1e293b"
color: "#f8fafc"
---

# 8. Speaker Notes

Add an HTML comment at the very bottom of a slide's content — it never
renders on screen, but shows up in the **presenter console**
(`gophern serve` → open `/presenter`).

```markdown
# Slide Title
Visible content.

<!--
Only you see this, in the presenter view's notes panel.
-->
```

A comment only counts as speaker notes if nothing but whitespace follows
it — an HTML comment used mid-content (or inside a code block) is left
alone and rendered normally.

---
layout: "default"
background: "#0f172a"
color: "#f8fafc"
---

# 9. Code Highlighting

Fenced code blocks are compiled server-side with Chroma — no client-side
JavaScript highlighter, no flash of unstyled code.

````markdown
```go
func main() {
    fmt.Println("gophern")
}
```
````

```go
func main() {
    fmt.Println("gophern")
}
```

---
layout: "default"
background: "#0f172a"
color: "#f8fafc"
---

# 10. Images & Assets

Reference images with plain Markdown — a **relative path, no leading `/`**:

```markdown
![alt text](asset/photo.png)
```

The file must live in an **`asset/` folder next to the `.md` file** —
not `assets/` (no `s`), not a subfolder of a subfolder, a **sibling** of
the deck:

```
my-deck/
├── deck.md
└── asset/
    └── photo.png
```

<!--
The relative path (no leading slash) is what makes this syntax work
identically across all three commands: `serve` maps `/asset/` to this
folder over HTTP, `export`'s headless-Chrome capture serves the deck's
own directory as the static root, and `html`'s self-contained output
ships no server at all — it keeps the path exactly as written and just
expects `asset/` copied alongside the output file. A leading-slash
`/asset/foo.png` breaks under `html` (opened as file://, or hosted at
any non-root path) — always write it relative.
-->

---
layout: "two-cols"
background: "#0f172a"
color: "#f8fafc"
---

# 11. Inline HTML & Styling

## Raw HTML passes through
Gophern's markdown renderer allows raw HTML tags directly in your slide
content — for one-off styling that frontmatter fields don't cover.

```markdown
<span style="color:#f472b6; font-weight:700;">
  Custom inline color
</span>

<div style="text-align:right; border:1px solid #333;">
  A hand-styled box
</div>
```

## Use sparingly
<span style="color:#f472b6; font-weight:700;">This text is pink</span> via
the exact snippet above. Prefer `background`/`color`/`headerFont`
frontmatter for whole-slide styling — reach for inline HTML only for
small, one-off cases those fields don't reach.

---
layout: "split-h"
ratio: "55/45"
background: "#1e293b"
color: "#f8fafc"
---

# 12. Split Layouts: `split-h`

::left::
Divide a slide into named regions with `::name::` markers. `split-h`
gives you `::left::` and `::right::`, each rendered independently — code
highlighting, lists, everything works the same as the main slide body.

Control the width split with `ratio: "55/45"` (any two numbers that sum
to whatever you like — they're read as relative weights).

::right::
```markdown
---
layout: "split-h"
ratio: "55/45"
---
::left::
Left content.

::right::
Right content.
```

---
layout: "split-v"
ratio: "40/60"
background: "#0f172a"
color: "#f8fafc"
---

# 13. Split Layouts: `split-v`

::top::
Same idea as `split-h`, but stacked vertically. Regions are `::top::` and
`::bottom::`, and `ratio` controls the height split instead of width.

::bottom::
```markdown
---
layout: "split-v"
ratio: "40/60"
---
::top::
Top content.

::bottom::
Bottom content.
```

---
layout: "split-3"
ratio: "30/40/30"
background: "#1e293b"
color: "#f8fafc"
---

# 14. Split Layouts: `split-3`

::left::
Three columns: `::left::`, `::center::`, `::right::`.

::center::
`ratio` takes three numbers here, one per column — this slide uses
`"30/40/30"` so the center column is a bit wider.

::right::
```markdown
layout: "split-3"
ratio: "30/40/30"
```

---
layout: "grid-4"
cols: "50/50"
rows: "50/50"
background: "#0f172a"
color: "#f8fafc"
---

# 15. Split Layouts: `grid-4`

::tl::
A 2×2 grid: `::tl::`, `::tr::`, `::bl::`, `::br::`.

::tr::
Two independent ratios — `cols` for the column split, `rows` for the row
split — since this layout has two axes at once.

::bl::
```markdown
layout: "grid-4"
cols: "60/40"
rows: "70/30"
```

::br::
Any region can be left out entirely — the grid cell just stays empty.

---
layout: "default"
background: "#1e293b"
color: "#f8fafc"
---

# 16. Custom Fonts: the basics

Set the deck's main fonts once, globally:

```markdown
---
fonts:
  sans: 'Space Grotesk'
  mono: 'JetBrains Mono'
---
```

`gophern serve` and `/presenter` automatically fetch the named font from
Google Fonts — no extra setup. `gophern export` and `gophern html` do
**not** fetch fonts (to stay fully offline, self-contained output); an
unavailable font falls back to the built-in stack (`Inter` / `Fira Code`)
instead.

---
headerFont: "Poppins, sans-serif"
background: "#0f172a"
color: "#f8fafc"
---

# 17. Custom Fonts: per-slide heading

This heading uses `headerFont`, set only on this slide's local frontmatter.

Override just **this slide's heading** with `headerFont` in its local
frontmatter — the body text and every other slide keep using the deck's
main font.

```markdown
---
headerFont: "Poppins, sans-serif"
---
# Only this heading uses Poppins
```

---
layout: "default"
background: "#1e293b"
color: "#f8fafc"
---

# 18. Custom Fonts: หลายฟอนต์ / Thai fallback

ข้อความไทยผสมกับ English ในสไลด์เดียวกัน — สไลด์นี้ (และทั้งเดค) ใช้
`fonts.sans: "Poppins, 'Noto Sans Thai'"` ที่ตั้งไว้ใน **global frontmatter**
บนสไลด์ปก (Slide 0) ดังนั้นข้อความไทยตรงนี้จึงแสดงถูกต้องแบบ live จริง

```markdown
---
fonts:
  sans: "Poppins, 'Noto Sans Thai'"
---
```

Any font field accepts a full comma-separated CSS font stack, not just
one name — list a script-specific fallback after your primary font.

<!--
Note: `fonts:` is a Presentation-level (global-only) field — it must live
in Slide 0's frontmatter to take effect, unlike `background`/`color`/
`headerFont` which work per-slide. This deck sets it once on the cover
slide, so every slide (including this one) already renders with the Thai
fallback active.
-->

---
layout: "default"
background: "#0f172a"
color: "#f8fafc"
---

# 19. Show/Hide Navigation Controls

The prev/next buttons and the `1 / N` slide-number indicator in the
**Presentation View** (`serve`, not `/presenter`) are hidden by default.
This deck turns them on via its global frontmatter:

```markdown
---
showControls: true
showSlideNumber: true
---
```

Leave them unset (or `false`) for a chrome-free presentation. The
fullscreen button in the bottom-right is always visible either way.

<!--
Both keys default to false (hidden) and only affect the live serve
Presentation View — the Presenter Console and exported HTML always show
their own controls unconditionally.
-->

---
layout: "default"
background: "#1e293b"
color: "#f8fafc"
fragments: true
---

# 20. Fragments: step-reveal list items

Set `fragments: true` in a slide's local frontmatter to reveal that
slide's list items one at a time instead of all at once.

- Press **→**, **Space**, or click "Next" to reveal this line
- Then this one
- Then this one — only once every item is shown does the same key move
  you on to the next slide
- Going **back** re-enters this slide with every item already visible,
  then hides them one at a time on further "Prev" presses

```markdown
---
fragments: true
---
# Title
- Revealed first
- Revealed second
- Revealed third
```

<!--
fragments only affects <li> elements rendered from this slide's own
markdown (every list on the slide, including nested ones, numbered in
document order) — it's a per-slide opt-in, not a deck-wide default.
-->

---
headingColors:
  h1: "linear-gradient(90deg, #f472b6, #60a5fa)"
  h2: "#22d3ee"
  h3: "#a3e635"
  h4: "#fbbf24"
background: "#1e293b"
color: "#f8fafc"
---

# 21. Heading Colors & Gradients
## A solid cyan H2
### A solid green H3
#### A solid amber H4

Give each heading level (`h1`-`h4`) its own color or gradient with
`headingColors` frontmatter — set deck-wide defaults on Slide 0, override
per level on any later slide (levels you don't list there keep the deck
default).

```markdown
---
headingColors:
  h1: "linear-gradient(90deg, #f472b6, #60a5fa)"
  h2: "#22d3ee"
---
```

<!--
Any value containing "gradient(" (linear-gradient, radial-gradient,
conic-gradient, case-insensitive) renders via the background-clip:text
technique; anything else is treated as a solid CSS color. Merge is
per-level: a level set in a slide's own headingColors overrides just that
level, every other level still falls back to whatever was set globally on
Slide 0.
-->

---
layout: "cover"
background: "linear-gradient(135deg, #1e3a8a 0%, #0f172a 100%)"
color: "#ffffff"
---

# That's the whole toolkit
### Markdown in, presentation out.

Project Home: [github.com/gophernment/gophern](https://github.com/gophernment/gophern)

Try editing this file and watch `gophern serve` hot-reload instantly.

<!--
Closing slide. Encourage the reader to open USAGE.md next to the rendered
output and experiment — every syntax on every prior slide is copy-pasteable.
-->
