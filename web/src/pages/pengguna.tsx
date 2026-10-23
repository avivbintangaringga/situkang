import { ArrowLeft, Plus } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router'

import { UserFormSheet } from '@/components/user-form-sheet'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useMe } from '@/hooks/use-auth'
import { useUsers } from '@/hooks/use-users'
import { formatTimestampDate } from '@/lib/format'

export function PenggunaPage() {
  const users = useUsers()
  const { data: me } = useMe()
  const [formOpen, setFormOpen] = useState(false)

  return (
    <div className="space-y-4">
      <Link
        to="/pengaturan"
        className="text-muted-foreground hover:text-foreground inline-flex min-h-11 items-center gap-1 text-sm"
      >
        <ArrowLeft className="size-4" aria-hidden />
        Pengaturan
      </Link>

      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Kelola pengguna</h1>
          <p className="text-muted-foreground text-sm">
            Semua akun adalah admin dan bisa mengelola data.
          </p>
        </div>
        <Button className="h-11 shrink-0 px-3" onClick={() => setFormOpen(true)}>
          <Plus className="size-4" aria-hidden />
          Tambah
        </Button>
      </div>

      {users.isPending && (
        <div className="space-y-2" aria-hidden>
          <Skeleton className="h-16 w-full rounded-xl" />
          <Skeleton className="h-16 w-full rounded-xl" />
          <Skeleton className="h-16 w-full rounded-xl" />
        </div>
      )}

      {users.isError && (
        <div className="rounded-xl border p-6 text-center">
          <p className="text-sm">
            {users.error instanceof Error
              ? users.error.message
              : 'Tidak dapat memuat daftar pengguna.'}
          </p>
          <Button
            variant="outline"
            className="mt-4 h-11 px-5"
            onClick={() => void users.refetch()}
          >
            Coba lagi
          </Button>
        </div>
      )}

      {!users.isPending && !users.isError && users.data && users.data.length === 0 && (
        <div className="rounded-xl border p-8 text-center">
          <p className="font-medium">Belum ada pengguna.</p>
          <p className="text-muted-foreground mt-1 text-sm">
            Tambahkan akun admin untuk mulai bekerja.
          </p>
          <Button className="mt-4 h-11 px-5" onClick={() => setFormOpen(true)}>
            Tambah pengguna
          </Button>
        </div>
      )}

      {!users.isPending && !users.isError && users.data && users.data.length > 0 && (
        <ul className="divide-y rounded-xl border">
          {users.data.map((user) => (
            <li
              key={user.id}
              className="flex min-h-14 items-center justify-between gap-3 px-4 py-3"
            >
              <div className="min-w-0">
                <p className="truncate font-medium">{user.username}</p>
                <p className="text-muted-foreground text-xs">
                  Dibuat {formatTimestampDate(user.created_at)}
                </p>
              </div>
              {me?.id === user.id && (
                <Badge variant="secondary" className="shrink-0 text-xs">
                  Anda
                </Badge>
              )}
            </li>
          ))}
        </ul>
      )}

      <UserFormSheet open={formOpen} onOpenChange={setFormOpen} />
    </div>
  )
}
