import { useMemo, useState } from 'react';
import './WritingAnswerReview.css';

// Issue categories match the grader's correctionIssues, plus "other" for
// anything it couldn't categorise.
const ISSUE_LABELS = {
  grammar: 'Ngữ pháp',
  vocabulary: 'Từ vựng',
  spelling: 'Chính tả',
  punctuation: 'Dấu câu',
  cohesion: 'Liên kết',
  task: 'Nội dung',
  other: 'Khác',
};

const escapeRegExp = (s) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

// placeCorrections finds each correction's span in the answer. The grader
// kept only spans found with whitespace collapsed, so any run of whitespace
// matches here too. Spans come in reading order, so each search starts after
// the previous match and falls back to the whole text; a span that only
// overlaps one already placed can't be marked, and is listed instead.
function placeCorrections(text, corrections) {
  const placed = [];
  const unplaced = [];
  let cursor = 0;
  corrections.forEach((correction) => {
    const words = (correction.span || '').trim().split(/\s+/).filter(Boolean);
    if (!words.length) {
      unplaced.push(correction);
      return;
    }
    const re = new RegExp(words.map(escapeRegExp).join('\\s+'), 'g');
    const overlaps = (m) => placed.some((p) => m.index < p.end && m.index + m[0].length > p.start);
    let match = null;
    for (const from of [cursor, 0]) {
      re.lastIndex = from;
      let m = re.exec(text);
      while (m && overlaps(m)) m = re.exec(text);
      if (m) {
        match = m;
        break;
      }
    }
    if (!match) {
      unplaced.push(correction);
      return;
    }
    placed.push({ start: match.index, end: match.index + match[0].length, correction });
    cursor = match.index + match[0].length;
  });
  placed.sort((a, b) => a.start - b.start);
  return { placed, unplaced };
}

function issueOf(correction) {
  return ISSUE_LABELS[correction.issue] ? correction.issue : 'other';
}

// WritingAnswerReview shows a graded writing answer with every correction
// marked where it occurs: click a mark to see the fix and why, step through
// them in order, or switch to the corrected text to read it all at once.
export default function WritingAnswerReview({ text, corrections }) {
  const [selected, setSelected] = useState(null);
  const [showFixed, setShowFixed] = useState(false);
  const { placed, unplaced } = useMemo(() => placeCorrections(text || '', corrections || []), [text, corrections]);

  const counts = useMemo(() => {
    const c = {};
    for (const p of placed) c[issueOf(p.correction)] = (c[issueOf(p.correction)] || 0) + 1;
    return c;
  }, [placed]);

  function select(i) {
    setSelected(selected === i ? null : i);
  }

  const elements = [];
  let pos = 0;
  placed.forEach((p, i) => {
    if (p.start > pos) elements.push(text.slice(pos, p.start));
    const original = text.slice(p.start, p.end);
    elements.push(
      <mark
        key={i}
        role="button"
        tabIndex={0}
        className={`wr-mark wr-issue-${issueOf(p.correction)} ${selected === i ? 'wr-mark-selected' : ''}`}
        onClick={() => select(i)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            select(i);
          }
        }}
        title={showFixed ? p.correction.explanation : `${ISSUE_LABELS[issueOf(p.correction)]}: ${p.correction.suggestion}`}
      >
        {showFixed ? (
          <>
            <del>{original}</del> <ins>{p.correction.suggestion}</ins>
          </>
        ) : (
          original
        )}
      </mark>,
    );
    pos = p.end;
  });
  if (pos < (text || '').length) elements.push(text.slice(pos));

  const current = selected != null ? placed[selected] : null;

  return (
    <div className="wr">
      {placed.length > 0 && (
        <div className="wr-toolbar">
          <div className="wr-legend">
            {Object.entries(ISSUE_LABELS)
              .filter(([issue]) => counts[issue])
              .map(([issue, label]) => (
                <span key={issue} className={`wr-chip wr-issue-${issue}`}>
                  {label} · {counts[issue]}
                </span>
              ))}
          </div>
          <button type="button" className="wr-toggle" onClick={() => setShowFixed(!showFixed)}>
            {showFixed ? 'Xem bài gốc' : 'Xem bản đã sửa'}
          </button>
        </div>
      )}

      <p className="wr-text">{elements}</p>

      {placed.length > 0 && (
        <div className="wr-panel" aria-live="polite">
          {current ? (
            <>
              <div className="wr-panel-head">
                <span className={`wr-chip wr-issue-${issueOf(current.correction)}`}>
                  {ISSUE_LABELS[issueOf(current.correction)]}
                </span>
                <span className="wr-panel-count">
                  Lỗi {selected + 1}/{placed.length}
                </span>
                <div className="wr-panel-nav">
                  <button type="button" onClick={() => setSelected(selected - 1)} disabled={selected === 0}>
                    ← Trước
                  </button>
                  <button type="button" onClick={() => setSelected(selected + 1)} disabled={selected === placed.length - 1}>
                    Sau →
                  </button>
                </div>
              </div>
              <p className="wr-panel-fix">
                <del>{text.slice(current.start, current.end)}</del> → <ins>{current.correction.suggestion}</ins>
              </p>
              {current.correction.explanation && <p className="wr-panel-why">{current.correction.explanation}</p>}
            </>
          ) : (
            <div className="wr-panel-head">
              <span className="wr-hint">
                {placed.length} lỗi được đánh dấu trên bài. Bấm vào chỗ được tô để xem cách sửa.
              </span>
              <div className="wr-panel-nav">
                <button type="button" onClick={() => setSelected(0)}>
                  Xem lần lượt →
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {unplaced.length > 0 && (
        <div className="wr-unplaced">
          <h3>Lỗi khác</h3>
          <ul>
            {unplaced.map((c, i) => (
              <li key={i}>
                <span className={`wr-chip wr-issue-${issueOf(c)}`}>{ISSUE_LABELS[issueOf(c)]}</span>
                <del>{c.span}</del> → <ins>{c.suggestion}</ins>
                {c.explanation && <span className="wr-panel-why"> — {c.explanation}</span>}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
