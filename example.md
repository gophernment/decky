---
title: "Decky: Go & htmx Presentation Engine"
author: "Gophernment"
theme: "slate"
aspectRatio: "16:9"
layout: "cover"
background: "linear-gradient(135deg, #0f172a 0%, #1e293b 100%)"
color: "#ffffff"
---

# Decky 🐹
### Go & htmx Markdown Presentation Engine

Press **Right Arrow**, **Space**, or **Page Down** to navigate.

<!-- 
Welcome to Decky! 
This presentation serves as both a live demonstration of Decky's features and its official documentation.
-->

---
layout: "default"
background: "#0f172a"
color: "#f8fafc"
---

# Core Philosophy

Decky is built to be a simple, lightweight, and modern presentation tool.

- **Developer First**: Write your slides in simple Markdown.
- **Aspect Ratio Lock**: Strictly maintains a professional 16:9 ratio.
- **Ultra Portable**: Serves local presentations via a Go binary, or exports everything into a single offline HTML file.
- **Real-Time Sync**: Synchronizes main view and presenter console using Server-Sent Events (SSE).

<!-- 
Explain why Decky is a great alternative to heavy JS frameworks like Slidev or Marp when you want lightweight Go-based servers.
-->

---
layout: "two-cols"
background: "#1e293b"
color: "#f8fafc"
---

# How It Works

## Slide Parsing
Slides are separated by `---` lines.
Each slide can contain custom local frontmatter (YAML) to override colors, layouts, and backgrounds.

## Code Highlighting
Fenced code blocks are compiled server-side and styled using Chroma syntax highlighting.

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, Decky!")
}
```

<!-- 
Highlight the use of Goldmark and Chroma on the server side for maximum parsing speed and safety.
-->

---
layout: "default"
background: "#0f172a"
color: "#f8fafc"
---

# Presenter Dashboard

Run `decky serve` and navigate to `/presenter` to open the presenter console.

- **Split Screen Previews**: Displays scaled-down previews of the current and next slides.
- **Clock & Timer**: Tracks your elapsed presentation time.
- **Speaker Notes**: Automatically extracts and displays HTML comments at the bottom of slides.
- **Sync Control**: Actions on either the presentation view or presenter console sync instantly via SSE.

<!-- 
Explain how HTMX's SSE extension connects to the server-sent event stream to update the preview screens instantly.
-->

---
layout: "default"
background: "#1e293b"
color: "#f8fafc"
---

# CLI Interface

Manage your slide decks directly from the command line.

### Start the Live Server
```bash
decky serve [-port 3325] example.md
```

### Export for Offline Sharing
```bash
decky export [-o output.html] example.md
```

<!-- 
Explain the two subcommands: serve starts http.ListenAndServe; export bundles all assets into a single static file.
-->

---
layout: "split-h"
ratio: "60/40"
background: "#0f172a"
color: "#f8fafc"
---

# Split Layouts: split-h

::left::
Use `::left::` and `::right::` markers to divide a slide into two
independently-authored regions. Set `ratio: "60/40"` to control the split.

::right::
```go
// Right region can hold its own
// code block, list, or any markdown.
func main() {
    fmt.Println("decky")
}
```

---
layout: "split-v"
ratio: "50/50"
background: "#1e293b"
color: "#f8fafc"
---

# Split Layouts: split-v

::top::
Top region — stacked vertically with `::top::` / `::bottom::`.

::bottom::
Bottom region. `layout: "split-v"` splits by row instead of by column.

---
layout: "split-3"
ratio: "33/34/33"
background: "#0f172a"
color: "#f8fafc"
---

# Split Layouts: split-3

::left::
Left third.

::center::
Center third.

::right::
Right third.

---
layout: "grid-4"
cols: "60/40"
rows: "70/30"
background: "#1e293b"
color: "#f8fafc"
---

# Split Layouts: grid-4

::tl::
Top-left

::tr::
Top-right

::bl::
Bottom-left

::br::
Bottom-right

<!--
Demo of all four split-region layouts: split-h, split-v, split-3, and
grid-4, each with a custom ratio via the ratio/cols/rows frontmatter fields.
-->

---
layout: "cover"
background: "linear-gradient(135deg, #1e3a8a 0%, #0f172a 100%)"
color: "#ffffff"
---

# Start Presenting with Decky!
### Simple, elegant, and blazing fast.

Project Home: [github.com/gophernment/decky](https://github.com/gophernment/decky)

<!-- 
End of the presentation deck. Prompt the user for questions or contributions.
-->
