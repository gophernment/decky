document.addEventListener('DOMContentLoaded', () => {
  const container = document.getElementById('slide-container');
  if (!container) return;

  const slides = Array.from(container.querySelectorAll('.slide'));
  const btnPrev = document.getElementById('btn-prev');
  const btnNext = document.getElementById('btn-next');
  const btnFullscreen = document.getElementById('btn-fullscreen');
  const progressBar = document.querySelector('.progress-bar');
  const slideNum = document.querySelector('.slide-number');

  let currentIndex = 0;

  // Fragment stepping (slide frontmatter: fragments: true). -1 means no
  // fragment on the current slide is revealed yet; each next/prev step
  // moves this by one until it runs out, only then does nav fall through
  // to actually changing slides.
  let fragmentIndex = -1;

  function getFragments(slideIndex) {
    return Array.from(slides[slideIndex].querySelectorAll('.fragment'));
  }

  function applyFragmentVisibility() {
    getFragments(currentIndex).forEach((el, i) => {
      el.classList.toggle('visible', i <= fragmentIndex);
    });
  }

  // Aspect Ratio Scaling (dimensions come from --slide-width/--slide-height,
  // set per-deck in the page's inline body style from the parsed aspectRatio).
  // Read the computed style from #slide-container (a descendant of <body>),
  // not document.documentElement (<html>): CSS custom properties only
  // inherit downward, so <html> never sees body's inline override and would
  // always report the :root default (960/540) regardless of the deck's
  // actual aspectRatio.
  function updateScale() {
    const containerStyle = getComputedStyle(container);
    const targetWidth = parseFloat(containerStyle.getPropertyValue('--slide-width')) || 960;
    const targetHeight = parseFloat(containerStyle.getPropertyValue('--slide-height')) || 540;
    const windowWidth = window.innerWidth;
    const windowHeight = window.innerHeight;

    const scaleX = windowWidth / targetWidth;
    const scaleY = windowHeight / targetHeight;

    // Scale down if the screen is smaller than the target slide viewport,
    // and keep it aligned to fit within the viewport bounds.
    // Multiply by 0.95 to give a 5% margin to prevent slide edges from sticking to screen boundaries.
    const scale = Math.min(scaleX, scaleY) * 0.95;

    // Apply scale to the CSS custom property on the document root
    document.documentElement.style.setProperty('--scale', Math.max(scale, 0.1).toString());
  }

  // Go to a specific slide index and update CSS transition classes
  function goToSlide(index) {
    if (index < 0 || index >= slides.length) return;

    // Entering a slide forward starts with no fragment revealed; entering
    // it backward (returning from a later slide) shows all of them, same
    // as reveal.js's default fragment behavior.
    const enteringBackward = index < currentIndex;
    currentIndex = index;
    fragmentIndex = enteringBackward ? getFragments(currentIndex).length - 1 : -1;
    applyFragmentVisibility();

    slides.forEach((slide, i) => {
      slide.classList.remove('active', 'past', 'future');
      if (i === currentIndex) {
        slide.classList.add('active');
      } else if (i < currentIndex) {
        slide.classList.add('past');
      } else {
        slide.classList.add('future');
      }
    });

    // Update progress bar width
    if (progressBar && slides.length > 0) {
      const progress = ((currentIndex + 1) / slides.length) * 100;
      progressBar.style.width = `${progress}%`;
    }

    // Update current/total slide text
    if (slideNum && slides.length > 0) {
      slideNum.textContent = `${currentIndex + 1} / ${slides.length}`;
    }

    // Toggle navigation button disabled states at boundaries
    if (btnPrev) {
      btnPrev.disabled = (currentIndex === 0);
    }
    if (btnNext) {
      btnNext.disabled = (currentIndex === slides.length - 1);
    }

    // Dispatch custom event to allow external hooks (e.g., SSE connection synchronizers) to listen to state changes
    document.dispatchEvent(new CustomEvent('slidechange', { 
      detail: { index: currentIndex } 
    }));
  }

  function nextSlide() {
    const total = getFragments(currentIndex).length;
    if (fragmentIndex < total - 1) {
      fragmentIndex++;
      applyFragmentVisibility();
      return;
    }
    if (currentIndex < slides.length - 1) {
      goToSlide(currentIndex + 1);
    }
  }

  function prevSlide() {
    if (fragmentIndex > -1) {
      fragmentIndex--;
      applyFragmentVisibility();
      return;
    }
    if (currentIndex > 0) {
      goToSlide(currentIndex - 1);
    }
  }

  // ---- Slide overview (thumbnail grid) ----
  // Clones every .slide into a scaled-down tile inside #overview, laid out as
  // a grid. Built lazily on first open; hot-reload does a full page reload so
  // it always rebuilds fresh. Clicking a tile jumps the deck there (which
  // broadcasts over SSE like any other nav) and closes the grid.
  const overviewEl = document.getElementById('overview');
  const btnOverview = document.getElementById('btn-overview');
  let overviewBuilt = false;

  function buildOverview() {
    if (overviewBuilt || !overviewEl || slides.length === 0) return;

    // The clone is rendered at the deck's native slide box and then CSS-scaled
    // as a whole (same trick the live view uses via --scale), so content
    // authored in rem/px stays proportional. --thumb-w is the tile width;
    // derive the scale factor and exact tile height from the deck dimensions.
    const cs = getComputedStyle(container);
    const slideW = parseFloat(cs.getPropertyValue('--slide-width')) || 960;
    const slideH = parseFloat(cs.getPropertyValue('--slide-height')) || 540;
    const thumbW = parseFloat(getComputedStyle(overviewEl).getPropertyValue('--thumb-w')) || 240;
    overviewEl.style.setProperty('--thumb-scale', (thumbW / slideW).toFixed(6));
    overviewEl.style.setProperty('--thumb-h', (thumbW * slideH / slideW).toFixed(2) + 'px');

    const frag = document.createDocumentFragment();
    slides.forEach((slide, i) => {
      const tile = document.createElement('button');
      tile.type = 'button';
      tile.className = 'thumb';
      tile.setAttribute('aria-label', `Go to slide ${i + 1}`);

      // Per-slide headingColors CSS is a <style> sibling emitted just before
      // the slide, scoped to #slide-N. Re-scope a copy to the clone's new id
      // so gradient/colored headings still show in the thumbnail.
      const styleSib = slide.previousElementSibling;
      if (styleSib && styleSib.tagName === 'STYLE') {
        const scoped = document.createElement('style');
        scoped.textContent = styleSib.textContent.replace(
          new RegExp('#slide-' + i + '\\b', 'g'), '#overview-slide-' + i);
        tile.appendChild(scoped);
      }

      const scaler = document.createElement('div');
      scaler.className = 'thumb-scaler';

      const clone = slide.cloneNode(true);
      clone.id = 'overview-slide-' + i;
      clone.classList.remove('active', 'past', 'future');
      clone.querySelectorAll('.fragment').forEach((f) => f.classList.add('visible'));
      clone.querySelectorAll('a, button, input, textarea, select').forEach((el) => {
        el.setAttribute('tabindex', '-1');
      });

      scaler.appendChild(clone);
      tile.appendChild(scaler);

      const num = document.createElement('span');
      num.className = 'thumb-num';
      num.textContent = String(i + 1);
      tile.appendChild(num);

      tile.addEventListener('click', () => {
        goToSlide(i);
        closeOverview();
      });
      frag.appendChild(tile);
    });
    overviewEl.appendChild(frag);
    overviewBuilt = true;
  }

  function markCurrentThumb() {
    if (!overviewBuilt) return;
    const tiles = overviewEl.querySelectorAll('.thumb');
    tiles.forEach((t, i) => t.classList.toggle('current', i === currentIndex));
  }

  function isOverviewOpen() {
    return overviewEl && !overviewEl.hidden;
  }

  function openOverview() {
    if (!overviewEl) return;
    buildOverview();
    overviewEl.hidden = false;
    document.body.classList.add('overview-open');
    markCurrentThumb();
    const current = overviewEl.querySelector('.thumb.current');
    if (current) current.scrollIntoView({ block: 'center' });
  }

  function closeOverview() {
    if (!overviewEl) return;
    overviewEl.hidden = true;
    document.body.classList.remove('overview-open');
  }

  function toggleOverview() {
    if (isOverviewOpen()) {
      closeOverview();
    } else {
      openOverview();
    }
  }

  if (btnOverview) {
    btnOverview.addEventListener('click', toggleOverview);
  }

  // Keep the highlight in sync when the slide changes while the grid is open
  // (e.g. another viewer navigates over SSE).
  document.addEventListener('slidechange', () => {
    if (isOverviewOpen()) markCurrentThumb();
  });

  // Bind Keyboard Navigation
  window.addEventListener('keydown', (e) => {
    const activeEl = document.activeElement;

    // Ignore keypresses if the focus is on input fields, textareas, select dropdowns, or contenteditable elements
    if (activeEl && (
        activeEl.tagName === 'INPUT' ||
        activeEl.tagName === 'TEXTAREA' ||
        activeEl.tagName === 'SELECT' ||
        activeEl.isContentEditable)) {
      return;
    }

    // If a button is focused, only ignore Spacebar to prevent double activation, but allow arrows
    if (activeEl && activeEl.tagName === 'BUTTON' && e.key === ' ') {
      return;
    }

    // `o` toggles the slide overview from anywhere.
    if (e.key === 'o' || e.key === 'O') {
      e.preventDefault();
      toggleOverview();
      return;
    }

    // While the overview covers the deck, only Escape (close) acts; swallow
    // deck-navigation keys so the hidden deck doesn't move underneath it.
    if (isOverviewOpen()) {
      if (e.key === 'Escape') {
        e.preventDefault();
        closeOverview();
      } else if (['ArrowRight', 'ArrowLeft', ' ', 'PageDown', 'PageUp'].includes(e.key)) {
        e.preventDefault();
      }
      return;
    }

    switch (e.key) {
      case 'ArrowRight':
      case ' ': // Spacebar
      case 'PageDown':
        e.preventDefault();
        nextSlide();
        break;
      case 'ArrowLeft':
      case 'PageUp':
        e.preventDefault();
        prevSlide();
        break;
    }
  });

  // Bind Navigation Buttons clicks
  if (btnPrev) {
    btnPrev.addEventListener('click', prevSlide);
  }
  if (btnNext) {
    btnNext.addEventListener('click', nextSlide);
  }

  // Toggle fullscreen on the whole page
  function toggleFullscreen() {
    if (document.fullscreenElement) {
      document.exitFullscreen();
    } else {
      document.documentElement.requestFullscreen();
    }
  }

  if (btnFullscreen) {
    btnFullscreen.addEventListener('click', toggleFullscreen);
    document.addEventListener('fullscreenchange', () => {
      btnFullscreen.textContent = document.fullscreenElement ? '⤢' : '⛶';
    });
  }

  // Bind scale updater to window resize
  window.addEventListener('resize', updateScale);

  // Run initial setups
  updateScale();
  goToSlide(0);

  let isHandlingSSE = false;

  // Hot-reload (triggered when the source markdown file changes) does a
  // full page reload, which would otherwise always snap back to slide 0
  // and lose the scroll position inside a long slide. Stash both in
  // sessionStorage right before reloading and restore them once the new
  // page's slide index is confirmed (per-tab only, not synced across
  // viewers — the SSE slide index is what stays synced across viewers).
  function saveScrollState() {
    const activeSlide = slides[currentIndex];
    sessionStorage.setItem('decky:scrollIndex', String(currentIndex));
    sessionStorage.setItem('decky:scrollTop', String(activeSlide ? activeSlide.scrollTop : 0));
  }

  function restoreScrollState(index) {
    const savedIndex = sessionStorage.getItem('decky:scrollIndex');
    const savedScroll = sessionStorage.getItem('decky:scrollTop');
    if (savedIndex === null || savedScroll === null) return;
    if (parseInt(savedIndex, 10) !== index) return;
    const activeSlide = slides[index];
    if (activeSlide) {
      activeSlide.scrollTop = parseInt(savedScroll, 10);
    }
  }

  if (window.location.protocol.startsWith('http')) {
    // Connect to Server-Sent Events stream
    const eventSource = new EventSource('/events');
    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.reload) {
          saveScrollState();
          window.location.reload();
          return;
        }
        const idx = data.slide;
        if (!isNaN(idx)) {
          if (idx !== currentIndex) {
            isHandlingSSE = true;
            goToSlide(idx);
            isHandlingSSE = false;
          }
          restoreScrollState(idx);
        }
      } catch (err) {
        console.error("Failed to parse SSE JSON payload:", err);
      }
    };
    eventSource.onerror = (err) => {
      console.error("SSE connection dropped, retrying...", err);
    };

    // Listen to slidechange event to synchronize other clients
    document.addEventListener('slidechange', (e) => {
      if (!isHandlingSSE) {
        const idx = e.detail.index;
        fetch('/api/slide', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ index: idx })
        }).catch(err => console.error('Failed to broadcast slide change:', err));
      }
    });
  }

  // Expose slide control functions globally for external integrations (such as Server-Sent Events controllers)
  window.goToSlide = goToSlide;
  window.getCurrentIndex = () => currentIndex;
  window.nextSlide = nextSlide;
  window.prevSlide = prevSlide;
});
