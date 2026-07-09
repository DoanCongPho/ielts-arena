import { useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import ReactMarkdown from 'react-markdown';
import { getScore, getSubmission, getTest } from '../lib/api';
import { safeParse } from '../lib/safeParse';
import { SKILL_CONFIG } from '../lib/skillConfig';
import ScoreResult from '../components/ScoreResult/ScoreResult';
import QuestionList from '../components/QuestionList/QuestionList';
import AutoGradeResult from '../components/AutoGradeResult/AutoGradeResult';
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
        <button className="practice-back-btn" onClick={() => navigate('/submissions')}>
          ← Bài đã làm
        </button>
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

// MultiUnitReview renders a reading/listening submission as a stacked block
// per passage/section, since a multi-unit test doesn't fit the single
// prompt-panel-vs-answer-panel layout used by writing/speaking. Listening
// units share ONE audio file at the test level, so the player lives once
// at the top of the review and each section header gets a "listen here"
// button that seeks it, rather than one <audio> per section.
function MultiUnitReview({ skill, test, content, payload, score, notGradedMessage }) {
  const audioRef = useRef(null);
  const units = skill === 'reading' ? (content?.passages || []) : (content?.sections || []);
  const scoreResults = score ? safeParse(score.details)?.results : undefined;
  const allQuestions = units.flatMap((u) => (u.question_groups || []).flatMap((g) => g.questions));
  const config = SKILL_CONFIG[skill];

  function seekAudio(time) {
    if (audioRef.current) {
      audioRef.current.currentTime = time;
      audioRef.current.play();
    }
  }

  return (
    <div className="submission-multi-review">
      <h2>{skill === 'reading' ? 'Reading' : 'Listening'} — {config.taskTypeLabel(test.task_type)}</h2>

      {skill === 'listening' && (
        <div className="attempt-audio-panel">
          <audio ref={audioRef} className="attempt-audio-player" controls src={content?.audio_url} />
        </div>
      )}

      {units.map((u, i) => (
        <section key={i} className="submission-multi-block">
          <h3 className="submission-multi-heading">
            {u.title || `${skill === 'reading' ? 'Passage' : 'Section'} ${i + 1}`}
            {skill === 'listening' && (
              <button type="button" className="submission-multi-seek-btn" onClick={() => seekAudio(u.section_start_time)}>
                ▶ Nghe đoạn này
              </button>
            )}
          </h3>
          <div className="submission-multi-body">
            <div className="attempt-prompt-panel">
              {skill === 'reading' ? (
                <div className="attempt-passage-text">
                  {(u.paragraphs || []).map((p, pi) => (
                    <p key={`${i}-${pi}`}>
                      {p.label && <strong>{p.label}. </strong>}
                      {p.text}
                    </p>
                  ))}
                </div>
              ) : (
                <p className="practice-status">
                  Đoạn ghi âm: {formatSeconds(u.section_start_time)} – {formatSeconds(u.section_end_time)}
                </p>
              )}
            </div>
            <div className="attempt-answer-panel">
              <QuestionList
                groups={u.question_groups}
                answers={payload?.answers}
                results={scoreResults}
                disabled
              />
            </div>
          </div>
        </section>
      ))}
      {notGradedMessage}
      {score && <AutoGradeResult score={score} questions={allQuestions} onSeek={skill === 'listening' ? seekAudio : undefined} />}
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
