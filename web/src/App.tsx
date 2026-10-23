import { Navigate, Route, Routes } from 'react-router'

import { AppShell } from '@/components/app-shell'
import { RequireAuth } from '@/components/require-auth'
import { AttendancePage } from '@/pages/attendance'
import { SharePage } from '@/pages/share'
import { HomePage } from '@/pages/home'
import { LoginPage } from '@/pages/login'
import { WorkersPage } from '@/pages/workers'
import { SettingsPage } from '@/pages/settings'
import { UsersPage } from '@/pages/users'
import { ProjectDetailPage } from '@/pages/project-detail'
import { ProjectsPage } from '@/pages/projects'
import { RecapDetailPage } from '@/pages/recap-detail'
import { RecapsPage } from '@/pages/recaps'
import { WorkerDetailPage } from '@/pages/worker-detail'

function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/share/:token" element={<SharePage />} />
      <Route element={<RequireAuth />}>
        <Route element={<AppShell />}>
          <Route path="/" element={<HomePage />} />
          <Route path="/attendance" element={<AttendancePage />} />
          <Route path="/workers" element={<WorkersPage />} />
          <Route path="/workers/:id" element={<WorkerDetailPage />} />
          <Route path="/projects" element={<ProjectsPage />} />
          <Route path="/projects/:id" element={<ProjectDetailPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="/users" element={<UsersPage />} />
          <Route path="/recaps" element={<RecapsPage />} />
          <Route path="/recaps/:id" element={<RecapDetailPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Route>
    </Routes>
  )
}

export default App
