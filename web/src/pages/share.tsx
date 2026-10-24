import { useParams } from 'react-router'

import { ThemeToggle } from '@/components/theme-toggle'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { usePublicRecap } from '@/hooks/use-recap-share'
import { formatDate, formatDays, formatRupiah } from '@/lib/format'
import { cn } from '@/lib/utils'

/**
 * Public read-only recap page. It lives outside the auth guard and the app
 * shell so a shared link can be opened by anyone.
 */
export function SharePage() {
  const params = useParams()
  const token = params.token ?? ''
  const { data, isPending, isError, refetch } = usePublicRecap(token)

  return (
    <div className="bg-muted/30 min-h-dvh">
      <header className="bg-background/90 sticky top-0 z-30 border-b backdrop-blur">
        <div className="mx-auto flex h-14 w-full max-w-2xl items-center justify-between px-4">
          <div className="flex items-center gap-2">
            <img src="/logo.svg" alt="" className="size-7" />
            <div className="flex flex-col leading-tight">
              <span className="text-base font-semibold">SiTukang</span>
              <span className="text-muted-foreground text-[11px]">Rekap gaji</span>
            </div>
          </div>
          <ThemeToggle />
        </div>
      </header>

      <main className="mx-auto w-full max-w-2xl space-y-4 px-4 py-5">
        {isPending && (
          <div className="space-y-2" aria-hidden>
            <Skeleton className="h-24 w-full rounded-xl" />
            <Skeleton className="h-20 w-full rounded-xl" />
            <Skeleton className="h-20 w-full rounded-xl" />
          </div>
        )}

        {isError && (
          <div className="bg-background rounded-xl border p-6 text-center">
            <p className="font-medium">Tautan tidak ditemukan atau rekap sudah tidak berlaku.</p>
            <Button
              variant="outline"
              className="mt-4 h-9 px-5"
              onClick={() => void refetch()}
            >
              Coba lagi
            </Button>
          </div>
        )}

        {data && (
          <>
            <section className="bg-background space-y-1 rounded-xl border p-4 shadow-sm">
              <h1 className="text-lg font-semibold tracking-tight">{data.recap.name}</h1>
              <p className="text-muted-foreground text-sm">
                {data.recap.start_date && data.recap.end_date
                  ? `Periode ${formatDate(data.recap.start_date)} - ${formatDate(data.recap.end_date)}`
                  : 'Semua yang belum dibayar'}
              </p>
              <span
                className={cn(
                  'inline-block rounded-full px-2 py-0.5 text-xs',
                  data.recap.status === 'draft'
                    ? 'bg-amber-500/10 text-amber-700 dark:text-amber-300'
                    : 'bg-muted text-muted-foreground',
                )}
              >
                {data.recap.status === 'draft' ? 'Draft' : 'Rekap final'}
              </span>
              {data.recap.status === 'draft' && (
                <p className="text-xs text-amber-700 dark:text-amber-300">
                  Masih draft, nominal dapat berubah sampai rekap difinalkan.
                </p>
              )}
            </section>

            <section className="bg-background grid grid-cols-2 gap-3 rounded-xl border p-4 shadow-sm sm:grid-cols-4">
              <div>
                <p className="text-muted-foreground text-xs">Gaji kotor</p>
                <p className="font-medium">{formatRupiah(data.totals.gross)}</p>
              </div>
              <div>
                <p className="text-muted-foreground text-xs">Kas bon</p>
                <p className="font-medium">{formatRupiah(data.totals.advances)}</p>
              </div>
              <div>
                <p className="text-muted-foreground text-xs">Tambahan</p>
                <p className="font-medium">{formatRupiah(data.totals.extras)}</p>
              </div>
              <div>
                <p className="text-muted-foreground text-xs">Total dibayar</p>
                <p className="font-semibold">{formatRupiah(data.totals.net)}</p>
              </div>
            </section>

            {data.workers.length === 0 ? (
              <div className="bg-background rounded-xl border p-8 text-center">
                <p className="font-medium">Belum ada data pada rekap ini.</p>
                <p className="text-muted-foreground mt-1 text-sm">
                  Rekap ini tidak memuat pekerja yang dibayar.
                </p>
              </div>
            ) : (
              <ul className="space-y-3">
                {data.workers.map((line) => (
                  <li key={line.worker_id} className="bg-background space-y-3 rounded-xl border p-4">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="truncate font-medium">{line.name}</p>
                        <p className="text-muted-foreground truncate text-xs">{line.role}</p>
                      </div>
                      <div className="text-right">
                        <p className="text-muted-foreground text-xs">
                          {formatDays(line.quarters)} hari
                        </p>
                        <p className="font-semibold">{formatRupiah(line.net)}</p>
                      </div>
                    </div>

                    <dl className="space-y-1 border-t pt-3 text-sm">
                      <div className="flex justify-between">
                        <dt className="text-muted-foreground">Gaji kotor</dt>
                        <dd>{formatRupiah(line.gross)}</dd>
                      </div>
                      <div className="flex justify-between">
                        <dt className="text-muted-foreground">Kas bon</dt>
                        <dd>- {formatRupiah(line.advances)}</dd>
                      </div>
                      <div className="flex justify-between">
                        <dt className="text-muted-foreground">Tambahan</dt>
                        <dd>{formatRupiah(line.extras)}</dd>
                      </div>
                      <div className="flex justify-between font-medium">
                        <dt>Total diterima</dt>
                        <dd>{formatRupiah(line.net)}</dd>
                      </div>
                    </dl>
                  </li>
                ))}
              </ul>
            )}

            <p className="text-muted-foreground pb-4 text-center text-xs">
              Dibagikan lewat SiTukang. Halaman ini hanya dapat dilihat.
            </p>
          </>
        )}
      </main>
    </div>
  )
}
