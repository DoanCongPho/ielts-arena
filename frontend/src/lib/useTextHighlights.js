import { useEffect, useMemo, useRef, useState } from 'react';
import { mergeRanges, textOffset } from './highlightText';
import { collectWords } from './vocab';

// useTextHighlights is the attempt pages' highlighter, like the one in the
// real IELTS on-screen test: select text anywhere inside the area (spread
// areaProps onto it) and a floating "Tô đậm" button (HighlightToolbar,
// given `selection`, `apply` and `toolbarRef`) marks it; clicking a mark
// removes it. Highlights are kept per `scope` — the active passage or
// section — then by the id of the HighlightableText they're in (see its
// data-highlight-key), so `ranges` is what the current scope's texts
// render. Never submitted.
//
// A review passes `storageKey`: its highlights are then kept in the
// browser for that test, and `words` lists them across every passage or
// section, with their sentence, as the learner's vocabulary (VocabPanel).
export function useTextHighlights(scope, { storageKey } = {}) {
  const areaRef = useRef(null);
  const toolbarRef = useRef(null);
  const [highlights, setHighlights] = useState({});
  // The full text of each highlighted element, by scope and key, so the
  // words can be read back while their passage isn't on screen.
  const [texts, setTexts] = useState({});
  const [selection, setSelection] = useState(null);

  useEffect(() => {
    setSelection(null);
  }, [scope]);

  // A review starts from what was kept for its test: the highlights made
  // while taking the test were for finding answers, not words to learn,
  // so they don't carry over into the vocabulary list.
  useEffect(() => {
    if (!storageKey) return;
    const saved = readStore(storageKey);
    setHighlights(saved?.highlights || {});
    setTexts(saved?.texts || {});
  }, [storageKey]);

  useEffect(() => {
    if (storageKey) writeStore(storageKey, { highlights, texts });
  }, [storageKey, highlights, texts]);

  useEffect(() => {
    function handleDocMouseDown(e) {
      if (toolbarRef.current?.contains(e.target)) return;
      if (areaRef.current?.contains(e.target)) return;
      setSelection(null);
    }
    document.addEventListener('mousedown', handleDocMouseDown);
    return () => document.removeEventListener('mousedown', handleDocMouseDown);
  }, []);

  // On mouseup, the nearest ancestor with data-highlight-key identifies
  // which text element was selected, and offsets are computed relative to
  // it, so distinct elements never share one bucket of character ranges.
  function onMouseUp() {
    const sel = window.getSelection();
    if (!sel || sel.isCollapsed || sel.rangeCount === 0) {
      setSelection(null);
      return;
    }
    const range = sel.getRangeAt(0);
    const anchorEl =
      range.commonAncestorContainer.nodeType === Node.TEXT_NODE
        ? range.commonAncestorContainer.parentElement
        : range.commonAncestorContainer;
    const targetEl = anchorEl?.closest?.('[data-highlight-key]');
    if (!targetEl) {
      setSelection(null);
      return;
    }
    const from = textOffset(targetEl, range.startContainer, range.startOffset);
    const to = textOffset(targetEl, range.endContainer, range.endOffset);
    const start = Math.min(from, to);
    const end = Math.max(from, to);
    if (start === end) {
      setSelection(null);
      return;
    }
    const rect = range.getBoundingClientRect();
    setSelection({
      key: targetEl.dataset.highlightKey,
      text: targetEl.textContent,
      start,
      end,
      x: rect.left + rect.width / 2,
      y: rect.top,
    });
  }

  function apply() {
    if (!selection) return;
    const { key, text, start, end } = selection;
    setHighlights((prev) => {
      const scoped = { ...(prev[scope] || {}) };
      scoped[key] = mergeRanges([...(scoped[key] || []), { start, end }]);
      return { ...prev, [scope]: scoped };
    });
    setTexts((prev) => ({ ...prev, [scope]: { ...(prev[scope] || {}), [key]: text } }));
    window.getSelection()?.removeAllRanges();
    setSelection(null);
  }

  // removeAt drops highlights by where they are: [{scope, key, start, end}].
  function removeAt(spots) {
    setHighlights((prev) => {
      const next = { ...prev };
      for (const { scope: s, key, start, end } of spots) {
        next[s] = { ...(next[s] || {}) };
        next[s][key] = (next[s][key] || []).filter((r) => !(r.start === start && r.end === end));
      }
      return next;
    });
  }

  const words = useMemo(() => collectWords(highlights, texts), [highlights, texts]);

  return {
    areaProps: { ref: areaRef, onMouseUp },
    ranges: highlights[scope] || {},
    remove: (key, start, end) => removeAt([{ scope, key, start, end }]),
    words,
    removeWord: (w) => removeAt(w.at),
    selection,
    apply,
    toolbarRef,
  };
}

// The browser may refuse storage (private mode, blocked site data); the
// highlights then just last the visit.
function readStore(key) {
  try {
    return JSON.parse(localStorage.getItem(key) || 'null');
  } catch {
    return null;
  }
}

function writeStore(key, value) {
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // ignore
  }
}
