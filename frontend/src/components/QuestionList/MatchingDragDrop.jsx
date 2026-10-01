import AnswerExplanation from '../AnswerExplanation/AnswerExplanation';
import HighlightableText from '../HighlightableText/HighlightableText';
import { ChipBank, DropSlot } from './ChipBank';
import { isSingleUse, useChipPicker } from './chipPicker';
import FigureSplit from './FigureSplit';
import MatchingGrid from './MatchingGrid';

function optionsFor(group, question) {
  if (group.question_type === 'map-plan-labelling') return group.location_key || [];
  if (question.options?.length) return question.options;
  return group.shared_options || [];
}

// MatchingDragDrop covers every single-answer "match a prompt to one item
// from a shared list" question_type (matching-headings/information/
// features/sentence-endings, matching, map-plan-labelling): a bank of
// option chips, and one drop slot per question (see ChipBank).
export default function MatchingDragDrop({ group, answers, onChange, disabled, results, highlights, onHighlightRemove, skill }) {
  const picker = useChipPicker({ disabled, onChange });

  // All questions in the group share one option pool when using
  // shared_options/location_key; matching-sentence-endings may instead give
  // each question its own options — in that case the "bank" is just that
  // question's own list, shown alongside its slot instead of once globally.
  const bankOptions = group.shared_options?.length
    ? group.shared_options
    : group.question_type === 'map-plan-labelling'
      ? group.location_key || []
      : null;

  const singleUse = isSingleUse(group, bankOptions);
  // Letters that may be used more than once go in a grid, as on the paper
  // (a map keeps its chips: they are placed on the picture's labels).
  if (bankOptions && !singleUse && group.question_type !== 'map-plan-labelling') {
    return (
      <MatchingGrid
        group={group}
        options={bankOptions}
        answers={answers}
        onChange={onChange}
        disabled={disabled}
        results={results}
        highlights={highlights}
        onHighlightRemove={onHighlightRemove}
      />
    );
  }

  const usedIds = new Set(group.questions.map((q) => answers?.[q.question_order]).filter(Boolean));
  const bank = (options) => (
    <ChipBank
      options={options}
      picker={picker}
      usedIds={usedIds}
      singleUse={singleUse}
      // On the listening page, where the answers go in against the
      // recording, a placed single-use option leaves the bank altogether.
      removeUsed={skill === 'listening'}
    />
  );

  return (
    <FigureSplit images={group.map_image_url ? [group.map_image_url] : []} alt="Sơ đồ">
      <div className="matching-dnd">
        {bankOptions && bank(bankOptions)}

        <div className="matching-dnd-questions">
          {group.questions.map((q) => {
            const order = q.question_order;
            const result = results?.[order];
            const options = bankOptions ? null : optionsFor(group, q);
            const value = result ? result.submitted_answer?.[0] ?? '' : answers?.[order] ?? '';
            const source = bankOptions || options;
            const chosen = source?.find((o) => o.id === value);
            const itemClass = result
              ? `question-item ${result.correct ? 'question-item-correct' : 'question-item-incorrect'}`
              : 'question-item';
            const textKey = `q-${order}-text`;

            return (
              <div key={order} id={`question-${order}`} className={itemClass}>
                <p className="question-item-text">
                  <span className="question-item-number">Câu {order}</span>{' '}
                  <HighlightableText id={textKey} text={q.text} ranges={highlights?.[textKey]} onRemoveRange={onHighlightRemove} />
                </p>

                {!bankOptions && options && bank(options)}

                <DropSlot order={order} chosen={chosen} picker={picker} placeholder="Kéo đáp án vào đây, hoặc bấm để chọn" />

                {result && !result.correct && (
                  <p className="question-item-correct-answer">Đáp án đúng: {(result.correct_answer || []).join(', ')}</p>
                )}
                {result && <AnswerExplanation order={order} />}
              </div>
            );
          })}
        </div>
      </div>
    </FigureSplit>
  );
}
