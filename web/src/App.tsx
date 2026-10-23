import { Navigate, Route, Routes } from 'react-router'

import { AppShell } from '@/components/app-shell'
import { RequireAuth } from '@/components/require-auth'
import { BerandaPage } from '@/pages/beranda'
import { LoginPage } from '@/pages/login'
import { PekerjaPage } from '@/pages/pekerja'

function App() {
  return (
    <Routes>
      <Route path="/masuk" element={<LoginPage />} />
      <Route element={<RequireAuth />}>
        <Route element={<AppShell />}>
          <Route path="/" element={<BerandaPage />} />
          <Route path="/pekerja" element={<PekerjaPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Route>
    </Routes>
  )
}

export default App
