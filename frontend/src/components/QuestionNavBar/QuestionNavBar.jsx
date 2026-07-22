import { useRef } from 'react';
import { isAnswered } from '../../lib/answerUtils';
import BandMeter from '../ui/BandMeter/BandMeter';
import './QuestionNavBar.css';

// QuestionNavBar is a bottom bar listing every question number in a test —
// click to jump straight to it (the page decides what "jump" means, e.g.
// switching passage/section tab then scrolling to it). Two modes: live
// (green once answered) and, once `results` is passed post-grading, review
// (green if correct, red if incorrect) — so a graded attempt stays
// navigable instead of losing the strip the moment it's submitted.
export default function QuestionNavBar({ questions, answers, results, onJump }) {
  const stripRef = useRef(null);
  const sorted = [...questions].sort((a, b) => a.question_order - b.question_order);
  const isReview = !!results;
  const correctCount = sorted.filter((q) => results?.[q.question_order]?.correct).length;
  const answeredCount = sorted.filter((q) => isAnswered(answers?.[q.question_order])).length;

  function scrollStrip(direction) {
    stripRef.current?.scrollBy({ left: direction * 240, behavior: 'smooth' });
  }

  return (
    <nav className="question-nav-bar">
      <div className="question-nav-bar-status">
        <span className="question-nav-bar-label text-label">Câu hỏi</span>
        <span className="question-nav-bar-count text-data-sm">
          {isReview ? `Đúng ${correctCount}/${sorted.length}` : `Đã làm ${answeredCount}/${sorted.length}`}
        </span>
        <BandMeter
          value={isReview ? correctCount : answeredCount}
          max={sorted.length || 1}
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
              {order}
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
