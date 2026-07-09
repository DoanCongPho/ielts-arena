import { safeParse } from '../../lib/safeParse';
import './AutoGradeResult.css';

// Sibling of ScoreResult, for auto-graded skills (reading/listening) whose
// score.details shape (correct_count/total_count/results) is structurally
// different from the LLM-graded ScoreDetails (criteria/corrections/model_answer).
// `results` is keyed by question_order (as a string); submitted/correct
// answers are always arrays. `onSeek(seconds)`, if provided, renders a
// "listen again" button on wrong listening answers that have a
// timestamp_hint, so the candidate can jump straight to the moment they
// got it wrong.
export default function AutoGradeResult({ score, questions, onSeek }) {
  const details = safeParse(score.details) || {};
  const { correct_count: correctCount = 0, total_count: totalCount = 0, results = {} } = details;

  const questionByOrder = Object.fromEntries((questions || []).map((q) => [q.question_order, q]));
  const orderedIds = questions
    ? questions.map((q) => String(q.question_order))
    : Object.keys(results).sort((a, b) => Number(a) - Number(b));

  return (
    <div className="autograde-result">
      <div className="autograde-result-band">
        <span className="autograde-result-band-value">{score.overall_band}</span>
        <span className="autograde-result-band-label">Overall Band</span>
        <span className="autograde-result-count">{correctCount}/{totalCount} câu đúng</span>
      </div>

      <div className="autograde-result-list">
        {orderedIds.map((order) => {
          const r = results[order];
          if (!r) return null;
          const question = questionByOrder[order];
          const label = question?.text || `Câu ${order}`;
          return (
            <div
              key={order}
              className={`autograde-result-item ${r.correct ? 'autograde-result-item-correct' : 'autograde-result-item-incorrect'}`}
            >
              <div className="autograde-result-item-header">
                <span>{label}</span>
                <span className={`autograde-result-badge ${r.correct ? 'autograde-result-badge-correct' : 'autograde-result-badge-incorrect'}`}>
                  {r.correct ? 'Đúng' : 'Sai'}
                </span>
              </div>
              <p className="autograde-result-item-answer">
                Bạn chọn: {(r.submitted_answer || []).join(', ') || '(bỏ trống)'}
              </p>
              {!r.correct && (
                <p className="autograde-result-item-answer autograde-result-item-correct-answer">
                  Đáp án đúng: {(r.correct_answer || []).join(', ')}
                </p>
              )}
              {!r.correct && onSeek && question?.timestamp_hint != null && (
                <button
                  type="button"
                  className="autograde-result-seek-btn"
                  onClick={() => onSeek(question.timestamp_hint)}
                >
                  ▶ Nghe lại đoạn này
                </button>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
