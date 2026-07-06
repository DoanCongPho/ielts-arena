const API_BASE = 'http://localhost:8080';

function authHeaders() {
  const token = localStorage.getItem('access_token');
  return token ? { Authorization: `Bearer ${token}` } : {};
}

// Multiple requests can hit a 401 around the same time (e.g. a page that
// fires several API calls on mount). Sharing one in-flight refresh promise
// stops that from turning into a burst of parallel /auth/refresh calls.
let refreshPromise = null;

function refreshAccessToken() {
  if (!refreshPromise) {
    refreshPromise = (async () => {
      const refreshToken = localStorage.getItem('refresh_token');
      if (!refreshToken) return false;
      try {
        const res = await fetch(`${API_BASE}/auth/refresh`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: refreshToken }),
        });
        if (!res.ok) return false;
        const body = await res.json();
        localStorage.setItem('access_token', body.data.access_token);
        localStorage.setItem('refresh_token', body.data.refresh_token);
        return true;
      } catch {
        return false;
      }
    })().finally(() => {
      refreshPromise = null;
    });
  }
  return refreshPromise;
}

function forceLogout() {
  localStorage.removeItem('access_token');
  localStorage.removeItem('refresh_token');
  window.location.href = '/login?reason=expired';
}

async function request(path, options = {}, retry = true) {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...authHeaders(),
      ...options.headers,
    },
  });

  if (res.status === 401 && retry) {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      return request(path, options, false);
    }
    forceLogout();
    throw new Error('Session expired, please sign in again');
  }

  const body = await res.json().catch(() => null);
  if (!res.ok) {
    throw new Error(body?.message || `Request failed (${res.status})`);
  }
  return body?.data;
}

export function listTests(skill, page = 1) {
  return request(`/api/tests?skill=${encodeURIComponent(skill)}&page=${page}`);
}

export function getTest(id) {
  return request(`/api/tests/${id}`);
}

export function submitAnswer(testId, payload) {
  return request('/api/submissions', {
    method: 'POST',
    body: JSON.stringify({ test_id: testId, payload }),
  });
}

export function getScore(submissionId) {
  return request(`/api/submissions/${submissionId}/score`);
}
