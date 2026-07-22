import { useNavigate } from 'react-router-dom';
import Card from '../components/ui/Card/Card';
import IconChip from '../components/ui/IconChip/IconChip';
import { isAdmin, clearUserRole } from '../lib/auth';
import './DashboardPage.css';

const SKILL_NAV = [
  { skill: 'reading', title: 'Luyện tập Reading', subtitle: 'Đọc hiểu & trả lời câu hỏi', path: '/practice/reading' },
  { skill: 'listening', title: 'Luyện tập Listening', subtitle: 'Nghe & trả lời câu hỏi', path: '/practice/listening' },
  { skill: 'writing', title: 'Luyện tập Writing', subtitle: 'Viết bài & nhận chấm điểm', path: '/practice/writing' },
];

export default function DashboardPage() {
  const navigate = useNavigate();

  function handleLogout() {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    clearUserRole();
    navigate('/login');
  }

  return (
    <div className="dashboard-page">
      <div className="dashboard-column">
        <span className="dashboard-tag text-label">IELTS Arena</span>
        <h1 className="text-h1">Dashboard</h1>
        <p className="dashboard-subtitle text-body-sm">Welcome back. Ready to practice?</p>

        <nav className="dashboard-nav-list">
          {SKILL_NAV.map(({ skill, title, subtitle, path }) => (
            <Card
              key={skill}
              as="button"
              type="button"
              interactive
              padding="compact"
              className="dashboard-nav-item"
              onClick={() => navigate(path)}
            >
              <IconChip icon={skill} />
              <span className="dashboard-nav-text">
                <span className="dashboard-nav-title text-h3">{title}</span>
                <span className="dashboard-nav-subtitle text-body-sm">{subtitle}</span>
              </span>
            </Card>
          ))}

          <Card
            as="button"
            type="button"
            interactive
            padding="compact"
            className="dashboard-nav-item"
            onClick={() => navigate('/submissions')}
          >
            <IconChip icon="history" />
            <span className="dashboard-nav-text">
              <span className="dashboard-nav-title text-h3">Bài đã làm</span>
              <span className="dashboard-nav-subtitle text-body-sm">Xem lại lịch sử luyện tập</span>
            </span>
          </Card>

          {isAdmin() && (
            <Card
              as="button"
              type="button"
              interactive
              padding="compact"
              className="dashboard-nav-item"
              onClick={() => navigate('/tests/create')}
            >
              <IconChip icon="add" />
              <span className="dashboard-nav-text">
                <span className="dashboard-nav-title text-h3">Tạo đề thi</span>
                <span className="dashboard-nav-subtitle text-body-sm">Dành cho quản trị viên</span>
              </span>
            </Card>
          )}

          <Card
            as="button"
            type="button"
            interactive
            padding="compact"
            className="dashboard-nav-item"
            onClick={handleLogout}
          >
            <IconChip icon="logout" />
            <span className="dashboard-nav-text">
              <span className="dashboard-nav-title text-h3">Logout</span>
            </span>
          </Card>
        </nav>
      </div>
    </div>
  );
}
