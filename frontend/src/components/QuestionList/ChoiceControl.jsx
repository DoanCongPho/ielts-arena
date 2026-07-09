const TFNG_OPTIONS = ['TRUE', 'FALSE', 'NOT GIVEN'].map((v) => ({ id: v, text: v }));
const YNNG_OPTIONS = ['YES', 'NO', 'NOT GIVEN'].map((v) => ({ id: v, text: v }));

function optionsFor(group, question) {
  if (group.question_type === 'true-false-not-given') return TFNG_OPTIONS;
  if (group.question_type === 'yes-no-not-given') return YNNG_OPTIONS;
  return question.options?.length ? question.options : group.shared_options || [];
}

function optionLabel(opt) {
  return opt.id === opt.text ? opt.text : `${opt.id}. ${opt.text}`;
}

// ChoiceControl covers fixed true/false/not-given and yes/no/not-given
// radios, multiple-choice radios, and multiple-choice-multi checkboxes
// (capped at select_count). The matching-* family and map-plan-labelling
// live in MatchingDragDrop instead — they're "match to one item from a
// shared list" rather than "pick from this question's own options".
export default function ChoiceControl({ group, answers, onChange, disabled, results }) {
  const isMulti = group.question_type === 'multiple-choice-multi';

  return (
    <div className="choice-control">
      {group.questions.map((q) => {
        const order = q.question_order;
        const result = results?.[order];
        const options = optionsFor(group, q);
        const liveValue = isMulti ? answers?.[order] ?? [] : answers?.[order] ?? '';
        const value = result ? (isMulti ? result.submitted_answer ?? [] : result.submitted_answer?.[0] ?? '') : liveValue;
        const itemClass = result
          ? `question-item ${result.correct ? 'question-item-correct' : 'question-item-incorrect'}`
          : 'question-item';

        function toggleMulti(optionId) {
          const current = Array.isArray(value) ? value : [];
          if (current.includes(optionId)) {
            onChange?.(order, current.filter((v) => v !== optionId));
          } else if (current.length < (group.select_count || options.length)) {
            onChange?.(order, [...current, optionId]);
          }
        }

        return (
          <div key={order} id={`question-${order}`} className={itemClass}>
            <p className="question-item-text">
              <span className="question-item-number">Câu {order}</span> {q.text}
            </p>

            <div className="question-item-options">
              {options.map((opt) => (
                <label key={opt.id} className="question-item-option">
                  <input
                    type={isMulti ? 'checkbox' : 'radio'}
                    name={isMulti ? undefined : `q-${order}`}
                    checked={isMulti ? Array.isArray(value) && value.includes(opt.id) : value === opt.id}
                    disabled={disabled}
                    onChange={() => (isMulti ? toggleMulti(opt.id) : onChange?.(order, opt.id))}
                  />
                  {optionLabel(opt)}
                </label>
              ))}
            </div>

            {result && !result.correct && (
              <p className="question-item-correct-answer">Đáp án đúng: {(result.correct_answer || []).join(', ')}</p>
            )}
          </div>
        );
      })}
    </div>
  );
}
