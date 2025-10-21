import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AuthProvider } from './contexts/AuthContext';
import { ProtectedRoute } from './components/ProtectedRoute';
import { Layout } from './components/Layout';
import { RoleGuard } from './components/RoleGuard';
import { Login } from './pages/Login';
import { Dashboard } from './pages/Dashboard';
import { ScanPage } from './pages/ScanPage';
import { DevicesPage } from './pages/DevicesPage';
import { ZonesPage } from './pages/ZonesPage';
import { ZoneDetailPage } from './pages/ZoneDetailPage';
import { JobsPage } from './pages/JobsPage';
import { MaintenancePage } from './pages/MaintenancePage';
import { CasesPage } from './pages/CasesPage';
import { AdminPage } from './pages/AdminPage';
import { ProfilePage } from './pages/ProfilePage';

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          {/* Public route */}
          <Route path="/login" element={<Login />} />

          {/* Protected routes */}
          <Route path="/" element={<ProtectedRoute><Layout><Dashboard /></Layout></ProtectedRoute>} />
          <Route path="/scan" element={<ProtectedRoute><Layout><ScanPage /></Layout></ProtectedRoute>} />
          <Route path="/devices" element={<ProtectedRoute><Layout><DevicesPage /></Layout></ProtectedRoute>} />
          <Route path="/zones" element={<ProtectedRoute><Layout><ZonesPage /></Layout></ProtectedRoute>} />
          <Route path="/zones/:id" element={<ProtectedRoute><Layout><ZoneDetailPage /></Layout></ProtectedRoute>} />
          <Route path="/cases" element={<ProtectedRoute><Layout><CasesPage /></Layout></ProtectedRoute>} />
          <Route path="/jobs" element={<ProtectedRoute><Layout><JobsPage /></Layout></ProtectedRoute>} />
          <Route path="/maintenance" element={<ProtectedRoute><Layout><MaintenancePage /></Layout></ProtectedRoute>} />
          <Route
            path="/admin"
            element={
              <ProtectedRoute>
                <Layout>
                  <RoleGuard requiredRoles={['admin', 'manager', 'warehouse_admin']}>
                    <AdminPage />
                  </RoleGuard>
                </Layout>
              </ProtectedRoute>
            }
          />
          <Route path="/profile" element={<ProtectedRoute><Layout><ProfilePage /></Layout></ProtectedRoute>} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}

export default App;
