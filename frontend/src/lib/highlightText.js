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

// splitLayeredSegments is splitHighlightSegments with a second, read-only
// layer: `evidence` ranges (passage text the answer key cites). Segments
// break wherever either layer starts or ends. A segment inside a user
// highlight carries the start/end of that whole highlight, so
// click-to-remove still matches the stored range when evidence splits it.
export function splitLayeredSegments(text, ranges, evidence) {
  const cuts = new Set([0, text.length]);
  for (const r of [...ranges, ...evidence]) {
    cuts.add(r.start);
    cuts.add(r.end);
  }
  const points = [...cuts].filter((p) => p >= 0 && p <= text.length).sort((a, b) => a - b);
  const segments = [];
  for (let i = 0; i < points.length - 1; i++) {
    const [a, b] = [points[i], points[i + 1]];
    const owner = ranges.find((r) => r.start <= a && b <= r.end);
    segments.push({
      text: text.slice(a, b),
      highlighted: !!owner,
      evidence: evidence.some((r) => r.start <= a && b <= r.end),
      start: owner?.start,
      end: owner?.end,
    });
  }
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
