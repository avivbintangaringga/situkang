import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
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
import {
  useCreateCashAdvance,
  useDeleteCashAdvance,
  useUpdateCashAdvance,
} from '@/hooks/use-cash-advances'
import { ApiError } from '@/lib/api'
import { digitsOnly, formatThousands, todayISO } from '@/lib/format'
import type { CashAdvance } from '@/lib/types'

const schema = z.object({
  date: z.string().min(1, 'Tanggal wajib diisi.'),
  amount: z.string().min(1, 'Jumlah wajib diisi.'),
  note: z.string().max(200, 'Catatan maksimal 200 karakter.'),
})

type Form = z.infer<typeof schema>

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  workerId: number
  advance?: CashAdvance | null
}

function valuesFor(advance?: CashAdvance | null): Form {
  if (!advance) {
    return { date: todayISO(), amount: '', note: '' }
  }
  return {
    date: advance.date,
    amount: String(advance.amount),
    note: advance.note,
  }
}

export function CashAdvanceSheet({ open, onOpenChange, workerId, advance }: Props) {
  const create = useCreateCashAdvance(workerId)
  const update = useUpdateCashAdvance(advance?.id ?? 0, workerId)
  const remove = useDeleteCashAdvance(workerId)
  const [confirmDelete, setConfirmDelete] = useState(false)

  const form = useForm<Form>({
    resolver: zodResolver(schema),
    defaultValues: valuesFor(advance),
  })

  useEffect(() => {
    if (!open) return
    form.reset(valuesFor(advance))
    setConfirmDelete(false)
  }, [open, advance, form])

  const pending = create.isPending || update.isPending
  const mutationError = create.error ?? update.error
  const amount = Number(form.watch('amount') || '0')

  function onSubmit(values: Form) {
    const input = {
      date: values.date,
      amount: Number(values.amount || '0'),
      note: values.note.trim(),
    }

    if (advance) {
      update.mutate(input, { onSuccess: () => onOpenChange(false) })
    } else {
      create.mutate(input, { onSuccess: () => onOpenChange(false) })
    }
  }

  function onDelete() {
    if (!advance) return
    remove.mutate(advance.id, {
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
          <SheetTitle>{advance ? 'Ubah kas bon' : 'Tambah kas bon'}</SheetTitle>
          <SheetDescription>
            Kas bon akan dipotong dari rekap gaji berikutnya.
          </SheetDescription>
        </SheetHeader>

        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 px-4" noValidate>
          <div className="space-y-2">
            <Label htmlFor="advance-date">Tanggal</Label>
            <Input
              id="advance-date"
              type="date"
              className="h-11"
              aria-invalid={Boolean(form.formState.errors.date)}
              {...form.register('date')}
            />
            {form.formState.errors.date && (
              <p className="text-destructive text-sm">
                {form.formState.errors.date.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="advance-amount">Jumlah</Label>
            <div className="relative">
              <span className="text-muted-foreground pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-sm">
                Rp
              </span>
              <Input
                id="advance-amount"
                className="h-11 pl-9"
                inputMode="numeric"
                autoComplete="off"
                value={formatThousands(form.watch('amount'))}
                onChange={(event) =>
                  form.setValue('amount', digitsOnly(event.target.value), {
                    shouldValidate: form.formState.isSubmitted,
                  })
                }
              />
            </div>
            {form.formState.errors.amount && (
              <p className="text-destructive text-sm">
                {form.formState.errors.amount.message}
              </p>
            )}
            {amount <= 0 && form.watch('amount') !== '' && (
              <p className="text-destructive text-sm">Jumlah harus lebih dari nol.</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="advance-note">Catatan (opsional)</Label>
            <Input
              id="advance-note"
              className="h-11"
              autoComplete="off"
              {...form.register('note')}
            />
            {form.formState.errors.note && (
              <p className="text-destructive text-sm">
                {form.formState.errors.note.message}
              </p>
            )}
          </div>

          {mutationError && (
            <p className="text-destructive text-sm" role="alert">
              {mutationError instanceof ApiError
                ? mutationError.message
                : 'Tidak dapat menyimpan kas bon.'}
            </p>
          )}

          <SheetFooter className="px-0">
            <Button
              type="submit"
              className="h-11 w-full"
              disabled={pending || amount <= 0}
            >
              {pending ? 'Menyimpan…' : 'Simpan'}
            </Button>

            {advance && (
              <Button
                type="button"
                variant="destructive"
                className="h-11 w-full"
                onClick={() => setConfirmDelete(true)}
              >
                Hapus kas bon
              </Button>
            )}
          </SheetFooter>
        </form>
      </SheetContent>

      <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus kas bon ini?</AlertDialogTitle>
            <AlertDialogDescription>
              Kas bon akan hilang dan tidak dipotong dari rekap.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onDelete} disabled={remove.isPending}>
              {remove.isPending ? 'Memproses…' : 'Hapus'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Sheet>
  )
}
