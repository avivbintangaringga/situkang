import { zodResolver } from '@hookform/resolvers/zod'
import { Check } from 'lucide-react'
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
  useCreateProject,
  useDeleteProject,
  useUpdateProject,
} from '@/hooks/use-projects'
import { useWorkers } from '@/hooks/use-workers'
import { ApiError } from '@/lib/api'
import type { Project, ProjectStatus } from '@/lib/types'
import { cn } from '@/lib/utils'

const statuses: { value: ProjectStatus; label: string }[] = [
  { value: 'aktif', label: 'Aktif' },
  { value: 'selesai', label: 'Selesai' },
  { value: 'ditunda', label: 'Ditunda' },
]

const projectSchema = z
  .object({
    name: z
      .string()
      .trim()
      .min(1, 'Nama proyek wajib diisi.')
      .max(100, 'Nama maksimal 100 karakter.'),
    status: z.enum(['aktif', 'selesai', 'ditunda']),
    start_date: z.string(),
    end_date: z.string(),
    notes: z.string().max(500, 'Catatan maksimal 500 karakter.'),
    worker_ids: z.array(z.number()),
  })
  .refine(
    (values) =>
      !values.start_date || !values.end_date || values.end_date >= values.start_date,
    { path: ['end_date'], message: 'Tanggal selesai tidak boleh sebelum tanggal mulai.' },
  )

type ProjectForm = z.infer<typeof projectSchema>

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  project?: Project | null
}

const emptyValues: ProjectForm = {
  name: '',
  status: 'aktif',
  start_date: '',
  end_date: '',
  notes: '',
  worker_ids: [],
}

function valuesFor(project?: Project | null): ProjectForm {
  if (!project) return emptyValues
  return {
    name: project.name,
    status: project.status,
    start_date: project.start_date ?? '',
    end_date: project.end_date ?? '',
    notes: project.notes,
    worker_ids: project.worker_ids,
  }
}

