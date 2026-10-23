import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { useSettings, useUpdateSettings } from '@/hooks/use-settings'
import { ApiError } from '@/lib/api'

// ISO weekday numbers: Monday = 1 ... Sunday = 7.
const weekdays = [
  { value: 1, label: 'Senin' },
  { value: 2, label: 'Selasa' },
  { value: 3, label: 'Rabu' },
  { value: 4, label: 'Kamis' },
  { value: 5, label: 'Jumat' },
  { value: 6, label: 'Sabtu' },
  { value: 7, label: 'Minggu' },
]

export function PengaturanPage() {
  const settings = useSettings()
  const update = useUpdateSettings()
  const [selection, setSelection] = useState<number[] | null>(null)

  const workingDays = selection ?? settings.data?.working_days ?? []

  function toggle(day: number) {
    setSelection(
      workingDays.includes(day)
        ? workingDays.filter((value) => value !== day)
        : [...workingDays, day].sort((a, b) => a - b),
    )
  }

  function save() {
    if (selection) {
      update.mutate(selection, { onSuccess: () => setSelection(null) })
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold tracking-tight">Pengaturan</h1>
        <p className="text-muted-foreground text-sm">
          Atur hari kerja. Absensi tetap bisa dicatat pada hari libur.
        </p>
      </div>

      {settings.isPending && <Skeleton className="h-80 w-full rounded-xl" />}

      {settings.isError && (
        <div className="rounded-xl border p-6 text-center">
          <p className="text-sm">
            {settings.error instanceof Error
              ? settings.error.message
              : 'Tidak dapat memuat pengaturan.'}
          </p>
          <Button
            variant="outline"
            className="mt-4 h-11 px-5"
            onClick={() => void settings.refetch()}
          >
            Coba lagi
          </Button>
        </div>
      )}

      {settings.data && (
        <div className="divide-y rounded-xl border">
          {weekdays.map((day) => (
            <label
              key={day.value}
              className="flex h-14 cursor-pointer items-center justify-between gap-3 px-4"
            >
              <span className="text-sm font-medium">{day.label}</span>
              <Switch
                checked={workingDays.includes(day.value)}
                onCheckedChange={() => toggle(day.value)}
                aria-label={`Hari kerja ${day.label}`}
              />
            </label>
          ))}
        </div>
      )}

      {update.error && (
        <p className="text-destructive text-sm" role="alert">
          {update.error instanceof ApiError
            ? update.error.message
            : 'Tidak dapat menyimpan pengaturan.'}
        </p>
      )}

      {settings.data && (
        <Button
          className="h-11 w-full"
          onClick={save}
          disabled={!selection || selection.length === 0 || update.isPending}
        >
          {update.isPending ? 'Menyimpan…' : 'Simpan'}
        </Button>
      )}

      {settings.data && workingDays.length === 0 && (
        <p className="text-destructive text-sm">Pilih minimal satu hari kerja.</p>
      )}
    </div>
  )
}
