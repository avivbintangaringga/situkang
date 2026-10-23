import { addDays, format, parseISO } from 'date-fns'
import { Banknote, ChevronLeft, ChevronRight, Lock } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router'

import { CashAdvanceSheet } from '@/components/cash-advance-sheet'
import { DatePicker } from '@/components/date-picker'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useAttendanceDay, useSaveDay } from '@/hooks/use-attendance'
import { useSettings } from '@/hooks/use-settings'
import { formatDate, formatRupiah, todayISO } from '@/lib/format'
import { cn } from '@/lib/utils'

const fractionOptions = {
  0: {
    label: 'Absen',
    aria: 'Absen',
    text: 'text-xs',
    active:
      'border-destructive/40 bg-destructive/10 text-destructive dark:border-destructive/60 dark:bg-destructive/20',
  },
  1: {
    label: '1/4',
    aria: 'Seperempat hari',
    text: 'text-sm',
    active:
      'border-amber-300 bg-amber-100 text-amber-900 dark:border-amber-700 dark:bg-amber-900/40 dark:text-amber-100',
  },
  2: {
    label: '1/2',
    aria: 'Setengah hari',
    text: 'text-sm',
    active:
      'border-sky-300 bg-sky-100 text-sky-900 dark:border-sky-700 dark:bg-sky-900/40 dark:text-sky-100',
  },
  3: {
    label: '3/4',
    aria: 'Tiga per empat hari',
    text: 'text-sm',
    active:
      'border-violet-300 bg-violet-100 text-violet-900 dark:border-violet-700 dark:bg-violet-900/40 dark:text-violet-100',
  },
  4: {
    label: 'Hadir',
    aria: 'Hadir sehari penuh',
    text: 'text-sm',
    active:
      'border-emerald-300 bg-emerald-100 text-emerald-900 dark:border-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-100',
  },
} as const

