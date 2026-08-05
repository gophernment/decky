# Usage Docs Refresh & CLI Hint Fix — Design

## Problem

1. `gophern serve`'s startup message tells users to run `gophern export -o output.html <file>` to get a standalone file — but `export` only produces PDF (default output `presentation.pdf`); only the separate `html` command produces `.html`. The hint is wrong and will fail or silently produce a mislabeled PDF.
2. There is no way to get a local copy of `USAGE.md` (the tutorial/example deck) without cloning the repo — `go install`-only users have no example file to run `gophern serve` against.
3. `USAGE.md`'s own "Three commands, that's it" section will become stale the moment a 4th command exists.
4. `README.md`'s "## Usage" section only documents `serve` and `export` — it never mentions the `html` command at all (a pre-existing staleness bug, found during this pass, not part of the original report).

## Goal

Fix the wrong hint, add a `gophern usage` command that drops a local `USAGE.md` copy, and bring both `USAGE.md` and `README.md` up to date with all CLI commands.

## Changes

### 1. Fix the serve startup hint (`internal/server/server.go`)

Replace the single wrong hint with two correct ones, one per export format:

```go
fmt.Fprintf(stdout, "\n  Presentation:  %s\n  Presenter:     %s\n\n  Export to PDF:   gophern export -o output.pdf %s\n  Export to HTML:  gophern html -o output.html %s\n\n",
	presentationURL, presenterURL, markdownFile, markdownFile)
```

No existing test pins the old string (confirmed via grep), so this is a safe rewrite.

### 2. Add `gophern usage [-o USAGE.md]` command (`main.go`)

- Embed the repo-root `USAGE.md` into the binary via `//go:embed USAGE.md` (a package-level `var usageGuide string` in `main.go`, since `main.go` already lives at repo root — same directory `USAGE.md` is in, which `go:embed` requires).
- New `flag.NewFlagSet("usage", ...)` following the exact shape of the existing `export`/`html` cases: a `-o` string flag defaulting to `"USAGE.md"`, a `Usage` func, `NArg()` is not required (no positional markdown-file argument — this command takes none).
- Behavior: `os.WriteFile(*output, []byte(usageGuide), 0o644)`, matching the existing commands' no-preflight-check-for-existing-file behavior (they already silently overwrite `-o`'s target; this command matches that established pattern, no new special-casing).
- On success, print a one-line confirmation with a next-step hint: `Wrote %s — run "gophern serve %s" to try it.\n`.
- Add `usage [-o USAGE.md]  Write the embedded usage guide to a file` to `printUsage`'s command list.

### 3. Update `USAGE.md` section 1

- "Three commands, that's it" → "Four commands, that's it".
- Add `gophern usage [-o USAGE.md]` to the code block, with a one-line description in the prose below (same paragraph style as the existing `serve`/`export`/`html` descriptions).

### 4. Update `README.md`

- "## Usage" intro line: "Gophern provides two main subcommands: `serve` and `export`" → mentions all four (`serve`, `export`, `html`, `usage`).
- Add a "### 3. Export Standalone HTML (`html`)" subsection after the existing "### 2. Export Standalone Slide Deck (`export`)" one, matching that subsection's structure and tone (command line example, one paragraph explaining self-contained/no-network-at-view-time behavior — mirrors `USAGE.md`'s existing `html` explanation).
- Add a "### 4. Get the Usage Guide (`usage`)" subsection: command example, one line noting it's useful for `go install`-only users without a repo clone.
- Add one bullet to the top "## Features" list for the self-contained HTML export (currently only the PDF exporter is mentioned there).

## Testing

- `internal/server`: no test currently asserts the old hint string; add one asserting the new message contains both `gophern export -o output.pdf` and `gophern html -o output.html`.
- `main_test.go`: add `TestCLIUsageCommand` — run `gophern usage -o <tempdir>/USAGE.md`, assert the file was written and its content matches the embedded `usageGuide` var, assert the confirmation message appears in stdout.
- No test changes needed for `USAGE.md`/`README.md` (prose-only, not asserted by any Go test — confirmed via grep in the prior session).
