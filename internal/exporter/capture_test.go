package exporter

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// This synthetic deck deliberately mirrors the REAL page templates'
// cascade structure (web/templates/export.html, presentation.html):
// --slide-width/--slide-height are set as an inline style on <body>, not on
// :root/<html>. #slide-container is a child of body and reads the vars via
// inheritance. Only --scale is left to the real :root/document.documentElement
// convention (app.js sets it there), which this test doesn't need to exercise.
//
// A synthetic page that instead declared --slide-width/--slide-height on
// :root (as the previous version of this test did) could not have caught the
// capture-mode override targeting the wrong element, because body has no own
// declaration to shadow a :root rule in that case. The base size below
// (800x400) is deliberately different from the requested capture size
// (200x100) so a failed override is visible as a wrong-sized screenshot
// instead of an accidental match.
const captureTestBaseWidth = 800
const captureTestBaseHeight = 400

// markerSizeCSSPx is a fixed-pixel-sized element inside slide 1, standing in
// for real slide content (headings, padding, etc. are all authored in
// rem/fixed units against the deck's native box, not in percentages of it).
// It anchors at the container's top-left corner so its rendered footprint
// can be measured directly from the captured image's pixels.
const markerSizeCSSPx = 50

func writeCaptureTestDeck(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	htmlPath := filepath.Join(dir, "deck.html")
	html := fmt.Sprintf(`<!DOCTYPE html>
<html><head><style>
  #slide-container { width: var(--slide-width); height: var(--slide-height); }
  .slide { display: none; width: 100%%; height: 100%%; position: relative; }
  .slide.active { display: block; }
  .slide:nth-child(1) { background: red; }
  .slide:nth-child(2) { background: blue; }
  #marker { position: absolute; top: 0; left: 0; width: %dpx; height: %dpx; background: lime; }
</style></head>
<body style="--slide-width: %dpx; --slide-height: %dpx;">
  <div id="slide-container">
    <div class="slide active">One<div id="marker"></div></div>
    <div class="slide">Two</div>
  </div>
  <script>
    window.goToSlide = function(i) {
      document.querySelectorAll('.slide').forEach(function(s, idx) {
        s.classList.toggle('active', idx === i);
      });
    };
  </script>
</body></html>`, markerSizeCSSPx, markerSizeCSSPx, captureTestBaseWidth, captureTestBaseHeight)
	if err := os.WriteFile(htmlPath, []byte(html), 0o644); err != nil {
		t.Fatalf("failed to write test html: %v", err)
	}
	return htmlPath
}

// markerExtentPixels measures how many pixels from the top-left corner of
// img match markerColor, scanning rightward along row 0 for width and
// downward along column 0 for height. Used to verify the marker div's
// rendered footprint, in physical pixels, against its known CSS size.
func markerExtentPixels(img image.Image, markerColor color.Color) (width, height int) {
	bounds := img.Bounds()
	wantR, wantG, wantB, wantA := markerColor.RGBA()

	matches := func(x, y int) bool {
		r, g, b, a := img.At(x, y).RGBA()
		return r == wantR && g == wantG && b == wantB && a == wantA
	}

	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		if !matches(x, bounds.Min.Y) {
			break
		}
		width++
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		if !matches(bounds.Min.X, y) {
			break
		}
		height++
	}
	return width, height
}

