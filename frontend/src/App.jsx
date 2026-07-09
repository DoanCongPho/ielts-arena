import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import ClickSpark from './components/ClickSpark/ClickSpark';
import LoginPage from './pages/LoginPage';
import RegisterPage from './pages/RegisterPage';
import DashboardPage from './pages/DashboardPage';
import SkillTestsPage from './pages/SkillTestsPage';
import WritingAttemptPage from './pages/WritingAttemptPage';
import ReadingAttemptPage from './pages/ReadingAttemptPage';
import ListeningAttemptPage from './pages/ListeningAttemptPage';
import SubmissionsPage from './pages/SubmissionsPage';
import SubmissionDetailPage from './pages/SubmissionDetailPage';
import CreateTestPage from './pages/CreateTestPage';

function PrivateRoute({ children }) {
  const token = localStorage.getItem('access_token');
  return token ? children : <Navigate to="/login" replace />;
}

export default function App() {
  return (
    <ClickSpark sparkColor="#0b0e14" sparkSize={10} sparkRadius={18} sparkCount={8} duration={500}>
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
            <PrivateRoute>
              <WritingAttemptPage />
            </PrivateRoute>
          } />
          <Route path="/practice/reading/:testId" element={
            <PrivateRoute>
              <ReadingAttemptPage />
            </PrivateRoute>
          } />
          <Route path="/practice/listening/:testId" element={
            <PrivateRoute>
              <ListeningAttemptPage />
            </PrivateRoute>
          } />
          <Route path="/submissions" element={
            <PrivateRoute>
              <SubmissionsPage />
            </PrivateRoute>
          } />
          <Route path="/submissions/:submissionId" element={
            <PrivateRoute>
              <SubmissionDetailPage />
            </PrivateRoute>
          } />
          <Route path="/tests/create" element={
            <PrivateRoute>
              <CreateTestPage />
            </PrivateRoute>
          } />
          <Route path="*" element={<Navigate to="/login" replace />} />
        </Routes>
      </BrowserRouter>
    </ClickSpark>
  );
}
