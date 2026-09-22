import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { getScore, getTest, submitAnswer, waitForGrading } from '../lib/api';
import { flattenQuestions } from '../lib/answerUtils';
import { safeParse } from '../lib/safeParse';
import { LISTENING_CHECK_SECONDS, useCountdown, useLeaveGuard } from '../lib/useAttemptSession';
import { SKILL_CONFIG } from '../lib/skillConfig';
import { useEvidenceReview } from '../lib/useEvidenceReview';
import { useSectionAudio } from '../lib/useSectionAudio';
import { useTextHighlights } from '../lib/useTextHighlights';
import QuestionList from '../components/QuestionList/QuestionList';
import { ReviewContext } from '../components/AnswerExplanation/ReviewContext';
import ListeningTranscript from '../components/ListeningTranscript/ListeningTranscript';
import ListeningExamPlayer from '../components/ListeningExamPlayer/ListeningExamPlayer';
import HighlightToolbar from '../components/HighlightToolbar/HighlightToolbar';
import { totalSeconds } from '../lib/listeningAudio';
import AutoGradeResult from '../components/AutoGradeResult/AutoGradeResult';
import QuestionNavBar from '../components/QuestionNavBar/QuestionNavBar';
import Button from '../components/ui/Button/Button';
import './PracticePage.css';
import './WritingAttemptPage.css';
import './ListeningAttemptPage.css';

export default function ListeningAttemptPage() {
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
  // The recording and the clock start together, on "Bắt đầu nghe".
  const [started, setStarted] = useState(false);

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

  const content = safeParse(test?.content_data);
  const sections = content?.sections || [];

  const inProgress = !!test && !score && !gradeFailed;
  const remaining = useCountdown(totalSeconds(content) + LISTENING_CHECK_SECONDS, inProgress && started, handleTimeUp);
  const confirmLeave = useLeaveGuard(inProgress);
  const audio = useSectionAudio(content, activeIndex);
  const review = useEvidenceReview(testId, !!score, sections, setActiveIndex);
  // Highlight question text, instructions and (in review) the transcript,
  // per section — as on the reading page.
  const highlight = useTextHighlights(activeIndex);

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

  // While the attempt is on, the recording plays straight through on its
  // own (ListeningExamPlayer) and switching section only changes what's on
  // screen. In review, it also seeks the review player to startTime within
  // that section's recording (useSectionAudio).
  function handleSelectSection(i, startTime) {
    if (!inProgress) audio.seek(i, startTime);
    setActiveIndex(i);
  }

  // Review: replay from where question `order` is answered.
  function handleListen(order) {
    const i = sections.findIndex((s) =>
      (s.question_groups || []).some((g) => g.questions.some((q) => q.question_order === order)),
    );
    if (i === -1) return;
    const question = allQuestions.find((q) => q.question_order === order);
    handleSelectSection(i, question?.timestamp_hint ?? sections[i]?.section_start_time);
    audio.play();
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

  const activeSection = sections[activeIndex];
  const activeGroups = activeSection?.question_groups || [];
  const allQuestions = sections.flatMap((s) => flattenQuestions(s.question_groups));
  const scoreResults = score ? safeParse(score.details)?.results : undefined;

  return (
    <div className="attempt-page attempt-page-focus">
      <header className="attempt-header">
        <button type="button" className="attempt-back" onClick={() => confirmLeave() && navigate('/practice/listening')} aria-label="Về danh sách đề" title="Về danh sách đề">
          ←
        </button>
        <span className={`attempt-timer ${inProgress && remaining <= 5 * 60 ? 'attempt-timer-low' : ''}`}>{formatTime(remaining)}</span>
      </header>

      {/* Taking the test: one full-width question column (tables and notes
          need the room), with the player docked at the bottom. Review:
          two columns, player + transcript beside the marked answers. */}
      <div className={`attempt-body ${inProgress ? 'listening-exam-layout' : ''}`} {...highlight.areaProps}>
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
          {inProgress ? (
            <p className="reading-highlight-hint listening-exam-hint">Bôi đen câu hỏi để tô đậm — bấm vào phần đã tô để bỏ.</p>
          ) : (
            <>
              <div className="attempt-audio-panel">
                <audio className="attempt-audio-player" controls {...audio.audioProps} />
              </div>
              <ListeningTranscript
                section={activeIndex}
                paragraphs={review.transcriptFor(activeIndex)}
                evidenceFor={review.evidenceFor}
                ranges={highlight.ranges}
                onRemoveRange={highlight.remove}
              />
            </>
          )}
        </div>

        <div className="attempt-answer-panel">
          {score && <AutoGradeResult score={score} skill="listening" compact />}

          {gradeFailed && (
            <div className="attempt-result">
              <p className="practice-status practice-error">
                Bài đã được lưu nhưng chấm điểm thất bại. Xem lại trong mục Lịch sử làm bài.
              </p>
            </div>
          )}

          {!gradeFailed && (
            <ReviewContext.Provider
              value={{ answerKey: review.answerKey, onLocate: review.locate, onListen: score ? handleListen : null, locateLabel: 'Xem trong transcript' }}
            >
              <QuestionList
                groups={activeGroups}
                answers={answers}
                onChange={score ? undefined : handleAnswerChange}
                disabled={submitting || !!score}
                results={scoreResults}
                highlights={highlight.ranges}
                onHighlightRemove={highlight.remove}
                skill="listening"
              />
            </ReviewContext.Provider>
          )}

          {error && <p className="practice-status practice-error">{error}</p>}

          {/* Submit only from the last section, like the end of a real paper. */}
          {!score && !gradeFailed && activeIndex === sections.length - 1 && (
            <Button variant="primary" className="attempt-submit-btn" onClick={handleSubmit} disabled={submitting}>
              {submitting ? 'Đang chấm điểm...' : 'Nộp bài'}
            </Button>
          )}
        </div>
      </div>

      <div className="listening-dock">
        {/* Mounted for the whole attempt so playback never restarts. */}
        {inProgress && (
          <div className="listening-dock-player">
            <ListeningExamPlayer content={content} started={started} onStart={() => setStarted(true)} />
          </div>
        )}
        {!gradeFailed && (
          <QuestionNavBar
            questions={allQuestions}
            answers={answers}
            results={scoreResults}
            onJump={handleJumpToQuestion}
          />
        )}
      </div>

      <HighlightToolbar selection={highlight.selection} onApply={highlight.apply} toolbarRef={highlight.toolbarRef} />
    </div>
  );
}

function formatTime(totalSeconds) {
  const m = Math.floor(totalSeconds / 60).toString().padStart(2, '0');
  const s = (totalSeconds % 60).toString().padStart(2, '0');
  return `${m}:${s}`;
}
