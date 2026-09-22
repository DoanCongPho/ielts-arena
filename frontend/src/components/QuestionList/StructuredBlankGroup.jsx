import AnswerExplanation from '../AnswerExplanation/AnswerExplanation';
import HighlightableText from '../HighlightableText/HighlightableText';
import Dropdown from './Dropdown';
import { resolveGapLines, resolveGapRows } from './gapText';

// GapControl renders one blank: a Dropdown populated from the group's word
// bank when summary-completion has one, otherwise a free-text <input>.
function GapControl({ question, answers, onChange, disabled, results, hasWordBank, wordBank }) {
  const order = question.question_order;
  const result = results?.[order];
  const value = result ? result.submitted_answer?.[0] ?? '' : answers?.[order] ?? '';
  const stateClass = result ? (result.correct ? 'gap-correct' : 'gap-incorrect') : '';

  if (hasWordBank) {
    return (
      <Dropdown
        id={`question-${order}`}
        wrapperClassName="gap-select-wrapper"
        triggerClassName={`gap-select ${stateClass}`}
        value={value}
        disabled={disabled}
        options={wordBank || []}
        placeholder={`(${order})`}
        onChange={(v) => onChange?.(order, v)}
      />
    );
  }

  return (
    <input
      id={`question-${order}`}
      type="text"
      className={`gap-input ${stateClass}`}
      value={value}
      disabled={disabled}
      onChange={(e) => onChange?.(order, e.target.value)}
      placeholder={`(${order})`}
    />
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

// StructuredBlankGroup covers question_types answered by filling blanks
// embedded in a group-level shared structure rather than one self-
// contained question each: summary-completion (text or word-bank select
// per gap, depending on has_word_bank), table-completion, note-completion,
// flow-chart-completion, and form-completion. The i-th "{{gap}}" marker
// found in the structure (in document order) is answered by the i-th
// entry in group.questions — see gapText.js.
export default function StructuredBlankGroup({ group, answers, onChange, disabled, results, highlights, onHighlightRemove }) {
  const shared = {
    answers,
    onChange,
    disabled,
    results,
    hasWordBank: group.has_word_bank,
    wordBank: group.word_bank,
    highlights,
    onHighlightRemove,
  };
  const key = (...parts) => [`s${group.group_order}`, ...parts].join('-');
  const plainText = (text, id) => (
    <HighlightableText id={id} text={text} ranges={highlights?.[id]} onRemoveRange={onHighlightRemove} />
  );

  return (
    <div className="structured-blank-group">
      {group.question_type === 'summary-completion' && (
        <p className="structured-text">{renderSegments(resolveGapLines([group.summary_text], group.questions)[0], shared, key('summary'))}</p>
      )}

      {group.question_type === 'note-completion' && group.note_structure && (
        <div className="structured-note">
          {group.note_structure.title && <h4 className="structured-heading">{plainText(group.note_structure.title, key('title'))}</h4>}
          <ul className="structured-note-list">
            {resolveGapLines(group.note_structure.items, group.questions).map((segments, i) => (
              <li key={i}>{renderSegments(segments, shared, key('note', i))}</li>
            ))}
          </ul>
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
          {group.form_structure.title && <h4 className="structured-heading">{plainText(group.form_structure.title, key('title'))}</h4>}
          <ul className="structured-note-list">
            {resolveGapLines(group.form_structure.fields, group.questions).map((segments, i) => (
              <li key={i}>{renderSegments(segments, shared, key('form', i))}</li>
            ))}
          </ul>
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
            {resolveGapRows(group.table_structure.rows, group.questions).map((row, ri) => (
              <tr key={ri}>
                {row.map((segments, ci) => (
                  <td key={ci}>{renderSegments(segments, shared, key('cell', ri, ci))}</td>
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
