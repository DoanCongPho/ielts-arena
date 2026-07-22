import { safeParse } from '../../lib/safeParse';
import ScoreCard from '../ui/ScoreCard/ScoreCard';
import './ScoreResult.css';

export default function ScoreResult({ score }) {
  const details = safeParse(score.details) || {};
  const criteria = details.criteria || {};
  const corrections = details.corrections || [];

  return (
    <div className="score-result">
      <ScoreCard skill="writing" band={score.overall_band} />

      <div className="score-result-criteria-list">
        {Object.entries(criteria).map(([name, c]) => (
          <div key={name} className="score-result-criteria-item">
            <div className="score-result-criteria-header">
              <span>{name}</span>
              <span className="score-result-criteria-score">{c.score}</span>
            </div>
            <p className="score-result-criteria-feedback">{c.feedback}</p>
          </div>
        ))}
      </div>

      {corrections.length > 0 && (
        <div className="score-result-corrections">
          <h3>Corrections</h3>
          {corrections.map((c, i) => (
            <div key={i} className="score-result-correction-item">
              <p className="score-result-correction-span">"{c.span}"</p>
              <p className="score-result-correction-issue">{c.issue}</p>
              <p className="score-result-correction-suggestion">→ {c.suggestion}</p>
            </div>
          ))}
        </div>
      )}

      {details.model_answer && (
        <div className="score-result-model-answer">
          <h3>Model Answer</h3>
          <p>{details.model_answer}</p>
        </div>
      )}
    </div>
  );
}
