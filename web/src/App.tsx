import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './contexts/AuthContext';
import { ProtectedRoute } from './components/ProtectedRoute';
import { Layout } from './components/Layout';
import { RoleGuard } from './components/RoleGuard';
import { Login } from './pages/Login';
import { ChangePassword } from './pages/ChangePassword';
import { Dashboard } from './pages/Dashboard';
import { ScanPage } from './pages/ScanPage';
import { ZonesPage } from './pages/ZonesPage';
import { ZoneDetailPage } from './pages/ZoneDetailPage';
import { JobsPage } from './pages/JobsPage';
import { MaintenancePage } from './pages/MaintenancePage';
import { CasesPage } from './pages/CasesPage';
import { ProductsPage } from './pages/ProductsPage';
import { AdminPage } from './pages/AdminPage';
import { ProfilePage } from './pages/ProfilePage';
import LabelDesignerPage from './pages/LabelDesignerPage';

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          {/* Public route */}
          <Route path="/login" element={<Login />} />

          {/* Password change route (requires auth but bypasses force check) */}
          <Route path="/change-password" element={<ProtectedRoute bypassForcePasswordChange><ChangePassword /></ProtectedRoute>} />
          <Route path="/" element={<ProtectedRoute><Layout><Dashboard /></Layout></ProtectedRoute>} />
          <Route path="/scan" element={<ProtectedRoute><Layout><ScanPage /></Layout></ProtectedRoute>} />
          {/* Redirect legacy links from RentalCore to the products page */}
          <Route path="/admin/devices" element={<Navigate to="/products" replace />} />
          <Route path="/admin/devices/*" element={<Navigate to="/products" replace />} />
          <Route path="/labels" element={<ProtectedRoute><Layout><LabelDesignerPage /></Layout></ProtectedRoute>} />
          <Route path="/zones" element={<ProtectedRoute><Layout><ZonesPage /></Layout></ProtectedRoute>} />
          <Route path="/zones/:id" element={<ProtectedRoute><Layout><ZoneDetailPage /></Layout></ProtectedRoute>} />
          <Route path="/cases" element={<ProtectedRoute><Layout><CasesPage /></Layout></ProtectedRoute>} />
          <Route
            path="/products"
            element={
              <ProtectedRoute>
                <Layout>
                  <RoleGuard requiredRoles={['admin', 'manager', 'warehouse_admin']}>
                    <ProductsPage />
                  </RoleGuard>
                </Layout>
              </ProtectedRoute>
            }
          />
          <Route
            path="/cables"
            element={<Navigate to="/products" replace />}
          />
          <Route path="/jobs" element={<ProtectedRoute><Layout><JobsPage /></Layout></ProtectedRoute>} />
          <Route path="/jobs/:id" element={<ProtectedRoute><Layout><JobsPage /></Layout></ProtectedRoute>} />
          <Route path="/maintenance" element={<ProtectedRoute><Layout><MaintenancePage /></Layout></ProtectedRoute>} />
          <Route
            path="/admin/*"
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