export function AbsensiPage() {
  const [date, setDate] = useState<string | null>(null)
  const { data, isPending, isError, error, refetch } = useAttendanceDay(date)
  const { mutate: saveDay, isPending: saving, error: saveError } = useSaveDay()
  const settings = useSettings()

  const [draft, setDraft] = useState<Record<number, number>>({})
  const [dirty, setDirty] = useState(false)
  const [advanceTarget, setAdvanceTarget] = useState<{ workerId: number; name: string } | null>(
    null,
  )

  const currentDate = data?.date ?? date ?? ''
  const syncedDate = useRef<string | null>(null)

  useEffect(() => {
    if (!data) return
    // Only load the draft when a different day arrives. A save triggers a
    // background refresh of the same day, and resetting here would briefly
    // flash the previous value back.
    if (syncedDate.current === data.date) return
    syncedDate.current = data.date
    // oxlint-disable-next-line react/set-state-in-effect -- load the day into the editable draft
    setDraft(
      Object.fromEntries(data.workers.map((worker) => [worker.worker_id, worker.quarters])),
    )
  }, [data])

  useEffect(() => {
    if (!dirty || !currentDate) return

    const timer = setTimeout(() => {
      const items = Object.entries(draft)
        .filter(([, quarters]) => quarters > 0)
        .map(([id, quarters]) => ({
          worker_id: Number(id),
          quarters,
        }))

      saveDay(
        { date: currentDate, items },
        {
          onSuccess: (saved) => {
            // Show the server result immediately; the cache may live under a
            // different "today" key than the saved date.
            setDraft(
              Object.fromEntries(saved.workers.map((worker) => [worker.worker_id, worker.quarters])),
            )
            setDirty(false)
          },
        },
      )
    }, 600)

    return () => clearTimeout(timer)
  }, [draft, dirty, currentDate, saveDay])

  function setQuarters(workerId: number, quarters: number) {
    setDraft((previous) => ({ ...previous, [workerId]: quarters }))
    setDirty(true)
  }

  function shiftDate(days: number) {
    if (!currentDate) return
    setDirty(false)
    setDate(format(addDays(parseISO(currentDate), days), 'yyyy-MM-dd'))
  }

  const enabledFractions = settings.data?.attendance_fractions ?? [1, 2, 3]
  const quarterOptions = [0, 1, 2, 3, 4].filter(
    (value) => value === 0 || value === 4 || enabledFractions.includes(value),
  )

  const workers = data?.workers ?? []
  const rows = workers.map((worker) => {
    const quarters = draft[worker.worker_id] ?? worker.quarters
    return {
      ...worker,
      quarters,
      preview: Math.floor((worker.snapshot * quarters) / 4),
    }
  })
  const present = rows.filter((row) => row.quarters > 0).length
  const total = rows.reduce((sum, row) => sum + row.preview, 0)

  const isoWeekday = currentDate
    ? ((parseISO(currentDate).getDay() + 6) % 7) + 1
    : 0
  const isNonWorkingDay =
    Boolean(currentDate) &&
    Boolean(settings.data) &&
    !settings.data?.working_days.includes(isoWeekday)

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
            className="size-9 shrink-0 p-0"
            onClick={() => shiftDate(-1)}
            disabled={!currentDate || saving}
            aria-label="Hari sebelumnya"
          >
            <ChevronLeft className="size-5" aria-hidden />
          </Button>
          <DatePicker
            value={currentDate}
            onChange={(value) => {
              setDirty(false)
              setDate(value)
            }}
            className="flex-1"
          />
          <Button
            variant="outline"
            className="size-9 shrink-0 p-0"
            onClick={() => shiftDate(1)}
            disabled={!currentDate || saving}
            aria-label="Hari berikutnya"
          >
            <ChevronRight className="size-5" aria-hidden />
          </Button>
        </div>

        <div className="flex items-center justify-between gap-2">
          <p className="text-sm font-medium">
            {currentDate ? formatDate(currentDate) : '-'}
            {isNonWorkingDay && (
              <span className="text-muted-foreground font-normal"> · Hari libur</span>
            )}
          </p>
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
            className="mt-4 h-9 px-5"
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
          <Button className="mt-4 h-9 px-5" render={<Link to="/pekerja" />}>
            Tambah pekerja
          </Button>
        </div>
      )}

      {!isPending && !isError && rows.length > 0 && (
        <ul className="space-y-2">
          {rows.map((row) => (
            <li key={row.worker_id} className="space-y-2.5 rounded-xl border p-3">
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
                className="bg-muted/60 flex gap-0.5 rounded-full p-0.5 sm:max-w-md"
              >
                {(quarterOptions.includes(row.quarters)
                  ? quarterOptions
                  : [...quarterOptions, row.quarters].sort((a, b) => a - b)
                ).map((value) => {
                  const option = fractionOptions[value as keyof typeof fractionOptions]
                  const active = row.quarters === value
                  return (
                    <button
                      key={value}
                      type="button"
                      disabled={row.locked}
                      aria-pressed={active}
                      aria-label={option.aria}
                      onClick={() => setQuarters(row.worker_id, value)}
                      className={cn(
                        'h-9 flex-1 rounded-full font-medium transition-colors disabled:opacity-50',
                        option.text,
                        active
                          ? cn(option.active, 'shadow-sm')
                          : 'text-muted-foreground hover:bg-background/80',
                      )}
                    >
                      {option.label}
                    </button>
                  )
                })}
              </div>

              {row.quarters > 0 && (
                <Button
                  variant="outline"
                  className="h-9 rounded-full px-4 text-xs"
                  onClick={() => {
                    setAdvanceTarget({ workerId: row.worker_id, name: row.name })
                  }}
                  disabled={row.locked || !currentDate}
                >
                  <Banknote className="size-3.5" aria-hidden />
                  Kas bon
                </Button>
              )}
            </li>
          ))}
        </ul>
      )}

      {advanceTarget && (
        <CashAdvanceSheet
          open={Boolean(advanceTarget)}
          onOpenChange={(open) => {
            if (!open) setAdvanceTarget(null)
          }}
          workerId={advanceTarget.workerId}
          defaultDate={currentDate}
        />
      )}

      <p className="text-muted-foreground text-xs">
        Bagian hari yang tampil bisa diubah di Pengaturan.
      </p>
    </div>
  )
}
