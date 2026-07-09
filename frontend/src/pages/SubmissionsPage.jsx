import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { listSubmissions } from '../lib/api';
import './PracticePage.css';
import './SubmissionsPage.css';

const STATUS_LABEL = {
  pending: 'Đang chờ',
  submitted: 'Đã nộp',
  graded: 'Đã chấm',
  failed: 'Chấm lỗi',
};

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
        <h1>Bài đã làm</h1>
        <button className="practice-back-btn" onClick={() => navigate('/dashboard')}>
          ← Dashboard
        </button>
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
            className="submission-card"
            onClick={() => navigate(`/submissions/${sub.id}`)}
          >
            <div className="submission-card-tags">
              <span className="test-tag test-tag-task">
                {SKILL_LABEL[sub.test_skill] || sub.test_skill}
                {sub.test_task_type ? ` · ${sub.test_task_type}` : ''}
              </span>
              <span className={`submission-status submission-status-${sub.status}`}>
                {STATUS_LABEL[sub.status] || sub.status}
              </span>
              {sub.overall_band != null && (
                <span className="test-tag test-tag-xp">Band {sub.overall_band}</span>
              )}
            </div>
            <span className="submission-card-date">{formatDate(sub.submitted_at)}</span>
          </div>
        ))}
      </div>

      {pagination && (
        <div className="practice-pagination">
          <button disabled={!pagination.has_prev} onClick={() => setPage((p) => p - 1)}>
            ← Trước
          </button>
          <span>
            Trang {pagination.page} / {pagination.total_pages || 1}
          </span>
          <button disabled={!pagination.has_next} onClick={() => setPage((p) => p + 1)}>
            Sau →
          </button>
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
