import { useState } from 'react';

function optionLabel(opt) {
  return opt.id === opt.text ? opt.text : `${opt.id}. ${opt.text}`;
}

function optionsFor(group, question) {
  if (group.question_type === 'map-plan-labelling') return group.location_key || [];
  if (question.options?.length) return question.options;
  return group.shared_options || [];
}

// MatchingDragDrop covers every single-answer "match a prompt to one item
// from a shared list" question_type (matching-headings/information/
// features/sentence-endings, matching, map-plan-labelling): a word bank of
// draggable option chips, and one drop slot per question. Options can also
// be assigned by clicking a chip (selecting it) then clicking a slot —
// necessary on touch devices, where HTML5 drag-and-drop doesn't work well.
export default function MatchingDragDrop({ group, answers, onChange, disabled, results }) {
  const [selectedChip, setSelectedChip] = useState(null);
  const [dragOverOrder, setDragOverOrder] = useState(null);

  // All questions in the group share one option pool when using
  // shared_options/location_key; matching-sentence-endings may instead give
  // each question its own options — in that case the "bank" is just that
  // question's own list, shown alongside its slot instead of once globally.
  const bankOptions = group.shared_options?.length
    ? group.shared_options
    : group.question_type === 'map-plan-labelling'
      ? group.location_key || []
      : null;

  const usedIds = new Set(group.questions.map((q) => answers?.[q.question_order]).filter(Boolean));

  function assign(order, optionId) {
    if (disabled) return;
    onChange?.(order, optionId);
    setSelectedChip(null);
  }

  function clear(order) {
    if (disabled) return;
    onChange?.(order, '');
  }

  function renderBank(options) {
    return (
      <div className="matching-dnd-bank">
        {options.map((opt) => {
          const usedElsewhere = group.allow_reuse === false && usedIds.has(opt.id);
          return (
            <button
              key={opt.id}
              type="button"
              draggable={!disabled && !usedElsewhere}
              disabled={disabled || usedElsewhere}
              className={`matching-chip ${selectedChip === opt.id ? 'matching-chip-selected' : ''} ${usedElsewhere ? 'matching-chip-used' : ''}`}
              onDragStart={(e) => e.dataTransfer.setData('text/plain', opt.id)}
              onClick={() => setSelectedChip((id) => (id === opt.id ? null : opt.id))}
            >
              {optionLabel(opt)}
            </button>
          );
        })}
      </div>
    );
  }

  return (
    <div className="matching-dnd">
      {group.map_image_url && <img className="choice-control-map" src={group.map_image_url} alt="Sơ đồ" />}
      {bankOptions && renderBank(bankOptions)}

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

          return (
            <div key={order} id={`question-${order}`} className={itemClass}>
              <p className="question-item-text">
                <span className="question-item-number">Câu {order}</span> {q.text}
              </p>

              {!bankOptions && options && renderBank(options)}

              <div
                className={`matching-drop-slot ${dragOverOrder === order ? 'matching-drop-slot-over' : ''} ${chosen ? 'matching-drop-slot-filled' : ''}`}
                onDragOver={(e) => {
                  if (disabled) return;
                  e.preventDefault();
                  setDragOverOrder(order);
                }}
                onDragLeave={() => setDragOverOrder((o) => (o === order ? null : o))}
                onDrop={(e) => {
                  e.preventDefault();
                  setDragOverOrder(null);
                  const optionId = e.dataTransfer.getData('text/plain');
                  if (optionId) assign(order, optionId);
                }}
                onClick={() => {
                  if (selectedChip) assign(order, selectedChip);
                }}
              >
                {chosen ? (
                  <span className="matching-drop-value">
                    {optionLabel(chosen)}
                    {!disabled && (
                      <button
                        type="button"
                        className="matching-drop-clear"
                        onClick={(e) => {
                          e.stopPropagation();
                          clear(order);
                        }}
                        aria-label="Bỏ đáp án"
                      >
                        ×
                      </button>
                    )}
                  </span>
                ) : (
                  <span className="matching-drop-placeholder">Kéo đáp án vào đây, hoặc bấm để chọn</span>
                )}
              </div>

              {result && !result.correct && (
                <p className="question-item-correct-answer">Đáp án đúng: {(result.correct_answer || []).join(', ')}</p>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
