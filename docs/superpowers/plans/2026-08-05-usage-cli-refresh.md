# Usage Docs Refresh & CLI Hint Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the wrong `serve` startup hint (tells users to run `export` for HTML, which only produces PDF), add a `gophern usage [-o USAGE.md]` command that drops a local copy of the tutorial deck, and bring `USAGE.md`/`README.md` up to date with all four CLI commands.

**Architecture:** A small `main.go` addition (new `usage` subcommand backed by `//go:embed USAGE.md`), a one-line fix in `internal/server/server.go`'s startup message, and prose updates to two markdown files. No new packages, no changes to the parser/exporter/template pipeline.

**Tech Stack:** Go 1.x standard library (`embed`, `flag`, `os`). No new dependencies.

## Global Constraints

- The `usage` command overwrites its `-o` target unconditionally, matching the existing `export`/`html` commands' behavior (neither checks for an existing file before writing) — no new special-casing (spec: "Add `gophern usage` command").
- No test currently pins the old `server.go` hint string (confirmed via grep) — safe to rewrite outright, no back-compat shim needed.
- `USAGE.md`/`README.md` prose is not asserted by any Go test — confirmed via grep; only manual/visual verification needed for those two tasks.

Spec: `docs/superpowers/specs/2026-08-05-usage-cli-refresh-design.md`

---

### Task 1: Fix the `serve` startup hint

**Files:**
- Modify: `internal/server/server.go:84`
- Test: `internal/server/server_test.go`

**Interfaces:**
- Produces: nothing consumed by later tasks — this is a self-contained message fix.

No existing test in `internal/server/server_test.go` calls `Start` directly — it would block forever on `ListenAndServe`, so there's no established pattern to follow for it. Instead, extract the message-building into a small pure function that `Start` calls, so the message can be tested without any networking at all.

- [ ] **Step 1: Write the failing test**

Add to `internal/server/server_test.go`:

```go
func TestStartupMessage_ShowsBothExportHints(t *testing.T) {
	msg := startupMessage("http://localhost:8080/", "http://localhost:8080/presenter", "test.md")

	if !strings.Contains(msg, "gophern export -o output.pdf test.md") {
		t.Errorf("expected PDF export hint, got: %s", msg)
	}
	if !strings.Contains(msg, "gophern html -o output.html test.md") {
		t.Errorf("expected HTML export hint, got: %s", msg)
	}
	if strings.Contains(msg, "gophern export -o output.html") {
		t.Errorf("expected the old incorrect hint (export producing .html) to be gone, got: %s", msg)
	}
}
```

