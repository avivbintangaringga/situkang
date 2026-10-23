import { Check } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Link } from 'react-router'

import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import type { Project } from '@/lib/types'
import { cn } from '@/lib/utils'

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  workerName: string
  projects: Project[]
  selected: number[]
  onSave: (projectIds: number[]) => void
}

export function AttendanceProjectSheet({
  open,
  onOpenChange,
  workerName,
  projects,
  selected,
  onSave,
}: Props) {
  const [picked, setPicked] = useState<number[]>(selected)

  useEffect(() => {
    if (!open) return
    // oxlint-disable-next-line react/set-state-in-effect -- copy the saved tags when the sheet opens
    setPicked(selected)
  }, [open, selected])

  function toggle(id: number) {
    setPicked((current) =>
      current.includes(id) ? current.filter((value) => value !== id) : [...current, id],
    )
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="bottom" className="max-h-[90dvh] overflow-y-auto">
        <SheetHeader>
          <SheetTitle>Proyek {workerName}</SheetTitle>
          <SheetDescription>
            Tandai proyek tempat tukang ini bekerja pada tanggal tersebut.
          </SheetDescription>
        </SheetHeader>

        <div className="space-y-2 px-4">
          {projects.length === 0 && (
            <div className="rounded-xl border p-6 text-center">
              <p className="text-sm">Belum ada proyek.</p>
              <p className="text-muted-foreground mt-1 text-xs">
                Tambahkan proyek dulu untuk bisa menandainya di absensi.
              </p>
              <Button className="mt-3 h-11 px-4" render={<Link to="/proyek" />}>
                Buka proyek
              </Button>
            </div>
          )}

          {projects.map((project) => {
            const active = picked.includes(project.id)
            return (
              <button
                key={project.id}
                type="button"
                aria-pressed={active}
                onClick={() => toggle(project.id)}
                className={cn(
                  'flex h-11 w-full items-center justify-between gap-3 rounded-lg border px-3 text-left text-sm transition-colors',
                  active ? 'border-primary bg-primary/5' : 'hover:bg-muted',
                )}
              >
                <span className="min-w-0 truncate">{project.name}</span>
                <span
                  className={cn(
                    'flex size-5 shrink-0 items-center justify-center rounded border',
                    active ? 'bg-primary text-primary-foreground border-primary' : '',
                  )}
                >
                  {active && <Check className="size-3.5" aria-hidden />}
                </span>
              </button>
            )
          })}
        </div>

        <SheetFooter>
          <Button
            className="h-11 w-full"
            onClick={() => {
              onSave(picked)
              onOpenChange(false)
            }}
          >
            Simpan
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
