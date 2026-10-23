import { useNavigate } from 'react-router'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { useLogout, useMe } from '@/hooks/use-auth'

export function BerandaPage() {
  const { data: user } = useMe()
  const logout = useLogout()
  const navigate = useNavigate()

  function onLogout() {
    logout.mutate(undefined, {
      onSettled: () => navigate('/masuk', { replace: true }),
    })
  }

  return (
    <div className="bg-background min-h-dvh">
      <header className="border-border bg-background/95 sticky top-0 border-b backdrop-blur">
        <div className="mx-auto flex h-14 w-full max-w-3xl items-center justify-between px-4">
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold">SiTukang</p>
            {user && (
              <p className="text-muted-foreground truncate text-xs">{user.username}</p>
            )}
          </div>
          <Button
            variant="outline"
            className="h-11 px-4"
            onClick={onLogout}
            disabled={logout.isPending}
          >
            {logout.isPending ? 'Keluar…' : 'Keluar'}
          </Button>
        </div>
      </header>

      <main className="mx-auto w-full max-w-3xl space-y-4 px-4 py-6">
        <Card>
          <CardHeader>
            <CardTitle>Selamat datang</CardTitle>
            <CardDescription>
              Catat absensi harian tukang, lalu buat rekap gaji mingguan.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-muted-foreground text-sm">
              Tahap autentikasi selesai. Menu absensi, rekap, pekerja, dan proyek
              menyusul.
            </p>
          </CardContent>
        </Card>
      </main>
    </div>
  )
}
