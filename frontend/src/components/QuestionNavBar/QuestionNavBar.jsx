import { flattenQuestions, isAnswered, questionNumberLabel } from '../../lib/answerUtils';
import BandMeter from '../ui/BandMeter/BandMeter';
import './QuestionNavBar.css';

// QuestionNavBar is a bottom bar listing every question number in a test —
// click to jump straight to it (the page decides what "jump" means, e.g.
// switching passage/section tab then scrolling to it). Two modes: live
// (green once answered) and, once `results` is passed post-grading, review
// (green if correct, red if incorrect) — so a graded attempt stays
// navigable instead of losing the strip the moment it's submitted.
// Counts are in marks, like the score: a question with `span` > 1 (a
// "choose TWO" multi-select) counts per key picked / per point earned.
//
// `units` are the test's passages (reading) or sections (listening). The
// numbers are grouped by part, each with its own count; every part keeps
// its numbers and their state on show, so any question is one click away,
// and the part on screen (`activeUnit`) is highlighted. A part's name
// jumps to its first question.
export default function QuestionNavBar({ units, activeUnit = 0, answers, results, onJump }) {
  const isReview = !!results;
  const spanOf = (q) => q.span || 1;

  const parts = (units || []).map((u, i) => ({
    label: `Part ${i + 1}`,
    questions: flattenQuestions(u.question_groups).sort((a, b) => a.question_order - b.question_order),
  }));

  function tally(questions) {
    let total = 0;
    let answered = 0;
    let correct = 0;
    for (const q of questions) {
      const span = spanOf(q);
      total += span;
      const r = results?.[q.question_order];
      if (r) correct += r.points ?? (r.correct ? span : 0);
      const a = answers?.[q.question_order];
      if (isAnswered(a)) answered += Array.isArray(a) ? Math.min(a.length, span) : 1;
    }
    return { total, done: isReview ? correct : answered };
  }

  const overall = tally(parts.flatMap((p) => p.questions));

  function pillClass(order) {
    if (isReview) {
      const r = results?.[order];
      if (!r) return '';
      return r.correct ? 'question-nav-pill-correct' : 'question-nav-pill-incorrect';
    }
    return isAnswered(answers?.[order]) ? 'question-nav-pill-done' : '';
  }

  return (
    <nav className="question-nav-bar">
      <div className="question-nav-bar-status">
        <span className="question-nav-bar-label text-label">Câu hỏi</span>
        <span className="question-nav-bar-count text-data-sm">
          {isReview ? `Đúng ${overall.done}/${overall.total}` : `Đã làm ${overall.done}/${overall.total}`}
        </span>
        <BandMeter value={overall.done} max={overall.total || 1} className="question-nav-bar-meter" />
      </div>

      <div className="question-nav-bar-parts">
        {parts.map((part, i) => {
          const { total, done } = tally(part.questions);
          const first = part.questions[0];
          const complete = !isReview && total > 0 && done === total;
          return (
            <div
              key={part.label}
              className={`question-nav-part ${i === activeUnit ? 'question-nav-part-active' : ''}`}
            >
              <button
                type="button"
                className={`question-nav-part-head ${complete ? 'question-nav-part-complete' : ''}`}
                onClick={() => first && onJump(first.question_order)}
                title={`Chuyển sang ${part.label}`}
              >
                <span className="question-nav-part-label">{part.label}</span>
                <span className="question-nav-part-count text-data-sm">
                  {done}/{total}
                </span>
              </button>
              <div className="question-nav-part-pills">
                {part.questions.map((q) => (
                  <button
                    key={q.question_order}
                    type="button"
                    className={`question-nav-pill ${pillClass(q.question_order)}`}
                    onClick={() => onJump(q.question_order)}
                  >
                    {questionNumberLabel(q.question_order, spanOf(q))}
                  </button>
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </nav>
  );
}
