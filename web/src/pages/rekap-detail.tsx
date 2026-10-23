import { ArrowLeft, Copy, FileSpreadsheet, Plus, Send, Share2 } from 'lucide-react'
import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router'

import { ExtraFeeSheet } from '@/components/extra-fee-sheet'
import { RecapShareSheet } from '@/components/recap-share-sheet'
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
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import {
  useDeleteRecap,
  useFinalizeRecap,
  useRecap,
  useReopenRecap,
  useSetRecapPaid,
} from '@/hooks/use-recaps'
import { api } from '@/lib/api'
import { formatDate, formatDays, formatRupiah } from '@/lib/format'
import type { RecapExtraFee } from '@/lib/types'
import { cn } from '@/lib/utils'

export function RekapDetailPage() {
  const params = useParams()
  const recapId = Number(params.id)
  const navigate = useNavigate()

  const { data, isPending, isError, error, refetch } = useRecap(recapId)
  const setPaid = useSetRecapPaid(recapId)
  const finalize = useFinalizeRecap(recapId)
  const reopen = useReopenRecap(recapId)
  const remove = useDeleteRecap(recapId)

  const [confirmFinal, setConfirmFinal] = useState(false)
  const [confirmReopen, setConfirmReopen] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [feeTarget, setFeeTarget] = useState<{ workerId: number; fee?: RecapExtraFee } | null>(
    null,
  )
  const [shareNote, setShareNote] = useState<string | null>(null)
  const [shareOpen, setShareOpen] = useState(false)

  async function whatsappText(workerId?: number) {
    const suffix = workerId ? `?worker_id=${workerId}` : ''
    return api.text(`/recaps/${recapId}/whatsapp${suffix}`)
  }

  async function copyText(workerId?: number) {
    try {
      const text = await whatsappText(workerId)
      await navigator.clipboard.writeText(text)
      setShareNote('Tersalin ke papan klip.')
    } catch {
      setShareNote('Gagal menyalin. Coba lagi.')
    }
  }

  async function sendWhatsApp(workerId?: number) {
    try {
      const text = await whatsappText(workerId)
      window.open(`https://wa.me/?text=${encodeURIComponent(text)}`, '_blank', 'noopener')
    } catch {
      setShareNote('Gagal membuka WhatsApp. Coba lagi.')
    }
  }

  const isDraft = data?.recap.status === 'draft'
  const actionError = finalize.error ?? reopen.error ?? remove.error ?? setPaid.error

  if (isPending) {
    return <Skeleton className="h-64 w-full rounded-xl" />
  }

  if (isError) {
    return (
      <div className="rounded-xl border p-6 text-center">
        <p className="text-sm">
          {error instanceof Error ? error.message : 'Tidak dapat memuat rekap.'}
        </p>
        <Button variant="outline" className="mt-4 h-9 px-5" onClick={() => void refetch()}>
          Coba lagi
        </Button>
      </div>
    )
  }

  if (!data) return null

  const { recap, workers, totals } = data

  return (
    <div className="space-y-4 pb-4">
      <Button variant="ghost" className="-ml-2 h-9 px-2" render={<Link to="/rekap" />}>
        <ArrowLeft className="size-4" aria-hidden />
        Rekap
      </Button>

      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-xl font-semibold tracking-tight">{recap.name}</h1>
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
          <p className="text-muted-foreground text-xs">#{recap.id}</p>
          <p className="text-muted-foreground text-sm">
            {formatDate(data.period_start)} - {formatDate(data.period_end)}
          </p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 rounded-xl border p-4 shadow-sm sm:grid-cols-4">
        <div>
          <p className="text-muted-foreground text-xs">Gaji kotor</p>
          <p className="font-medium">{formatRupiah(totals.gross)}</p>
        </div>
        <div>
          <p className="text-muted-foreground text-xs">Kas bon</p>
          <p className="font-medium">{formatRupiah(totals.advances)}</p>
        </div>
        <div>
          <p className="text-muted-foreground text-xs">Tambahan</p>
          <p className="font-medium">{formatRupiah(totals.extras)}</p>
        </div>
        <div>
          <p className="text-muted-foreground text-xs">Total dibayar</p>
          <p className="font-semibold">{formatRupiah(totals.net)}</p>
        </div>
      </div>

      <div className="flex flex-wrap gap-2">
        <Button
          variant="outline"
          className="h-9 flex-1"
          render={<a href={`/api/recaps/${recapId}/export.xlsx`} />}
        >
          <FileSpreadsheet className="size-4" aria-hidden />
          Excel
        </Button>
        <Button
          variant="outline"
          className="h-9 flex-1"
          onClick={() => void copyText()}
        >
          <Copy className="size-4" aria-hidden />
          Salin semua
        </Button>
        <Button
          variant="outline"
          className="h-9 flex-1"
          onClick={() => void sendWhatsApp()}
        >
          <Send className="size-4" aria-hidden />
          Kirim WA
        </Button>
      </div>

      {shareNote && (
        <p className="text-muted-foreground text-xs" role="status">
          {shareNote}
        </p>
      )}

      {actionError && (
        <p className="text-destructive text-sm" role="alert">
          {actionError instanceof Error ? actionError.message : 'Terjadi kesalahan.'}
        </p>
      )}

      {isDraft ? (
        <div className="flex gap-2">
          <Button
            className="h-9 flex-1"
            onClick={() => setConfirmFinal(true)}
            disabled={workers.length === 0 || finalize.isPending}
          >
            {finalize.isPending ? 'Memfinalkan…' : 'Finalkan rekap'}
          </Button>
          <Button
            variant="destructive"
            className="h-9 px-4"
            onClick={() => setConfirmDelete(true)}
          >
            Hapus
          </Button>
        </div>
      ) : (
        <div className="flex gap-2">
          <Button
            variant="outline"
            className="h-9 flex-1"
            onClick={() => setShareOpen(true)}
          >
            <Share2 className="size-4" aria-hidden />
            Bagikan
          </Button>
          <Button
            variant="outline"
            className="h-9 flex-1"
            onClick={() => setConfirmReopen(true)}
            disabled={reopen.isPending}
          >
            {reopen.isPending ? 'Membuka…' : 'Buka kembali rekap'}
          </Button>
        </div>
      )}

      {workers.length === 0 && (
        <div className="rounded-xl border p-8 text-center">
          <p className="font-medium">Tidak ada data pada periode ini.</p>
          <p className="text-muted-foreground mt-1 text-sm">
            Catat absensi atau kas bon terlebih dahulu.
          </p>
        </div>
      )}

      <ul className="space-y-3">
        {workers.map((line) => (
          <li key={line.worker_id} className="space-y-3 rounded-xl border p-4">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="truncate font-medium">{line.name}</p>
                <p className="text-muted-foreground truncate text-xs">{line.role}</p>
              </div>
              <div className="text-right">
                <p className="font-semibold">{formatRupiah(line.net)}</p>
                <p className="text-muted-foreground text-xs">
                  {formatDays(line.quarters)} hari
                </p>
              </div>
            </div>

            {line.warning > 0 && (
              <p className="rounded-lg bg-amber-50 p-2 text-xs text-amber-800 dark:bg-amber-500/10 dark:text-amber-200">
                Ada {line.warning} hari belum dibayar sebelum tanggal ini.
              </p>
            )}

            <div className="space-y-1 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Gaji kotor</span>
                <span>{formatRupiah(line.gross)}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Kas bon</span>
                <span>- {formatRupiah(line.advances)}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Tambahan</span>
                <span>{formatRupiah(line.extras)}</span>
              </div>
            </div>

            {line.extra_fees.length > 0 && (
              <ul className="space-y-1">
                {line.extra_fees.map((fee) => (
                  <li key={fee.id}>
                    <button
                      type="button"
                      disabled={!isDraft}
                      onClick={() => setFeeTarget({ workerId: line.worker_id, fee })}
                      className="flex w-full items-center justify-between rounded-lg border px-2 py-1.5 text-xs disabled:opacity-70"
                    >
                      <span className="truncate">{fee.description}</span>
                      <span className={cn(fee.amount < 0 && 'text-destructive')}>
                        {formatRupiah(fee.amount)}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            )}

            <div className="flex items-center justify-between gap-2 border-t pt-3">
              {isDraft ? (
                <label className="flex items-center gap-2 text-sm">
                  <Switch
                    checked={line.paid}
                    onCheckedChange={(paid) =>
                      setPaid.mutate({ workerId: line.worker_id, paid })
                    }
                    aria-label={`Dibayar ${line.name}`}
                  />
                  {line.paid ? 'Dibayar' : 'Belum dibayar'}
                </label>
              ) : (
                <span className="text-muted-foreground text-sm">Sudah dibayar</span>
              )}

              <div className="flex items-center gap-1">
                <Button
                  variant="ghost"
                  className="size-9 p-0"
                  aria-label={`Salin WhatsApp ${line.name}`}
                  onClick={() => void copyText(line.worker_id)}
                >
                  <Copy className="size-4" aria-hidden />
                </Button>
                <Button
                  variant="ghost"
                  className="size-9 p-0"
                  aria-label={`Kirim WhatsApp ${line.name}`}
                  onClick={() => void sendWhatsApp(line.worker_id)}
                >
                  <Send className="size-4" aria-hidden />
                </Button>
                {isDraft && (
                  <Button
                    variant="outline"
                    className="h-9 px-3 text-xs"
                    onClick={() => setFeeTarget({ workerId: line.worker_id })}
                  >
                    <Plus className="size-3.5" aria-hidden />
                    Tambahan
                  </Button>
                )}
              </div>
            </div>
          </li>
        ))}
      </ul>

      {feeTarget && (
        <ExtraFeeSheet
          open={Boolean(feeTarget)}
          onOpenChange={(open) => {
            if (!open) setFeeTarget(null)
          }}
          recapId={recapId}
          workerId={feeTarget.workerId}
          fee={feeTarget.fee}
        />
      )}

      <RecapShareSheet open={shareOpen} onOpenChange={setShareOpen} recapId={recapId} />

      <AlertDialog open={confirmFinal} onOpenChange={setConfirmFinal}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Finalkan rekap ini?</AlertDialogTitle>
            <AlertDialogDescription>
              Absensi dan kas bon yang sudah dibayar akan terkunci. Pekerja yang
              belum dibayar tetap terbawa ke rekap berikutnya.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => finalize.mutate(undefined, { onSuccess: () => setConfirmFinal(false) })}
            >
              Finalkan
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={confirmReopen} onOpenChange={setConfirmReopen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Buka kembali rekap ini?</AlertDialogTitle>
            <AlertDialogDescription>
              Data yang terkunci akan kembali bisa diubah. Hanya rekap final
              terakhir yang dapat dibuka kembali.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => reopen.mutate(undefined, { onSuccess: () => setConfirmReopen(false) })}
            >
              Buka kembali
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus draft rekap?</AlertDialogTitle>
            <AlertDialogDescription>
              Tambahan pada draft ini juga akan terhapus. Absensi dan kas bon
              tetap tersimpan.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction
              onClick={() =>
                remove.mutate(undefined, { onSuccess: () => navigate('/rekap') })
              }
            >
              Hapus
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
