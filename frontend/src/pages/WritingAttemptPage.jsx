import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import ReactMarkdown from 'react-markdown';
import { getScore, getTest, submitAnswer, waitForGrading } from '../lib/api';
import { safeParse } from '../lib/safeParse';
import { ATTEMPT_SECONDS, WRITING_SECONDS, useCountdown, useElapsed, useLeaveGuard } from '../lib/useAttemptSession';
import ScoreResult from '../components/ScoreResult/ScoreResult';
import Button from '../components/ui/Button/Button';
import './WritingAttemptPage.css';

// Timed ("thi thử") runs the task's exam clock and hands the essay in when
// it runs out; practice ("luyện tập") has no clock and waits for the
// learner to submit. The last choice is remembered per browser.
const MODE_KEY = 'writing-mode';

function loadMode() {
  try {
    return localStorage.getItem(MODE_KEY) === 'practice' ? 'practice' : 'timed';
  } catch {
    return 'timed';
  }
}

function saveMode(mode) {
  try {
    localStorage.setItem(MODE_KEY, mode);
  } catch {
    // Storage blocked (private window): the choice just isn't remembered.
  }
}

export default function WritingAttemptPage() {
  const { testId } = useParams();
  const navigate = useNavigate();

  const [test, setTest] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // null until the learner picks a mode; nothing is timed before that.
  const [mode, setMode] = useState(null);
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

  const timed = mode === 'timed';
  const allowance = WRITING_SECONDS[test?.task_type] ?? ATTEMPT_SECONDS;
  const inProgress = !!test && !!mode && !score && !gradeFailed;
  const remaining = useCountdown(allowance, inProgress && timed, handleTimeUp);
  const elapsed = useElapsed(inProgress);
  const confirmLeave = useLeaveGuard(inProgress);

  function start(chosen) {
    saveMode(chosen);
    setMode(chosen);
  }

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
      const queued = await submitAnswer(Number(testId), { text, mode, elapsed_seconds: elapsed });
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
  const taskLabel = test.task_type === 'task1' ? 'Writing Task 1' : 'Writing Task 2';

  if (!mode) {
    const preferred = loadMode();
    return (
      <div className="attempt-page attempt-page-focus">
        <header className="attempt-header">
          <button type="button" className="attempt-back" onClick={() => navigate('/practice/writing')} aria-label="Về danh sách đề" title="Về danh sách đề">
            ←
          </button>
        </header>
        <div className="attempt-mode-card">
          <h2>{taskLabel}</h2>
          <p className="attempt-mode-lead">Chọn cách làm bài:</p>
          <div className="attempt-mode-options">
            <Button variant={preferred === 'timed' ? 'primary' : 'secondary'} className="attempt-mode-option" onClick={() => start('timed')}>
              <strong>Thi thử · {Math.round(allowance / 60)} phút</strong>
              <span>Tính giờ như thi thật, hết giờ tự động nộp bài.</span>
            </Button>
            <Button variant={preferred === 'practice' ? 'primary' : 'secondary'} className="attempt-mode-option" onClick={() => start('practice')}>
              <strong>Luyện tập</strong>
              <span>Không giới hạn thời gian, nộp khi bạn viết xong.</span>
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="attempt-page attempt-page-focus">
      <header className="attempt-header">
        <button type="button" className="attempt-back" onClick={() => confirmLeave() && navigate('/practice/writing')} aria-label="Về danh sách đề" title="Về danh sách đề">
          ←
        </button>
        {timed ? (
          <span className={`attempt-timer ${inProgress && remaining <= 5 * 60 ? 'attempt-timer-low' : ''}`} title="Thời gian còn lại">
            {formatTime(remaining)}
          </span>
        ) : (
          <span className="attempt-timer" title="Thời gian đã viết (luyện tập, không giới hạn)">
            Luyện tập · {formatTime(elapsed)}
          </span>
        )}
      </header>

      <div className="attempt-body">
        <div className="attempt-prompt-panel">
          <h2>{taskLabel}</h2>
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
