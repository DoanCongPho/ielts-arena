import { useNavigate } from 'react-router-dom';
import Card from '../components/ui/Card/Card';
import IconChip from '../components/ui/IconChip/IconChip';
import { isAdmin, clearUserRole } from '../lib/auth';
import './DashboardPage.css';

const SKILL_NAV = [
  { skill: 'reading', title: 'Luyện tập Reading', subtitle: 'Đọc hiểu & trả lời câu hỏi', path: '/practice/reading' },
  { skill: 'listening', title: 'Luyện tập Listening', subtitle: 'Nghe & trả lời câu hỏi', path: '/practice/listening' },
  { skill: 'writing', title: 'Luyện tập Writing', subtitle: 'Viết bài & nhận chấm điểm', path: '/practice/writing' },
  { skill: 'speaking', title: 'Luyện tập Speaking', subtitle: 'Thi nói với giám khảo & chấm phát âm', path: '/practice/speaking' },
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
        <h1 className="text-display-sm">Dashboard</h1>
        <p className="dashboard-subtitle text-body">Welcome back. Ready to practice?</p>

        <nav className="dashboard-nav-list">
          {SKILL_NAV.map(({ skill, title, subtitle, path }) => (
            <Card
              key={skill}
              as="button"
              type="button"
              interactive
              padding="default"
              className="dashboard-nav-item"
              onClick={() => navigate(path)}
            >
              <IconChip icon={skill} size={52} />
              <span className="dashboard-nav-text">
                <span className="dashboard-nav-title text-h2">{title}</span>
                <span className="dashboard-nav-subtitle text-body">{subtitle}</span>
              </span>
            </Card>
          ))}

          <Card
            as="button"
            type="button"
            interactive
            padding="default"
            className="dashboard-nav-item"
            onClick={() => navigate('/submissions')}
          >
            <IconChip icon="history" size={52} />
            <span className="dashboard-nav-text">
              <span className="dashboard-nav-title text-h2">Bài đã làm</span>
              <span className="dashboard-nav-subtitle text-body">Xem lại lịch sử luyện tập</span>
            </span>
          </Card>

          {isAdmin() && (
            <Card
              as="button"
              type="button"
              interactive
              padding="default"
              className="dashboard-nav-item"
              onClick={() => navigate('/tests/create')}
            >
              <IconChip icon="add" size={52} />
              <span className="dashboard-nav-text">
                <span className="dashboard-nav-title text-h2">Tạo đề thi</span>
                <span className="dashboard-nav-subtitle text-body">Dành cho quản trị viên</span>
              </span>
            </Card>
          )}

          <Card
            as="button"
            type="button"
            interactive
            padding="default"
            className="dashboard-nav-item"
            onClick={handleLogout}
          >
            <IconChip icon="logout" size={52} />
            <span className="dashboard-nav-text">
              <span className="dashboard-nav-title text-h2">Logout</span>
            </span>
          </Card>
        </nav>
      </div>
    </div>
  );
}
