import { useReview } from './ReviewContext';
import './AnswerExplanation.css';

// AnswerExplanation shows, under a graded question, why its answer is right
// and — when the answer key cites the passage — a button that jumps to and
// highlights that spot. `label` prefixes the explanation when it isn't
// sitting right under its own question (structured gap groups list theirs
// below the whole structure).
export default function AnswerExplanation({ order, label }) {
  const { answerKey, onLocate } = useReview();
  const entry = answerKey?.[order];
  const hasEvidence = entry?.evidence?.length > 0 && !!onLocate;
  if (!entry || (!entry.explanation && !hasEvidence)) return null;

  return (
    <details className="answer-explanation">
      <summary className="answer-explanation-summary">
        {label ? `${label} — giải thích` : 'Giải thích'}
      </summary>
      {entry.explanation && <p className="answer-explanation-text">{entry.explanation}</p>}
      {hasEvidence && (
        <button type="button" className="answer-explanation-locate" onClick={() => onLocate(order)}>
          Xem vị trí trong bài
        </button>
      )}
    </details>
  );
}
