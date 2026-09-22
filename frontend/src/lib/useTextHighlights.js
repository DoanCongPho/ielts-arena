import { useEffect, useRef, useState } from 'react';
import { mergeRanges, textOffset } from './highlightText';

// useTextHighlights is the attempt pages' highlighter, like the one in the
// real IELTS on-screen test: select text anywhere inside the area (spread
// areaProps onto it) and a floating "Tô đậm" button (HighlightToolbar,
// given `selection`, `apply` and `toolbarRef`) marks it; clicking a mark
// removes it. Highlights are kept per `scope` — the active passage or
// section — then by the id of the HighlightableText they're in (see its
// data-highlight-key), so `ranges` is what the current scope's texts
// render. Frontend-only, never submitted.
export function useTextHighlights(scope) {
  const areaRef = useRef(null);
  const toolbarRef = useRef(null);
  const [highlights, setHighlights] = useState({});
  const [selection, setSelection] = useState(null);

  useEffect(() => {
    setSelection(null);
  }, [scope]);

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
    setSelection({ key: targetEl.dataset.highlightKey, start, end, x: rect.left + rect.width / 2, y: rect.top });
  }

  function apply() {
    if (!selection) return;
    const { key, start, end } = selection;
    setHighlights((prev) => {
      const scoped = { ...(prev[scope] || {}) };
      scoped[key] = mergeRanges([...(scoped[key] || []), { start, end }]);
      return { ...prev, [scope]: scoped };
    });
    window.getSelection()?.removeAllRanges();
    setSelection(null);
  }

  function remove(key, start, end) {
    setHighlights((prev) => {
      const scoped = { ...(prev[scope] || {}) };
      scoped[key] = (scoped[key] || []).filter((r) => !(r.start === start && r.end === end));
      return { ...prev, [scope]: scoped };
    });
  }

  return { areaProps: { ref: areaRef, onMouseUp }, ranges: highlights[scope] || {}, remove, selection, apply, toolbarRef };
}
