import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import Layout from './components/Layout';
import LoginPage from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';
import AlertsPage from './pages/AlertsPage';
import ActionsPage from './pages/ActionsPage';
import RulesPage from './pages/RulesPage';
import CreateRulePage from './pages/CreateRulePage';
import EventsPage from './pages/EventsPage';
import UsersPage from './pages/UsersPage';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuth();
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" replace />;
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  const { user } = useAuth();
  if (!user) return <Navigate to="/login" replace />;
  if (user.role !== 'admin') return <Navigate to="/agents" replace />;
  return <>{children}</>;
}

function AppRoutes() {
  const { isAuthenticated } = useAuth();
  return (
    <Routes>
      <Route path="/login" element={isAuthenticated ? <Navigate to="/agents" replace /> : <LoginPage />} />
      <Route path="/" element={<ProtectedRoute><Layout /></ProtectedRoute>}>
        <Route index element={<Navigate to="/agents" replace />} />
        <Route path="agents"       element={<DashboardPage />} />
        <Route path="alerts"       element={<AlertsPage />} />
        <Route path="actions"      element={<ActionsPage />} />
        <Route path="rules"        element={<RulesPage />} />
        <Route path="rules/create" element={<CreateRulePage />} />
        <Route path="rules/edit/:id" element={<CreateRulePage />} />
        <Route path="events"       element={<EventsPage />} />
        <Route path="users"        element={<AdminRoute><UsersPage /></AdminRoute>} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <AppRoutes />
      </BrowserRouter>
    </AuthProvider>
  );
}
