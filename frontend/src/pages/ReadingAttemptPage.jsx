import { useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { getScore, getTest, submitAnswer } from '../lib/api';
import { safeParse } from '../lib/safeParse';
import { splitHighlightSegments, mergeRanges, textOffset } from '../lib/highlightText';
import { SKILL_CONFIG } from '../lib/skillConfig';
import QuestionList from '../components/QuestionList/QuestionList';
import AutoGradeResult from '../components/AutoGradeResult/AutoGradeResult';
import QuestionNavBar from '../components/QuestionNavBar/QuestionNavBar';
import './PracticePage.css';
import './WritingAttemptPage.css';
import './ReadingAttemptPage.css';

export default function ReadingAttemptPage() {
  const { testId } = useParams();
  const navigate = useNavigate();
  const passageRef = useRef(null);
  const toolbarRef = useRef(null);

  const [test, setTest] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [activeIndex, setActiveIndex] = useState(0);
  const [answers, setAnswers] = useState({});
  const [seconds, setSeconds] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [score, setScore] = useState(null);
  const [gradeFailed, setGradeFailed] = useState(false);

  // Text highlighting in the passage panel — a plain frontend convenience
  // (like the highlighter tool in the real IELTS on-screen test), keyed by
  // passage index then paragraph label, never submitted to the backend.
  const [highlights, setHighlights] = useState({});
  const [selectionInfo, setSelectionInfo] = useState(null);

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

  useEffect(() => {
    if (score || gradeFailed) return;
    const id = setInterval(() => setSeconds((s) => s + 1), 1000);
    return () => clearInterval(id);
  }, [score, gradeFailed]);

  useEffect(() => {
    setSelectionInfo(null);
  }, [activeIndex]);

  useEffect(() => {
    function handleDocMouseDown(e) {
      if (toolbarRef.current?.contains(e.target)) return;
      if (passageRef.current?.contains(e.target)) return;
      setSelectionInfo(null);
    }
    document.addEventListener('mousedown', handleDocMouseDown);
    return () => document.removeEventListener('mousedown', handleDocMouseDown);
  }, []);

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

  function handlePassageMouseUp() {
    const sel = window.getSelection();
    if (!sel || sel.isCollapsed || sel.rangeCount === 0) {
      setSelectionInfo(null);
      return;
    }
    const range = sel.getRangeAt(0);
    const anchorEl = range.commonAncestorContainer.nodeType === Node.TEXT_NODE
      ? range.commonAncestorContainer.parentElement
      : range.commonAncestorContainer;
    const paragraphEl = anchorEl?.closest?.('[data-paragraph-label]');
    if (!paragraphEl) {
      setSelectionInfo(null);
      return;
    }
    const label = paragraphEl.dataset.paragraphLabel;
    const from = textOffset(paragraphEl, range.startContainer, range.startOffset);
    const to = textOffset(paragraphEl, range.endContainer, range.endOffset);
    const start = Math.min(from, to);
    const end = Math.max(from, to);
    if (start === end) {
      setSelectionInfo(null);
      return;
    }
    const rect = range.getBoundingClientRect();
    setSelectionInfo({ label, start, end, x: rect.left + rect.width / 2, y: rect.top });
  }

  function applyHighlight() {
    if (!selectionInfo) return;
    const { label, start, end } = selectionInfo;
    setHighlights((prev) => {
      const passageMap = { ...(prev[activeIndex] || {}) };
      passageMap[label] = mergeRanges([...(passageMap[label] || []), { start, end }]);
      return { ...prev, [activeIndex]: passageMap };
    });
    window.getSelection()?.removeAllRanges();
    setSelectionInfo(null);
  }

  function removeHighlight(label, start, end) {
    setHighlights((prev) => {
      const passageMap = { ...(prev[activeIndex] || {}) };
      passageMap[label] = (passageMap[label] || []).filter((r) => !(r.start === start && r.end === end));
      return { ...prev, [activeIndex]: passageMap };
    });
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
  const passages = content?.passages || [];
  const activePassage = passages[activeIndex];
  const activeGroups = activePassage?.question_groups || [];
  const allQuestions = passages.flatMap((p) => (p.question_groups || []).flatMap((g) => g.questions));

  return (
    <div className="attempt-page reading-attempt-page">
      <header className="attempt-header">
        <button className="practice-back-btn" onClick={() => navigate('/practice/reading')}>
          ← Danh sách đề
        </button>
        <span className="attempt-timer">{formatTime(seconds)}</span>
      </header>

      <div className="attempt-body reading-attempt-body">
        <div className="attempt-prompt-panel reading-scroll-panel">
          <h2>Reading — {SKILL_CONFIG.reading.taskTypeLabel(test.task_type)}</h2>
          {passages.length > 1 && (
            <nav className="skill-tabs attempt-multi-tabs">
              {passages.map((p, i) => (
                <button
                  key={i}
                  className={`skill-tab ${i === activeIndex ? 'active' : ''}`}
                  onClick={() => setActiveIndex(i)}
                  disabled={submitting || !!score || gradeFailed}
                >
                  {p.title || `Passage ${i + 1}`}
                </button>
              ))}
            </nav>
          )}
          <p className="reading-highlight-hint">Bôi đen văn bản để tô đậm — bấm vào phần đã tô để bỏ.</p>
          <div className="attempt-passage-text" ref={passageRef} onMouseUp={handlePassageMouseUp}>
            {activePassage?.title && <h3 className="reading-passage-title">{activePassage.title}</h3>}
            {(activePassage?.paragraphs || []).map((p, pi) => {
              const ranges = highlights[activeIndex]?.[p.label] || [];
              const segments = splitHighlightSegments(p.text, ranges);
              return (
                <p key={`${activeIndex}-${pi}`}>
                  {p.label && <strong>{p.label}. </strong>}
                  <span data-paragraph-label={p.label}>
                    {segments.map((seg, i) =>
                      seg.highlighted ? (
                        <mark
                          key={i}
                          className="reading-highlight-mark"
                          title="Bấm để bỏ tô đậm"
                          onClick={() => removeHighlight(p.label, seg.start, seg.end)}
                        >
                          {seg.text}
                        </mark>
                      ) : (
                        <span key={i}>{seg.text}</span>
                      ),
                    )}
                  </span>
                </p>
              );
            })}
          </div>
        </div>

        <div className="attempt-answer-panel reading-scroll-panel">
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
            </>
          )}

          {gradeFailed && (
            <div className="attempt-result">
              <p className="practice-status practice-error">
                Bài đã được lưu nhưng chấm điểm thất bại. Vui lòng thử lại sau.
              </p>
            </div>
          )}

          {score && <AutoGradeResult score={score} questions={allQuestions} />}
        </div>
      </div>

      {!score && !gradeFailed && (
        <QuestionNavBar questions={allQuestions} answers={answers} onJump={handleJumpToQuestion} />
      )}

      {selectionInfo && (
        <button
          ref={toolbarRef}
          type="button"
          className="reading-highlight-toolbar"
          style={{ left: selectionInfo.x, top: selectionInfo.y }}
          onClick={applyHighlight}
        >
          🖍 Tô đậm
        </button>
      )}
    </div>
  );
}

function formatTime(totalSeconds) {
  const m = Math.floor(totalSeconds / 60).toString().padStart(2, '0');
  const s = (totalSeconds % 60).toString().padStart(2, '0');
  return `${m}:${s}`;
}
