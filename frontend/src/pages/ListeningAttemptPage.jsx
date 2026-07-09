import { useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { getScore, getTest, submitAnswer } from '../lib/api';
import { safeParse } from '../lib/safeParse';
import { isAnswered } from '../lib/answerUtils';
import { SKILL_CONFIG } from '../lib/skillConfig';
import QuestionList from '../components/QuestionList/QuestionList';
import AutoGradeResult from '../components/AutoGradeResult/AutoGradeResult';
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
  const answeredCount = allQuestions.filter((q) => isAnswered(answers[q.question_order])).length;

  return (
    <div className="attempt-page">
      <header className="attempt-header">
        <button className="practice-back-btn" onClick={() => navigate('/practice/listening')}>
          ← Danh sách đề
        </button>
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
                  disabled={submitting || !!score || gradeFailed}
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
          {!score && !gradeFailed && (
            <>
              <QuestionList
                groups={activeGroups}
                answers={answers}
                onChange={handleAnswerChange}
                disabled={submitting}
              />
              {error && <p className="practice-status practice-error">{error}</p>}
              <button className="attempt-submit-btn" onClick={handleSubmit} disabled={submitting}>
                {submitting ? 'Đang chấm điểm...' : 'Nộp bài'}
              </button>
              <p className="attempt-answer-hint">
                Đã trả lời {answeredCount}/{allQuestions.length} câu
              </p>
            </>
          )}

          {gradeFailed && (
            <div className="attempt-result">
              <p className="practice-status practice-error">
                Bài đã được lưu nhưng chấm điểm thất bại. Vui lòng thử lại sau.
              </p>
            </div>
          )}

          {score && (
            <AutoGradeResult
              score={score}
              questions={allQuestions}
              onSeek={(t) => {
                if (audioRef.current) {
                  audioRef.current.currentTime = t;
                  audioRef.current.play();
                }
              }}
            />
          )}
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
