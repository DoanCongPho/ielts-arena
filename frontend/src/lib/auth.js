// Role is cached in localStorage at login time (see LoginPage.jsx) so pages
// can gate admin-only UI (like the create-test flow) without an extra
// network round-trip. It's a display/UX convenience only — the backend
// (middleware.RequireAdmin) is the actual enforcement point.
export function getUserRole() {
  return localStorage.getItem('user_role') || '';
}

export function isAdmin() {
  return getUserRole() === 'admin';
}

export function clearUserRole() {
  localStorage.removeItem('user_role');
}