export function ProjectFormSheet({ open, onOpenChange, project }: Props) {
  const isEdit = Boolean(project)
  const create = useCreateProject()
  const update = useUpdateProject(project?.id ?? 0)
  const remove = useDeleteProject()
  const workers = useWorkers('', false)
  const [confirmDelete, setConfirmDelete] = useState(false)

  const form = useForm<ProjectForm>({
    resolver: zodResolver(projectSchema),
    defaultValues: valuesFor(project),
  })
  const selectedWorkerIds = form.watch('worker_ids')

  useEffect(() => {
    if (!open) return
    form.reset(valuesFor(project))
    setConfirmDelete(false)
  }, [open, project, form])

  const pending = create.isPending || update.isPending
  const mutationError = create.error ?? update.error

  function onSubmit(values: ProjectForm) {
    const input = {
      name: values.name,
      status: values.status,
      start_date: values.start_date,
      end_date: values.end_date,
      notes: values.notes,
      worker_ids: values.worker_ids,
    }

    if (project) {
      update.mutate(input, { onSuccess: () => onOpenChange(false) })
    } else {
      create.mutate(input, { onSuccess: () => onOpenChange(false) })
    }
  }

  function toggleWorker(workerId: number) {
    const current = form.getValues('worker_ids')
    const next = current.includes(workerId)
      ? current.filter((id) => id !== workerId)
      : [...current, workerId]
    form.setValue('worker_ids', next, { shouldValidate: true })
  }

  function onDelete() {
    if (!project) return
    remove.mutate(project.id, {
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
          <SheetTitle>{isEdit ? 'Ubah proyek' : 'Tambah proyek'}</SheetTitle>
          <SheetDescription>
            Pilih tukang yang bekerja di proyek ini. Upah tukang tidak berubah.
          </SheetDescription>
        </SheetHeader>

        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="flex flex-col gap-4 px-4"
          noValidate
        >
          <div className="space-y-2">
            <Label htmlFor="project-name">Nama</Label>
            <Input
              id="project-name"
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
            <Label>Status</Label>
            <div className="grid grid-cols-3 gap-2">
              {statuses.map((option) => {
                const active = form.watch('status') === option.value
                return (
                  <button
                    key={option.value}
                    type="button"
                    aria-pressed={active}
                    onClick={() =>
                      form.setValue('status', option.value, { shouldValidate: true })
                    }
                    className={cn(
                      'h-9 rounded-lg border text-sm font-medium transition-colors',
                      active ? 'border-primary bg-primary/5' : 'hover:bg-muted',
                    )}
                  >
                    {option.label}
                  </button>
                )
              })}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <Label htmlFor="project-start">Mulai</Label>
              <DatePicker
                id="project-start"
                value={form.watch('start_date')}
                onChange={(value) =>
                  form.setValue('start_date', value, { shouldValidate: true })
                }
                onClear={() => form.setValue('start_date', '')}
                placeholder="Mulai"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="project-end">Selesai</Label>
              <DatePicker
                id="project-end"
                value={form.watch('end_date')}
                onChange={(value) =>
                  form.setValue('end_date', value, { shouldValidate: true })
                }
                onClear={() => form.setValue('end_date', '')}
                placeholder="Selesai"
              />
            </div>
          </div>
          {form.formState.errors.end_date && (
            <p className="text-destructive text-sm">
              {form.formState.errors.end_date.message}
            </p>
          )}

          <div className="space-y-2">
            <Label>Tukang</Label>
            <p className="text-muted-foreground text-xs">
              Pilih tukang yang ditugaskan ke proyek ini.
            </p>

            {workers.isPending && (
              <p className="text-muted-foreground text-sm">Memuat tukang…</p>
            )}

            {workers.isError && (
              <div className="rounded-lg border p-4 text-center">
                <p className="text-sm">Tidak dapat memuat daftar tukang.</p>
                <Button
                  type="button"
                  variant="outline"
                  className="mt-2 h-9 px-4"
                  onClick={() => void workers.refetch()}
                >
                  Coba lagi
                </Button>
              </div>
            )}

            {workers.data && workers.data.length === 0 && (
              <div className="rounded-lg border p-4 text-center">
                <p className="text-sm">Belum ada tukang aktif.</p>
                <p className="text-muted-foreground mt-1 text-xs">
                  Tambahkan tukang dulu di menu Pekerja.
                </p>
              </div>
            )}

            {workers.data && workers.data.length > 0 && (
              <ul className="space-y-2">
                {workers.data.map((worker) => {
                  const active = selectedWorkerIds.includes(worker.id)
                  return (
                    <li key={worker.id}>
                      <button
                        type="button"
                        aria-pressed={active}
                        onClick={() => toggleWorker(worker.id)}
                        className={cn(
                          'flex h-9 w-full items-center justify-between gap-3 rounded-lg border px-3 text-left text-sm transition-colors',
                          active ? 'border-primary bg-primary/5' : 'hover:bg-muted',
                        )}
                      >
                        <span className="min-w-0">
                          <span className="block truncate">{worker.name}</span>
                          <span className="text-muted-foreground block truncate text-xs">
                            {worker.role}
                          </span>
                        </span>
                        <span
                          className={cn(
                            'flex size-5 shrink-0 items-center justify-center rounded border',
                            active ? 'bg-primary text-primary-foreground border-primary' : '',
                          )}
                        >
                          {active && <Check className="size-3.5" aria-hidden />}
                        </span>
                      </button>
                    </li>
                  )
                })}
              </ul>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="project-notes">Catatan</Label>
            <textarea
              id="project-notes"
              rows={3}
              className="border-input focus-visible:border-ring focus-visible:ring-ring/50 w-full rounded-lg border bg-transparent px-3 py-2 text-sm outline-none focus-visible:ring-[3px]"
              placeholder="Opsional"
              {...form.register('notes')}
            />
            {form.formState.errors.notes && (
              <p className="text-destructive text-sm">
                {form.formState.errors.notes.message}
              </p>
            )}
          </div>

          {mutationError && (
            <p className="text-destructive text-sm" role="alert">
              {mutationError instanceof ApiError
                ? mutationError.message
                : 'Tidak dapat menyimpan proyek.'}
            </p>
          )}

          <SheetFooter className="px-0">
            <Button type="submit" className="h-9 w-full" disabled={pending}>
              {pending ? 'Menyimpan…' : 'Simpan'}
            </Button>

            {isEdit && (
              <Button
                type="button"
                variant="destructive"
                className="h-9 w-full"
                onClick={() => setConfirmDelete(true)}
              >
                Hapus proyek
              </Button>
            )}
          </SheetFooter>
        </form>
      </SheetContent>

      <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus proyek ini?</AlertDialogTitle>
            <AlertDialogDescription>
              Proyek akan dihapus beserta daftar tukangnya. Catatan absensi
              tidak berubah.
            </AlertDialogDescription>
          </AlertDialogHeader>
          {remove.error && (
            <p className="text-destructive text-sm" role="alert">
              {remove.error instanceof ApiError
                ? remove.error.message
                : 'Tidak dapat menghapus proyek.'}
            </p>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={onDelete} disabled={remove.isPending}>
              {remove.isPending ? 'Menghapus…' : 'Hapus'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Sheet>
  )
}
