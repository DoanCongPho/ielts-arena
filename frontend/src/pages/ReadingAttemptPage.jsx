import { useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { getScore, getTest, submitAnswer } from '../lib/api';
import { safeParse } from '../lib/safeParse';
import { mergeRanges, textOffset } from '../lib/highlightText';
import { SKILL_CONFIG } from '../lib/skillConfig';
import QuestionList from '../components/QuestionList/QuestionList';
import AutoGradeResult from '../components/AutoGradeResult/AutoGradeResult';
import QuestionNavBar from '../components/QuestionNavBar/QuestionNavBar';
import HighlightableText from '../components/HighlightableText/HighlightableText';
import Button from '../components/ui/Button/Button';
import './PracticePage.css';
import './WritingAttemptPage.css';
import './ReadingAttemptPage.css';

export default function ReadingAttemptPage() {
  const { testId } = useParams();
  const navigate = useNavigate();
  const highlightAreaRef = useRef(null);
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

  // Text highlighting — works anywhere in either panel (passage paragraphs,
  // group instructions, question text), a plain frontend convenience (like
  // the highlighter tool in the real IELTS on-screen test). Keyed by
  // passage index, then a highlight key identifying which text element
  // (`p-{i}` for a paragraph, `group-{order}-instructions`, `q-{order}-text`
  // — see data-highlight-key on HighlightableText). Never submitted to the
  // backend.
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
      if (highlightAreaRef.current?.contains(e.target)) return;
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

  // Fires on mouseup anywhere in either panel (passage or questions) — the
  // nearest ancestor with data-highlight-key identifies which text element
  // was selected (a paragraph, a group's instructions, or a question's
  // text), and offsets are computed relative to that element specifically,
  // so distinct elements never share one bucket of character ranges.
  function handleHighlightAreaMouseUp() {
    const sel = window.getSelection();
    if (!sel || sel.isCollapsed || sel.rangeCount === 0) {
      setSelectionInfo(null);
      return;
    }
    const range = sel.getRangeAt(0);
    const anchorEl = range.commonAncestorContainer.nodeType === Node.TEXT_NODE
      ? range.commonAncestorContainer.parentElement
      : range.commonAncestorContainer;
    const targetEl = anchorEl?.closest?.('[data-highlight-key]');
    if (!targetEl) {
      setSelectionInfo(null);
      return;
    }
    const key = targetEl.dataset.highlightKey;
    const from = textOffset(targetEl, range.startContainer, range.startOffset);
    const to = textOffset(targetEl, range.endContainer, range.endOffset);
    const start = Math.min(from, to);
    const end = Math.max(from, to);
    if (start === end) {
      setSelectionInfo(null);
      return;
    }
    const rect = range.getBoundingClientRect();
    setSelectionInfo({ key, start, end, x: rect.left + rect.width / 2, y: rect.top });
  }

  function applyHighlight() {
    if (!selectionInfo) return;
    const { key, start, end } = selectionInfo;
    setHighlights((prev) => {
      const passageMap = { ...(prev[activeIndex] || {}) };
      passageMap[key] = mergeRanges([...(passageMap[key] || []), { start, end }]);
      return { ...prev, [activeIndex]: passageMap };
    });
    window.getSelection()?.removeAllRanges();
    setSelectionInfo(null);
  }

  function removeHighlight(key, start, end) {
    setHighlights((prev) => {
      const passageMap = { ...(prev[activeIndex] || {}) };
      passageMap[key] = (passageMap[key] || []).filter((r) => !(r.start === start && r.end === end));
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
  const scoreResults = score ? safeParse(score.details)?.results : undefined;

  return (
    <div className="attempt-page reading-attempt-page">
      <header className="attempt-header">
        <Button variant="secondary" onClick={() => navigate('/practice/reading')}>
          ← Danh sách đề
        </Button>
        <span className="attempt-timer">{formatTime(seconds)}</span>
      </header>

      <div className="attempt-body reading-attempt-body" ref={highlightAreaRef} onMouseUp={handleHighlightAreaMouseUp}>
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
                <p key={`${activeIndex}-${pi}`}>
                  {p.label && <strong>{p.label}. </strong>}
                  <HighlightableText id={key} text={p.text} ranges={highlights[activeIndex]?.[key]} onRemoveRange={removeHighlight} />
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
              highlights={highlights[activeIndex]}
              onHighlightRemove={removeHighlight}
              skill="reading"
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
