import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import ReactMarkdown from 'react-markdown';
import { getScore, getTest, submitAnswer, waitForGrading } from '../lib/api';
import { safeParse } from '../lib/safeParse';
import { ATTEMPT_SECONDS, useCountdown, useLeaveGuard } from '../lib/useAttemptSession';
import ScoreResult from '../components/ScoreResult/ScoreResult';
import Button from '../components/ui/Button/Button';
import './WritingAttemptPage.css';

export default function WritingAttemptPage() {
  const { testId } = useParams();
  const navigate = useNavigate();

  const [test, setTest] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [text, setText] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [score, setScore] = useState(null);
  const [gradeFailed, setGradeFailed] = useState(false);

  useEffect(() => {
    getTest(testId)
      .then(setTest)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [testId]);

  const inProgress = !!test && !score && !gradeFailed;
  const remaining = useCountdown(ATTEMPT_SECONDS, inProgress, handleTimeUp);
  const confirmLeave = useLeaveGuard(inProgress);

  // Time's up: hand in whatever has been written. An empty answer isn't
  // worth an LLM grading call, so it's not submitted.
  function handleTimeUp() {
    if (submitting) return;
    if (text.trim()) handleSubmit();
    else setError('Hết giờ — bạn chưa viết gì nên bài không được nộp.');
  }

  async function handleSubmit() {
    setSubmitting(true);
    setError('');
    try {
      // The API only queues the answer; a background worker grades it, so
      // poll the submission until its status settles.
      const queued = await submitAnswer(Number(testId), { text });
      const settled = await waitForGrading(queued.id);
      if (settled.status === 'graded') {
        setScore(await getScore(settled.id));
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
    <div className="attempt-page attempt-page-focus">
      <header className="attempt-header">
        <button type="button" className="attempt-back" onClick={() => confirmLeave() && navigate('/practice/writing')} aria-label="Về danh sách đề" title="Về danh sách đề">
          ←
        </button>
        <span className={`attempt-timer ${inProgress && remaining <= 5 * 60 ? 'attempt-timer-low' : ''}`}>{formatTime(remaining)}</span>
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
              <Button variant="primary" className="attempt-submit-btn" onClick={handleSubmit} disabled={submitting || !text.trim()}>
                {submitting ? 'Đang chấm điểm...' : 'Nộp bài'}
              </Button>
            </>
          )}

          {gradeFailed && (
            <div className="attempt-result">
              <p className="practice-status practice-error">
                Bài đã được lưu nhưng chấm điểm thất bại. Xem lại trong mục Lịch sử làm bài.
              </p>
            </div>
          )}

          {score && <ScoreResult score={score} />}
        </div>
      </div>
    </div>
  );
}

function formatTime(totalSeconds) {
  const m = Math.floor(totalSeconds / 60).toString().padStart(2, '0');
  const s = (totalSeconds % 60).toString().padStart(2, '0');
  return `${m}:${s}`;
}
