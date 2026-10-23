import { useEffect, useState } from 'react'

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
  useAddExtraFee,
  useDeleteExtraFee,
  useUpdateExtraFee,
} from '@/hooks/use-recaps'
import { ApiError } from '@/lib/api'
import { digitsOnly, formatThousands } from '@/lib/format'
import type { RecapExtraFee } from '@/lib/types'
import { cn } from '@/lib/utils'

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  recapId: number
  workerId: number
  fee?: RecapExtraFee | null
}

export function ExtraFeeSheet({ open, onOpenChange, recapId, workerId, fee }: Props) {
  const add = useAddExtraFee(recapId)
  const update = useUpdateExtraFee(recapId)
  const remove = useDeleteExtraFee(recapId)

  const [kind, setKind] = useState<'bonus' | 'potongan'>('bonus')
  const [amount, setAmount] = useState('')
  const [description, setDescription] = useState('')

  /* oxlint-disable react/set-state-in-effect -- reset the form when the sheet opens */
  useEffect(() => {
    if (!open) return
    if (fee) {
      setKind(fee.amount < 0 ? 'potongan' : 'bonus')
      setAmount(String(Math.abs(fee.amount)))
      setDescription(fee.description)
    } else {
      setKind('bonus')
      setAmount('')
      setDescription('')
    }
  }, [open, fee])
  /* oxlint-enable react/set-state-in-effect */

  const error = add.error ?? update.error ?? remove.error
  const pending = add.isPending || update.isPending || remove.isPending
  const signedAmount = Number(amount || '0') * (kind === 'potongan' ? -1 : 1)
  const invalid = signedAmount === 0 || description.trim() === ''

  function submit() {
    const payload = { amount: signedAmount, description: description.trim() }
    if (fee) {
      update.mutate({ feeId: fee.id, ...payload }, { onSuccess: () => onOpenChange(false) })
    } else {
      add.mutate({ worker_id: workerId, ...payload }, { onSuccess: () => onOpenChange(false) })
    }
  }

  function onDelete() {
    if (!fee) return
    remove.mutate(fee.id, { onSuccess: () => onOpenChange(false) })
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

        <div className="space-y-4 px-4">
          <div className="grid grid-cols-2 gap-2">
            {(['bonus', 'potongan'] as const).map((option) => (
              <button
                key={option}
                type="button"
                onClick={() => setKind(option)}
                className={cn(
                  'h-11 rounded-lg border text-sm font-medium capitalize',
                  kind === option ? 'border-primary bg-primary/5' : 'hover:bg-muted',
                )}
              >
                {option === 'bonus' ? 'Bonus' : 'Potongan'}
              </button>
            ))}
          </div>

          <div className="space-y-2">
            <Label htmlFor="fee-amount">Nominal</Label>
            <div className="relative">
              <span className="text-muted-foreground pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-sm">
                Rp
              </span>
              <Input
                id="fee-amount"
                className="h-11 pl-9"
                inputMode="numeric"
                value={formatThousands(amount)}
                onChange={(event) => setAmount(digitsOnly(event.target.value))}
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="fee-description">Keterangan</Label>
            <Input
              id="fee-description"
              className="h-11"
              autoComplete="off"
              placeholder="Lembur, denda, uang makan…"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
            />
          </div>

          {error && (
            <p className="text-destructive text-sm" role="alert">
              {error instanceof ApiError ? error.message : 'Tidak dapat menyimpan tambahan.'}
            </p>
          )}

          <SheetFooter className="px-0">
            <Button className="h-11 w-full" onClick={submit} disabled={pending || invalid}>
              {pending ? 'Menyimpan…' : 'Simpan'}
            </Button>
            {fee && (
              <Button
                variant="destructive"
                className="h-11 w-full"
                onClick={onDelete}
                disabled={pending}
              >
                Hapus tambahan
              </Button>
            )}
          </SheetFooter>
        </div>
      </SheetContent>
    </Sheet>
  )
}
