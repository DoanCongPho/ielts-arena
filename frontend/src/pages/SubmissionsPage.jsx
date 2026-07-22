import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { listSubmissions } from '../lib/api';
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

export default function SubmissionsPage() {
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const [submissions, setSubmissions] = useState([]);
  const [pagination, setPagination] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
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
  }, [page]);

  return (
    <div className="practice-page">
      <header className="practice-header">
        <h1 className="text-h1">Bài đã làm</h1>
        <Button variant="secondary" onClick={() => navigate('/dashboard')}>
          ← Dashboard
        </Button>
      </header>

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
            <StatusBadge status={sub.status} className="submission-row-status" />
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
