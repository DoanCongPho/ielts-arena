// Empty by default: the app talks to its own origin and a proxy routes
// /api to the backend (nginx in production, Vite's dev server locally).
// That keeps every request same-origin, so there is no CORS anywhere in
// the stack. Set VITE_API_BASE only to point a build at a remote API.
export const API_BASE = import.meta.env.VITE_API_BASE || '';

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
        const res = await fetch(`${API_BASE}/api/auth/refresh`, {
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
  localStorage.removeItem('user_role');
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

// getAnswerKey returns a graded test's {questions, transcripts}: answers,
// explanations and evidence keyed by question_order, plus each listening
// section's transcript. 403 until the user has a graded attempt at it.
export function getAnswerKey(testId) {
  return request(`/api/tests/${testId}/answer-key`);
}

export function createTest(payload) {
  return request('/api/tests', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

// submitAnswer queues an answer for grading. The API answers 202 with a
// "pending" submission — grading happens in a background worker — so the
// caller follows up with waitForGrading() rather than reading a score off
// this response.
export function submitAnswer(testId, payload) {
  return request('/api/submissions', {
    method: 'POST',
    body: JSON.stringify({ test_id: testId, payload }),
  });
}

// Statuses that mean the grading worker is done with a submission, one way
// or another. Anything else ("pending", "grading") means keep waiting.
const SETTLED_STATUSES = new Set(['graded', 'failed']);

export function isSettled(status) {
  return SETTLED_STATUSES.has(status);
}

/**
 * Polls a submission until grading settles, then returns it.
 *
 * Writing/speaking go out to an LLM, so this can legitimately take tens of
 * seconds; reading/listening are graded in-process and usually settle on
 * the first or second poll. onTick is called with each intermediate
 * submission so the UI can show progress.
 */
export async function waitForGrading(submissionId, { intervalMs = 1500, timeoutMs = 180000, onTick } = {}) {
  const deadline = Date.now() + timeoutMs;

  for (;;) {
    const sub = await getSubmission(submissionId);
    if (isSettled(sub.status)) {
      // Grading just finished, which may have granted XP server-side —
      // let ProfileHud know to refetch without prop-drilling it down.
      window.dispatchEvent(new Event('profile:refresh'));
      return sub;
    }
    if (onTick) onTick(sub);
    if (Date.now() >= deadline) {
      throw new Error('Bài đang được chấm lâu hơn dự kiến. Bạn có thể xem lại trong mục Lịch sử làm bài.');
    }
    await new Promise((resolve) => setTimeout(resolve, intervalMs));
  }
}

export function getScore(submissionId) {
  return request(`/api/submissions/${submissionId}/score`);
}

export function listSubmissions(page = 1) {
  return request(`/api/submissions?page=${page}`);
}

export function getSubmission(id) {
  return request(`/api/submissions/${id}`);
}

export function getProfile() {
  return request('/api/profile');
}

export function setEquippedFrame(frameLevel) {
  return request('/api/profile/frame', {
    method: 'PUT',
    body: JSON.stringify({ frame_level: frameLevel }),
  });
}
