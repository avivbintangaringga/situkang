import { ChevronRight, Users } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router'

import { AccountSettings } from '@/components/account-settings'
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
  const [fractionSelection, setFractionSelection] = useState<number[] | null>(null)

  const workingDays = selection ?? settings.data?.working_days ?? []
  const fractions = fractionSelection ?? settings.data?.attendance_fractions ?? []
  const dirty = selection !== null || fractionSelection !== null

  function toggle(day: number) {
    setSelection(
      workingDays.includes(day)
        ? workingDays.filter((value) => value !== day)
        : [...workingDays, day].sort((a, b) => a - b),
    )
  }

  function toggleFraction(value: number) {
    setFractionSelection(
      fractions.includes(value)
        ? fractions.filter((item) => item !== value)
        : [...fractions, value].sort((a, b) => a - b),
    )
  }

  function save() {
    update.mutate(
      { working_days: workingDays, attendance_fractions: fractions },
      {
        onSuccess: () => {
          setSelection(null)
          setFractionSelection(null)
        },
      },
    )
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold tracking-tight">Pengaturan</h1>
        <p className="text-muted-foreground text-sm">
          Atur hari kerja. Absensi tetap bisa dicatat pada hari libur.
        </p>
      </div>

      <AccountSettings />

      <Link
        to="/pengguna"
        className="hover:bg-muted/50 flex min-h-14 items-center justify-between gap-3 rounded-xl border p-4 transition-colors"
      >
        <span className="flex min-w-0 items-center gap-3">
          <Users className="text-muted-foreground size-5 shrink-0" aria-hidden />
          <span className="min-w-0">
            <span className="block font-medium">Kelola pengguna</span>
            <span className="text-muted-foreground block text-sm">
              Lihat dan tambah akun admin.
            </span>
          </span>
        </span>
        <ChevronRight className="text-muted-foreground size-5 shrink-0" aria-hidden />
      </Link>

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
            className="mt-4 h-9 px-5"
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
        <>
          <div>
            <h2 className="text-sm font-semibold">Bagian hari</h2>
            <p className="text-muted-foreground text-sm">
              Pilih bagian hari yang bisa dipilih saat absensi. Penuh dan absen
              selalu tersedia.
            </p>
          </div>

          <div className="divide-y rounded-xl border">
            {[
              { value: 1, label: 'Seperempat hari (1/4)' },
              { value: 2, label: 'Setengah hari (1/2)' },
              { value: 3, label: 'Tiga per empat hari (3/4)' },
            ].map((fraction) => (
              <label
                key={fraction.value}
                className="flex h-14 cursor-pointer items-center justify-between gap-3 px-4"
              >
                <span className="text-sm font-medium">{fraction.label}</span>
                <Switch
                  checked={fractions.includes(fraction.value)}
                  onCheckedChange={() => toggleFraction(fraction.value)}
                  aria-label={`Aktifkan ${fraction.label}`}
                />
              </label>
            ))}
          </div>
        </>
      )}

      {settings.data && (
        <Button
          className="h-9 w-full"
          onClick={save}
          disabled={!dirty || workingDays.length === 0 || update.isPending}
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
