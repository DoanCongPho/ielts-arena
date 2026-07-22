import { useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import ReactMarkdown from 'react-markdown';
import { getScore, getSubmission, getTest } from '../lib/api';
import { safeParse } from '../lib/safeParse';
import { SKILL_CONFIG } from '../lib/skillConfig';
import ScoreResult from '../components/ScoreResult/ScoreResult';
import QuestionList from '../components/QuestionList/QuestionList';
import AutoGradeResult from '../components/AutoGradeResult/AutoGradeResult';
import QuestionNavBar from '../components/QuestionNavBar/QuestionNavBar';
import Button from '../components/ui/Button/Button';
import './PracticePage.css';
import './WritingAttemptPage.css';

export default function SubmissionDetailPage() {
  const { submissionId } = useParams();
  const navigate = useNavigate();

  const [submission, setSubmission] = useState(null);
  const [test, setTest] = useState(null);
  const [score, setScore] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError('');

    getSubmission(submissionId)
      .then(async (sub) => {
        if (cancelled) return;
        setSubmission(sub);
        const testPromise = getTest(sub.test_id);
        const scorePromise = sub.status === 'graded' ? getScore(submissionId) : Promise.resolve(null);
        const [testResult, scoreResult] = await Promise.all([testPromise, scorePromise]);
        if (cancelled) return;
        setTest(testResult);
        setScore(scoreResult);
      })
      .catch((err) => {
        if (!cancelled) setError(err.message);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [submissionId]);

  if (loading) return <div className="attempt-page"><p className="practice-status">Đang tải...</p></div>;
  if (error) return <div className="attempt-page"><p className="practice-status practice-error">{error}</p></div>;

  const content = safeParse(test?.content_data);
  const payload = safeParse(submission?.payload);
  const skill = test?.skill;
  const isAutoGraded = skill === 'reading' || skill === 'listening';
  const notGradedMessage = submission?.status !== 'graded' && (
    <p className="practice-status practice-error" style={{ marginTop: 16 }}>
      {submission?.status === 'failed'
        ? 'Bài này chấm điểm thất bại.'
        : 'Bài này chưa được chấm điểm.'}
    </p>
  );

  return (
    <div className="attempt-page">
      <header className="attempt-header">
        <Button variant="secondary" onClick={() => navigate('/submissions')}>
          ← Bài đã làm
        </Button>
        <span className="attempt-timer">{formatDate(submission?.submitted_at)}</span>
      </header>

      {isAutoGraded ? (
        <MultiUnitReview
          skill={skill}
          test={test}
          content={content}
          payload={payload}
          score={score}
          notGradedMessage={notGradedMessage}
        />
      ) : (
        <div className="attempt-body">
          <div className="attempt-prompt-panel">
            <h2>{test?.task_type === 'task1' ? 'Writing Task 1' : test?.task_type === 'task2' ? 'Writing Task 2' : test?.task_type}</h2>
            {content?.image_url && (
              <img className="attempt-chart" src={content.image_url} alt="Task chart" />
            )}
            <div className="attempt-prompt-text">
              <ReactMarkdown>{content?.prompt || ''}</ReactMarkdown>
            </div>
          </div>

          <div className="attempt-answer-panel">
            <h2 className="submission-answer-heading">Bài làm của bạn</h2>
            <p className="submission-answer-text">{payload?.text}</p>
            {notGradedMessage}
            {score && <ScoreResult score={score} />}
          </div>
        </div>
      )}
    </div>
  );
}

// MultiUnitReview mirrors the live attempt page's layout — a tabbed
// passage/section switcher with a two-pane body — instead of stacking
// every unit vertically, so reviewing a 3-passage test works the same way
// as taking it: switch tabs, don't scroll through everything at once.
function MultiUnitReview({ skill, test, content, payload, score, notGradedMessage }) {
  const audioRef = useRef(null);
  const [activeIndex, setActiveIndex] = useState(0);

  // Set after clicking a QuestionNavBar pill for a question in a different
  // passage/section — the target element doesn't exist until the tab
  // switch re-renders, so the actual scroll happens in an effect keyed on
  // this (same pattern as ReadingAttemptPage/ListeningAttemptPage).
  const [pendingScrollOrder, setPendingScrollOrder] = useState(null);

  const units = skill === 'reading' ? (content?.passages || []) : (content?.sections || []);
  const activeUnit = units[activeIndex];
  const activeGroups = activeUnit?.question_groups || [];
  const scoreResults = score ? safeParse(score.details)?.results : undefined;
  const allQuestions = units.flatMap((u) => (u.question_groups || []).flatMap((g) => g.questions));
  const config = SKILL_CONFIG[skill];

  useEffect(() => {
    if (pendingScrollOrder == null) return;
    const el = document.getElementById(`question-${pendingScrollOrder}`);
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' });
      setPendingScrollOrder(null);
    }
  }, [pendingScrollOrder, activeIndex]);

  function handleSelectUnit(i) {
    setActiveIndex(i);
    if (skill === 'listening' && audioRef.current) {
      audioRef.current.currentTime = units[i]?.section_start_time ?? 0;
    }
  }

  function handleJumpToQuestion(order) {
    const targetIndex = units.findIndex((u) =>
      (u.question_groups || []).some((g) => g.questions.some((q) => q.question_order === order)),
    );
    if (targetIndex === -1) return;
    handleSelectUnit(targetIndex);
    if (skill === 'listening' && audioRef.current) {
      const question = allQuestions.find((q) => q.question_order === order);
      if (question?.timestamp_hint != null) {
        audioRef.current.currentTime = question.timestamp_hint;
      }
    }
    setPendingScrollOrder(order);
  }

  return (
    <div className="submission-multi-review">
      {score && <AutoGradeResult score={score} skill={skill} compact />}

      <div className="attempt-body">
        <div className="attempt-prompt-panel">
          <h2>{skill === 'reading' ? 'Reading' : 'Listening'} — {config.taskTypeLabel(test.task_type)}</h2>

          {units.length > 1 && (
            <nav className="skill-tabs attempt-multi-tabs">
              {units.map((u, i) => (
                <button
                  key={i}
                  className={`skill-tab ${i === activeIndex ? 'active' : ''}`}
                  onClick={() => handleSelectUnit(i)}
                >
                  {u.title || `${skill === 'reading' ? 'Passage' : 'Section'} ${i + 1}`}
                </button>
              ))}
            </nav>
          )}

          {skill === 'reading' ? (
            <div className="attempt-passage-text">
              {activeUnit?.title && <h3 className="reading-passage-title">{activeUnit.title}</h3>}
              {(activeUnit?.paragraphs || []).map((p, pi) => (
                <p key={pi}>
                  {p.label && <strong>{p.label}. </strong>}
                  {p.text}
                </p>
              ))}
            </div>
          ) : (
            <>
              <div className="attempt-audio-panel">
                <audio ref={audioRef} className="attempt-audio-player" controls src={content?.audio_url} />
              </div>
              <p className="practice-status">
                Đoạn ghi âm: {formatSeconds(activeUnit?.section_start_time)} – {formatSeconds(activeUnit?.section_end_time)}
              </p>
            </>
          )}
        </div>

        <div className="attempt-answer-panel">
          <QuestionList
            groups={activeGroups}
            answers={payload?.answers}
            results={scoreResults}
            disabled
            skill={skill}
          />
        </div>
      </div>

      {notGradedMessage}

      {allQuestions.length > 0 && (
        <QuestionNavBar
          questions={allQuestions}
          answers={payload?.answers}
          results={scoreResults}
          onJump={handleJumpToQuestion}
        />
      )}
    </div>
  );
}

function formatDate(iso) {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString('vi-VN', { dateStyle: 'medium', timeStyle: 'short' });
}

function formatSeconds(totalSeconds) {
  const s = Math.round(totalSeconds || 0);
  const m = Math.floor(s / 60).toString().padStart(2, '0');
  const rem = (s % 60).toString().padStart(2, '0');
  return `${m}:${rem}`;
}
