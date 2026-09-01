# Decky 🐹

**Decky** is a professional, local Markdown presentation engine built with Go and `htmx`. It compiles standard Markdown files into sleek, interactive online slideshows featuring a synchronized presenter console, real-time Server-Sent Events (SSE) state synchronization, and self-contained PDF or HTML exporters.

---

## Features

- **Blazing Fast Markdown Rendering**: Uses Goldmark for GFM-compliant markdown compiling.
- **Server-Side Code Highlighting**: Syntax highlights code blocks using Chroma, compiling them into inline CSS styles for zero external dependency rendering.
- **Aspect Ratio Lock**: Strictly maintains a professional 16:9 layout viewport, auto-scaling to fit the browser window.
- **Presenter Dashboard**: Features a clock, elapsed timer, current/next slide previews, and real-time speaker notes display.
- **SSE Real-Time Sync**: Synchronizes slide navigation in real-time between the main viewer and presenter console.
- **Self-Contained Export**: Renders every slide via a locally installed headless Chrome and assembles them into a single, portable PDF that can be opened offline anywhere.
- **Self-Contained HTML Export**: Bundles every slide, plus all CSS and JS, into a single static HTML file that needs no server or network access to view.

---

## Installation

Ensure you have [Go](https://go.dev/) (version 1.22 or higher) installed. 

### Install Remotely
You can install Decky directly using Go's package manager:

```bash
go install github.com/gophernment/decky@latest
```

Ensure your Go bin directory (`$GOPATH/bin` or `~/go/bin`) is added to your system's `PATH`.

### Install Locally
Alternatively, clone the repository and install it locally:

```bash
git clone https://github.com/gophernment/decky.git
cd decky
go install .
```

---

## Usage

Decky provides four subcommands: `serve`, `export`, `html`, and `usage`.

### 1. Run Live Presentation Server (`serve`)
Start the local HTTP server to display slides and enable presenter synchronization:

```bash
decky serve [-port 3325] example.md
```

The port also defaults to the `PORT` environment variable when `-port` isn't given (`-port` always wins if both are set):

```bash
PORT=3000 decky serve example.md
```

- Open the **Presentation View** at `http://localhost:3325/`.
- Open the **Presenter Console** at `http://localhost:3325/presenter`.

Navigating slides on either window (using arrow keys, space, or buttons) will automatically sync the other window instantly via Server-Sent Events.

A fullscreen toggle button always sits in the bottom-right corner of the **Presentation View**. The prev/next nav buttons and the slide-number indicator (`1 / 12`) are hidden by default — enable them per deck with `showControls` / `showSlideNumber` in the global frontmatter (see [Show/Hide Navigation Controls](#showhide-navigation-controls)).

### 2. Export Standalone Slide Deck (`export`)
Export the presentation into a single self-contained PDF for distribution or offline use:

```bash
decky export [-o output.pdf] example.md
```

Each slide is rendered through a locally installed headless Chrome/Chromium (required for `export`; `serve` does not need it) and captured as a full-resolution image, one per PDF page — so the exported file looks exactly like the live view, including gradients, backgrounds, and syntax-highlighted code, with no server or browser needed to view it afterward.

### 3. Export Standalone HTML (`html`)
Export the presentation into a single self-contained HTML file — no server required to view it:

```bash
decky html [-o output.html] example.md
```

Everything is inlined into the one output file (CSS, JS, and every slide's syntax-highlighted code), so it never needs network access at view time and can be hosted as a static page (e.g. GitHub Pages) or opened directly from disk. Unlike `export`, this does not require a local Chrome/Chromium install.

### 4. Get the Usage Guide (`usage`)
Write a local copy of the tutorial deck (`USAGE.md`) to disk — useful if you installed with `go install` and don't have the repo cloned:

```bash
decky usage [-o USAGE.md]
```

Then view it live with `decky serve USAGE.md`.

---

## Presentation Syntax

### Slide Delimiter
Slides are separated by three hyphens (`---`) on a blank line:

```markdown
# Slide One
Content goes here.

---

# Slide Two
Another slide.
```

### Global & Local Frontmatter
You can define slide properties using YAML frontmatter blocks at the beginning of the slides.
- **Global settings** should be placed on the first slide (Slide 0).
- **Local settings** override themes, layouts, or backgrounds for individual slides.

```markdown
---
title: "Project Pitch"
theme: "slate"
layout: "cover"
background: "linear-gradient(135deg, #0f172a 0%, #1e293b 100%)"
color: "#ffffff"
---
# Main Cover Slide
```

### Supported Layouts
- `default`: Normal vertical flex layout.
- `cover`: Centered cover page layout with gradient background support.
- `two-cols`: Dual-column layout (useful for side-by-side text/images or text/code blocks).

### Custom Fonts
Set the deck's main fonts with `fonts.sans` / `fonts.mono` in the global frontmatter (first slide). Override just one slide's heading font with `headerFont` in that slide's local frontmatter — everything else on the slide keeps using the main font.

```markdown
---
title: "Project Pitch"
fonts:
  sans: 'Space Grotesk'
  mono: 'JetBrains Mono'
---
# Uses the main font

---
headerFont: "Poppins, sans-serif"
---
# This slide's heading uses its own font
```

> **Note:** `serve` mode (and `/presenter`) automatically link to [Google Fonts](https://fonts.google.com) for any custom font you set (`fonts.sans`, `fonts.mono`, `headerFont`), so most Google Font names just work — no extra setup needed. This does reach out to Google's CDN over the network. `export` renders with only locally-available or built-in fonts — if the font you named isn't installed on the machine running `export`, the PDF falls back to the built-in stack (`Inter` for sans, `Fira Code` for mono) instead.

#### Multiple fonts / non-Latin scripts (e.g. Thai)

Each font field accepts a full comma-separated CSS font stack, not just one name. This is useful when your primary font has no glyphs for a script like Thai — list a script-specific font after it as a fallback:

```markdown
---
title: "งานนำเสนอ"
fonts:
  sans: "Poppins, 'Noto Sans Thai'"
---
# หัวข้อ (Poppins สำหรับ Latin, Noto Sans Thai สำหรับไทย)
```

The browser tries each font in order per character, so Latin text renders in `Poppins` and Thai text automatically falls through to `Noto Sans Thai`. In `serve`/`presenter`, every real font name in the stack is fetched from Google Fonts (generic CSS keywords like `sans-serif` are ignored, not fetched).

### Show/Hide Navigation Controls
The prev/next buttons and the slide-number indicator (`1 / 12`) in the **Presentation View** are hidden by default. Turn them on per deck with `showControls` / `showSlideNumber` in the global frontmatter (first slide):

```markdown
---
title: "Project Pitch"
showControls: true
showSlideNumber: true
---
# Main Cover Slide
```

The fullscreen toggle button is always visible regardless of these settings. This config only affects `serve`'s Presentation View — the Presenter Console and `export` output are unaffected.

### Speaker Notes
Add speaker notes inside an HTML comment (`<!-- ... -->`) placed at the very bottom of the slide block:

```markdown
# Slide Title
Slide content here.

<!-- 
Remember to mention the architecture schema here!
-->
```

---

## License

This project is licensed under the [MIT License](LICENSE).
