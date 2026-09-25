import { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { getSubmission, isSettled } from '../../lib/api';
import { SETTLED_EVENT, WATCH_EVENT, forgetGrading, watchedGrades } from '../../lib/gradingWatch';
import './GradingNotifier.css';

const POLL_MS = 5000;
const TOAST_MS = 15000;

// GradingNotifier polls the submissions being graded in the background and
// shows a notification when each one settles, wherever the learner is.
// Mounted once, inside the router.
export default function GradingNotifier() {
  const navigate = useNavigate();
  const [toasts, setToasts] = useState([]);
  const timer = useRef(null);

  const dismiss = useCallback((id) => setToasts((ts) => ts.filter((t) => t.id !== id)), []);

  const check = useCallback(async () => {
    const watched = watchedGrades();
    if (!watched.length || !localStorage.getItem('access_token')) return;
    for (const w of watched) {
      let sub;
      try {
        sub = await getSubmission(w.id);
      } catch {
        continue; // offline or signed out: try again next round
      }
      if (!isSettled(sub.status)) continue;
      forgetGrading(w.id);
      setToasts((ts) => [...ts.filter((t) => t.id !== w.id), { id: w.id, label: w.label, status: sub.status }]);
      setTimeout(() => dismiss(w.id), TOAST_MS);
      // Grading may have granted XP; the list and review pages refresh too.
      window.dispatchEvent(new Event('profile:refresh'));
      window.dispatchEvent(new CustomEvent(SETTLED_EVENT, { detail: { submission: sub } }));
    }
  }, [dismiss]);

  useEffect(() => {
    function schedule() {
      clearTimeout(timer.current);
      timer.current = setTimeout(async () => {
        await check();
        if (watchedGrades().length) schedule();
      }, POLL_MS);
    }
    if (watchedGrades().length) schedule();
    window.addEventListener(WATCH_EVENT, schedule);
    return () => {
      clearTimeout(timer.current);
      window.removeEventListener(WATCH_EVENT, schedule);
    };
  }, [check]);

  if (!toasts.length) return null;
  return (
    <div className="grading-toasts" role="status" aria-live="polite">
      {toasts.map((t) => (
        <div key={t.id} className={`grading-toast grading-toast-${t.status}`}>
          <span className="grading-toast-icon" aria-hidden>{t.status === 'graded' ? '✓' : '!'}</span>
          <span className="grading-toast-text">
            {t.status === 'graded'
              ? <><strong>{t.label}</strong> đã chấm xong.</>
              : <><strong>{t.label}</strong> chấm điểm thất bại.</>}
          </span>
          <button
            type="button"
            className="grading-toast-link"
            onClick={() => {
              dismiss(t.id);
              navigate(`/submissions/${t.id}`);
            }}
          >
            Xem kết quả
          </button>
          <button type="button" className="grading-toast-close" aria-label="Đóng" onClick={() => dismiss(t.id)}>
            ×
          </button>
        </div>
      ))}
    </div>
  );
}
