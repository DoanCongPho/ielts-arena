// PlainInputGroup covers question_types where each question has its own
// text prompt and a single free-text answer: sentence-completion,
// short-answer, and diagram-label-completion (which additionally shows the
// group's diagram image above the question list).
export default function PlainInputGroup({ group, answers, onChange, disabled, results }) {
  return (
    <div className="plain-input-group">
      {group.diagram_image_url && <img className="plain-input-diagram" src={group.diagram_image_url} alt="Sơ đồ" />}

      {group.questions.map((q) => {
        const order = q.question_order;
        const result = results?.[order];
        const value = result ? result.submitted_answer?.[0] ?? '' : answers?.[order] ?? '';
        const itemClass = result
          ? `question-item ${result.correct ? 'question-item-correct' : 'question-item-incorrect'}`
          : 'question-item';

        return (
          <div key={order} id={`question-${order}`} className={itemClass}>
            <p className="question-item-text">
              <span className="question-item-number">Câu {order}</span> {q.text}
              {group.word_limit ? <span className="question-item-hint"> (tối đa {group.word_limit} từ)</span> : null}
            </p>
            <input
              type="text"
              className="question-item-input"
              value={value}
              disabled={disabled}
              onChange={(e) => onChange?.(order, e.target.value)}
              placeholder="Nhập câu trả lời..."
            />
            {result && !result.correct && (
              <p className="question-item-correct-answer">Đáp án đúng: {(result.correct_answer || []).join(', ')}</p>
            )}
          </div>
        );
      })}
    </div>
  );
}
