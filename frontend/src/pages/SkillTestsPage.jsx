import { useEffect, useState } from 'react';
import { Navigate, useNavigate, useParams } from 'react-router-dom';
import { listTests } from '../lib/api';
import { safeParse } from '../lib/safeParse';
import { SKILLS, SKILL_CONFIG } from '../lib/skillConfig';
import './PracticePage.css';

export default function SkillTestsPage() {
  const navigate = useNavigate();
  const { skill } = useParams();
  const config = SKILL_CONFIG[skill];

  const [taskFilter, setTaskFilter] = useState('all');
  const [page, setPage] = useState(1);
  const [tests, setTests] = useState([]);
  const [pagination, setPagination] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Reset the task filter whenever the skill changes so a filter key from a
  // different skill's filter set (e.g. "task1" while switching to reading)
  // doesn't linger and hide everything.
  useEffect(() => {
    setTaskFilter('all');
    setPage(1);
  }, [skill]);

  useEffect(() => {
    if (!config?.enabled) return;
    let cancelled = false;
    setLoading(true);
    setError('');
    listTests(skill, page)
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
  }, [skill, page, config?.enabled]);

  if (!config || !config.enabled) {
    return <Navigate to="/dashboard" replace />;
  }

  // Prefix match instead of exact match: for Writing, filter keys are the
  // exact task_type values (task1/task2) and a string always starts with
  // itself, so this is equivalent for that skill. For Reading/Listening,
  // filter keys are categories ("passage"/"section"/"test") that should
  // match any numbered task_type in that category (passage1, passage2, ...)
  // without needing a separate filter pill per number.
  const visibleTests = tests.filter(
    (t) => taskFilter === 'all' || t.task_type.startsWith(taskFilter),
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
        {SKILLS.map((s) => {
          const sConfig = SKILL_CONFIG[s.key];
          return (
            <button
              key={s.key}
              className={`skill-tab ${s.key === skill ? 'active' : sConfig?.enabled ? '' : 'disabled'}`}
              disabled={!sConfig?.enabled}
              title={!sConfig?.enabled ? 'Sắp ra mắt' : undefined}
              onClick={() => navigate(`/practice/${s.key}`)}
            >
              {s.label}
            </button>
          );
        })}
      </nav>

      <div className="task-filters">
        {config.taskFilters.map((f) => (
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
          const summary = config.cardSummary(content);
          return (
            <div
              key={t.id}
              className="test-card"
              onClick={() => navigate(config.attemptPath(t.id))}
            >
              {t.thumbnail_url && (
                <div className="test-card-media">
                  <img className="test-card-thumb" src={t.thumbnail_url} alt="" />
                  <span className="test-card-media-badge">+{t.xp_gain} XP</span>
                  <span className="test-card-section-tag">
                    {config.taskTypeLabel(t.task_type)}
                  </span>
                </div>
              )}
              <div className="test-card-body">
                {!t.thumbnail_url && (
                  <div className="test-card-tags">
                    <span className="test-tag test-tag-task">
                      {config.taskTypeLabel(t.task_type)}
                    </span>
                    {content?.image_url && <span className="test-tag test-tag-chart">Có biểu đồ</span>}
                    <span className="test-tag test-tag-xp">+{t.xp_gain} XP</span>
                  </div>
                )}
                {summary && <p className="test-card-prompt">{summary}</p>}
                {t.thumbnail_url && content?.image_url && (
                  <p className="test-card-subtitle">• Có biểu đồ</p>
                )}
              </div>
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
