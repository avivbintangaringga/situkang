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
import { DatePicker } from '@/components/date-picker'
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
  useCreateExtraFee,
  useDeleteExtraFee,
  useUpdateExtraFee,
} from '@/hooks/use-extra-fees'
import { ApiError } from '@/lib/api'
import { digitsOnly, formatThousands, todayISO } from '@/lib/format'
import type { ExtraFee } from '@/lib/types'
import { cn } from '@/lib/utils'

const schema = z.object({
  date: z.string().min(1, 'Tanggal wajib diisi.'),
  amount: z
    .string()
    .min(1, 'Nominal wajib diisi.')
    .refine((value) => Number(value) > 0, 'Nominal harus lebih dari nol.'),
  description: z
    .string()
    .trim()
    .min(1, 'Keterangan wajib diisi.')
    .max(200, 'Keterangan maksimal 200 karakter.'),
})

type Form = z.infer<typeof schema>

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  workerId: number
  fee?: ExtraFee | null
  defaultDate?: string
}

function valuesFor(fee?: ExtraFee | null, defaultDate?: string): Form {
  if (!fee) {
    return { date: defaultDate ?? todayISO(), amount: '', description: '' }
  }
  return {
    date: fee.date,
    amount: String(Math.abs(fee.amount)),
    description: fee.description,
  }
}

export function DayExtraFeeSheet({
  open,
  onOpenChange,
  workerId,
  fee,
  defaultDate,
}: Props) {
  const create = useCreateExtraFee(workerId)
  const update = useUpdateExtraFee(fee?.id ?? 0, workerId)
  const remove = useDeleteExtraFee(workerId)
  const [kind, setKind] = useState<'bonus' | 'potongan'>('bonus')
  const [confirmDelete, setConfirmDelete] = useState(false)

  const form = useForm<Form>({
    resolver: zodResolver(schema),
    defaultValues: valuesFor(fee, defaultDate),
  })

  /* oxlint-disable react/set-state-in-effect -- reset the form when the sheet opens */
  useEffect(() => {
    if (!open) return
    form.reset(valuesFor(fee, defaultDate))
    setKind(fee && fee.amount < 0 ? 'potongan' : 'bonus')
    setConfirmDelete(false)
  }, [open, fee, defaultDate, form])
  /* oxlint-enable react/set-state-in-effect */

  const pending = create.isPending || update.isPending
  const mutationError = create.error ?? update.error
  const signedAmount = Number(form.watch('amount') || '0') * (kind === 'potongan' ? -1 : 1)

  function onSubmit(values: Form) {
    const input = {
      date: values.date,
      amount: signedAmount,
      description: values.description.trim(),
    }

    if (fee) {
      update.mutate(input, { onSuccess: () => onOpenChange(false) })
    } else {
      create.mutate(input, { onSuccess: () => onOpenChange(false) })
    }
  }

  function onDelete() {
    if (!fee) return
    remove.mutate(fee.id, {
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
          <SheetTitle>{fee ? 'Ubah tambahan' : 'Tambah tambahan'}</SheetTitle>
          <SheetDescription>
            Bonus menambah upah, potongan mengurangi upah.
          </SheetDescription>
        </SheetHeader>

        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 px-4" noValidate>
          <div className="grid grid-cols-2 gap-2">
            {(['bonus', 'potongan'] as const).map((option) => (
              <button
                key={option}
                type="button"
                onClick={() => setKind(option)}
                className={cn(
                  'h-9 rounded-lg border text-sm font-medium',
                  kind === option ? 'border-primary bg-primary/5' : 'hover:bg-muted',
                )}
              >
                {option === 'bonus' ? 'Bonus' : 'Potongan'}
              </button>
            ))}
          </div>

          <div className="space-y-2">
            <Label htmlFor="extra-date">Tanggal</Label>
            <DatePicker
              id="extra-date"
              value={form.watch('date')}
              onChange={(value) =>
                form.setValue('date', value, { shouldValidate: true })
              }
            />
            {form.formState.errors.date && (
              <p className="text-destructive text-sm">
                {form.formState.errors.date.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="extra-amount">Nominal</Label>
            <div className="relative">
              <span className="text-muted-foreground pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-sm">
                Rp
              </span>
              <Input
                id="extra-amount"
                className="h-9 pl-9"
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
          </div>

          <div className="space-y-2">
            <Label htmlFor="extra-description">Keterangan</Label>
            <Input
              id="extra-description"
              className="h-9"
              autoComplete="off"
              placeholder="Lembur, denda, uang makan…"
              {...form.register('description')}
            />
            {form.formState.errors.description && (
              <p className="text-destructive text-sm">
                {form.formState.errors.description.message}
              </p>
            )}
          </div>

          {mutationError && (
            <p className="text-destructive text-sm" role="alert">
              {mutationError instanceof ApiError
                ? mutationError.message
                : 'Tidak dapat menyimpan tambahan.'}
            </p>
          )}

          <SheetFooter className="px-0">
            <Button type="submit" className="h-9 w-full" disabled={pending}>
              {pending ? 'Menyimpan…' : 'Simpan'}
            </Button>

            {fee && (
              <Button
                type="button"
                variant="destructive"
                className="h-9 w-full"
                onClick={() => setConfirmDelete(true)}
              >
                Hapus tambahan
              </Button>
            )}
          </SheetFooter>
        </form>
      </SheetContent>

      <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus tambahan ini?</AlertDialogTitle>
            <AlertDialogDescription>
              Tambahan akan hilang dan tidak dihitung pada rekap.
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
