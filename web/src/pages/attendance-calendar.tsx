import { format } from 'date-fns'
import { ArrowLeft } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router'

import { MonthCalendar, type CalendarMarker } from '@/components/month-calendar'
import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { useAttendanceCalendar } from '@/hooks/use-attendance'
import { formatDate, formatRupiah, quartersLabel } from '@/lib/format'
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

export function AttendanceCalendarPage() {
  const [month, setMonth] = useState(currentMonth)
  const [selectedDate, setSelectedDate] = useState<string | null>(null)
  const calendar = useAttendanceCalendar(month)

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

  const selectedDay = calendar.data?.days.find((day) => day.date === selectedDate)
  const selectedEarned = (selectedDay?.workers ?? []).reduce(
    (sum, worker) => sum + worker.earned,
    0,
  )

  return (
    <div className="space-y-4">
      <Button variant="ghost" className="-ml-2 h-11 px-2" render={<Link to="/attendance" />}>
        <ArrowLeft className="size-4" aria-hidden />
        Absensi
      </Button>

      <div>
        <h1 className="text-xl font-semibold tracking-tight">Kalender absensi</h1>
        <p className="text-muted-foreground text-sm">
          Lihat hari yang sudah tercatat dan tukang yang hadir.
        </p>
      </div>

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
            onSelectDate={setSelectedDate}
          />

          <div className="bg-muted/50 grid grid-cols-3 gap-2 rounded-xl px-3 py-3 text-center">
            <div>
              <p className="text-lg font-semibold">{calendar.data.totals.days}</p>
              <p className="text-muted-foreground text-[11px]">Hari tercatat</p>
            </div>
            <div>
              <p className="text-lg font-semibold">{calendar.data.totals.workers}</p>
              <p className="text-muted-foreground text-[11px]">Tukang-hari</p>
            </div>
            <div>
              <p className="truncate text-lg font-semibold">
                {formatRupiah(calendar.data.totals.earned)}
              </p>
              <p className="text-muted-foreground text-[11px]">Total upah</p>
            </div>
          </div>

          {calendar.data.days.length === 0 && (
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

      <Sheet
        open={Boolean(selectedDate)}
        onOpenChange={(open) => {
          if (!open) setSelectedDate(null)
        }}
      >
        <SheetContent side="bottom" className="max-h-[80dvh] overflow-y-auto">
          <SheetHeader>
            <SheetTitle>{selectedDate ? formatDate(selectedDate) : ''}</SheetTitle>
            <SheetDescription>
              {selectedDay
                ? `Total upah hari ini ${formatRupiah(selectedEarned)}`
                : 'Belum ada absensi pada tanggal ini.'}
            </SheetDescription>
          </SheetHeader>

          <div className="px-4 pb-4">
            {!selectedDay || selectedDay.workers.length === 0 ? (
              <p className="text-muted-foreground rounded-xl border p-6 text-center text-sm">
                Tidak ada tukang yang hadir pada tanggal ini.
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
                      <div className="flex items-center justify-between gap-3">
                        <span className="text-muted-foreground min-w-0 truncate text-xs">
                          {projects}
                        </span>
                        <span className="shrink-0 text-xs font-medium">
                          {formatRupiah(worker.earned)}
                        </span>
                      </div>
                    </li>
                  )
                })}
              </ul>
            )}
          </div>
        </SheetContent>
      </Sheet>
    </div>
  )
}
