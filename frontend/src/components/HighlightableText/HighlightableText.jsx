import { splitHighlightSegments, splitLayeredSegments } from '../../lib/highlightText';
import './HighlightableText.css';

// HighlightableText renders `text` split into plain/highlighted segments
// per `ranges` (character offsets — see lib/highlightText). It carries a
// `data-highlight-key` attribute that the nearest ancestor's mouseup
// handler (see ReadingAttemptPage) uses to find which text was selected
// and compute offsets against, and click-to-remove on already-highlighted
// spans. Used for the reading passage paragraphs as well as question
// instructions/text, so keywords can be marked anywhere on the page.
// `evidence`, in a graded review, marks the passage text the answer key
// cites as a separate, non-removable layer.
export default function HighlightableText({ id, text, ranges, evidence, onRemoveRange, as: Tag = 'span', className }) {
  const segments = evidence?.length
    ? splitLayeredSegments(text, ranges || [], evidence)
    : splitHighlightSegments(text, ranges || []);
  return (
    <Tag data-highlight-key={id} className={className}>
      {segments.map((seg, i) => {
        if (seg.highlighted) {
          return (
            <mark
              key={i}
              className={`reading-highlight-mark ${seg.evidence ? 'reading-evidence-mark' : ''}`}
              title="Bấm để bỏ tô đậm"
              onClick={() => onRemoveRange?.(id, seg.start, seg.end)}
            >
              {seg.text}
            </mark>
          );
        }
        if (seg.evidence) {
          return (
            <mark key={i} className="reading-evidence-mark">
              {seg.text}
            </mark>
          );
        }
        return <span key={i}>{seg.text}</span>;
      })}
    </Tag>
  );
}