func TestCaptureSlides_OneImagePerSlide(t *testing.T) {
	if !chromeAvailable() {
		t.Skip("no local Chrome/Chromium found, skipping headless capture test")
	}

	htmlPath := writeCaptureTestDeck(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const cssWidth, cssHeight = 200, 100
	const deviceScale = 2.0
	const wantImgWidth, wantImgHeight = int(cssWidth * deviceScale), int(cssHeight * deviceScale)

	images, _, err := captureSlides(ctx, "file://"+htmlPath, 2, cssWidth, cssHeight, deviceScale)
	if err != nil {
		t.Fatalf("captureSlides failed: %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}
	for i, img := range images {
		if len(img) == 0 {
			t.Errorf("image %d is empty", i)
			continue
		}
		cfg, err := png.DecodeConfig(bytes.NewReader(img))
		if err != nil {
			t.Errorf("image %d: failed to decode PNG: %v", i, err)
			continue
		}
		// This is the crux of the capture-mode override bug: the real page
		// templates set --slide-width/--slide-height as an inline style on
		// <body>, so the capture override must target body (not :root) to
		// actually win the cascade. If it targets :root only, the page keeps
		// rendering at its base size (captureTestBaseWidth x
		// captureTestBaseHeight here) instead of the requested capture
		// resolution, and this assertion catches it.
		if cfg.Width != wantImgWidth || cfg.Height != wantImgHeight {
			t.Errorf("image %d: got %dx%d, want %dx%d (capture-mode override failed to force fixed dimensions)",
				i, cfg.Width, cfg.Height, wantImgWidth, wantImgHeight)
		}
	}
}

// TestCaptureSlides_PreservesContentProportionsAtHigherDeviceScale guards
// against the bug where a higher-resolution capture was achieved by
// directly enlarging the --slide-width/--slide-height CSS box (e.g. to 2x),
// instead of keeping the box at its native size and rasterizing it at a
// higher device pixel ratio. Enlarging the box left fixed-size content
// (rem-based padding/fonts in the real templates; a fixed-px marker div
// here) the same absolute size while the box around it grew, so content
// shrank relative to the background. The fix captures at the native CSS
// size with a device scale factor, which must scale content and box
// together.
func TestCaptureSlides_PreservesContentProportionsAtHigherDeviceScale(t *testing.T) {
	if !chromeAvailable() {
		t.Skip("no local Chrome/Chromium found, skipping headless capture test")
	}

	htmlPath := writeCaptureTestDeck(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const cssWidth, cssHeight = 200, 100
	const deviceScale = 2.0

	images, _, err := captureSlides(ctx, "file://"+htmlPath, 1, cssWidth, cssHeight, deviceScale)
	if err != nil {
		t.Fatalf("captureSlides failed: %v", err)
	}
	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}

	img, err := png.Decode(bytes.NewReader(images[0]))
	if err != nil {
		t.Fatalf("failed to decode PNG: %v", err)
	}

	gotWidth, gotHeight := markerExtentPixels(img, color.RGBA{0, 255, 0, 255})

	wantExtent := int(markerSizeCSSPx * deviceScale) // 100
	const tolerance = 3                              // antialiasing at the marker's edge

	if abs(gotWidth-wantExtent) > tolerance || abs(gotHeight-wantExtent) > tolerance {
		t.Errorf("marker rendered at %dx%d px, want ~%dx%d px (marker's CSS size %dpx * device scale %v)."+
			" Getting ~%dpx (marker's CSS size unchanged) would mean the CSS box was resized directly"+
			" instead of rasterized at a higher device scale, reproducing the content-shrinks-relative-to-"+
			"background bug.",
			gotWidth, gotHeight, wantExtent, wantExtent, markerSizeCSSPx, deviceScale, markerSizeCSSPx)
	}
}

// writeOverflowTestDeck builds a two-slide page mirroring the real
// templates' id and scroll structure: each slide is id="slide-N", fills the
// container, and has overflow-y:auto (so the live view scrolls). Slide 0's
// content is deliberately far taller than the box; slide 1's fits.
func writeOverflowTestDeck(t *testing.T, boxHeightCSSPx int) string {
	t.Helper()
	dir := t.TempDir()
	htmlPath := filepath.Join(dir, "deck.html")
	html := fmt.Sprintf(`<!DOCTYPE html>
<html><head><style>
  #slide-container { width: var(--slide-width); height: var(--slide-height); overflow: hidden; }
  .slide { display: none; width: 100%%; height: 100%%; overflow-y: auto; box-sizing: border-box; }
  .slide.active { display: block; }
</style></head>
<body style="--slide-width: 400px; --slide-height: %dpx;">
  <div id="slide-container">
    <div class="slide active" id="slide-0"><div style="height: %dpx;">tall</div></div>
    <div class="slide" id="slide-1"><div style="height: 10px;">short</div></div>
  </div>
  <script>
    window.goToSlide = function(i) {
      document.querySelectorAll('.slide').forEach(function(s, idx) {
        s.classList.toggle('active', idx === i);
      });
    };
  </script>
</body></html>`, boxHeightCSSPx, boxHeightCSSPx*3)
	if err := os.WriteFile(htmlPath, []byte(html), 0o644); err != nil {
		t.Fatalf("failed to write overflow test html: %v", err)
	}
	return htmlPath
}

// TestCaptureSlides_WarnsOnOverflow verifies captureSlides reports a warning
// for a slide whose content is taller than the fixed slide box (the PDF
// crops it, unlike the scrollable live view) and stays silent for one that
// fits.
func TestCaptureSlides_WarnsOnOverflow(t *testing.T) {
	if !chromeAvailable() {
		t.Skip("no local Chrome/Chromium found, skipping headless capture test")
	}

	const boxHeight = 120
	htmlPath := writeOverflowTestDeck(t, boxHeight)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, warnings, err := captureSlides(ctx, "file://"+htmlPath, 2, 400, boxHeight, 1.0)
	if err != nil {
		t.Fatalf("captureSlides failed: %v", err)
	}

	if len(warnings) != 1 {
		t.Fatalf("got %d warnings %v, want exactly 1 (for slide 1 only)", len(warnings), warnings)
	}
	if !bytes.Contains([]byte(warnings[0]), []byte("slide 1")) {
		t.Errorf("warning does not name slide 1: %q", warnings[0])
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
