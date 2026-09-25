import { useEffect, useState } from 'react';
import { Navigate, useNavigate, useParams } from 'react-router-dom';
import { listTests } from '../lib/api';
import { safeParse } from '../lib/safeParse';
import { SKILLS, SKILL_CONFIG, seriesLabel } from '../lib/skillConfig';
import Button from '../components/ui/Button/Button';
import SkillTag from '../components/ui/SkillTag/SkillTag';
import Card from '../components/ui/Card/Card';
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

  const activeFilter = config?.taskFilters?.find((f) => f.key === taskFilter);
  const filterQuery = JSON.stringify(activeFilter?.query || {});

  useEffect(() => {
    if (!config?.enabled) return;
    let cancelled = false;
    setLoading(true);
    setError('');
    listTests(skill, page, JSON.parse(filterQuery))
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
  }, [skill, page, filterQuery, config?.enabled]);

  if (!config || !config.enabled) {
    return <Navigate to="/dashboard" replace />;
  }

  // The API already applied the filter; this page shows what it returned.
  const groups = groupBySeries(tests);

  // Inside a book the test number is what tells cards apart ("Test 3").
  function cardLabel(t) {
    return t.test_number ? `Test ${t.test_number}` : config.taskTypeLabel(t.task_type);
  }

  return (
    <div className="practice-page">
      <header className="practice-header">
        <h1 className="text-h1">IELTS Arena</h1>
        <Button variant="secondary" onClick={() => navigate('/dashboard')}>
          ← Dashboard
        </Button>
      </header>

      <nav className="skill-tabs">
        {SKILLS.map((s) => {
          const sConfig = SKILL_CONFIG[s.key];
          return (
            <button
              key={s.key}
              className={
                `skill-tab ${s.key === skill ? 'active' : ''}` +
                (!sConfig?.enabled ? ' skill-tab-coming-soon' : '')
              }
              disabled={!sConfig?.enabled}
              title={!sConfig?.enabled ? 'Sắp ra mắt' : undefined}
              onClick={() => navigate(`/practice/${s.key}`)}
            >
              {s.label}
              {sConfig?.comingSoon && <span className="skill-tab-badge text-label">Soon</span>}
            </button>
          );
        })}
      </nav>

      <div className="task-filters">
        {config.taskFilters.map((f) => (
          <button
            key={f.key}
            className={`task-filter-pill ${taskFilter === f.key ? 'active' : ''}`}
            onClick={() => {
              setTaskFilter(f.key);
              setPage(1);
            }}
          >
            {f.label}
          </button>
        ))}
      </div>

      {loading && <p className="practice-status">Đang tải đề...</p>}
      {error && <p className="practice-status practice-error">{error}</p>}

      {!loading && !error && tests.length === 0 && (
        <p className="practice-status">Chưa có đề nào cho bộ lọc này.</p>
      )}

      {groups.map((g) => (
        <section key={g.key} className="test-group">
          {g.label && <h2 className="test-group-title text-h3">{g.label}</h2>}
          <div className="test-grid">
            {g.tests.map((t) => {
              const content = safeParse(t.content_data);
              const summary = config.cardSummary(content);
              return (
                <Card
                  key={t.id}
                  padding="compact"
                  className="test-card"
                  onClick={() => navigate(config.attemptPath(t.id))}
                >
                  {t.thumbnail_url && (
                    <div className="test-card-media">
                      <img className="test-card-thumb" src={t.thumbnail_url} alt="" />
                      <span className="test-card-xp-chip text-data-sm">+{t.xp_gain} XP</span>
                      <span className="test-card-section-tag">
                        <SkillTag skill={skill}>{cardLabel(t)}</SkillTag>
                      </span>
                    </div>
                  )}
                  <div className="test-card-body">
                    {!t.thumbnail_url && (
                      <div className="test-card-tags">
                        <SkillTag skill={skill}>{cardLabel(t)}</SkillTag>
                        {content?.image_url && (
                          <span className="test-tag-neutral text-label">Có biểu đồ</span>
                        )}
                        <span className="test-card-xp-chip text-data-sm">+{t.xp_gain} XP</span>
                      </div>
                    )}
                    {summary && <p className="test-card-prompt text-body-sm">{summary}</p>}
                    {t.thumbnail_url && content?.image_url && (
                      <p className="test-card-subtitle text-body-sm">• Có biểu đồ</p>
                    )}
                  </div>
                </Card>
              );
            })}
          </div>
        </section>
      ))}

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

// groupBySeries splits tests (already ordered by the API: books first, then
// the rest) into one group per book, keeping that order. Tests outside any
// series form a trailing "Khác" group — unlabelled when nothing on the page
// belongs to a series, so skills without books look as before.
function groupBySeries(tests) {
  const groups = [];
  const byKey = new Map();
  for (const t of tests) {
    const key = t.series ? `${t.series}-${t.volume}` : 'other';
    let g = byKey.get(key);
    if (!g) {
      g = { key, label: seriesLabel(t) || 'Khác', tests: [] };
      byKey.set(key, g);
      groups.push(g);
    }
    g.tests.push(t);
  }
  if (groups.length === 1 && groups[0].key === 'other') groups[0].label = null;
  return groups;
}
