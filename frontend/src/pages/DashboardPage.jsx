import { useNavigate } from 'react-router-dom';

export default function DashboardPage() {
  const navigate = useNavigate();

  function handleLogout() {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    navigate('/login');
  }

  return (
    <div style={{
      minHeight: '100vh',
      background: '#0a0a0f',
      color: '#ffffff',
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      gap: '16px',
      fontFamily: '-apple-system, BlinkMacSystemFont, sans-serif',
    }}>
      <h1 style={{ fontSize: '32px', fontWeight: 700 }}>Dashboard</h1>
      <p style={{ color: 'rgba(255,255,255,0.5)' }}>Welcome to IELTS Arena! 🎉</p>
      <button
        onClick={handleLogout}
        style={{
          marginTop: '16px',
          padding: '10px 24px',
          background: 'rgba(255,255,255,0.08)',
          border: '1px solid rgba(255,255,255,0.12)',
          borderRadius: '8px',
          color: '#ffffff',
          cursor: 'pointer',
          fontSize: '14px',
        }}
      >
        Logout
      </button>
    </div>
  );
}
