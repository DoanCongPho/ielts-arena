import ReactMarkdown from 'react-markdown';
import { safeParse } from '../../lib/safeParse';
import ScoreCard from '../ui/ScoreCard/ScoreCard';
import './ScoreResult.css';

export default function ScoreResult({ score }) {
  const details = safeParse(score.details) || {};
  const criteria = details.criteria || {};

  return (
    <div className="score-result">
      <ScoreCard skill="writing" band={score.overall_band} />
      {details.word_count > 0 && <p className="score-result-word-count">{details.word_count} từ</p>}

      <div className="score-result-criteria-list">
        {Object.entries(criteria).map(([name, c]) => (
          <div key={name} className="score-result-criteria-item">
            <div className="score-result-criteria-header">
              <span>{name}</span>
              <span className="score-result-criteria-score">{c.score}</span>
            </div>
            <p className="score-result-criteria-feedback">{c.feedback}</p>
            {c.improvements?.length > 0 && (
              <ul className="score-result-criteria-improvements">
                {c.improvements.map((tip, i) => <li key={i}>{tip}</li>)}
              </ul>
            )}
          </div>
        ))}
      </div>

      {details.model_answer && (
        <div className="score-result-model-answer">
          <h3>
            Bài mẫu
            {details.model_answer_source === 'llm' && <span className="score-result-ai-tag">AI viết</span>}
          </h3>
          <ReactMarkdown>{details.model_answer}</ReactMarkdown>
        </div>
      )}
    </div>
  );
}
