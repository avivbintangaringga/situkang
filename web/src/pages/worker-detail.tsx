import { addDays, endOfMonth, format, parseISO, startOfMonth, startOfWeek } from 'date-fns'
import { ArrowLeft, Lock, Plus } from 'lucide-react'
import { useState } from 'react'
import { Link, useParams } from 'react-router'

import { CashAdvanceSheet } from '@/components/cash-advance-sheet'
import { MonthCalendar, type CalendarMarker } from '@/components/month-calendar'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useWorkerWeek } from '@/hooks/use-attendance'
import { useCashAdvances } from '@/hooks/use-cash-advances'
import { useProjects } from '@/hooks/use-projects'
import { useWorker } from '@/hooks/use-workers'
import {
  formatDate,
  formatDateShort,
  formatDays,
  formatRupiah,
  quartersLabel,
  todayISO,
} from '@/lib/format'
import type { CashAdvance } from '@/lib/types'

const quarterColors: Record<number, string> = {
  1: 'bg-amber-400',
  2: 'bg-sky-400',
  3: 'bg-violet-400',
  4: 'bg-emerald-500',
}

export function WorkerDetailPage() {
  const params = useParams()
  const workerId = Number(params.id)

  const { data: worker, isPending, isError, error, refetch } = useWorker(workerId)
  const advances = useCashAdvances(workerId)
  const projects = useProjects()

  const [advanceOpen, setAdvanceOpen] = useState(false)
  const [editingAdvance, setEditingAdvance] = useState<CashAdvance | null>(null)
  const [month, setMonth] = useState(() => todayISO().slice(0, 7))

  // oxlint-disable-next-line react/purity -- one snapshot of "this week" per mount
  const weekStart = startOfWeek(new Date(), { weekStartsOn: 1 })
  const from = format(weekStart, 'yyyy-MM-dd')
  const to = format(addDays(weekStart, 6), 'yyyy-MM-dd')
  const week = useWorkerWeek(workerId, from, to)

  const monthStart = startOfMonth(parseISO(`${month}-01`))
  const monthEnd = endOfMonth(monthStart)
  const monthFrom = format(monthStart, 'yyyy-MM-dd')
  const monthTo = format(monthEnd, 'yyyy-MM-dd')
  const monthAttendance = useWorkerWeek(workerId, monthFrom, monthTo)

  const total = (week.data ?? []).reduce((sum, entry) => sum + entry.earned, 0)
  const monthQuarters = (monthAttendance.data ?? []).reduce(
    (sum, entry) => sum + entry.quarters,
    0,
  )
  const monthEarned = (monthAttendance.data ?? []).reduce(
    (sum, entry) => sum + entry.earned,
    0,
  )

  const monthMarkers: Record<string, CalendarMarker[]> = {}
  for (const entry of monthAttendance.data ?? []) {
    monthMarkers[entry.date] = [
      {
        className: `${quarterColors[entry.quarters] ?? 'bg-muted-foreground'} size-2 rounded-full`,
        label: `${quartersLabel(entry.quarters)} · ${formatRupiah(entry.earned)}`,
      },
    ]
  }

  const advancesTotal = (advances.data ?? []).reduce(
    (sum, advance) => sum + advance.amount,
    0,
  )
  const assignedProjects = (projects.data ?? []).filter((project) =>
    project.worker_ids.includes(workerId),
  )

  function openAdvance(advance: CashAdvance | null) {
    setEditingAdvance(advance)
    setAdvanceOpen(true)
  }

  return (
    <div className="space-y-6">
      <Button variant="ghost" className="-ml-2 h-9 px-2" render={<Link to="/attendance" />}>
        <ArrowLeft className="size-4" aria-hidden />
        Absensi
      </Button>

      {isPending && <Skeleton className="h-24 w-full rounded-xl" />}

      {isError && (
        <div className="rounded-xl border p-6 text-center">
          <p className="text-sm">
            {error instanceof Error ? error.message : 'Tidak dapat memuat pekerja.'}
          </p>
          <Button
            variant="outline"
            className="mt-4 h-9 px-5"
            onClick={() => void refetch()}
          >
            Coba lagi
          </Button>
        </div>
      )}

      {worker && (
        <>
          <div>
            <h1 className="text-xl font-semibold tracking-tight">{worker.name}</h1>
            <p className="text-muted-foreground text-sm">
              {worker.role} · {formatRupiah(worker.daily_fee)}/hari
            </p>
          </div>

          <section className="space-y-2">
            <h2 className="text-sm font-semibold">Proyek</h2>

            {projects.isPending && <Skeleton className="h-16 w-full rounded-xl" />}

            {projects.isError && (
              <div className="rounded-xl border p-6 text-center">
                <p className="text-sm">Tidak dapat memuat proyek.</p>
                <Button
                  variant="outline"
                  className="mt-4 h-9 px-5"
                  onClick={() => void projects.refetch()}
                >
                  Coba lagi
                </Button>
              </div>
            )}

            {projects.data && assignedProjects.length === 0 && (
              <div className="rounded-xl border p-6 text-center">
                <p className="text-sm">Belum ditugaskan ke proyek.</p>
              </div>
            )}

            {assignedProjects.length > 0 && (
              <ul className="divide-y rounded-xl border">
                {assignedProjects.map((project) => (
                  <li key={project.id} className="px-3 py-2.5 text-sm">
                    {project.name}
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section className="space-y-2">
            <div className="flex items-baseline justify-between">
              <h2 className="text-sm font-semibold">Absensi minggu ini</h2>
              <span className="text-sm font-medium">{formatRupiah(total)}</span>
            </div>

            {week.isPending && <Skeleton className="h-40 w-full rounded-xl" />}

            {week.isError && (
              <div className="rounded-xl border p-6 text-center text-sm">
                Tidak dapat memuat absensi minggu ini.
              </div>
            )}

            {week.data && (
              <ul className="divide-y rounded-xl border">
                {Array.from({ length: 7 }, (_, index) => {
                  const iso = format(addDays(weekStart, index), 'yyyy-MM-dd')
                  const entry = week.data.find((item) => item.date === iso)

                  return (
                    <li
                      key={iso}
                      className="flex items-center justify-between gap-3 px-3 py-2.5"
                    >
                      <div className="min-w-0">
                        <p className="truncate text-sm">{formatDate(iso)}</p>
                        <p className="text-muted-foreground flex items-center gap-1 text-xs">
                          {entry ? quartersLabel(entry.quarters) : 'Tidak hadir'}
                          {entry?.locked && (
                            <>
                              <Lock className="size-3" aria-hidden />
                              terkunci
                            </>
                          )}
                        </p>
                      </div>
                      <span className="shrink-0 text-sm font-medium">
                        {entry ? formatRupiah(entry.earned) : '-'}
                      </span>
                    </li>
                  )
                })}
              </ul>
            )}
          </section>

          <section className="space-y-2">
            <div className="flex items-baseline justify-between gap-3">
              <h2 className="text-sm font-semibold">Absensi bulanan</h2>
              <span className="text-muted-foreground text-xs">
                {formatDays(monthQuarters)} hari · {formatRupiah(monthEarned)}
              </span>
            </div>

            {monthAttendance.isPending && <Skeleton className="h-72 w-full rounded-xl" />}

            {monthAttendance.isError && (
              <div className="rounded-xl border p-6 text-center text-sm">
                Tidak dapat memuat absensi bulan ini.
              </div>
            )}

            {!monthAttendance.isPending && !monthAttendance.isError && (
              <>
                <MonthCalendar
                  month={month}
                  onMonthChange={setMonth}
                  markers={monthMarkers}
                />
                <p className="text-muted-foreground flex flex-wrap items-center gap-3 text-xs">
                  <span className="flex items-center gap-1">
                    <span className="size-2 rounded-full bg-amber-400" aria-hidden />
                    ¼ hari
                  </span>
                  <span className="flex items-center gap-1">
                    <span className="size-2 rounded-full bg-sky-400" aria-hidden />
                    ½ hari
                  </span>
                  <span className="flex items-center gap-1">
                    <span className="size-2 rounded-full bg-violet-400" aria-hidden />
                    ¾ hari
                  </span>
                  <span className="flex items-center gap-1">
                    <span className="size-2 rounded-full bg-emerald-500" aria-hidden />
                    Penuh
                  </span>
                </p>
              </>
            )}
          </section>

          <section className="space-y-2">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-sm font-semibold">Kas bon</h2>
                <p className="text-muted-foreground text-xs">
                  Total {formatRupiah(advancesTotal)}
                </p>
              </div>
              <Button className="h-9 px-3" onClick={() => openAdvance(null)}>
                <Plus className="size-4" aria-hidden />
                Tambah
              </Button>
            </div>

            {advances.isPending && <Skeleton className="h-24 w-full rounded-xl" />}

            {advances.isError && (
              <div className="rounded-xl border p-6 text-center text-sm">
                Tidak dapat memuat kas bon.
              </div>
            )}

            {advances.data && advances.data.length === 0 && (
              <div className="rounded-xl border p-6 text-center">
                <p className="text-sm">Belum ada kas bon.</p>
              </div>
            )}

            {advances.data && advances.data.length > 0 && (
              <ul className="divide-y rounded-xl border">
                {advances.data.map((advance) => (
                  <li key={advance.id}>
                    <button
                      type="button"
                      onClick={() => openAdvance(advance)}
                      className="hover:bg-muted/50 flex w-full items-center justify-between gap-3 px-3 py-3 text-left"
                    >
                      <div className="min-w-0">
                        <p className="truncate text-sm">
                          {formatDateShort(advance.date)}
                          {advance.note ? ` · ${advance.note}` : ''}
                        </p>
                        {advance.locked && (
                          <p className="text-muted-foreground flex items-center gap-1 text-xs">
                            <Lock className="size-3" aria-hidden />
                            Terkunci rekap
                          </p>
                        )}
                      </div>
                      <span className="shrink-0 text-sm font-medium">
                        {formatRupiah(advance.amount)}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <CashAdvanceSheet
            open={advanceOpen}
            onOpenChange={setAdvanceOpen}
            workerId={workerId}
            advance={editingAdvance}
          />
        </>
      )}
    </div>
  )
}
