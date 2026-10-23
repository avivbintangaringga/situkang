import { format } from 'date-fns'
import { Plus } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router'

import { MonthCalendar, type CalendarMarker } from '@/components/month-calendar'
import { ProjectFormSheet } from '@/components/project-form-sheet'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useProjectTimeline, useProjects } from '@/hooks/use-projects'
import { useWorkers } from '@/hooks/use-workers'
import { formatDateShort } from '@/lib/format'
import type { ProjectStatus } from '@/lib/types'
import { cn } from '@/lib/utils'

const statusLabel: Record<ProjectStatus, string> = {
  aktif: 'Aktif',
  selesai: 'Selesai',
  ditunda: 'Ditunda',
}

const projectColors = [
  'bg-sky-500',
  'bg-rose-500',
  'bg-amber-500',
  'bg-violet-500',
  'bg-emerald-500',
  'bg-cyan-500',
  'bg-orange-500',
  'bg-pink-500',
  'bg-indigo-500',
  'bg-teal-500',
]

function currentMonth(): string {
  return format(new Date(), 'yyyy-MM')
}

export function ProyekPage() {
  const [month, setMonth] = useState(currentMonth)
  const projects = useProjects()
  const timeline = useProjectTimeline(month)
  const workers = useWorkers('', true)

  const workerNames = new Map((workers.data ?? []).map((worker) => [worker.id, worker.name]))

  const [formOpen, setFormOpen] = useState(false)

  const colorByProject = new Map<number, string>()
  const markers: Record<string, CalendarMarker[]> = {}

  for (const [index, project] of (timeline.data?.projects ?? []).entries()) {
    const color = projectColors[index % projectColors.length]
    colorByProject.set(project.id, color)
    for (const date of project.tagged_days) {
      const list = markers[date] ?? []
      list.push({ className: cn('size-1.5 rounded-full', color), label: project.name })
      markers[date] = list
    }
  }

  for (const [date, list] of Object.entries(markers)) {
    if (list.length <= 3) continue
    const hidden = list.length - 3
    markers[date] = [
      ...list.slice(0, 3),
      {
        className: 'text-[8px] leading-none font-medium text-muted-foreground',
        label: `+${hidden} proyek lain`,
      },
    ]
  }

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Proyek</h1>
          <p className="text-muted-foreground text-sm">
            Daftar proyek dan linimasa kehadiran tukang.
          </p>
        </div>
        <Button className="h-11 shrink-0 px-3" onClick={() => setFormOpen(true)}>
          <Plus className="size-4" aria-hidden />
          Tambah
        </Button>
      </div>

      <section className="space-y-3">
        <h2 className="text-sm font-semibold">Linimasa</h2>

        {timeline.isPending && <Skeleton className="h-72 w-full rounded-xl" />}

        {timeline.isError && (
          <div className="rounded-xl border p-6 text-center">
            <p className="text-sm">
              {timeline.error instanceof Error
                ? timeline.error.message
                : 'Tidak dapat memuat linimasa.'}
            </p>
            <Button
              variant="outline"
              className="mt-4 h-11 px-5"
              onClick={() => void timeline.refetch()}
            >
              Coba lagi
            </Button>
          </div>
        )}

        {!timeline.isPending && !timeline.isError && timeline.data && (
          <>
            <MonthCalendar month={month} onMonthChange={setMonth} markers={markers} />

            {timeline.data.projects.length === 0 ? (
              <div className="rounded-xl border p-6 text-center">
                <p className="text-sm">Belum ada proyek pada bulan ini.</p>
                <p className="text-muted-foreground mt-1 text-xs">
                  Proyek yang punya tanggal atau punya tukang dengan absensi akan
                  tampil di sini.
                </p>
              </div>
            ) : (
              <ul className="grid grid-cols-1 gap-1.5 sm:grid-cols-2">
                {timeline.data.projects.map((project) => (
                  <li key={project.id} className="flex items-center gap-2 text-xs">
                    <span
                      className={cn(
                        'size-2.5 shrink-0 rounded-full',
                        colorByProject.get(project.id),
                      )}
                      aria-hidden
                    />
                    <span className="truncate">{project.name}</span>
                  </li>
                ))}
              </ul>
            )}

            <p className="text-muted-foreground text-xs">
              Titik menandai hari saat tukang yang ditugaskan hadir. Maksimal tiga
              titik per hari.
            </p>
          </>
        )}
      </section>

      <section className="space-y-2">
        <h2 className="text-sm font-semibold">Semua proyek</h2>

        {projects.isPending && (
          <div className="space-y-2" aria-hidden>
            <Skeleton className="h-20 w-full rounded-xl" />
            <Skeleton className="h-20 w-full rounded-xl" />
          </div>
        )}

        {projects.isError && (
          <div className="rounded-xl border p-6 text-center">
            <p className="text-sm">
              {projects.error instanceof Error
                ? projects.error.message
                : 'Tidak dapat memuat proyek.'}
            </p>
            <Button
              variant="outline"
              className="mt-4 h-11 px-5"
              onClick={() => void projects.refetch()}
            >
              Coba lagi
            </Button>
          </div>
        )}

        {!projects.isPending && !projects.isError && projects.data && projects.data.length === 0 && (
          <div className="rounded-xl border p-8 text-center">
            <p className="font-medium">Belum ada proyek.</p>
            <p className="text-muted-foreground mt-1 text-sm">
              Tambahkan proyek untuk menugaskan tukang ke tempat kerja.
            </p>
            <Button className="mt-4 h-11 px-5" onClick={() => setFormOpen(true)}>
              Tambah proyek
            </Button>
          </div>
        )}

        {!projects.isPending && !projects.isError && projects.data && projects.data.length > 0 && (
          <ul className="space-y-2">
            {projects.data.map((project) => {
              const assignedNames = project.worker_ids
                .map((id) => workerNames.get(id))
                .filter((name): name is string => Boolean(name))
              const workerLabel =
                project.worker_ids.length === 0
                  ? 'Belum ada tukang'
                  : assignedNames.length === project.worker_ids.length
                    ? assignedNames.join(' · ')
                    : `${project.worker_ids.length} tukang`

              return (
                <li key={project.id}>
                  <Link
                    to={`/proyek/${project.id}`}
                    className="bg-card hover:bg-muted/50 flex w-full items-center justify-between gap-3 rounded-xl border p-4 text-left transition-colors"
                  >
                    <div className="min-w-0 space-y-1">
                      <p className="truncate font-medium">{project.name}</p>
                      <p className="text-muted-foreground truncate text-xs">
                        {project.start_date && project.end_date
                          ? `${formatDateShort(project.start_date)} - ${formatDateShort(project.end_date)}`
                          : 'Tanggal belum diisi'}
                        {project.notes ? ` · ${project.notes}` : ''}
                      </p>
                      <p className="text-muted-foreground truncate text-xs">
                        Tukang: {workerLabel}
                      </p>
                    </div>
                    <Badge variant="secondary" className="shrink-0 text-xs">
                      {statusLabel[project.status]}
                    </Badge>
                  </Link>
                </li>
              )
            })}
          </ul>
        )}
      </section>

      <ProjectFormSheet open={formOpen} onOpenChange={setFormOpen} project={null} />
    </div>
  )
}
