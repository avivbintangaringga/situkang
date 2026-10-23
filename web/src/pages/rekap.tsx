import { Plus } from 'lucide-react'
import { useState } from 'react'
import { useNavigate } from 'react-router'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { useCreateRecap, useRecaps } from '@/hooks/use-recaps'
import { ApiError } from '@/lib/api'
import { formatDate, formatDateShort } from '@/lib/format'
import type { RecapMode } from '@/lib/types'
import { cn } from '@/lib/utils'

const modes: { value: RecapMode; label: string; description: string }[] = [
  {
    value: 'all_unsettled',
    label: 'Semua yang belum dibayar',
    description: 'Semua absensi dan kas bon yang belum masuk rekap.',
  },
  {
    value: 'since_monday',
    label: 'Sejak Senin',
    description: 'Hanya minggu ini. Tidak cocok untuk mengejar pembayaran minggu lalu.',
  },
  {
    value: 'custom',
    label: 'Rentang tanggal',
    description: 'Pilih tanggal mulai dan selesai sendiri.',
  },
]

export function RekapPage() {
  const recaps = useRecaps()
  const navigate = useNavigate()

  const [open, setOpen] = useState(false)
  const [mode, setMode] = useState<RecapMode>('all_unsettled')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const create = useCreateRecap()

  function submit() {
    create.mutate(
      {
        mode,
        start_date: mode === 'custom' ? startDate : undefined,
        end_date: mode === 'custom' ? endDate : undefined,
      },
      {
        onSuccess: (recap) => {
          setOpen(false)
          navigate(`/rekap/${recap.id}`)
        },
      },
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Rekap</h1>
          <p className="text-muted-foreground text-sm">
            Rekap gaji mingguan tukang.
          </p>
        </div>
        <Button className="h-11 shrink-0 px-3" onClick={() => setOpen(true)}>
          <Plus className="size-4" aria-hidden />
          Buat
        </Button>
      </div>

      {recaps.isPending && (
        <div className="space-y-2" aria-hidden>
          <Skeleton className="h-24 w-full rounded-xl" />
          <Skeleton className="h-24 w-full rounded-xl" />
        </div>
      )}

      {recaps.isError && (
        <div className="rounded-xl border p-6 text-center">
          <p className="text-sm">
            {recaps.error instanceof Error
              ? recaps.error.message
              : 'Tidak dapat memuat rekap.'}
          </p>
          <Button
            variant="outline"
            className="mt-4 h-11 px-5"
            onClick={() => void recaps.refetch()}
          >
            Coba lagi
          </Button>
        </div>
      )}

      {recaps.data && recaps.data.length === 0 && (
        <div className="rounded-xl border p-8 text-center">
          <p className="font-medium">Belum ada rekap.</p>
          <p className="text-muted-foreground mt-1 text-sm">
            Buat rekap dari absensi dan kas bon yang belum dibayar.
          </p>
          <Button className="mt-4 h-11 px-5" onClick={() => setOpen(true)}>
            Buat rekap
          </Button>
        </div>
      )}

      {recaps.data && recaps.data.length > 0 && (
        <ul className="space-y-2">
          {recaps.data.map((recap) => (
            <li key={recap.id}>
              <button
                type="button"
                onClick={() => navigate(`/rekap/${recap.id}`)}
                className="hover:bg-muted/50 flex w-full items-center justify-between gap-3 rounded-xl border p-4 text-left"
              >
                <div className="min-w-0 space-y-1">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">Rekap #{recap.id}</span>
                    <span
                      className={cn(
                        'rounded-full px-2 py-0.5 text-xs',
                        recap.status === 'final'
                          ? 'bg-muted text-muted-foreground'
                          : 'bg-primary/10 text-primary',
                      )}
                    >
                      {recap.status === 'final' ? 'Final' : 'Draft'}
                    </span>
                  </div>
                  <p className="text-muted-foreground truncate text-sm">
                    {recap.start_date && recap.end_date
                      ? `${formatDateShort(recap.start_date)} – ${formatDate(recap.end_date)}`
                      : 'Semua yang belum dibayar'}
                  </p>
                </div>
              </button>
            </li>
          ))}
        </ul>
      )}

      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent side="bottom" className="max-h-[90dvh] overflow-y-auto">
          <SheetHeader>
            <SheetTitle>Buat rekap</SheetTitle>
            <SheetDescription>
              Pilih periode. Hanya satu draft rekap yang bisa ada.
            </SheetDescription>
          </SheetHeader>

          <div className="space-y-4 px-4">
            <div className="space-y-2">
              {modes.map((option) => (
                <button
                  key={option.value}
                  type="button"
                  onClick={() => setMode(option.value)}
                  className={cn(
                    'w-full rounded-xl border p-3 text-left transition-colors',
                    mode === option.value ? 'border-primary bg-primary/5' : 'hover:bg-muted/50',
                  )}
                >
                  <p className="text-sm font-medium">{option.label}</p>
                  <p className="text-muted-foreground text-xs">{option.description}</p>
                </button>
              ))}
            </div>

            {mode === 'custom' && (
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-2">
                  <Label htmlFor="recap-start">Mulai</Label>
                  <Input
                    id="recap-start"
                    type="date"
                    className="h-11"
                    value={startDate}
                    onChange={(event) => setStartDate(event.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="recap-end">Selesai</Label>
                  <Input
                    id="recap-end"
                    type="date"
                    className="h-11"
                    value={endDate}
                    onChange={(event) => setEndDate(event.target.value)}
                  />
                </div>
              </div>
            )}

            {create.error && (
              <p className="text-destructive text-sm" role="alert">
                {create.error instanceof ApiError
                  ? create.error.message
                  : 'Tidak dapat membuat rekap.'}
              </p>
            )}

            <SheetFooter className="px-0">
              <Button
                className="h-11 w-full"
                onClick={submit}
                disabled={
                  create.isPending ||
                  (mode === 'custom' && (!startDate || !endDate))
                }
              >
                {create.isPending ? 'Membuat…' : 'Buat rekap'}
              </Button>
            </SheetFooter>
          </div>
        </SheetContent>
      </Sheet>
    </div>
  )
}
