package exporter

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
)

// chromeAvailable reports whether a local Chrome/Chromium install can be
// launched headlessly. Used to skip PDF export (and its tests) with a clear
// message instead of failing deep inside chromedp on machines without a
// browser installed.
func chromeAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, chromedp.DefaultExecAllocatorOptions[:]...)
	defer cancel()

	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	err := chromedp.Run(browserCtx, chromedp.Navigate("about:blank"))
	return err == nil
}

// captureStyleOverride forces a fixed, unscaled render at the deck's native
// cssWidthPx/cssHeightPx and hides on-screen navigation chrome, so the
// screenshot shows only slide content regardless of what device pixel size
// the live-view scale-to-fit logic would otherwise pick.
//
// The CSS box size is deliberately kept at the deck's native dimensions
// (e.g. 960x540), NOT multiplied up for a higher-resolution capture: slide
// content (font sizes, padding, etc.) is authored in rem/fixed units against
// that native box, exactly like the live view, which achieves crisp
// scaling via a CSS transform (--scale) rather than by resizing the box
// itself. Resizing the box directly (an earlier, buggy version of this
// override did that) leaves rem-sized content the same absolute size while
// the box grows, so content shrinks relative to the background. Higher
// resolution is instead obtained via Chrome's device scale factor
// (see captureSlides' deviceScale param), which rasterizes the same layout
// at more physical pixels without changing anything's relative size.
//
// --slide-width/--slide-height are set as an inline style on <body> by the
// real page templates (web/templates/export.html, presentation.html), so the
// override must target body too: a descendant's own declaration for a custom
// property always wins over an ancestor's, even an ancestor :root rule with
// !important. --scale, by contrast, is set on document.documentElement
// (i.e. :root) by app.js, so overriding it on :root is correct.
const captureStyleOverrideTemplate = `
body { --slide-width: %dpx !important; --slide-height: %dpx !important; }
:root { --scale: 1 !important; }
* { transition: none !important; }
.nav-buttons, .progress-bar-container, .slide-number { display: none !important; }
`

// allImagesLoadedExpr resolves once every <img> currently in the document
// has either finished loading or failed. All slides' markup is present in
// the DOM from the initial page load (goToSlide only toggles CSS visibility,
// it doesn't create/destroy nodes), so waiting for this once before the
// capture loop covers every slide's images, not just the first one.
// Without this wait, a slide could be screenshotted before its images (in
// particular any large asset/ image, or simply one on a slow disk) finished
// decoding, producing a blank/missing image in that slide's capture.
const allImagesLoadedExpr = `
Promise.all(Array.from(document.images).map(function(img) {
	if (img.complete) return true;
	return new Promise(function(resolve) {
		img.addEventListener('load', function() { resolve(true); });
		img.addEventListener('error', function() { resolve(true); });
	});
})).then(function() { return true; })
`

// captureSlides renders each slide of the deck at pageURL to a PNG, in slide
// order. pageURL must be a fully-qualified URL the caller is responsible for
// constructing (e.g. an http://127.0.0.1:port/... URL served from the
// deck's own directory, so both relative asset/foo.png and root-absolute
// /asset/foo.png references resolve exactly as they do under `decky
// serve` — a bare file:// URL cannot resolve root-absolute paths, since
// those resolve against the filesystem root instead of the deck's
// directory). The CSS layout is sized at the deck's native cssWidthPx x
// cssHeightPx (so content proportions match the live view exactly);
// deviceScale controls how many physical pixels each CSS pixel rasterizes
// to (e.g. 2.0 for a sharper, higher-resolution image), so each output
// image is cssWidthPx*deviceScale x cssHeightPx*deviceScale pixels.
// slideCount must match the number of ".slide" elements the page renders.
func captureSlides(ctx context.Context, pageURL string, slideCount, cssWidthPx, cssHeightPx int, deviceScale float64) ([][]byte, error) {
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, chromedp.DefaultExecAllocatorOptions[:]...)
	defer cancel()

	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	styleOverride := fmt.Sprintf(captureStyleOverrideTemplate, cssWidthPx, cssHeightPx)

	images := make([][]byte, 0, slideCount)

	var imagesLoaded bool
	tasks := chromedp.Tasks{
		chromedp.Navigate(pageURL),
		chromedp.EmulateViewport(int64(cssWidthPx), int64(cssHeightPx), chromedp.EmulateScale(deviceScale)),
		chromedp.WaitVisible("#slide-container", chromedp.ByID),
		chromedp.Evaluate(fmt.Sprintf(`
			(function() {
				var style = document.createElement('style');
				style.textContent = %q;
				document.head.appendChild(style);
			})();
		`, styleOverride), nil),
		chromedp.Poll(allImagesLoadedExpr, &imagesLoaded, chromedp.WithPollingTimeout(15*time.Second)),
	}
	if err := chromedp.Run(browserCtx, tasks); err != nil {
		return nil, fmt.Errorf("captureSlides: failed to load %s: %w", pageURL, err)
	}

	for i := 0; i < slideCount; i++ {
		var buf []byte
		err := chromedp.Run(browserCtx, chromedp.Tasks{
			chromedp.Evaluate(fmt.Sprintf(`window.goToSlide(%d)`, i), nil),
			chromedp.Screenshot("#slide-container", &buf, chromedp.NodeVisible, chromedp.ByID),
		})
		if err != nil {
			return nil, fmt.Errorf("captureSlides: failed to capture slide %d: %w", i+1, err)
		}
		images = append(images, buf)
	}

	return images, nil
}
