import { useNavigate } from 'react-router-dom';
import HoverGrid from '../components/HoverGrid/HoverGrid';
import './DashboardPage.css';

export default function DashboardPage() {
  const navigate = useNavigate();

  function handleLogout() {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    navigate('/login');
  }

  return (
    <div className="dashboard-page">
      <HoverGrid />
      <div className="dashboard-card">
        <span className="dashboard-tag">IELTS ARENA</span>
        <h1>Dashboard</h1>
        <p>Welcome back. Ready to practice?</p>
        <button className="dashboard-btn dashboard-btn-primary" onClick={() => navigate('/practice/writing')}>
          Luyện tập Writing
        </button>
        <button className="dashboard-btn" onClick={handleLogout}>
          Logout
        </button>
      </div>
    </div>
  );
}
