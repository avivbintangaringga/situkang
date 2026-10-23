import { Plus, Search } from 'lucide-react'
import { useDeferredValue, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { WorkerFormSheet } from '@/components/worker-form-sheet'
import { useWorkers } from '@/hooks/use-workers'
import { formatRupiah } from '@/lib/format'
import type { Worker } from '@/lib/types'

export function PekerjaPage() {
  const [search, setSearch] = useState('')
  const [includeInactive, setIncludeInactive] = useState(false)
  const deferredSearch = useDeferredValue(search)

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<Worker | null>(null)

  const { data, isPending, isError, error, refetch } = useWorkers(
    deferredSearch,
    includeInactive,
  )

  function openCreate() {
    setEditing(null)
    setFormOpen(true)
  }

  function openEdit(worker: Worker) {
    setEditing(worker)
    setFormOpen(true)
  }

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Pekerja</h1>
          <p className="text-muted-foreground text-sm">
            Daftar tukang beserta upah hariannya.
          </p>
        </div>
        <Button className="h-11 shrink-0 px-3" onClick={openCreate}>
          <Plus className="size-4" aria-hidden />
          Tambah
        </Button>
      </div>

      <div className="space-y-3">
        <div className="relative">
          <Search
            className="text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2"
            aria-hidden
          />
          <Input
            className="h-11 pl-9"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Cari nama atau peran…"
            aria-label="Cari pekerja"
          />
        </div>

        <div className="flex items-center justify-between">
          <Label htmlFor="include-inactive" className="text-sm font-normal">
            Tampilkan yang nonaktif
          </Label>
          <Switch
            id="include-inactive"
            checked={includeInactive}
            onCheckedChange={setIncludeInactive}
          />
        </div>
      </div>

      {isPending && (
        <div className="space-y-2" aria-hidden>
          <Skeleton className="h-20 w-full rounded-xl" />
          <Skeleton className="h-20 w-full rounded-xl" />
          <Skeleton className="h-20 w-full rounded-xl" />
        </div>
      )}

      {isError && (
        <div className="rounded-xl border p-6 text-center">
          <p className="text-sm">
            {error instanceof Error
              ? error.message
              : 'Tidak dapat memuat daftar pekerja.'}
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

      {!isPending && !isError && data && data.length === 0 && (
        <div className="rounded-xl border p-8 text-center">
          <p className="font-medium">
            {search ? 'Tidak ada pekerja yang cocok.' : 'Belum ada pekerja.'}
          </p>
          <p className="text-muted-foreground mt-1 text-sm">
            {search
              ? 'Coba kata kunci lain.'
              : 'Tambahkan tukang pertama untuk mulai mencatat absensi.'}
          </p>
          {!search && (
            <Button className="mt-4 h-11 px-5" onClick={openCreate}>
              Tambah pekerja
            </Button>
          )}
        </div>
      )}

      {!isPending && !isError && data && data.length > 0 && (
        <ul className="space-y-2">
          {data.map((worker) => (
            <li key={worker.id}>
              <button
                type="button"
                onClick={() => openEdit(worker)}
                className="bg-card hover:bg-muted/50 flex w-full items-center justify-between gap-3 rounded-xl border p-4 text-left transition-colors"
              >
                <div className="min-w-0 space-y-1">
                  <div className="flex items-center gap-2">
                    <span className="truncate font-medium">{worker.name}</span>
                    {!worker.active && (
                      <Badge variant="secondary" className="text-xs">
                        Nonaktif
                      </Badge>
                    )}
                  </div>
                  <p className="text-muted-foreground truncate text-sm">
                    {worker.role}
                  </p>
                </div>
                <span className="shrink-0 text-sm font-medium">
                  {formatRupiah(worker.daily_fee)}
                  <span className="text-muted-foreground block text-xs font-normal">
                    per hari
                  </span>
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}

      <WorkerFormSheet
        open={formOpen}
        onOpenChange={setFormOpen}
        worker={editing}
      />
    </div>
  )
}
