import { Navigate, Route, Routes } from 'react-router'

import { AppShell } from '@/components/app-shell'
import { RequireAuth } from '@/components/require-auth'
import { AbsensiPage } from '@/pages/absensi'
import { BerandaPage } from '@/pages/beranda'
import { LoginPage } from '@/pages/login'
import { PekerjaPage } from '@/pages/pekerja'
import { PengaturanPage } from '@/pages/pengaturan'
import { ProjectDetailPage } from '@/pages/project-detail'
import { ProyekPage } from '@/pages/proyek'
import { RekapDetailPage } from '@/pages/rekap-detail'
import { RekapPage } from '@/pages/rekap'
import { WorkerDetailPage } from '@/pages/worker-detail'

function App() {
  return (
    <Routes>
      <Route path="/masuk" element={<LoginPage />} />
      <Route element={<RequireAuth />}>
        <Route element={<AppShell />}>
          <Route path="/" element={<BerandaPage />} />
          <Route path="/absensi" element={<AbsensiPage />} />
          <Route path="/pekerja" element={<PekerjaPage />} />
          <Route path="/pekerja/:id" element={<WorkerDetailPage />} />
          <Route path="/proyek" element={<ProyekPage />} />
          <Route path="/proyek/:id" element={<ProjectDetailPage />} />
          <Route path="/pengaturan" element={<PengaturanPage />} />
          <Route path="/rekap" element={<RekapPage />} />
          <Route path="/rekap/:id" element={<RekapDetailPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Route>
    </Routes>
  )
}

export default App
