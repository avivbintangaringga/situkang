import { Link } from 'react-router'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { useMe } from '@/hooks/use-auth'

export function BerandaPage() {
  const { data: user } = useMe()

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold tracking-tight">
          Halo{user ? `, ${user.username}` : ''}
        </h1>
        <p className="text-muted-foreground text-sm">
          Catat absensi harian tukang, lalu buat rekap gaji mingguan.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Mulai dari pekerja</CardTitle>
          <CardDescription>
            Daftarkan tukang beserta upah hariannya sebelum mencatat absensi.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button className="h-11 px-5" render={<Link to="/pekerja" />}>
            Kelola pekerja
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Belum ada rekap</CardTitle>
          <CardDescription>
            Rekap gaji mingguan akan tampil di sini setelah absensi tercatat.
          </CardDescription>
        </CardHeader>
      </Card>
    </div>
  )
}
