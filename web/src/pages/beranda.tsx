import { ClipboardCheck, Receipt } from 'lucide-react'
import { Link } from 'react-router'

import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useMe } from '@/hooks/use-auth'
import { useDashboard } from '@/hooks/use-dashboard'
import { formatDays, formatRupiah } from '@/lib/format'

export function BerandaPage() {
  const { data: user } = useMe()
  const dashboard = useDashboard()

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

      <div className="flex flex-wrap gap-2">
        <Button className="h-11 flex-1" render={<Link to="/absensi" />}>
          <ClipboardCheck className="size-4" aria-hidden />
          Catat absensi
        </Button>
        <Button variant="outline" className="h-11 flex-1" render={<Link to="/rekap" />}>
          <Receipt className="size-4" aria-hidden />
          Rekap gaji
        </Button>
      </div>

      {dashboard.data?.draft_recap_id && (
        <div className="rounded-xl border border-amber-200 bg-amber-50 p-4 text-amber-900">
          <p className="text-sm font-medium">Ada draft rekap yang belum difinalkan.</p>
          <p className="mt-1 text-xs">
            Periksa dan finalkan agar pembayaran tercatat.
          </p>
          <Button
            variant="outline"
            className="mt-3 h-11 px-4"
            render={<Link to={`/rekap/${dashboard.data.draft_recap_id}`} />}
          >
            Buka draft rekap
          </Button>
        </div>
      )}

      {dashboard.isPending && (
        <div className="space-y-2" aria-hidden>
          <Skeleton className="h-28 w-full rounded-xl" />
          <Skeleton className="h-20 w-full rounded-xl" />
          <Skeleton className="h-20 w-full rounded-xl" />
        </div>
      )}

      {dashboard.isError && (
        <div className="rounded-xl border p-6 text-center">
          <p className="text-sm">
            {dashboard.error instanceof Error
              ? dashboard.error.message
              : 'Tidak dapat memuat ringkasan.'}
          </p>
          <Button
            variant="outline"
            className="mt-4 h-11 px-5"
            onClick={() => void dashboard.refetch()}
          >
            Coba lagi
          </Button>
        </div>
      )}

      {dashboard.data && (
        <>
          <div className="bg-muted/40 grid grid-cols-2 gap-3 rounded-xl border p-4">
            <div>
              <p className="text-muted-foreground text-xs">Gaji kotor</p>
              <p className="font-medium">{formatRupiah(dashboard.data.totals.gross)}</p>
            </div>
            <div>
              <p className="text-muted-foreground text-xs">Kas bon</p>
              <p className="font-medium">{formatRupiah(dashboard.data.totals.advances)}</p>
            </div>
            <div>
              <p className="text-muted-foreground text-xs">Hari tercatat</p>
              <p className="font-medium">{formatDays(dashboard.data.totals.quarters)} hari</p>
            </div>
            <div>
              <p className="text-muted-foreground text-xs">Total belum dibayar</p>
              <p className="font-semibold">{formatRupiah(dashboard.data.totals.net)}</p>
            </div>
          </div>

          <section className="space-y-2">
            <h2 className="text-sm font-semibold">Belum dibayar</h2>

            {dashboard.data.workers.length === 0 ? (
              <div className="rounded-xl border p-8 text-center">
                <p className="font-medium">Semua sudah dibayar.</p>
                <p className="text-muted-foreground mt-1 text-sm">
                  Belum ada absensi atau kas bon yang menunggu rekap.
                </p>
              </div>
            ) : (
              <ul className="space-y-2">
                {dashboard.data.workers.map((worker) => (
                  <li
                    key={worker.worker_id}
                    className="flex items-start justify-between gap-3 rounded-xl border p-4"
                  >
                    <div className="min-w-0">
                      <p className="truncate font-medium">{worker.name}</p>
                      <p className="text-muted-foreground truncate text-xs">{worker.role}</p>
                      <p className="text-muted-foreground mt-1 text-xs">
                        {formatDays(worker.quarters)} hari ·{' '}
                        {formatRupiah(worker.gross)}
                        {worker.advances > 0
                          ? ` · kas bon ${formatRupiah(worker.advances)}`
                          : ''}
                      </p>
                    </div>
                    <span className="shrink-0 text-right font-semibold">
                      {formatRupiah(worker.net)}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </section>
        </>
      )}
    </div>
  )
}
