import { splitHighlightSegments } from '../../lib/highlightText';

// HighlightableText renders `text` split into plain/highlighted segments
// per `ranges` (character offsets — see lib/highlightText). It carries a
// `data-highlight-key` attribute that the nearest ancestor's mouseup
// handler (see ReadingAttemptPage) uses to find which text was selected
// and compute offsets against, and click-to-remove on already-highlighted
// spans. Used for the reading passage paragraphs as well as question
// instructions/text, so keywords can be marked anywhere on the page.
export default function HighlightableText({ id, text, ranges, onRemoveRange, as: Tag = 'span', className }) {
  const segments = splitHighlightSegments(text, ranges || []);
  return (
    <Tag data-highlight-key={id} className={className}>
      {segments.map((seg, i) =>
        seg.highlighted ? (
          <mark
            key={i}
            className="reading-highlight-mark"
            title="Bấm để bỏ tô đậm"
            onClick={() => onRemoveRange?.(id, seg.start, seg.end)}
          >
            {seg.text}
          </mark>
        ) : (
          <span key={i}>{seg.text}</span>
        ),
      )}
    </Tag>
  );
}
