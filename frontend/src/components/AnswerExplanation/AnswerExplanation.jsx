import { useReview } from './ReviewContext';
import './AnswerExplanation.css';

// AnswerExplanation shows, under a graded question, why its answer is right
// and — when the answer key cites the passage or transcript — a button that
// jumps to and highlights that spot; for listening, also one that replays
// the recording from where the answer is heard. `label` prefixes it when
// it isn't sitting right under its own question (structured gap groups
// list theirs below the whole structure).
export default function AnswerExplanation({ order, label }) {
  const { answerKey, onLocate, onListen, locateLabel } = useReview();
  const entry = answerKey?.[order];
  const hasEvidence = entry?.evidence?.length > 0 && !!onLocate;
  if (!entry || (!entry.explanation && !hasEvidence && !onListen)) return null;

  const title = entry.explanation ? 'Giải thích' : 'Đáp án trong bài';
  return (
    <details className="answer-explanation">
      <summary className="answer-explanation-summary">{label ? `${label} — ${title.toLowerCase()}` : title}</summary>
      {entry.explanation && <p className="answer-explanation-text">{entry.explanation}</p>}
      <div className="answer-explanation-actions">
        {hasEvidence && (
          <button type="button" className="answer-explanation-locate" onClick={() => onLocate(order)}>
            {locateLabel || 'Xem vị trí trong bài'}
          </button>
        )}
        {onListen && (
          <button type="button" className="answer-explanation-locate" onClick={() => onListen(order)}>
            ▶ Nghe lại đoạn này
          </button>
        )}
      </div>
    </details>
  );
}
