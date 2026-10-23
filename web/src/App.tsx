import { Navigate, Route, Routes } from 'react-router'

import { RequireAuth } from '@/components/require-auth'
import { BerandaPage } from '@/pages/beranda'
import { LoginPage } from '@/pages/login'

function App() {
  return (
    <Routes>
      <Route path="/masuk" element={<LoginPage />} />
      <Route element={<RequireAuth />}>
        <Route path="/" element={<BerandaPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  )
}

export default App
