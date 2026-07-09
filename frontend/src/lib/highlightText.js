// splitHighlightSegments turns one paragraph's plain text plus a set of
// non-overlapping {start,end} character ranges into an ordered list of
// segments for rendering, marking which ones fall inside a highlight.
export function splitHighlightSegments(text, ranges) {
  if (!ranges || ranges.length === 0) return [{ text, highlighted: false }];
  const sorted = [...ranges].sort((a, b) => a.start - b.start);
  const segments = [];
  let cursor = 0;
  for (const { start, end } of sorted) {
    if (start > cursor) segments.push({ text: text.slice(cursor, start), highlighted: false });
    segments.push({ text: text.slice(start, end), highlighted: true, start, end });
    cursor = end;
  }
  if (cursor < text.length) segments.push({ text: text.slice(cursor), highlighted: false });
  return segments;
}

// mergeRanges collapses overlapping/adjacent ranges so re-highlighting over
// an existing highlight (or two highlights that end up touching) doesn't
// produce nested/duplicate <mark> segments.
export function mergeRanges(ranges) {
  const sorted = [...ranges].sort((a, b) => a.start - b.start);
  const merged = [];
  for (const r of sorted) {
    const last = merged[merged.length - 1];
    if (last && r.start <= last.end) {
      last.end = Math.max(last.end, r.end);
    } else {
      merged.push({ ...r });
    }
  }
  return merged;
}

// textOffset computes how many characters into `container`'s full text
// content a given (node, offset) DOM position is — i.e. converts a
// Range boundary into a plain-text character index, walking the
// container's text nodes in document order. Needed because the container
// may already be split into multiple text/<mark> nodes from prior
// highlights, so a raw DOM offset isn't a plain-text offset on its own.
export function textOffset(container, targetNode, targetOffset) {
  let offset = 0;
  let found = false;

  function walk(node) {
    if (found) return;
    if (node === targetNode) {
      offset += targetOffset;
      found = true;
      return;
    }
    if (node.nodeType === Node.TEXT_NODE) {
      offset += node.textContent.length;
    } else {
      for (const child of node.childNodes) {
        walk(child);
        if (found) return;
      }
    }
  }

  walk(container);
  return offset;
}
