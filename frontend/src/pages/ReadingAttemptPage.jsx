import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { getScore, getTest, submitAnswer, waitForGrading } from '../lib/api';
import { flattenQuestions } from '../lib/answerUtils';
import { safeParse } from '../lib/safeParse';
import { ATTEMPT_SECONDS, useCountdown, useLeaveGuard } from '../lib/useAttemptSession';
import { useTextHighlights } from '../lib/useTextHighlights';
import { useEvidenceReview } from '../lib/useEvidenceReview';
import { SKILL_CONFIG } from '../lib/skillConfig';
import QuestionList from '../components/QuestionList/QuestionList';
import { ReviewContext } from '../components/AnswerExplanation/ReviewContext';
import AutoGradeResult from '../components/AutoGradeResult/AutoGradeResult';
import QuestionNavBar from '../components/QuestionNavBar/QuestionNavBar';
import HighlightableText from '../components/HighlightableText/HighlightableText';
import HighlightToolbar from '../components/HighlightToolbar/HighlightToolbar';
import Button from '../components/ui/Button/Button';
import './PracticePage.css';
import './WritingAttemptPage.css';
import './ReadingAttemptPage.css';

export default function ReadingAttemptPage() {
  const { testId } = useParams();
  const navigate = useNavigate();

  const [test, setTest] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [activeIndex, setActiveIndex] = useState(0);
  const [answers, setAnswers] = useState({});
  const [submitting, setSubmitting] = useState(false);
  const [score, setScore] = useState(null);
  const [gradeFailed, setGradeFailed] = useState(false);

  // Text highlighting anywhere in either panel (passage paragraphs, group
  // instructions, question text), per passage.
  const highlight = useTextHighlights(activeIndex);

  // Set after clicking a QuestionNavBar pill for a question in a different
  // passage — the target element doesn't exist until the passage switch
  // re-renders, so the actual scroll happens in an effect keyed on this.
  const [pendingScrollOrder, setPendingScrollOrder] = useState(null);

  useEffect(() => {
    getTest(testId)
      .then(setTest)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [testId]);

  const content = safeParse(test?.content_data);
  const passages = content?.passages || [];

  const inProgress = !!test && !score && !gradeFailed;
  const remaining = useCountdown(ATTEMPT_SECONDS, inProgress, handleTimeUp);
  const confirmLeave = useLeaveGuard(inProgress);
  const review = useEvidenceReview(testId, !!score, passages, setActiveIndex);

  // Runs after a QuestionNavBar jump: if the target question is in a
  // different passage, setActiveIndex above re-renders that passage's
  // question list first, and only then does its DOM node exist to scroll to.
  useEffect(() => {
    if (pendingScrollOrder == null) return;
    const el = document.getElementById(`question-${pendingScrollOrder}`);
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' });
      setPendingScrollOrder(null);
    }
  }, [pendingScrollOrder, activeIndex]);

  function handleAnswerChange(questionOrder, value) {
    setAnswers((prev) => ({ ...prev, [questionOrder]: value }));
  }

  function handleJumpToQuestion(order) {
    const targetPassageIndex = passages.findIndex((p) =>
      (p.question_groups || []).some((g) => g.questions.some((q) => q.question_order === order)),
    );
    if (targetPassageIndex === -1) return;
    setActiveIndex(targetPassageIndex);
    setPendingScrollOrder(order);
  }

  // Time's up: hand in the answers as they stand, like a real paper.
  function handleTimeUp() {
    if (!submitting) handleSubmit();
  }

  async function handleSubmit() {
    setSubmitting(true);
    setError('');
    try {
      // The API only queues the answer; a background worker grades it, so
      // poll the submission until its status settles.
      const queued = await submitAnswer(Number(testId), { answers });
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

  const activePassage = passages[activeIndex];
  const activeGroups = activePassage?.question_groups || [];
  const allQuestions = passages.flatMap((p) => flattenQuestions(p.question_groups));
  const scoreResults = score ? safeParse(score.details)?.results : undefined;

  return (
    <div className="attempt-page reading-attempt-page attempt-page-focus">
      <header className="attempt-header">
        <button type="button" className="attempt-back" onClick={() => confirmLeave() && navigate('/practice/reading')} aria-label="Về danh sách đề" title="Về danh sách đề">
          ←
        </button>
        <span className={`attempt-timer ${inProgress && remaining <= 5 * 60 ? 'attempt-timer-low' : ''}`}>{formatTime(remaining)}</span>
      </header>

      <div className="attempt-body reading-attempt-body" {...highlight.areaProps}>
        <div className="attempt-prompt-panel reading-scroll-panel">
          <h2>Reading — {SKILL_CONFIG.reading.taskTypeLabel(test.task_type)}</h2>
          {passages.length > 1 && (
            <nav className="skill-tabs attempt-multi-tabs">
              {passages.map((p, i) => (
                <button
                  key={i}
                  className={`skill-tab ${i === activeIndex ? 'active' : ''}`}
                  onClick={() => setActiveIndex(i)}
                  disabled={submitting}
                >
                  {p.title || `Passage ${i + 1}`}
                </button>
              ))}
            </nav>
          )}
          <p className="reading-highlight-hint">Bôi đen văn bản để tô đậm (đề hoặc câu hỏi) — bấm vào phần đã tô để bỏ.</p>
          <div className="attempt-passage-text">
            {activePassage?.title && <h3 className="reading-passage-title">{activePassage.title}</h3>}
            {(activePassage?.paragraphs || []).map((p, pi) => {
              const key = `p-${pi}`;
              return (
                <p key={`${activeIndex}-${pi}`} id={`passage-${activeIndex}-p-${pi}`}>
                  {p.label && <strong>{p.label}. </strong>}
                  <HighlightableText
                    id={key}
                    text={p.text}
                    ranges={highlight.ranges[key]}
                    evidence={review.evidenceFor(activeIndex, pi)}
                    onRemoveRange={highlight.remove}
                  />
                </p>
              );
            })}
          </div>
        </div>

        <div className="attempt-answer-panel reading-scroll-panel">
          {score && <AutoGradeResult score={score} skill="reading" compact />}

          {gradeFailed && (
            <div className="attempt-result">
              <p className="practice-status practice-error">
                Bài đã được lưu nhưng chấm điểm thất bại. Xem lại trong mục Lịch sử làm bài.
              </p>
            </div>
          )}

          {!gradeFailed && (
            <ReviewContext.Provider value={{ answerKey: review.answerKey, onLocate: review.locate }}>
              <QuestionList
                groups={activeGroups}
                answers={answers}
                onChange={score ? undefined : handleAnswerChange}
                disabled={submitting || !!score}
                results={scoreResults}
                highlights={highlight.ranges}
                onHighlightRemove={highlight.remove}
                skill="reading"
              />
            </ReviewContext.Provider>
          )}

          {error && <p className="practice-status practice-error">{error}</p>}

          {/* Submit only from the last passage, like the end of a real paper. */}
          {!score && !gradeFailed && activeIndex === passages.length - 1 && (
            <Button variant="primary" className="attempt-submit-btn" onClick={handleSubmit} disabled={submitting}>
              {submitting ? 'Đang chấm điểm...' : 'Nộp bài'}
            </Button>
          )}
        </div>
      </div>

      {!gradeFailed && (
        <QuestionNavBar
          questions={allQuestions}
          answers={answers}
          results={scoreResults}
          onJump={handleJumpToQuestion}
        />
      )}

      <HighlightToolbar selection={highlight.selection} onApply={highlight.apply} toolbarRef={highlight.toolbarRef} />
    </div>
  );
}

function formatTime(totalSeconds) {
  const m = Math.floor(totalSeconds / 60).toString().padStart(2, '0');
  const s = (totalSeconds % 60).toString().padStart(2, '0');
  return `${m}:${s}`;
}
