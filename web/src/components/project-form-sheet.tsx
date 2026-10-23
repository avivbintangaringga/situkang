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
  useCreateProject,
  useDeleteProject,
  useUpdateProject,
} from '@/hooks/use-projects'
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
}

function valuesFor(project?: Project | null): ProjectForm {
  if (!project) return emptyValues
  return {
    name: project.name,
    status: project.status,
    start_date: project.start_date ?? '',
    end_date: project.end_date ?? '',
    notes: project.notes,
  }
}

export function ProjectFormSheet({ open, onOpenChange, project }: Props) {
  const isEdit = Boolean(project)
  const create = useCreateProject()
  const update = useUpdateProject(project?.id ?? 0)
  const remove = useDeleteProject()
  const [confirmDelete, setConfirmDelete] = useState(false)

  const form = useForm<ProjectForm>({
    resolver: zodResolver(projectSchema),
    defaultValues: valuesFor(project),
  })

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
    }

    if (project) {
      update.mutate(input, { onSuccess: () => onOpenChange(false) })
    } else {
      create.mutate(input, { onSuccess: () => onOpenChange(false) })
    }
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
            Proyek hanya menjadi tanda pada absensi. Upah tukang tidak berubah.
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
              className="h-11"
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
                      'h-11 rounded-lg border text-sm font-medium transition-colors',
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
              <Input
                id="project-start"
                type="date"
                className="h-11"
                {...form.register('start_date')}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="project-end">Selesai</Label>
              <Input
                id="project-end"
                type="date"
                className="h-11"
                aria-invalid={Boolean(form.formState.errors.end_date)}
                {...form.register('end_date')}
              />
            </div>
          </div>
          {form.formState.errors.end_date && (
            <p className="text-destructive text-sm">
              {form.formState.errors.end_date.message}
            </p>
          )}

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
            <Button type="submit" className="h-11 w-full" disabled={pending}>
              {pending ? 'Menyimpan…' : 'Simpan'}
            </Button>

            {isEdit && (
              <Button
                type="button"
                variant="destructive"
                className="h-11 w-full"
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
              Proyek yang sudah ditandai di absensi tidak dapat dihapus. Lepas
              tanda proyeknya terlebih dahulu.
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
