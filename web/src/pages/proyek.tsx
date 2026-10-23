import { addMonths, format, getDaysInMonth, parseISO } from 'date-fns'
import { ChevronLeft, ChevronRight, Plus } from 'lucide-react'
import { useState } from 'react'

import { ProjectFormSheet } from '@/components/project-form-sheet'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useProjectTimeline, useProjects } from '@/hooks/use-projects'
import { useWorkers } from '@/hooks/use-workers'
import { formatDateShort } from '@/lib/format'
import type { Project, ProjectStatus, TimelineProject } from '@/lib/types'
import { cn } from '@/lib/utils'

const statusLabel: Record<ProjectStatus, string> = {
  aktif: 'Aktif',
  selesai: 'Selesai',
  ditunda: 'Ditunda',
}

function currentMonth(): string {
  return format(new Date(), 'yyyy-MM')
}

function dayISO(month: string, day: number): string {
  return `${month}-${String(day).padStart(2, '0')}`
}

function TimelineRow({ project, month, days }: { project: TimelineProject; month: string; days: number }) {
  const tagged = new Set(project.tagged_days)
  const cells = Array.from({ length: days }, (_, index) => {
    const day = index + 1
    const iso = dayISO(month, day)
    const inSpan =
      (!project.start_date || iso >= project.start_date) &&
      (!project.end_date || iso <= project.end_date)
    return { day, iso, inSpan, isTagged: tagged.has(iso) }
  })

  return (
    <li className="space-y-1.5 rounded-xl border p-3">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="truncate text-sm font-medium">{project.name}</p>
          <p className="text-muted-foreground text-xs">
            {project.start_date && project.end_date
              ? `${formatDateShort(project.start_date)} - ${formatDateShort(project.end_date)}`
              : project.start_date
                ? `Mulai ${formatDateShort(project.start_date)}`
                : project.end_date
                  ? `Selesai ${formatDateShort(project.end_date)}`
                  : 'Tanpa tanggal'}
          </p>
        </div>
        <Badge variant="secondary" className="shrink-0 text-xs">
          {statusLabel[project.status]}
        </Badge>
      </div>

      <div className="flex gap-1 overflow-hidden">
        <div
          role="img"
          aria-label={`Linimasa ${project.name}`}
          className="grid w-full gap-px"
          style={{ gridTemplateColumns: `repeat(${days}, minmax(0, 1fr))` }}
        >
          {cells.map((cell) => (
            <span
              key={cell.iso}
              title={`${cell.day}${cell.isTagged ? ' · tukang hadir' : ''}`}
              className={cn(
                'h-5 rounded-[2px]',
                cell.isTagged
                  ? 'bg-primary'
                  : cell.inSpan
                    ? 'bg-primary/25'
                    : 'bg-muted',
              )}
            />
          ))}
        </div>
      </div>
    </li>
  )
}

export function ProyekPage() {
  const [month, setMonth] = useState(currentMonth)
  const projects = useProjects()
  const timeline = useProjectTimeline(month)
  const workers = useWorkers('', true)

  const workerNames = new Map((workers.data ?? []).map((worker) => [worker.id, worker.name]))

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<Project | null>(null)

  function openCreate() {
    setEditing(null)
    setFormOpen(true)
  }

  function openEdit(project: Project) {
    setEditing(project)
    setFormOpen(true)
  }

  function shiftMonth(offset: number) {
    setMonth(format(addMonths(parseISO(`${month}-01`), offset), 'yyyy-MM'))
  }

  const days = timeline.data?.days ?? getDaysInMonth(parseISO(`${month}-01`))
  const monthLabel = format(parseISO(`${month}-01`), 'MMMM yyyy')

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Proyek</h1>
          <p className="text-muted-foreground text-sm">
            Daftar proyek dan linimasa kehadiran tukang.
          </p>
        </div>
        <Button className="h-11 shrink-0 px-3" onClick={openCreate}>
          <Plus className="size-4" aria-hidden />
          Tambah
        </Button>
      </div>

      <section className="space-y-3">
        <div className="flex items-center justify-between gap-2">
          <h2 className="text-sm font-semibold">Linimasa</h2>
          <div className="flex items-center gap-1">
            <Button
              variant="outline"
              className="size-11 p-0"
              onClick={() => shiftMonth(-1)}
              aria-label="Bulan sebelumnya"
            >
              <ChevronLeft className="size-5" aria-hidden />
            </Button>
            <span className="min-w-[7.5rem] text-center text-sm font-medium capitalize">
              {monthLabel}
            </span>
            <Button
              variant="outline"
              className="size-11 p-0"
              onClick={() => shiftMonth(1)}
              aria-label="Bulan berikutnya"
            >
              <ChevronRight className="size-5" aria-hidden />
            </Button>
          </div>
        </div>

        {timeline.isPending && <Skeleton className="h-40 w-full rounded-xl" />}

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
            <div
              className="text-muted-foreground grid gap-px px-3 text-center text-[9px] leading-4"
              style={{ gridTemplateColumns: `repeat(${days}, minmax(0, 1fr))` }}
              aria-hidden
            >
              {Array.from({ length: days }, (_, index) => (
                <span key={index}>{index + 1}</span>
              ))}
            </div>

            {timeline.data.projects.length === 0 ? (
              <div className="rounded-xl border p-6 text-center">
                <p className="text-sm">Belum ada proyek pada bulan ini.</p>
                <p className="text-muted-foreground mt-1 text-xs">
                  Proyek yang punya tanggal atau punya tukang dengan absensi akan
                  tampil di sini.
                </p>
              </div>
            ) : (
              <ul className="space-y-2">
                {timeline.data.projects.map((project) => (
                  <TimelineRow
                    key={project.id}
                    project={project}
                    month={month}
                    days={days}
                  />
                ))}
              </ul>
            )}
            <p className="text-muted-foreground flex flex-wrap items-center gap-3 text-xs">
              <span className="flex items-center gap-1">
                <span className="bg-primary/25 size-3 rounded-[2px]" aria-hidden />
                Rentang proyek
              </span>
              <span className="flex items-center gap-1">
                <span className="bg-primary size-3 rounded-[2px]" aria-hidden />
                Hari tercatat
              </span>
            </p>
            <p className="text-muted-foreground text-xs">
              Hari tercatat adalah hari saat tukang yang ditugaskan hadir.
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
            <Button className="mt-4 h-11 px-5" onClick={openCreate}>
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
                  <button
                    type="button"
                    onClick={() => openEdit(project)}
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
                  </button>
                </li>
              )
            })}
          </ul>
        )}
      </section>

      <ProjectFormSheet open={formOpen} onOpenChange={setFormOpen} project={editing} />
    </div>
  )
}
