import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import ReactMarkdown from 'react-markdown';
import { getScore, getTest, submitAnswer } from '../lib/api';
import './WritingAttemptPage.css';

export default function WritingAttemptPage() {
  const { testId } = useParams();
  const navigate = useNavigate();

  const [test, setTest] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [text, setText] = useState('');
  const [seconds, setSeconds] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [score, setScore] = useState(null);
  const [gradeFailed, setGradeFailed] = useState(false);

  useEffect(() => {
    getTest(testId)
      .then(setTest)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [testId]);

  useEffect(() => {
    if (score || gradeFailed) return;
    const id = setInterval(() => setSeconds((s) => s + 1), 1000);
    return () => clearInterval(id);
  }, [score, gradeFailed]);

  async function handleSubmit() {
    setSubmitting(true);
    setError('');
    try {
      const sub = await submitAnswer(Number(testId), { text });
      if (sub.status === 'graded') {
        const result = await getScore(sub.id);
        setScore(result);
      } else {
        setGradeFailed(true);
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setSubmitting(false);
    }
  }

  if (loading) return <div className="attempt-page"><p className="practice-status">Đang tải đề...</p></div>;
  if (error && !test) return <div className="attempt-page"><p className="practice-status practice-error">{error}</p></div>;

  const content = safeParse(test?.content_data);
  const wordCount = text.trim() ? text.trim().split(/\s+/).length : 0;

  return (
    <div className="attempt-page">
      <header className="attempt-header">
        <button className="practice-back-btn" onClick={() => navigate('/practice/writing')}>
          ← Danh sách đề
        </button>
        <span className="attempt-timer">{formatTime(seconds)}</span>
      </header>

      <div className="attempt-body">
        <div className="attempt-prompt-panel">
          <h2>{test.task_type === 'task1' ? 'Writing Task 1' : 'Writing Task 2'}</h2>
          {content?.image_url && (
            <img className="attempt-chart" src={content.image_url} alt="Task chart" />
          )}
          <div className="attempt-prompt-text">
            <ReactMarkdown>{content?.prompt || ''}</ReactMarkdown>
          </div>
        </div>

        <div className="attempt-answer-panel">
          {!score && !gradeFailed && (
            <>
              <div className="attempt-answer-meta">
                <span>Word count: {wordCount}</span>
              </div>
              <textarea
                className="attempt-textarea"
                value={text}
                onChange={(e) => setText(e.target.value)}
                placeholder="Viết bài của bạn ở đây..."
                disabled={submitting}
              />
              {error && <p className="practice-status practice-error">{error}</p>}
              <button className="attempt-submit-btn" onClick={handleSubmit} disabled={submitting || !text.trim()}>
                {submitting ? 'Đang chấm điểm...' : 'Nộp bài'}
              </button>
            </>
          )}

          {gradeFailed && (
            <div className="attempt-result">
              <p className="practice-status practice-error">
                Bài đã được lưu nhưng chấm điểm thất bại. Vui lòng thử lại sau.
              </p>
            </div>
          )}

          {score && <ScoreResult score={score} />}
        </div>
      </div>
    </div>
  );
}

function ScoreResult({ score }) {
  const details = safeParse(score.details) || {};
  const criteria = details.criteria || {};
  const corrections = details.corrections || [];

  return (
    <div className="attempt-result">
      <div className="attempt-band">
        <span className="attempt-band-value">{score.overall_band}</span>
        <span className="attempt-band-label">Overall Band</span>
      </div>

      <div className="criteria-list">
        {Object.entries(criteria).map(([name, c]) => (
          <div key={name} className="criteria-item">
            <div className="criteria-item-header">
              <span>{name}</span>
              <span className="criteria-score">{c.score}</span>
            </div>
            <p className="criteria-feedback">{c.feedback}</p>
          </div>
        ))}
      </div>

      {corrections.length > 0 && (
        <div className="corrections-list">
          <h3>Corrections</h3>
          {corrections.map((c, i) => (
            <div key={i} className="correction-item">
              <p className="correction-span">"{c.span}"</p>
              <p className="correction-issue">{c.issue}</p>
              <p className="correction-suggestion">→ {c.suggestion}</p>
            </div>
          ))}
        </div>
      )}

      {details.model_answer && (
        <div className="model-answer">
          <h3>Model Answer</h3>
          <p>{details.model_answer}</p>
        </div>
      )}
    </div>
  );
}

function safeParse(raw) {
  if (raw == null) return null;
  try {
    return typeof raw === 'string' ? JSON.parse(raw) : raw;
  } catch {
    return null;
  }
}

function formatTime(totalSeconds) {
  const m = Math.floor(totalSeconds / 60).toString().padStart(2, '0');
  const s = (totalSeconds % 60).toString().padStart(2, '0');
  return `${m}:${s}`;
}
