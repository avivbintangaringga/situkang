import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

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
import { useCreateUser } from '@/hooks/use-users'
import { ApiError } from '@/lib/api'

const userSchema = z
  .object({
    username: z
      .string()
      .trim()
      .min(3, 'Nama pengguna minimal 3 karakter.')
      .max(50, 'Nama pengguna maksimal 50 karakter.'),
    password: z.string().min(8, 'Kata sandi minimal 8 karakter.'),
    confirm_password: z.string().min(1, 'Ulangi kata sandi.'),
  })
  .refine((values) => values.password === values.confirm_password, {
    path: ['confirm_password'],
    message: 'Kata sandi tidak sama.',
  })

type UserForm = z.infer<typeof userSchema>

const emptyValues: UserForm = {
  username: '',
  password: '',
  confirm_password: '',
}

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function UserFormSheet({ open, onOpenChange }: Props) {
  const create = useCreateUser()
  const form = useForm<UserForm>({
    resolver: zodResolver(userSchema),
    defaultValues: emptyValues,
  })

  function handleOpenChange(next: boolean) {
    if (next) {
      form.reset(emptyValues)
      create.reset()
    }
    onOpenChange(next)
  }

  function onSubmit(values: UserForm) {
    create.mutate(
      { username: values.username, password: values.password },
      { onSuccess: () => onOpenChange(false) },
    )
  }

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetContent side="bottom" className="max-h-[90dvh] overflow-y-auto">
        <SheetHeader>
          <SheetTitle>Tambah pengguna</SheetTitle>
          <SheetDescription>
            Akun baru bisa masuk dan mengelola absensi serta rekap gaji.
          </SheetDescription>
        </SheetHeader>

        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="flex flex-col gap-4 px-4"
          noValidate
        >
          <div className="space-y-2">
            <Label htmlFor="user-username">Nama pengguna</Label>
            <Input
              id="user-username"
              className="h-9"
              autoComplete="off"
              aria-invalid={Boolean(form.formState.errors.username)}
              {...form.register('username')}
            />
            {form.formState.errors.username && (
              <p className="text-destructive text-sm">
                {form.formState.errors.username.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="user-password">Kata sandi</Label>
            <Input
              id="user-password"
              type="password"
              className="h-9"
              autoComplete="new-password"
              aria-invalid={Boolean(form.formState.errors.password)}
              {...form.register('password')}
            />
            {form.formState.errors.password && (
              <p className="text-destructive text-sm">
                {form.formState.errors.password.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="user-confirm-password">Ulangi kata sandi</Label>
            <Input
              id="user-confirm-password"
              type="password"
              className="h-9"
              autoComplete="new-password"
              aria-invalid={Boolean(form.formState.errors.confirm_password)}
              {...form.register('confirm_password')}
            />
            {form.formState.errors.confirm_password && (
              <p className="text-destructive text-sm">
                {form.formState.errors.confirm_password.message}
              </p>
            )}
          </div>

          {create.error && (
            <p className="text-destructive text-sm" role="alert">
              {create.error instanceof ApiError
                ? create.error.message
                : 'Tidak dapat menambah pengguna.'}
            </p>
          )}

          <SheetFooter className="px-0">
            <Button type="submit" className="h-9 w-full" disabled={create.isPending}>
              {create.isPending ? 'Menyimpan…' : 'Simpan'}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}