(`server_test.go` already imports `"strings"` — check before adding a duplicate import.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server/... -run TestStartupMessage_ShowsBothExportHints -v`
Expected: FAIL — compile error, `startupMessage` doesn't exist yet.

- [ ] **Step 3: Extract the function and fix the message**

In `internal/server/server.go`, replace line 84:

```go
	fmt.Fprintf(stdout, "\n  Presentation:  %s\n  Presenter:     %s\n\n  Export to a standalone file:\n    gophern export -o output.html %s\n\n",
		presentationURL, presenterURL, markdownFile)
```

with:

```go
	fmt.Fprint(stdout, startupMessage(presentationURL, presenterURL, markdownFile))
```

And add this new function near `Start` (e.g. right after it):

```go
// startupMessage builds the text Start prints once the server is ready:
// the two view URLs, plus a hint for each of the two ways to export a
// standalone copy of the deck (PDF via `export`, HTML via `html` — two
// separate commands, two separate flags, shown as two separate lines so
// neither is mistaken for producing the other's output format).
func startupMessage(presentationURL, presenterURL, markdownFile string) string {
	return fmt.Sprintf("\n  Presentation:  %s\n  Presenter:     %s\n\n  Export to PDF:   gophern export -o output.pdf %s\n  Export to HTML:  gophern html -o output.html %s\n\n",
		presentationURL, presenterURL, markdownFile, markdownFile)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/server/... -run TestStartupMessage_ShowsBothExportHints -v`
Expected: PASS

- [ ] **Step 5: Run the full server test suite to check for regressions**

Run: `go test ./internal/server/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/server/server.go internal/server/server_test.go
git commit -m "fix: correct serve startup hint to show both export and html commands"
```

---

### Task 2: Add `gophern usage [-o USAGE.md]` command

**Files:**
- Modify: `main.go`
- Test: `main_test.go`

**Interfaces:**
- Produces: nothing consumed by later tasks — self-contained CLI command.
- Consumes: `USAGE.md` at the repo root, embedded at build time via `//go:embed USAGE.md` — this task must run *after* Task 3 has finalized `USAGE.md`'s content is not required; embedding just captures whatever `USAGE.md` contains at build time, so order relative to Task 3 doesn't matter functionally, but do this task first since Task 3 will reference the `usage` command's existence in `USAGE.md`'s own prose.

- [ ] **Step 1: Write the failing test**

Add to `main_test.go`:

```go
func TestCLIUsageCommand(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "USAGE.md")

	output, err := runCLI("usage", "-o", outPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v (output: %s)", err, output)
	}
	if !strings.Contains(output, "Wrote "+outPath) {
		t.Errorf("expected confirmation message, got: %s", output)
	}

	written, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected file to be written: %v", err)
	}
	if !strings.Contains(string(written), "# Gophern Usage Guide") {
		t.Errorf("expected written file to contain the usage guide's title, got %d bytes starting with: %.80s", len(written), written)
	}
}

func TestCLIUsageCommandDefaultOutput(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)

	output, err := runCLI("usage")
	if err != nil {
		t.Fatalf("expected no error, got: %v (output: %s)", err, output)
	}
	if !strings.Contains(output, "Wrote USAGE.md") {
		t.Errorf("expected default-output confirmation message, got: %s", output)
	}
	if _, err := os.Stat(filepath.Join(dir, "USAGE.md")); err != nil {
		t.Errorf("expected USAGE.md to be written to cwd: %v", err)
	}
}
```

`main_test.go` will need `"os"` and `"path/filepath"` added to its import block (currently `bytes`, `io`, `strings`, `testing` — check the file's current imports before editing so you don't duplicate an existing one).

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test . -run TestCLIUsageCommand -v`
Expected: FAIL — compile error or `unknown command: usage` (the command doesn't exist yet).

- [ ] **Step 3: Add the embed directive and the `usage` case**

At the top of `main.go`, add the embed import and directive. `main.go` is at the repo root, the same directory as `USAGE.md`, so a direct embed works:

```go
import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/gophernment/gophern/internal/exporter"
	"github.com/gophernment/gophern/internal/server"
)

//go:embed USAGE.md
var usageGuide string
```

Add a new `case "usage":` branch in the `run` function's switch, after the `case "html":` block (before `default:`):

```go
	case "usage":
		usageCmd := flag.NewFlagSet("usage", flag.ContinueOnError)
		usageCmd.SetOutput(stderr)
		output := usageCmd.String("o", "USAGE.md", "Output file path")
		usageCmd.Usage = func() {
			fmt.Fprintln(usageCmd.Output(), "Usage: gophern usage [-o USAGE.md]")
			fmt.Fprintln(usageCmd.Output(), "Options:")
			usageCmd.PrintDefaults()
		}
		if err := usageCmd.Parse(args[2:]); err != nil {
			return err
		}
		if err := os.WriteFile(*output, []byte(usageGuide), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Wrote %s — run \"gophern serve %s\" to try it.\n", *output, *output)
		return nil
```

Add a line to `printUsage`, after the `html` line:

```go
	fmt.Fprintln(w, "  html [-o output.html] <file.md>  Export to a single self-contained HTML file")
	fmt.Fprintln(w, "  usage [-o USAGE.md]  Write the embedded usage guide to a file")
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test . -run TestCLIUsage -v`
Expected: PASS (this also re-runs `TestCLIUsageNoFile`-style pre-existing tests if the `-run` pattern matches them — that's fine, they should still pass unchanged)

- [ ] **Step 5: Run the full top-level test suite to check for regressions**

Run: `go build ./... && go test .`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add main.go main_test.go
git commit -m "feat: add usage command to write the embedded USAGE.md guide to disk"
```

---

### Task 3: Update `USAGE.md` and `README.md`

**Files:**
- Modify: `USAGE.md:43-56` (section 1's command list and prose)
- Modify: `README.md:3,7-14,44,73-75` (intro line, Features list, Usage intro, new `html`/`usage` subsections)

**Interfaces:**
- Consumes: the `usage` command's existence and behavior from Task 2 (to describe it accurately).
- Produces: nothing (terminal documentation task).

- [ ] **Step 1: Update `USAGE.md` section 1**

In `USAGE.md`, replace:

```markdown
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
```

with:

```markdown
### Four commands, that's it
```bash
gophern serve [-port 8080] USAGE.md        # live server + hot reload
gophern export [-o output.pdf] USAGE.md    # single PDF, one page per slide
gophern html [-o output.html] USAGE.md     # single self-contained HTML file
gophern usage [-o USAGE.md]                # write this guide to a local file
```

`serve` starts a local HTTP server with a `/presenter` console and
auto-reloads whenever you save the `.md` file. `export` drives a local
headless Chrome to capture each slide and assemble a PDF — the thing to
hand out or attach to an email. `html` bundles the CSS, JS, and every
slide into one self-contained HTML file with no server required to view
it — the thing to host as a static page (e.g. GitHub Pages) so a live
gophern process isn't required just to view the deck online. `usage`
writes this exact tutorial deck to disk — handy if you installed with
`go install` and don't have this repo cloned to read it from.
```

- [ ] **Step 2: Update `README.md`'s intro line (line 3)**

Replace:

```markdown
**Gophern** is a professional, local Markdown presentation engine built with Go and `htmx`. It compiles standard Markdown files into sleek, interactive online slideshows featuring a synchronized presenter console, real-time Server-Sent Events (SSE) state synchronization, and a self-contained PDF exporter.
```

with:

```markdown
**Gophern** is a professional, local Markdown presentation engine built with Go and `htmx`. It compiles standard Markdown files into sleek, interactive online slideshows featuring a synchronized presenter console, real-time Server-Sent Events (SSE) state synchronization, and self-contained PDF or HTML exporters.
```

- [ ] **Step 3: Add a Features bullet (after line 14)**

After the existing `- **Self-Contained Export**: ...` bullet, add:

```markdown
- **Self-Contained HTML Export**: Bundles every slide, plus all CSS and JS, into a single static HTML file that needs no server or network access to view.
```

- [ ] **Step 4: Update the `## Usage` intro line (line 44)**

Replace:

```markdown
Gophern provides two main subcommands: `serve` and `export`.
```

with:

```markdown
Gophern provides four subcommands: `serve`, `export`, `html`, and `usage`.
```

- [ ] **Step 5: Add `html` and `usage` subsections (after the existing `### 2. Export Standalone Slide Deck (\`export\`)` section, i.e. after its closing paragraph and before the next `---`)**

```markdown
### 3. Export Standalone HTML (`html`)
Export the presentation into a single self-contained HTML file — no server required to view it:

```bash
gophern html [-o output.html] example.md
```

Everything is inlined into the one output file (CSS, JS, and every slide's syntax-highlighted code), so it never needs network access at view time and can be hosted as a static page (e.g. GitHub Pages) or opened directly from disk. Unlike `export`, this does not require a local Chrome/Chromium install.

### 4. Get the Usage Guide (`usage`)
Write a local copy of the tutorial deck (`USAGE.md`) to disk — useful if you installed with `go install` and don't have the repo cloned:

```bash
gophern usage [-o USAGE.md]
```

Then view it live with `gophern serve USAGE.md`.
```

- [ ] **Step 6: Verify the deck still parses and serves**

Run: `go build -o /tmp/gophern-verify . && /tmp/gophern-verify usage -o /tmp/usage-check.md && diff /tmp/usage-check.md USAGE.md && echo "embed matches source"`
Expected: `embed matches source` printed (confirms the embed directive picked up the just-edited `USAGE.md`, not a stale build cache), then `rm /tmp/gophern-verify /tmp/usage-check.md`.

Run: `go build ./... && go test ./...`
Expected: PASS (README.md/USAGE.md prose changes don't affect any test, this just confirms nothing else broke).

- [ ] **Step 7: Commit**

```bash
git add USAGE.md README.md
git commit -m "docs: document the usage and html commands in USAGE.md and README.md"
```
