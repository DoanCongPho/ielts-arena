import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { listTests } from '../lib/api';
import './PracticePage.css';

const SKILLS = [
  { key: 'reading', label: 'Reading' },
  { key: 'listening', label: 'Listening' },
  { key: 'writing', label: 'Writing' },
  { key: 'speaking', label: 'Speaking' },
];

const TASK_FILTERS = [
  { key: 'all', label: 'Tất cả' },
  { key: 'task1', label: 'Task 1' },
  { key: 'task2', label: 'Task 2' },
];

export default function WritingTestsPage() {
  const navigate = useNavigate();
  const [taskFilter, setTaskFilter] = useState('all');
  const [page, setPage] = useState(1);
  const [tests, setTests] = useState([]);
  const [pagination, setPagination] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError('');
    listTests('writing', page)
      .then((data) => {
        if (cancelled) return;
        setTests(data.data || []);
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

  const visibleTests = tests.filter(
    (t) => taskFilter === 'all' || t.task_type === taskFilter,
  );

  return (
    <div className="practice-page">
      <header className="practice-header">
        <h1>IELTS Arena</h1>
        <button className="practice-back-btn" onClick={() => navigate('/dashboard')}>
          ← Dashboard
        </button>
      </header>

      <nav className="skill-tabs">
        {SKILLS.map((s) => (
          <button
            key={s.key}
            className={`skill-tab ${s.key === 'writing' ? 'active' : 'disabled'}`}
            disabled={s.key !== 'writing'}
            title={s.key !== 'writing' ? 'Sắp ra mắt' : undefined}
          >
            {s.label}
          </button>
        ))}
      </nav>

      <div className="task-filters">
        {TASK_FILTERS.map((f) => (
          <button
            key={f.key}
            className={`task-filter-pill ${taskFilter === f.key ? 'active' : ''}`}
            onClick={() => setTaskFilter(f.key)}
          >
            {f.label}
          </button>
        ))}
      </div>

      {loading && <p className="practice-status">Đang tải đề...</p>}
      {error && <p className="practice-status practice-error">{error}</p>}

      {!loading && !error && visibleTests.length === 0 && (
        <p className="practice-status">Chưa có đề nào cho bộ lọc này.</p>
      )}

      <div className="test-grid">
        {visibleTests.map((t) => {
          const content = safeParse(t.content_data);
          return (
            <div
              key={t.id}
              className="test-card"
              onClick={() => navigate(`/practice/writing/${t.id}`)}
            >
              <div className="test-card-tags">
                <span className="test-tag test-tag-task">
                  {t.task_type === 'task1' ? 'Task 1' : 'Task 2'}
                </span>
                {content?.image_url && <span className="test-tag test-tag-chart">Có biểu đồ</span>}
                <span className="test-tag test-tag-xp">+{t.xp_gain} XP</span>
              </div>
              <p className="test-card-prompt">{content?.prompt}</p>
            </div>
          );
        })}
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

function safeParse(raw) {
  try {
    return typeof raw === 'string' ? JSON.parse(raw) : raw;
  } catch {
    return null;
  }
}
