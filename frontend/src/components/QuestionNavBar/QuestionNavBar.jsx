import { useRef } from 'react';
import { isAnswered, questionNumberLabel } from '../../lib/answerUtils';
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
export default function QuestionNavBar({ questions, answers, results, onJump }) {
  const stripRef = useRef(null);
  const sorted = [...questions].sort((a, b) => a.question_order - b.question_order);
  const isReview = !!results;
  const spanOf = (q) => q.span || 1;
  const totalMarks = sorted.reduce((n, q) => n + spanOf(q), 0);
  const correctCount = sorted.reduce((n, q) => {
    const r = results?.[q.question_order];
    return n + (r ? r.points ?? (r.correct ? spanOf(q) : 0) : 0);
  }, 0);
  const answeredCount = sorted.reduce((n, q) => {
    const a = answers?.[q.question_order];
    if (!isAnswered(a)) return n;
    return n + (Array.isArray(a) ? Math.min(a.length, spanOf(q)) : 1);
  }, 0);

  function scrollStrip(direction) {
    stripRef.current?.scrollBy({ left: direction * 240, behavior: 'smooth' });
  }

  return (
    <nav className="question-nav-bar">
      <div className="question-nav-bar-status">
        <span className="question-nav-bar-label text-label">Câu hỏi</span>
        <span className="question-nav-bar-count text-data-sm">
          {isReview ? `Đúng ${correctCount}/${totalMarks}` : `Đã làm ${answeredCount}/${totalMarks}`}
        </span>
        <BandMeter
          value={isReview ? correctCount : answeredCount}
          max={totalMarks || 1}
          className="question-nav-bar-meter"
        />
      </div>

      <button type="button" className="question-nav-bar-arrow" onClick={() => scrollStrip(-1)} aria-label="Cuộn trái">
        ‹
      </button>

      <div className="question-nav-bar-strip" ref={stripRef}>
        {sorted.map((q) => {
          const order = q.question_order;
          const stateClass = isReview
            ? results?.[order]
              ? results[order].correct
                ? 'question-nav-pill-correct'
                : 'question-nav-pill-incorrect'
              : ''
            : isAnswered(answers?.[order])
              ? 'question-nav-pill-done'
              : '';
          return (
            <button
              key={order}
              type="button"
              className={`question-nav-pill ${stateClass}`}
              onClick={() => onJump(order)}
            >
              {questionNumberLabel(order, spanOf(q))}
            </button>
          );
        })}
      </div>

      <button type="button" className="question-nav-bar-arrow" onClick={() => scrollStrip(1)} aria-label="Cuộn phải">
        ›
      </button>
    </nav>
  );
}
