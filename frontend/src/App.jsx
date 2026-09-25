import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import ProfileHud from './components/ProfileHud/ProfileHud';
import GradingNotifier from './components/GradingNotifier/GradingNotifier';
import LoginPage from './pages/LoginPage';
import RegisterPage from './pages/RegisterPage';
import DashboardPage from './pages/DashboardPage';
import SkillTestsPage from './pages/SkillTestsPage';
import WritingAttemptPage from './pages/WritingAttemptPage';
import ReadingAttemptPage from './pages/ReadingAttemptPage';
import ListeningAttemptPage from './pages/ListeningAttemptPage';
import SpeakingAttemptPage from './pages/SpeakingAttemptPage';
import SpeakingComposerPage from './pages/SpeakingComposerPage';
import SubmissionsPage from './pages/SubmissionsPage';
import SubmissionDetailPage from './pages/SubmissionDetailPage';
import CreateTestPage from './pages/CreateTestPage';
import { isAdmin } from './lib/auth';

// `focus` hides the ProfileHud — used on attempt pages and the review of
// a submission, where the test takes the whole screen and the candidate
// navigates away on their own.
function PrivateRoute({ children, focus = false }) {
  const token = localStorage.getItem('access_token');
  if (!token) return <Navigate to="/login" replace />;
  return (
    <>
      {!focus && <ProfileHud />}
      {children}
    </>
  );
}

// Gates a route to admin accounts — the button that links here is already
// hidden for non-admins (DashboardPage), this is the defense-in-depth layer
// for anyone who navigates to the URL directly. The backend
// (middleware.RequireAdmin) is the real enforcement point either way.
function AdminRoute({ children }) {
  const token = localStorage.getItem('access_token');
  if (!token) return <Navigate to="/login" replace />;
  if (!isAdmin()) return <Navigate to="/dashboard" replace />;
  return (
    <>
      <ProfileHud />
      {children}
    </>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/dashboard" element={
          <PrivateRoute>
            <DashboardPage />
          </PrivateRoute>
        } />
        <Route path="/practice/:skill" element={
          <PrivateRoute>
            <SkillTestsPage />
          </PrivateRoute>
        } />
        <Route path="/practice/writing/:testId" element={
          <PrivateRoute focus>
            <WritingAttemptPage />
          </PrivateRoute>
        } />
        <Route path="/practice/reading/:testId" element={
          <PrivateRoute focus>
            <ReadingAttemptPage />
          </PrivateRoute>
        } />
        <Route path="/practice/listening/:testId" element={
          <PrivateRoute focus>
            <ListeningAttemptPage />
          </PrivateRoute>
        } />
        <Route path="/practice/speaking/custom/new" element={
          <PrivateRoute>
            <SpeakingComposerPage />
          </PrivateRoute>
        } />
        <Route path="/practice/speaking/custom/:testId/edit" element={
          <PrivateRoute>
            <SpeakingComposerPage />
          </PrivateRoute>
        } />
        <Route path="/practice/speaking/:testId" element={
          <PrivateRoute focus>
            <SpeakingAttemptPage />
          </PrivateRoute>
        } />
        <Route path="/submissions" element={
          <PrivateRoute>
            <SubmissionsPage />
          </PrivateRoute>
        } />
        <Route path="/submissions/:submissionId" element={
          <PrivateRoute focus>
            <SubmissionDetailPage />
          </PrivateRoute>
        } />
        <Route path="/tests/create" element={
          <AdminRoute>
            <CreateTestPage />
          </AdminRoute>
        } />
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
      <GradingNotifier />
    </BrowserRouter>
  );
}
