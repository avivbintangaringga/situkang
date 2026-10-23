import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { Link } from 'react-router'
import { z } from 'zod'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
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
import { Switch } from '@/components/ui/switch'
import {
  useCreateWorker,
  useDeleteWorker,
  useUpdateWorker,
} from '@/hooks/use-workers'
import { ApiError } from '@/lib/api'
import { digitsOnly, formatThousands } from '@/lib/format'
import type { Worker } from '@/lib/types'
import { useState } from 'react'

const workerSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, 'Nama wajib diisi.')
    .max(100, 'Nama maksimal 100 karakter.'),
  role: z
    .string()
    .trim()
    .min(1, 'Peran wajib diisi.')
    .max(50, 'Peran maksimal 50 karakter.'),
  fee: z.string().min(1, 'Upah harian wajib diisi.'),
  active: z.boolean(),
})

type WorkerForm = z.infer<typeof workerSchema>

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  worker?: Worker | null
}

const emptyValues: WorkerForm = { name: '', role: '', fee: '0', active: true }

function valuesFor(worker?: Worker | null): WorkerForm {
  if (!worker) return emptyValues
  return {
    name: worker.name,
    role: worker.role,
    fee: String(worker.daily_fee),
    active: worker.active,
  }
}

export function WorkerFormSheet({ open, onOpenChange, worker }: Props) {
  const isEdit = Boolean(worker)
  const create = useCreateWorker()
  const update = useUpdateWorker(worker?.id ?? 0)
  const remove = useDeleteWorker()
  const [confirmDelete, setConfirmDelete] = useState(false)

  const form = useForm<WorkerForm>({
    resolver: zodResolver(workerSchema),
    defaultValues: valuesFor(worker),
  })

  useEffect(() => {
    if (!open) return
    form.reset(valuesFor(worker))
    setConfirmDelete(false)
  }, [open, worker, form])

  const feeValue = formatThousands(form.watch('fee'))
  const pending = create.isPending || update.isPending
  const mutationError = create.error ?? update.error

  function onSubmit(values: WorkerForm) {
    const input = {
      name: values.name,
      role: values.role,
      daily_fee: Number(values.fee || '0'),
      active: values.active,
    }

    if (worker) {
      update.mutate(input, { onSuccess: () => onOpenChange(false) })
    } else {
      create.mutate(input, { onSuccess: () => onOpenChange(false) })
    }
  }

  function onDelete() {
    if (!worker) return
    remove.mutate(worker.id, {
      onSuccess: () => {
        setConfirmDelete(false)
        onOpenChange(false)
      },
    })
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="bottom" className="max-h-[90dvh] overflow-y-auto">
        <SheetHeader>
          <SheetTitle>{isEdit ? 'Ubah pekerja' : 'Tambah pekerja'}</SheetTitle>
          <SheetDescription>
            {isEdit
              ? 'Perubahan upah berlaku untuk absensi berikutnya, bukan yang sudah tercatat.'
              : 'Pekerja tidak perlu akun. Cukup nama, peran, dan upah harian.'}
          </SheetDescription>
        </SheetHeader>

        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="flex flex-col gap-4 px-4"
          noValidate
        >
          <div className="space-y-2">
            <Label htmlFor="worker-name">Nama</Label>
            <Input
              id="worker-name"
              className="h-9"
              autoComplete="off"
              aria-invalid={Boolean(form.formState.errors.name)}
              {...form.register('name')}
            />
            {form.formState.errors.name && (
              <p className="text-destructive text-sm">
                {form.formState.errors.name.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="worker-role">Peran</Label>
            <Input
              id="worker-role"
              className="h-9"
              placeholder="Tukang batu, kenek, mandor…"
              autoComplete="off"
              aria-invalid={Boolean(form.formState.errors.role)}
              {...form.register('role')}
            />
            {form.formState.errors.role && (
              <p className="text-destructive text-sm">
                {form.formState.errors.role.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="worker-fee">Upah harian</Label>
            <div className="relative">
              <span className="text-muted-foreground pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-sm">
                Rp
              </span>
              <Input
                id="worker-fee"
                className="h-9 pl-9"
                inputMode="numeric"
                autoComplete="off"
                value={feeValue}
                onChange={(event) =>
                  form.setValue('fee', digitsOnly(event.target.value), {
                    shouldValidate: form.formState.isSubmitted,
                  })
                }
              />
            </div>
            {form.formState.errors.fee && (
              <p className="text-destructive text-sm">
                {form.formState.errors.fee.message}
              </p>
            )}
          </div>

          {isEdit && (
            <div className="flex items-center justify-between rounded-lg border px-3 py-2">
              <div>
                <Label htmlFor="worker-active">Aktif</Label>
                <p className="text-muted-foreground text-xs">
                  Nonaktif menyembunyikan pekerja dari daftar absensi.
                </p>
              </div>
              <Switch
                id="worker-active"
                checked={form.watch('active')}
                onCheckedChange={(checked) => form.setValue('active', checked)}
              />
            </div>
          )}

          {mutationError && (
            <p className="text-destructive text-sm" role="alert">
              {mutationError instanceof ApiError
                ? mutationError.message
                : 'Tidak dapat menyimpan pekerja.'}
            </p>
          )}

          <SheetFooter className="px-0">
            <Button type="submit" className="h-9 w-full" disabled={pending}>
              {pending ? 'Menyimpan…' : 'Simpan'}
            </Button>

            {isEdit && worker && (
              <Button
                type="button"
                variant="outline"
                className="h-9 w-full"
                render={<Link to={`/pekerja/${worker.id}`} />}
              >
                Lihat detail dan kas bon
              </Button>
            )}

            {isEdit && (
              <Button
                type="button"
                variant="destructive"
                className="h-9 w-full"
                onClick={() => setConfirmDelete(true)}
              >
                Nonaktifkan pekerja
              </Button>
            )}
          </SheetFooter>
        </form>
      </SheetContent>

      <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Nonaktifkan pekerja ini?</AlertDialogTitle>
            <AlertDialogDescription>
              Riwayat absensi dan rekapnya tetap tersimpan. Pekerja akan hilang
              dari daftar aktif dan tidak muncul di layar absensi.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onDelete} disabled={remove.isPending}>
              {remove.isPending ? 'Memproses…' : 'Nonaktifkan'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Sheet>
  )
}
