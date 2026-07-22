import { useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { getScore, getTest, submitAnswer } from '../lib/api';
import { safeParse } from '../lib/safeParse';
import { SKILL_CONFIG } from '../lib/skillConfig';
import QuestionList from '../components/QuestionList/QuestionList';
import AutoGradeResult from '../components/AutoGradeResult/AutoGradeResult';
import QuestionNavBar from '../components/QuestionNavBar/QuestionNavBar';
import Button from '../components/ui/Button/Button';
import './PracticePage.css';
import './WritingAttemptPage.css';
import './ListeningAttemptPage.css';

export default function ListeningAttemptPage() {
  const { testId } = useParams();
  const navigate = useNavigate();
  const audioRef = useRef(null);

  const [test, setTest] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [activeIndex, setActiveIndex] = useState(0);
  const [answers, setAnswers] = useState({});
  const [seconds, setSeconds] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [score, setScore] = useState(null);
  const [gradeFailed, setGradeFailed] = useState(false);

  // Set after clicking a QuestionNavBar pill for a question in a different
  // section — the target element doesn't exist until the section switch
  // re-renders, so the actual scroll happens in an effect keyed on this.
  const [pendingScrollOrder, setPendingScrollOrder] = useState(null);

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

  // All sections share ONE audio file; switching the active section tab
  // seeks the shared player to that section's start time instead of
  // swapping the <audio> src.
  function handleSelectSection(i, startTime) {
    setActiveIndex(i);
    if (audioRef.current) {
      audioRef.current.currentTime = startTime ?? 0;
    }
  }

  function handleJumpToQuestion(order) {
    const targetSectionIndex = sections.findIndex((s) =>
      (s.question_groups || []).some((g) => g.questions.some((q) => q.question_order === order)),
    );
    if (targetSectionIndex === -1) return;
    const question = allQuestions.find((q) => q.question_order === order);
    handleSelectSection(targetSectionIndex, question?.timestamp_hint ?? sections[targetSectionIndex]?.section_start_time);
    setPendingScrollOrder(order);
  }

  async function handleSubmit() {
    setSubmitting(true);
    setError('');
    try {
      const sub = await submitAnswer(Number(testId), { answers });
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
  const sections = content?.sections || [];
  const activeSection = sections[activeIndex];
  const activeGroups = activeSection?.question_groups || [];
  const allQuestions = sections.flatMap((s) => (s.question_groups || []).flatMap((g) => g.questions));
  const scoreResults = score ? safeParse(score.details)?.results : undefined;

  return (
    <div className="attempt-page">
      <header className="attempt-header">
        <Button variant="secondary" onClick={() => navigate('/practice/listening')}>
          ← Danh sách đề
        </Button>
        <span className="attempt-timer">{formatTime(seconds)}</span>
      </header>

      <div className="attempt-body">
        <div className="attempt-prompt-panel">
          <h2>Listening — {SKILL_CONFIG.listening.taskTypeLabel(test.task_type)}</h2>
          {sections.length > 1 && (
            <nav className="skill-tabs attempt-multi-tabs">
              {sections.map((s, i) => (
                <button
                  key={i}
                  className={`skill-tab ${i === activeIndex ? 'active' : ''}`}
                  onClick={() => handleSelectSection(i, s.section_start_time)}
                  disabled={submitting}
                >
                  {s.title || `Section ${i + 1}`}
                </button>
              ))}
            </nav>
          )}
          <div className="attempt-audio-panel">
            <audio ref={audioRef} className="attempt-audio-player" controls src={content?.audio_url} />
          </div>
        </div>

        <div className="attempt-answer-panel">
          {score && <AutoGradeResult score={score} skill="listening" compact />}

          {gradeFailed && (
            <div className="attempt-result">
              <p className="practice-status practice-error">
                Bài đã được lưu nhưng chấm điểm thất bại. Vui lòng thử lại sau.
              </p>
            </div>
          )}

          {!gradeFailed && (
            <QuestionList
              groups={activeGroups}
              answers={answers}
              onChange={score ? undefined : handleAnswerChange}
              disabled={submitting || !!score}
              results={scoreResults}
              skill="listening"
            />
          )}

          {error && <p className="practice-status practice-error">{error}</p>}

          {!score && !gradeFailed && (
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
    </div>
  );
}

function formatTime(totalSeconds) {
  const m = Math.floor(totalSeconds / 60).toString().padStart(2, '0');
  const s = (totalSeconds % 60).toString().padStart(2, '0');
  return `${m}:${s}`;
}
