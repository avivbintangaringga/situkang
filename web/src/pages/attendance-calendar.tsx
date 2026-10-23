import { format } from 'date-fns'
import { ArrowLeft, X } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router'

import { MonthCalendar, type CalendarMarker } from '@/components/month-calendar'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useAttendanceCalendar } from '@/hooks/use-attendance'
import { formatDate, quartersLabel } from '@/lib/format'
import { cn } from '@/lib/utils'

const quarterColors: Record<number, string> = {
  1: 'bg-amber-400',
  2: 'bg-sky-400',
  3: 'bg-violet-400',
  4: 'bg-emerald-500',
}

function currentMonth(): string {
  return format(new Date(), 'yyyy-MM')
}

function today(): string {
  return format(new Date(), 'yyyy-MM-dd')
}

export function AttendanceCalendarPage() {
  const [month, setMonth] = useState(currentMonth)
  const [selectionStart, setSelectionStart] = useState<string | null>(today)
  const [selectionEnd, setSelectionEnd] = useState<string | null>(null)

  const isRange =
    selectionStart !== null &&
    selectionEnd !== null &&
    selectionStart !== selectionEnd
  const queryRange =
    selectionStart && selectionEnd && selectionStart !== selectionEnd
      ? { from: selectionStart, to: selectionEnd }
      : null
  const calendar = useAttendanceCalendar(month, queryRange)

  const markers: Record<string, CalendarMarker[]> = {}
  for (const day of calendar.data?.days ?? []) {
    const list = day.workers.map((worker) => ({
      className: cn('size-1.5 rounded-full', quarterColors[worker.quarters] ?? 'bg-muted-foreground'),
      label: `${worker.name} (${quartersLabel(worker.quarters)})`,
    }))
    if (list.length > 3) {
      markers[day.date] = [
        ...list.slice(0, 3),
        {
          className: 'text-[8px] leading-none font-medium text-muted-foreground',
          label: `+${list.length - 3} tukang lain`,
        },
      ]
    } else {
      markers[day.date] = list
    }
  }

  const selectedDay = calendar.data?.days.find((day) => day.date === selectionStart)

  const rangeWorkers = new Map<number, { name: string; count: number }>()
  if (isRange) {
    for (const day of calendar.data?.days ?? []) {
      for (const worker of day.workers) {
        const current = rangeWorkers.get(worker.worker_id)
        if (current) {
          current.count += 1
        } else {
          rangeWorkers.set(worker.worker_id, { name: worker.name, count: 1 })
        }
      }
    }
  }
  const rangeWorkerList = [...rangeWorkers.entries()]
    .map(([workerId, worker]) => ({ workerId, ...worker }))
    .sort((a, b) => a.name.localeCompare(b.name, 'id'))

  const hasSelection = selectionStart !== null
  const selectionLabel =
    isRange && selectionStart && selectionEnd
      ? `${formatDate(selectionStart)} - ${formatDate(selectionEnd)}`
      : selectionStart
        ? formatDate(selectionStart)
        : null

  function selectDate(date: string) {
    if (!selectionStart || selectionEnd) {
      setSelectionStart(date)
      setSelectionEnd(null)
      return
    }

    if (date < selectionStart) {
      setSelectionStart(date)
      setSelectionEnd(selectionStart)
      return
    }

    setSelectionEnd(date)
  }

  function clearSelection() {
    setSelectionStart(null)
    setSelectionEnd(null)
  }

  return (
    <div className="space-y-4">
      <Button variant="ghost" className="-ml-2 h-11 px-2" render={<Link to="/attendance" />}>
        <ArrowLeft className="size-4" aria-hidden />
        Absensi
      </Button>

      <h1 className="text-xl font-semibold tracking-tight">Kalender absensi</h1>

      {hasSelection && (
        <div className="flex items-center justify-between gap-2 rounded-full border bg-muted/50 py-1 pl-3 pr-1">
          <p className="min-w-0 truncate text-xs font-medium">{selectionLabel}</p>
          <Button
            type="button"
            variant="outline"
            className="h-8 shrink-0 gap-1 rounded-full px-3 text-xs"
            onClick={clearSelection}
          >
            <X className="size-3.5" aria-hidden />
            Hapus pilihan
          </Button>
        </div>
      )}

      {calendar.isPending && (
        <div className="space-y-2" aria-hidden>
          <Skeleton className="h-64 w-full rounded-xl" />
          <Skeleton className="h-16 w-full rounded-xl" />
        </div>
      )}

      {calendar.isError && (
        <div className="rounded-xl border p-6 text-center">
          <p className="text-sm">
            {calendar.error instanceof Error
              ? calendar.error.message
              : 'Tidak dapat memuat kalender absensi.'}
          </p>
          <Button
            variant="outline"
            className="mt-4 h-9 px-5"
            onClick={() => void calendar.refetch()}
          >
            Coba lagi
          </Button>
        </div>
      )}

      {!calendar.isPending && !calendar.isError && calendar.data && (
        <>
          <MonthCalendar
            month={month}
            onMonthChange={setMonth}
            markers={markers}
            selection={isRange ? { start: selectionStart, end: selectionEnd } : undefined}
            selectedDate={isRange ? null : selectionStart}
            onSelectDate={selectDate}
          />

          {hasSelection && (
            <section className="space-y-2" aria-live="polite">

              {isRange ? (
                rangeWorkerList.length === 0 ? (
                  <p className="text-muted-foreground rounded-xl border p-6 text-center text-sm">
                    Tidak ada absensi pada rentang tanggal ini.
                  </p>
                ) : (
                  <ul className="divide-y rounded-xl border">
                    {rangeWorkerList.map((worker) => (
                      <li key={worker.workerId} className="space-y-0.5 px-3 py-2.5">
                        <div className="flex items-center justify-between gap-3">
                          <span className="truncate text-sm font-medium">{worker.name}</span>
                          <span className="text-muted-foreground shrink-0 text-xs">
                            {worker.count} kali hadir
                          </span>
                        </div>
                      </li>
                    ))}
                  </ul>
                )
              ) : !selectedDay || selectedDay.workers.length === 0 ? (
                <p className="text-muted-foreground rounded-xl border p-6 text-center text-sm">
                  Tidak ada absensi pada tanggal ini.
                </p>
              ) : (
                <ul className="divide-y rounded-xl border">
                  {selectedDay.workers.map((worker) => {
                    const projects =
                      worker.projects.length > 0
                        ? worker.projects.map((project) => project.name).join(' · ')
                        : 'Belum ada proyek'
                    return (
                      <li key={worker.worker_id} className="space-y-0.5 px-3 py-2.5">
                        <div className="flex items-center justify-between gap-3">
                          <span className="truncate text-sm font-medium">{worker.name}</span>
                          <span className="text-muted-foreground shrink-0 text-xs">
                            {quartersLabel(worker.quarters)}
                          </span>
                        </div>
                        <span className="text-muted-foreground block min-w-0 truncate text-xs">
                          {projects}
                        </span>
                      </li>
                    )
                  })}
                </ul>
              )}
            </section>
          )}

          {!hasSelection && calendar.data.days.length === 0 && (
            <div className="rounded-xl border p-8 text-center">
              <p className="font-medium">Belum ada absensi pada bulan ini.</p>
              <p className="text-muted-foreground mt-1 text-sm">
                Catat absensi dulu, lalu kembali ke kalender ini.
              </p>
              <Button className="mt-4 h-9 px-5" render={<Link to="/attendance" />}>
                Buka absensi
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
