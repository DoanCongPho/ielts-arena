import AnswerExplanation from '../AnswerExplanation/AnswerExplanation';
import HighlightableText from '../HighlightableText/HighlightableText';
import { ChipBank, DropSlot } from './ChipBank';
import { isSingleUse, useChipPicker } from './chipPicker';
import { inferMarkup, resolveGapCells, resolveGapLines, resolveMarkedLines } from './gapText';

// GapControl renders one blank as its question number in a badge followed
// by the answer field: a slot to drop a word into when summary-completion
// has a word list (the bank sits above the text, see ChipBank), otherwise
// a free-text <input> drawn as an underline, like the paper test.
function GapControl({ question, answers, onChange, disabled, results, wordBank, picker }) {
  const order = question.question_order;
  const result = results?.[order];
  const value = result ? result.submitted_answer?.[0] ?? '' : answers?.[order] ?? '';
  const stateClass = result ? (result.correct ? 'gap-correct' : 'gap-incorrect') : '';

  return (
    <span className="gap-field">
      <span className={`gap-number ${stateClass}`}>{order}</span>
      {wordBank ? (
        <DropSlot
          id={`question-${order}`}
          order={order}
          chosen={wordBank.find((o) => o.id === value)}
          picker={picker}
          inline
          stateClass={stateClass}
          placeholder="Kéo từ vào"
        />
      ) : (
        <input
          id={`question-${order}`}
          type="text"
          className={`gap-input ${stateClass}`}
          value={value}
          disabled={disabled}
          aria-label={`Câu ${order}`}
          onChange={(e) => onChange?.(order, e.target.value)}
        />
      )}
    </span>
  );
}

// renderSegments renders one line of a structure: its text runs, each
// highlightable under its own key (`${keyPrefix}-${i}`), and its gaps.
function renderSegments(segments, sharedProps, keyPrefix) {
  const { highlights, onHighlightRemove } = sharedProps;
  return segments.map((seg, i) => {
    if (seg.type !== 'text') return <GapControl key={i} question={seg.question} {...sharedProps} />;
    const id = `${keyPrefix}-${i}`;
    return <HighlightableText key={i} id={id} text={seg.value} ranges={highlights?.[id]} onRemoveRange={onHighlightRemove} />;
  });
}

// MarkedLines renders resolved structure lines (resolveMarkedLines):
// subheadings, bullets indented by level, and plain lines.
function MarkedLines({ lines, shared, keyPrefix }) {
  return (
    <div className="structured-lines">
      {lines.map((line, i) => {
        const content = renderSegments(line.segments, shared, `${keyPrefix}-${i}`);
        if (line.kind === 'heading') {
          return (
            <h5 key={i} className="structured-subheading">
              {content}
            </h5>
          );
        }
        const bullet = line.kind === 'bullet' ? ` structured-bullet structured-bullet-${Math.min(line.level, 2)}` : '';
        return (
          <div key={i} className={`structured-line${bullet}`}>
            {content}
          </div>
        );
      })}
    </div>
  );
}

// StructuredBlankGroup covers question_types answered by filling blanks
// embedded in a group-level shared structure rather than one self-
// contained question each: summary-completion (text or word-bank select
// per gap, depending on has_word_bank), table-completion, note-completion,
// flow-chart-completion, and form-completion. The i-th "{{gap}}" marker
// found in the structure (in document order) is answered by the i-th
// entry in group.questions — see gapText.js.
export default function StructuredBlankGroup({ group, answers, onChange, disabled, results, highlights, onHighlightRemove, skill }) {
  const picker = useChipPicker({ disabled, onChange });
  const wordBank = group.has_word_bank && group.word_bank?.length ? group.word_bank : null;
  const shared = {
    answers,
    onChange,
    disabled,
    results,
    wordBank,
    picker,
    highlights,
    onHighlightRemove,
  };
  const key = (...parts) => [`s${group.group_order}`, ...parts].join('-');
  const plainText = (text, id) => (
    <HighlightableText id={id} text={text} ranges={highlights?.[id]} onRemoveRange={onHighlightRemove} />
  );

  return (
    <div className="structured-blank-group">
      {wordBank && (
        <ChipBank
          options={wordBank}
          picker={picker}
          usedIds={new Set(group.questions.map((q) => answers?.[q.question_order]).filter(Boolean))}
          singleUse={isSingleUse(group, wordBank)}
          removeUsed={skill === 'listening'}
        />
      )}

      {group.question_type === 'summary-completion' && (
        <div className="structured-text">
          <MarkedLines
            lines={resolveMarkedLines(inferMarkup(String(group.summary_text ?? '').split('\n'), { title: true }), group.questions)}
            shared={shared}
            keyPrefix={key('summary')}
          />
        </div>
      )}

      {group.question_type === 'note-completion' && group.note_structure && (
        <div className="structured-note">
          {group.note_structure.title && <h4 className="structured-title">{plainText(group.note_structure.title, key('title'))}</h4>}
          <MarkedLines
            lines={resolveMarkedLines(inferMarkup(group.note_structure.items, { title: !group.note_structure.title }), group.questions)}
            shared={shared}
            keyPrefix={key('note')}
          />
        </div>
      )}

      {group.question_type === 'flow-chart-completion' && group.flow_structure && (
        <ol className="structured-flow">
          {resolveGapLines(group.flow_structure.steps, group.questions).map((segments, i) => (
            <li key={i}>{renderSegments(segments, shared, key('flow', i))}</li>
          ))}
        </ol>
      )}

      {group.question_type === 'form-completion' && group.form_structure && (
        <div className="structured-form">
          {group.form_structure.title && <h4 className="structured-title">{plainText(group.form_structure.title, key('title'))}</h4>}
          <MarkedLines
            lines={resolveMarkedLines(inferMarkup(group.form_structure.fields, { headings: false }), group.questions)}
            shared={shared}
            keyPrefix={key('form')}
          />
        </div>
      )}

      {group.question_type === 'table-completion' && group.table_structure && (
        <table className="structured-table">
          <thead>
            <tr>
              {group.table_structure.columns.map((col, i) => (
                <th key={i}>{plainText(col, key('col', i))}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {resolveGapCells(group.table_structure.rows, group.questions).map((row, ri) => (
              <tr key={ri}>
                {row.map((lines, ci) => (
                  <td key={ci}>
                    <MarkedLines lines={lines} shared={shared} keyPrefix={key('cell', ri, ci)} />
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {/* Gaps sit inline in the structure, so their explanations are
          listed underneath it instead. */}
      {results && (
        <div className="structured-explanations">
          {group.questions.map((q) => (
            <AnswerExplanation key={q.question_order} order={q.question_order} label={`Câu ${q.question_order}`} />
          ))}
        </div>
      )}
    </div>
  );
}
