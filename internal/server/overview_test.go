package server_test

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"github.com/gophernment/decky/internal/server"
)

// chromeAvailableForOverview reports whether a local headless Chrome can be
// launched, so the browser-driven overview test skips cleanly (with a
// message) on machines without one instead of failing inside chromedp.
// Mirrors internal/exporter's chromeAvailable; kept local to this package.
func chromeAvailableForOverview() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, chromedp.DefaultExecAllocatorOptions[:]...)
	defer cancel()
	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	return chromedp.Run(browserCtx, chromedp.Navigate("about:blank")) == nil
}

// TestOverviewMode is a browser smoke test for the slide-overview feature
// (web/static/js/app.js + styles.css): pressing `o` builds a thumbnail grid
// with one tile per slide, clicking a tile jumps the deck to that slide and
// closes the grid, and `Escape` closes it too.
func TestOverviewMode(t *testing.T) {
	if !chromeAvailableForOverview() {
		t.Skip("no local Chrome/Chromium found, skipping overview browser test")
	}

	tmpFile, err := os.CreateTemp("", "test_overview_*.md")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	deck := `---
title: Overview Test
---
# Slide One
first
---
# Slide Two
second
---
# Slide Three
third
`
	if _, err := tmpFile.WriteString(deck); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	srv := server.NewServer(tmpFile.Name())
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), chromedp.DefaultExecAllocatorOptions[:]...)
	defer cancelAlloc()
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, cancelTimeout := context.WithTimeout(ctx, 30*time.Second)
	defer cancelTimeout()

	var (
		hiddenBefore  bool
		thumbCount    int
		hiddenOpen    bool
		currentIndex  int
		hiddenAfter   bool
		hiddenOnEsc   bool
		reopenedCount int
	)

	err = chromedp.Run(ctx,
		chromedp.Navigate(ts.URL),
		chromedp.WaitVisible("#slide-container", chromedp.ByID),

		// Overview starts closed and unbuilt.
		chromedp.Evaluate(`document.getElementById('overview').hidden`, &hiddenBefore),

		// `o` opens it and builds one tile per slide.
		chromedp.KeyEvent("o"),
		chromedp.WaitVisible("#overview .thumb", chromedp.ByQuery),
		chromedp.Evaluate(`document.getElementById('overview').hidden`, &hiddenOpen),
		chromedp.Evaluate(`document.querySelectorAll('#overview .thumb').length`, &thumbCount),

		// Clicking the third tile jumps the deck there and closes the grid.
		chromedp.Click(`#overview .thumb:nth-child(3)`, chromedp.ByQuery),
		chromedp.Evaluate(`window.getCurrentIndex()`, &currentIndex),
		chromedp.Evaluate(`document.getElementById('overview').hidden`, &hiddenAfter),

		// Reopen, then Escape closes it.
		chromedp.KeyEvent("o"),
		chromedp.WaitVisible("#overview .thumb", chromedp.ByQuery),
		chromedp.Evaluate(`document.querySelectorAll('#overview .thumb').length`, &reopenedCount),
		chromedp.KeyEvent(kb.Escape),
		chromedp.Evaluate(`document.getElementById('overview').hidden`, &hiddenOnEsc),
	)
	if err != nil {
		t.Fatalf("chromedp run failed: %v", err)
	}

	if !hiddenBefore {
		t.Error("overview should be hidden before it is opened")
	}
	if hiddenOpen {
		t.Error("overview should be visible after pressing 'o'")
	}
	if thumbCount != 3 {
		t.Errorf("expected 3 thumbnails (one per slide), got %d", thumbCount)
	}
	if currentIndex != 2 {
		t.Errorf("clicking the 3rd thumbnail should select slide index 2, got %d", currentIndex)
	}
	if !hiddenAfter {
		t.Error("overview should close after a thumbnail is clicked")
	}
	if reopenedCount != 3 {
		t.Errorf("expected 3 thumbnails after reopening, got %d", reopenedCount)
	}
	if !hiddenOnEsc {
		t.Error("Escape should close the overview")
	}
}
