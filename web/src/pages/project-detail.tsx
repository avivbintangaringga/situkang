import { format } from 'date-fns'
import { ArrowLeft, Pencil } from 'lucide-react'
import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router'

import { MonthCalendar, type CalendarMarker } from '@/components/month-calendar'
import { ProjectFormSheet } from '@/components/project-form-sheet'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useProject, useProjectCalendar } from '@/hooks/use-projects'
import { useWorkers } from '@/hooks/use-workers'
import { formatDate, quartersLabel } from '@/lib/format'
import type { ProjectStatus } from '@/lib/types'
import { cn } from '@/lib/utils'

const statusLabel: Record<ProjectStatus, string> = {
  aktif: 'Aktif',
  selesai: 'Selesai',
  ditunda: 'Ditunda',
}

const quarterColors: Record<number, string> = {
  1: 'bg-amber-400',
  2: 'bg-sky-400',
  3: 'bg-violet-400',
  4: 'bg-emerald-500',
}

function currentMonth(): string {
  return format(new Date(), 'yyyy-MM')
}

export function ProjectDetailPage() {
  const params = useParams()
  const navigate = useNavigate()
  const projectId = Number(params.id)

  const project = useProject(projectId)
  const workers = useWorkers('', true)
  const [month, setMonth] = useState(currentMonth)
  const calendar = useProjectCalendar(projectId, month)
  const [formOpen, setFormOpen] = useState(false)

  const workerByID = new Map((workers.data ?? []).map((worker) => [worker.id, worker]))

  const markers: Record<string, CalendarMarker[]> = {}
  for (const day of calendar.data?.days ?? []) {
    const list = day.workers.map((worker) => {
      const color = quarterColors[worker.quarters] ?? 'bg-muted-foreground'
      return {
        className: cn('size-1.5 rounded-full', color),
        label: `${worker.name} (${quartersLabel(worker.quarters)})`,
      }
    })
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

  if (project.isPending) {
    return <Skeleton className="h-64 w-full rounded-xl" />
  }

  if (project.isError) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" className="-ml-2 h-11 px-2" render={<Link to="/proyek" />}>
          <ArrowLeft className="size-4" aria-hidden />
          Proyek
        </Button>
        <div className="rounded-xl border p-6 text-center">
          <p className="text-sm">
            {project.error instanceof Error
              ? project.error.message
              : 'Tidak dapat memuat proyek.'}
          </p>
          <Button
            variant="outline"
            className="mt-4 h-11 px-5"
            onClick={() => void project.refetch()}
          >
            Coba lagi
          </Button>
        </div>
      </div>
    )
  }

  if (!project.data) return null

  const data = project.data

  return (
    <div className="space-y-4">
      <Button variant="ghost" className="-ml-2 h-11 px-2" render={<Link to="/proyek" />}>
        <ArrowLeft className="size-4" aria-hidden />
        Proyek
      </Button>

      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <h1 className="truncate text-xl font-semibold tracking-tight">{data.name}</h1>
            <Badge variant="secondary" className="shrink-0 text-xs">
              {statusLabel[data.status]}
            </Badge>
          </div>
          <p className="text-muted-foreground mt-1 text-sm">
            {data.start_date && data.end_date
              ? `${formatDate(data.start_date)} - ${formatDate(data.end_date)}`
              : data.start_date
                ? `Mulai ${formatDate(data.start_date)}`
                : data.end_date
                  ? `Selesai ${formatDate(data.end_date)}`
                  : 'Tanggal belum diisi'}
          </p>
        </div>
        <Button
          variant="outline"
          className="h-11 shrink-0 px-3"
          onClick={() => setFormOpen(true)}
        >
          <Pencil className="size-4" aria-hidden />
          Ubah
        </Button>
      </div>

      {data.notes && (
        <section className="bg-muted/40 rounded-xl border p-4">
          <h2 className="text-sm font-semibold">Catatan</h2>
          <p className="text-muted-foreground mt-1 text-sm whitespace-pre-line">
            {data.notes}
          </p>
        </section>
      )}

      <section className="space-y-2">
        <div className="flex items-baseline justify-between gap-3">
          <h2 className="text-sm font-semibold">Tukang</h2>
          <span className="text-muted-foreground text-xs">
            {data.worker_ids.length} ditugaskan
          </span>
        </div>

        {workers.isPending && <Skeleton className="h-20 w-full rounded-xl" />}

        {workers.isError && (
          <div className="rounded-xl border p-6 text-center">
            <p className="text-sm">Tidak dapat memuat daftar tukang.</p>
            <Button
              variant="outline"
              className="mt-4 h-11 px-5"
              onClick={() => void workers.refetch()}
            >
              Coba lagi
            </Button>
          </div>
        )}

        {!workers.isPending && !workers.isError && data.worker_ids.length === 0 && (
          <div className="rounded-xl border p-6 text-center">
            <p className="text-sm">Belum ada tukang yang ditugaskan.</p>
            <p className="text-muted-foreground mt-1 text-xs">
              Ubah proyek untuk memilih tukang.
            </p>
          </div>
        )}

        {data.worker_ids.length > 0 && (
          <ul className="divide-y rounded-xl border">
            {data.worker_ids.map((id) => {
              const worker = workerByID.get(id)
              return (
                <li key={id} className="flex items-center justify-between gap-3 px-3 py-3">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">
                      {worker?.name ?? `Tukang #${id}`}
                    </p>
                    <p className="text-muted-foreground truncate text-xs">
                      {worker?.role ?? '-'}
                    </p>
                  </div>
                </li>
              )
            })}
          </ul>
        )}
      </section>

      <section className="space-y-2">
        <div className="flex items-baseline justify-between gap-3">
          <h2 className="text-sm font-semibold">Kalender kehadiran</h2>
          <span className="text-muted-foreground text-xs">
            {calendar.data?.days.length ?? 0} hari tercatat
          </span>
        </div>

        {calendar.isPending && <Skeleton className="h-72 w-full rounded-xl" />}

        {calendar.isError && (
          <div className="rounded-xl border p-6 text-center">
            <p className="text-sm">
              {calendar.error instanceof Error
                ? calendar.error.message
                : 'Tidak dapat memuat kalender proyek.'}
            </p>
            <Button
              variant="outline"
              className="mt-4 h-11 px-5"
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
              range={{ start: data.start_date, end: data.end_date }}
            />
            <p className="text-muted-foreground text-xs">
              Titik adalah tukang yang ditugaskan dan hadir pada hari itu.
              Penyorotan menunjukkan rentang tanggal proyek.
            </p>
          </>
        )}
      </section>

      <ProjectFormSheet
        open={formOpen}
        onOpenChange={setFormOpen}
        project={data}
        onDeleted={() => navigate('/proyek')}
      />
    </div>
  )
}
