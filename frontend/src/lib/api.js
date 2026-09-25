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

// listTests fetches one page of official tests. filter narrows them on the
// server ({task_type, series}), so pagination counts only matching tests.
export function listTests(skill, page = 1, filter = {}) {
  const params = new URLSearchParams({ skill, page: String(page) });
  for (const [k, v] of Object.entries(filter)) {
    if (v) params.set(k, v);
  }
  return request(`/api/tests?${params}`);
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

// --- Speaking ---

// getSpeakingScript returns the examiner's script for a speaking test:
// {test_id, mode, lines: [{kind, part, text, question_id, seconds, audio_url}]}.
// audio_url is missing while a line's examiner recording is still being
// made; the runner reads that line with browser speech instead.
export function getSpeakingScript(testId) {
  return request(`/api/speaking/tests/${testId}/script`);
}

// requestUploadSlots returns `count` {key, upload_url} pairs to PUT
// recordings to. ext is the recording's container (webm, ogg, mp4, m4a).
export function requestUploadSlots(count, ext) {
  return request('/api/speaking/uploads', {
    method: 'POST',
    body: JSON.stringify({ count, ext }),
  });
}

// uploadRecording PUTs a recording to its upload link. The link is either
// straight to the storage bucket or a signed path on this server, so it
// carries its own authorisation and gets no bearer token.
export async function uploadRecording(uploadUrl, blob) {
  for (let attempt = 1; ; attempt++) {
    try {
      const res = await fetch(uploadUrl, {
        method: 'PUT',
        body: blob,
        headers: { 'Content-Type': blob.type || 'application/octet-stream' },
      });
      if (res.ok) return;
      if (attempt >= 3) throw new Error(`Tải bản ghi âm lên thất bại (${res.status})`);
    } catch (err) {
      if (attempt >= 3) throw err;
    }
    await new Promise((resolve) => setTimeout(resolve, 1000 * attempt));
  }
}

export function getRecordings(submissionId) {
  return request(`/api/speaking/submissions/${submissionId}/recordings`);
}

export function listCustomSpeaking(page = 1) {
  return request(`/api/speaking/custom?page=${page}`);
}

// createCustomSpeaking saves a custom test: {title, part1?, part2?, part3?},
// each part {bank_test_id} or {custom: <part content>}. Resolves to
// {test, warnings}.
export function createCustomSpeaking(body) {
  return request('/api/speaking/custom', { method: 'POST', body: JSON.stringify(body) });
}

export function updateCustomSpeaking(id, body) {
  return request(`/api/speaking/custom/${id}`, { method: 'PUT', body: JSON.stringify(body) });
}

export function deleteCustomSpeaking(id) {
  return request(`/api/speaking/custom/${id}`, { method: 'DELETE' });
}

// generateSpeakingParts drafts Part 1 and/or Part 3 around a Part 2 cue
// card, for the user to edit. Nothing is saved.
export function generateSpeakingParts(part2, { part1 = false, part3 = true } = {}) {
  return request('/api/speaking/custom/generate', {
    method: 'POST',
    body: JSON.stringify({ part2, part1, part3 }),
  });
}
