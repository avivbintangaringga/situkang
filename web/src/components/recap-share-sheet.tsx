import { Copy, Link2, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'

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
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { useCreateRecapShare, useRevokeRecapShare } from '@/hooks/use-recap-share'
import { ApiError } from '@/lib/api'
import type { RecapShare } from '@/lib/types'

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  recapId: number
}

export function RecapShareSheet({ open, onOpenChange, recapId }: Props) {
  const create = useCreateRecapShare(recapId)
  const revoke = useRevokeRecapShare(recapId)
  const createShare = create.mutate

  const [link, setLink] = useState<RecapShare | null>(null)
  const [feedback, setFeedback] = useState<string | null>(null)
  const [confirmRevoke, setConfirmRevoke] = useState(false)

  /* oxlint-disable react/set-state-in-effect -- fetch or create the link when the sheet opens */
  useEffect(() => {
    if (!open) return
    setLink(null)
    setFeedback(null)
    createShare(undefined, { onSuccess: (data) => setLink(data) })
  }, [open, createShare])
  /* oxlint-enable react/set-state-in-effect */

  const fullUrl = link
    ? `${typeof window === 'undefined' ? '' : window.location.origin}${link.url}`
    : ''

  async function copyLink() {
    try {
      await navigator.clipboard.writeText(fullUrl)
      setFeedback('Tautan tersalin.')
    } catch {
      setFeedback('Gagal menyalin. Coba lagi.')
    }
  }

  function createLink() {
    setFeedback(null)
    createShare(undefined, { onSuccess: (data) => setLink(data) })
  }

  function revokeLink() {
    revoke.mutate(undefined, {
      onSuccess: () => {
        setLink(null)
        setFeedback('Tautan dicabut. Rekap ini tidak lagi bisa dibuka lewat tautan.')
        setConfirmRevoke(false)
      },
    })
  }

  const error = create.error ?? revoke.error

  return (
    <>
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent side="bottom" className="max-h-[90dvh] overflow-y-auto">
          <SheetHeader>
            <SheetTitle>Bagikan rekap</SheetTitle>
            <SheetDescription>
              Siapa pun yang memiliki tautan dapat melihat rekap ini tanpa masuk.
            </SheetDescription>
          </SheetHeader>

          <div className="space-y-4 px-4">
            {create.isPending && !link && (
              <div className="space-y-2" aria-hidden>
                <Skeleton className="h-9 w-full rounded-lg" />
                <Skeleton className="h-9 w-full rounded-lg" />
              </div>
            )}

            {!create.isPending && !link && (
              <div className="space-y-3 rounded-xl border p-4 text-center">
                <Link2 className="text-muted-foreground mx-auto size-6" aria-hidden />
                <div>
                  <p className="font-medium">Belum ada tautan aktif</p>
                  <p className="text-muted-foreground mt-1 text-sm">
                    Buat tautan untuk membagikan rekap final ini.
                  </p>
                </div>
                <Button className="h-9 w-full" onClick={createLink} disabled={create.isPending}>
                  Buat tautan
                </Button>
              </div>
            )}

            {link && (
              <div className="space-y-3">
                <div className="space-y-2">
                  <p className="text-sm font-medium">Tautan publik</p>
                  <Input
                    readOnly
                    className="h-9 text-xs"
                    value={fullUrl}
                    aria-label="Tautan publik rekap"
                    onFocus={(event) => event.currentTarget.select()}
                  />
                </div>
                <div className="flex gap-2">
                  <Button className="h-9 flex-1" onClick={() => void copyLink()}>
                    <Copy className="size-4" aria-hidden />
                    Salin tautan
                  </Button>
                  <Button
                    variant="destructive"
                    className="h-9 px-4"
                    onClick={() => setConfirmRevoke(true)}
                    disabled={revoke.isPending}
                  >
                    <Trash2 className="size-4" aria-hidden />
                    Cabut
                  </Button>
                </div>
              </div>
            )}

            {feedback && (
              <p className="text-muted-foreground text-xs" role="status">
                {feedback}
              </p>
            )}

            {error && (
              <p className="text-destructive text-sm" role="alert">
                {error instanceof ApiError ? error.message : 'Tidak dapat membuat tautan.'}
              </p>
            )}
          </div>

          <SheetFooter>
            <Button variant="ghost" className="h-9 w-full" onClick={() => onOpenChange(false)}>
              Tutup
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <AlertDialog open={confirmRevoke} onOpenChange={setConfirmRevoke}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Cabut tautan ini?</AlertDialogTitle>
            <AlertDialogDescription>
              Tautan yang sudah dibagikan tidak bisa dibuka lagi. Kamu dapat membuat
              tautan baru kapan saja.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={revokeLink}>Cabut tautan</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
