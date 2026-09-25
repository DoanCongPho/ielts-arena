import { useEffect, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { isSettled, listSubmissions } from '../lib/api';
import { SETTLED_EVENT } from '../lib/gradingWatch';
import Button from '../components/ui/Button/Button';
import SkillTag from '../components/ui/SkillTag/SkillTag';
import StatusBadge from '../components/ui/StatusBadge/StatusBadge';
import './PracticePage.css';
import './SubmissionsPage.css';

const SKILL_LABEL = {
  writing: 'Writing',
  speaking: 'Speaking',
  reading: 'Reading',
  listening: 'Listening',
};

// While any submission on the page is still being graded, the list reloads
// itself this often, so statuses and bands appear without a manual refresh.
const REFRESH_MS = 5000;

export default function SubmissionsPage() {
  const navigate = useNavigate();
  const location = useLocation();
  // Set by an attempt page that just handed in (see watchGrading).
  const [justSubmitted, setJustSubmitted] = useState(location.state?.justSubmitted || null);
  const [page, setPage] = useState(1);
  const [submissions, setSubmissions] = useState([]);
  const [pagination, setPagination] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [reload, setReload] = useState(0);

  // Drop the "just submitted" state from history once read, so reloading
  // this page doesn't bring the notice back.
  useEffect(() => {
    if (location.state?.justSubmitted) navigate('.', { replace: true, state: null });
  }, [location.state, navigate]);

  const anyUnsettled = submissions.some((s) => !isSettled(s.status));

  // Reload while something is being graded, and as soon as GradingNotifier
  // reports one settled.
  useEffect(() => {
    if (!anyUnsettled) return undefined;
    const id = setInterval(() => setReload((n) => n + 1), REFRESH_MS);
    return () => clearInterval(id);
  }, [anyUnsettled]);
  useEffect(() => {
    const onSettled = () => setReload((n) => n + 1);
    window.addEventListener(SETTLED_EVENT, onSettled);
    return () => window.removeEventListener(SETTLED_EVENT, onSettled);
  }, []);

  useEffect(() => {
    let cancelled = false;
    // Only the first load shows the loading state; refreshes swap the
    // rows in place so the list doesn't flicker.
    if (reload === 0) setLoading(true);
    setError('');
    listSubmissions(page)
      .then((data) => {
        if (cancelled) return;
        setSubmissions(data.data || []);
        setPagination(data.pagination);
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
  }, [page, reload]);

  const justSubmittedRow = justSubmitted && submissions.find((s) => s.id === justSubmitted.id);
  const justSubmittedDone = justSubmittedRow && isSettled(justSubmittedRow.status);

  return (
    <div className="practice-page">
      <header className="practice-header">
        <h1 className="text-h1">Bài đã làm</h1>
        <Button variant="secondary" onClick={() => navigate('/dashboard')}>
          ← Dashboard
        </Button>
      </header>

      {justSubmitted && (
        <div className={`submission-notice ${justSubmittedDone ? 'submission-notice-done' : ''}`}>
          <span className="submission-notice-text">
            {justSubmittedDone ? (
              <><strong>{justSubmitted.label}</strong> đã chấm xong.</>
            ) : (
              <>
                Đã nộp <strong>{justSubmitted.label}</strong>. Bài đang được chấm — bạn không cần chờ ở đây, cứ làm bài
                khác; khi có kết quả sẽ có thông báo.
              </>
            )}
          </span>
          {justSubmittedDone && (
            <Button variant="primary" onClick={() => navigate(`/submissions/${justSubmitted.id}`)}>
              Xem kết quả
            </Button>
          )}
          <button type="button" className="submission-notice-close" aria-label="Đóng" onClick={() => setJustSubmitted(null)}>
            ×
          </button>
        </div>
      )}

      {loading && <p className="practice-status">Đang tải...</p>}
      {error && <p className="practice-status practice-error">{error}</p>}

      {!loading && !error && submissions.length === 0 && (
        <p className="practice-status">Bạn chưa làm bài nào.</p>
      )}

      <div className="submission-list">
        {submissions.map((sub) => (
          <div
            key={sub.id}
            className="submission-row"
            onClick={() => navigate(`/submissions/${sub.id}`)}
          >
            <SkillTag skill={sub.test_skill} className="submission-row-skill">
              {SKILL_LABEL[sub.test_skill] || sub.test_skill}
              {sub.test_task_type ? ` · ${sub.test_task_type}` : ''}
            </SkillTag>
            <span className="submission-row-date text-data-sm">{formatDate(sub.submitted_at)}</span>
            <span className="submission-row-band text-data-sm">
              {sub.overall_band != null ? `Band ${sub.overall_band}` : ''}
            </span>
            <StatusBadge status={sub.status} className="submission-row-status" busy={!isSettled(sub.status)} />
          </div>
        ))}
      </div>

      {pagination && (
        <div className="practice-pagination">
          <Button variant="secondary" disabled={!pagination.has_prev} onClick={() => setPage((p) => p - 1)}>
            ← Trước
          </Button>
          <span className="text-data-sm">
            Trang {pagination.page} / {pagination.total_pages || 1}
          </span>
          <Button variant="secondary" disabled={!pagination.has_next} onClick={() => setPage((p) => p + 1)}>
            Sau →
          </Button>
        </div>
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
