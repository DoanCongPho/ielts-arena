import { useRef } from 'react';
import { isAnswered } from '../../lib/answerUtils';
import './QuestionNavBar.css';

// QuestionNavBar is a bottom bar listing every question number in a test —
// green once answered, click to jump straight to it (the page decides what
// "jump" means, e.g. switching passage/section tab then scrolling to it).
export default function QuestionNavBar({ questions, answers, onJump }) {
  const stripRef = useRef(null);
  const sorted = [...questions].sort((a, b) => a.question_order - b.question_order);
  const answeredCount = sorted.filter((q) => isAnswered(answers?.[q.question_order])).length;

  function scrollStrip(direction) {
    stripRef.current?.scrollBy({ left: direction * 240, behavior: 'smooth' });
  }

  return (
    <nav className="question-nav-bar">
      <div className="question-nav-bar-status">
        <span className="question-nav-bar-label">Câu hỏi</span>
        <span className="question-nav-bar-count">
          Đã làm {answeredCount}/{sorted.length}
        </span>
      </div>

      <button type="button" className="question-nav-bar-arrow" onClick={() => scrollStrip(-1)} aria-label="Cuộn trái">
        ‹
      </button>

      <div className="question-nav-bar-strip" ref={stripRef}>
        {sorted.map((q) => {
          const done = isAnswered(answers?.[q.question_order]);
          return (
            <button
              key={q.question_order}
              type="button"
              className={`question-nav-pill ${done ? 'question-nav-pill-done' : ''}`}
              onClick={() => onJump(q.question_order)}
            >
              {q.question_order}
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
