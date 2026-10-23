import { addDays, format, parseISO } from 'date-fns'
import { ChevronLeft, ChevronRight, Lock, Tag } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Link } from 'react-router'

import { AttendanceProjectSheet } from '@/components/attendance-project-sheet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { useAttendanceDay, useSaveDay } from '@/hooks/use-attendance'
import { useProjects } from '@/hooks/use-projects'
import { formatDate, formatRupiah, todayISO } from '@/lib/format'
import { cn } from '@/lib/utils'

const quarterOptions = [
  { value: 0, label: '—', aria: 'Absen' },
  { value: 1, label: '¼', aria: 'Seperempat hari' },
  { value: 2, label: '½', aria: 'Setengah hari' },
  { value: 3, label: '¾', aria: 'Tiga per empat hari' },
  { value: 4, label: '1', aria: 'Sehari penuh' },
]

export function AbsensiPage() {
  const [date, setDate] = useState<string | null>(null)
  const { data, isPending, isError, error, refetch } = useAttendanceDay(date)
  const { mutate: saveDay, isPending: saving, error: saveError } = useSaveDay()
  const projects = useProjects()

  const [draft, setDraft] = useState<Record<number, number>>({})
  const [projectDraft, setProjectDraft] = useState<Record<number, number[]>>({})
  const [dirty, setDirty] = useState(false)
  const [projectTarget, setProjectTarget] = useState<{ workerId: number; name: string } | null>(
    null,
  )

  const currentDate = data?.date ?? date ?? ''

  useEffect(() => {
    if (!data || dirty) return
    // oxlint-disable-next-line react/set-state-in-effect -- sync the editable draft with server data
    setDraft(
      Object.fromEntries(data.workers.map((worker) => [worker.worker_id, worker.quarters])),
    )
    // oxlint-disable-next-line react/set-state-in-effect -- sync the editable tags with server data
    setProjectDraft(
      Object.fromEntries(
        data.workers.map((worker) => [
          worker.worker_id,
          worker.projects.map((project) => project.id),
        ]),
      ),
    )
  }, [data, dirty])

  useEffect(() => {
    if (!dirty || !currentDate) return

    const timer = setTimeout(() => {
      const items = Object.entries(draft)
        .filter(([, quarters]) => quarters > 0)
        .map(([id, quarters]) => ({
          worker_id: Number(id),
          quarters,
          project_ids: projectDraft[Number(id)] ?? [],
        }))

      saveDay(
        { date: currentDate, items },
        { onSuccess: () => setDirty(false) },
      )
    }, 600)

    return () => clearTimeout(timer)
  }, [draft, projectDraft, dirty, currentDate, saveDay])

  function setQuarters(workerId: number, quarters: number) {
    setDraft((previous) => ({ ...previous, [workerId]: quarters }))
    setDirty(true)
  }

  function saveProjects(workerId: number, projectIds: number[]) {
    setProjectDraft((previous) => ({ ...previous, [workerId]: projectIds }))
    setDirty(true)
  }

  function shiftDate(days: number) {
    if (!currentDate) return
    setDirty(false)
    setDate(format(addDays(parseISO(currentDate), days), 'yyyy-MM-dd'))
  }

  const workers = data?.workers ?? []
  const projectList = projects.data ?? []
  const rows = workers.map((worker) => {
    const quarters = draft[worker.worker_id] ?? worker.quarters
    const projectIds =
      projectDraft[worker.worker_id] ?? worker.projects.map((project) => project.id)
    return {
      ...worker,
      quarters,
      projectIds,
      tags: projectList.filter((project) => projectIds.includes(project.id)),
      preview: Math.floor((worker.snapshot * quarters) / 4),
    }
  })
  const present = rows.filter((row) => row.quarters > 0).length
  const total = rows.reduce((sum, row) => sum + row.preview, 0)

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold tracking-tight">Absensi</h1>
        <p className="text-muted-foreground text-sm">
          Ketuk bagian hari untuk setiap tukang. Perubahan tersimpan otomatis.
        </p>
      </div>

      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            className="size-11 shrink-0 p-0"
            onClick={() => shiftDate(-1)}
            disabled={!currentDate || saving}
            aria-label="Hari sebelumnya"
          >
            <ChevronLeft className="size-5" aria-hidden />
          </Button>
          <Input
            type="date"
            className="h-11 flex-1"
            value={currentDate}
            onChange={(event) => {
              setDirty(false)
              setDate(event.target.value || null)
            }}
            aria-label="Tanggal absensi"
          />
          <Button
            variant="outline"
            className="size-11 shrink-0 p-0"
            onClick={() => shiftDate(1)}
            disabled={!currentDate || saving}
            aria-label="Hari berikutnya"
          >
            <ChevronRight className="size-5" aria-hidden />
          </Button>
        </div>

        <div className="flex items-center justify-between gap-2">
          <p className="text-sm font-medium">{currentDate ? formatDate(currentDate) : '—'}</p>
          {currentDate !== todayISO() && (
            <Button
              variant="ghost"
              className="h-9 px-3 text-xs"
              onClick={() => {
                setDirty(false)
                setDate(null)
              }}
            >
              Hari ini
            </Button>
          )}
        </div>
      </div>

      {rows.length > 0 && (
        <div className="bg-muted/50 flex items-center justify-between rounded-lg px-3 py-2 text-sm">
          <span>
            Hadir <strong>{present}</strong> dari {rows.length} pekerja
          </span>
          <span className="font-medium">{formatRupiah(total)}</span>
        </div>
      )}

      <p className="text-muted-foreground h-4 text-xs" role="status">
        {saving
          ? 'Menyimpan…'
          : dirty
            ? 'Belum tersimpan'
            : saveError
              ? saveError.message
              : 'Tersimpan'}
      </p>

      {isPending && (
        <div className="space-y-2" aria-hidden>
          <Skeleton className="h-28 w-full rounded-xl" />
          <Skeleton className="h-28 w-full rounded-xl" />
          <Skeleton className="h-28 w-full rounded-xl" />
        </div>
      )}

      {isError && (
        <div className="rounded-xl border p-6 text-center">
          <p className="text-sm">
            {error instanceof Error ? error.message : 'Tidak dapat memuat absensi.'}
          </p>
          <Button
            variant="outline"
            className="mt-4 h-11 px-5"
            onClick={() => void refetch()}
          >
            Coba lagi
          </Button>
        </div>
      )}

      {!isPending && !isError && rows.length === 0 && (
        <div className="rounded-xl border p-8 text-center">
          <p className="font-medium">Belum ada pekerja aktif.</p>
          <p className="text-muted-foreground mt-1 text-sm">
            Tambahkan pekerja dulu sebelum mencatat absensi.
          </p>
          <Button className="mt-4 h-11 px-5" render={<Link to="/pekerja" />}>
            Tambah pekerja
          </Button>
        </div>
      )}

      {!isPending && !isError && rows.length > 0 && (
        <ul className="space-y-2">
          {rows.map((row) => (
            <li key={row.worker_id} className="space-y-3 rounded-xl border p-3">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="truncate font-medium">{row.name}</p>
                    {row.locked && (
                      <span className="text-muted-foreground flex items-center gap-1 text-xs">
                        <Lock className="size-3" aria-hidden />
                        Terkunci
                      </span>
                    )}
                  </div>
                  <p className="text-muted-foreground truncate text-xs">{row.role}</p>
                </div>
                <div className="shrink-0 text-right">
                  <p className="text-sm font-medium">{formatRupiah(row.preview)}</p>
                  <Link
                    to={`/pekerja/${row.worker_id}`}
                    className="text-muted-foreground text-xs underline underline-offset-2"
                  >
                    Lihat minggu
                  </Link>
                </div>
              </div>

              <div
                role="group"
                aria-label={`Absensi ${row.name}`}
                className="grid grid-cols-5 gap-1"
              >
                {quarterOptions.map((option) => {
                  const active = row.quarters === option.value
                  return (
                    <button
                      key={option.value}
                      type="button"
                      disabled={row.locked}
                      aria-pressed={active}
                      aria-label={option.aria}
                      onClick={() => setQuarters(row.worker_id, option.value)}
                      className={cn(
                        'h-11 rounded-lg border text-sm font-medium transition-colors disabled:opacity-50',
                        active
                          ? 'bg-primary text-primary-foreground border-primary'
                          : 'hover:bg-muted',
                      )}
                    >
                      {option.label}
                    </button>
                  )
                })}
              </div>

              {row.quarters > 0 && (
                <div className="flex items-center justify-between gap-2">
                  <span className="text-muted-foreground truncate text-xs">
                    {row.tags.length > 0
                      ? row.tags.map((project) => project.name).join(' · ')
                      : 'Belum ada proyek'}
                  </span>
                  <Button
                    variant="outline"
                    className="h-11 shrink-0 px-3 text-xs"
                    onClick={() => {
                      setProjectTarget({ workerId: row.worker_id, name: row.name })
                    }}
                    disabled={row.locked || projects.isPending}
                  >
                    <Tag className="size-4" aria-hidden />
                    Proyek
                  </Button>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}

      <AttendanceProjectSheet
        open={Boolean(projectTarget)}
        onOpenChange={(open) => {
          if (!open) setProjectTarget(null)
        }}
        workerName={projectTarget?.name ?? ''}
        projects={projectList}
        selected={
          projectTarget ? (projectDraft[projectTarget.workerId] ?? []) : []
        }
        onSave={(projectIds) => {
          if (projectTarget) saveProjects(projectTarget.workerId, projectIds)
        }}
      />

      <p className="text-muted-foreground text-xs">
        — absen · ¼ seperempat · ½ setengah · ¾ tiga per empat · 1 penuh
      </p>
    </div>
  )
}
