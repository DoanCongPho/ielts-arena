import { useEffect, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import Card from '../components/ui/Card/Card';
import Button from '../components/ui/Button/Button';
import { API_BASE } from '../lib/api';
import './AuthPage.css';

function EyeIcon({ open }) {
  return open ? (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
      <circle cx="12" cy="12" r="3"/>
    </svg>
  ) : (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/>
      <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/>
      <line x1="1" y1="1" x2="23" y2="23"/>
    </svg>
  );
}

export default function LoginPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [sessionExpired] = useState(() => searchParams.get('reason') === 'expired');
  const [form, setForm] = useState({ email: '', password: '' });
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  // Strip the ?reason=expired param from the URL once read, so refreshing
  // the login page later doesn't keep re-showing the notice.
  useEffect(() => {
    if (searchParams.get('reason')) {
      setSearchParams({}, { replace: true });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function handleChange(e) {
    setForm({ ...form, [e.target.name]: e.target.value });
  }

  async function handleSubmit(e) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      });
      const data = await res.json();
      if (!res.ok) { setError(data.message || 'Login failed'); return; }
      localStorage.setItem('access_token', data.data.access_token);
      localStorage.setItem('refresh_token', data.data.refresh_token);
      localStorage.setItem('user_role', data.data.user?.role || '');
      navigate('/dashboard');
    } catch {
      setError('Network error. Please try again.');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="auth-page">
      <Card className="auth-card-wrapper">
        <div className="auth-card-header">
          <h1 className="text-display-sm">IELTS Arena</h1>
          <p className="text-body-sm">Welcome back! Sign in to continue.</p>
        </div>

        <form onSubmit={handleSubmit} className="auth-form">
          {sessionExpired && !error && (
            <div className="auth-notice">Phiên đăng nhập đã hết hạn, vui lòng đăng nhập lại.</div>
          )}
          {error && <div className="auth-error">{error}</div>}

          <div className="auth-field">
            <label className="text-label" htmlFor="email">Email</label>
            <input
              id="email" name="email" type="email"
              placeholder="you@example.com"
              value={form.email} onChange={handleChange}
              required autoComplete="email"
            />
          </div>

          <div className="auth-field">
            <label className="text-label" htmlFor="password">Password</label>
            <div className="auth-password-wrapper">
              <input
                id="password" name="password"
                type={showPassword ? 'text' : 'password'}
                placeholder="••••••••"
                value={form.password} onChange={handleChange}
                required autoComplete="current-password"
              />
              <button
                type="button"
                className="auth-eye-btn"
                onClick={() => setShowPassword(v => !v)}
                tabIndex={-1}
                aria-label={showPassword ? 'Hide password' : 'Show password'}
              >
                <EyeIcon open={showPassword} />
              </button>
            </div>
          </div>

          <Button type="submit" variant="primary" className="auth-submit-btn" disabled={loading}>
            {loading ? 'Signing in...' : 'Sign In'}
          </Button>
        </form>

        <p className="auth-switch">
          Don&apos;t have an account?{' '}
          <Link to="/register">Register</Link>
        </p>
      </Card>
    </div>
  );
}
