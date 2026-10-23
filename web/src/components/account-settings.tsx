import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useMe } from '@/hooks/use-auth'
import { useChangePassword, useUpdateUsername } from '@/hooks/use-users'
import { ApiError } from '@/lib/api'

const usernameSchema = z.object({
  username: z
    .string()
    .trim()
    .min(3, 'Nama pengguna minimal 3 karakter.')
    .max(50, 'Nama pengguna maksimal 50 karakter.'),
})

type UsernameForm = z.infer<typeof usernameSchema>

const passwordSchema = z
  .object({
    current_password: z.string().min(1, 'Kata sandi saat ini wajib diisi.'),
    new_password: z.string().min(8, 'Kata sandi baru minimal 8 karakter.'),
    confirm_password: z.string().min(1, 'Ulangi kata sandi baru.'),
  })
  .refine((values) => values.new_password === values.confirm_password, {
    path: ['confirm_password'],
    message: 'Kata sandi baru tidak sama.',
  })

type PasswordForm = z.infer<typeof passwordSchema>

const emptyPassword: PasswordForm = {
  current_password: '',
  new_password: '',
  confirm_password: '',
}

export function AccountSettings() {
  const { data: user } = useMe()
  const updateUsername = useUpdateUsername()
  const changePassword = useChangePassword()

  const usernameForm = useForm<UsernameForm>({
    resolver: zodResolver(usernameSchema),
    defaultValues: { username: '' },
  })
  const passwordForm = useForm<PasswordForm>({
    resolver: zodResolver(passwordSchema),
    defaultValues: emptyPassword,
  })

  const [usernameNotice, setUsernameNotice] = useState('')
  const [passwordNotice, setPasswordNotice] = useState('')

  useEffect(() => {
    if (!user) return
    usernameForm.reset({ username: user.username })
  }, [user, usernameForm])

  function onSubmitUsername(values: UsernameForm) {
    setUsernameNotice('')
    updateUsername.mutate(values.username, {
      onSuccess: () => setUsernameNotice('Tersimpan.'),
    })
  }

  function onSubmitPassword(values: PasswordForm) {
    setPasswordNotice('')
    changePassword.mutate(
      {
        current_password: values.current_password,
        new_password: values.new_password,
      },
      {
        onSuccess: () => {
          passwordForm.reset(emptyPassword)
          setPasswordNotice('Kata sandi diperbarui.')
        },
      },
    )
  }

  return (
    <section className="space-y-3">
      <div>
        <h2 className="text-sm font-semibold">Akun</h2>
        <p className="text-muted-foreground text-sm">
          Ubah nama pengguna dan kata sandi akun Anda.
        </p>
      </div>

      <form
        onSubmit={usernameForm.handleSubmit(onSubmitUsername)}
        className="space-y-3 rounded-xl border p-4"
        noValidate
      >
        <div className="space-y-2">
          <Label htmlFor="account-username">Nama pengguna</Label>
          <Input
            id="account-username"
            className="h-9"
            autoComplete="username"
            aria-invalid={Boolean(usernameForm.formState.errors.username)}
            {...usernameForm.register('username')}
          />
          {usernameForm.formState.errors.username && (
            <p className="text-destructive text-sm">
              {usernameForm.formState.errors.username.message}
            </p>
          )}
        </div>

        {updateUsername.error && (
          <p className="text-destructive text-sm" role="alert">
            {updateUsername.error instanceof ApiError
              ? updateUsername.error.message
              : 'Tidak dapat mengubah nama pengguna.'}
          </p>
        )}

        {usernameNotice && (
          <p className="text-sm text-emerald-600 dark:text-emerald-400" role="status">
            {usernameNotice}
          </p>
        )}

        <Button
          type="submit"
          className="h-9 w-full"
          disabled={updateUsername.isPending}
        >
          {updateUsername.isPending ? 'Menyimpan…' : 'Simpan'}
        </Button>
      </form>

      <form
        onSubmit={passwordForm.handleSubmit(onSubmitPassword)}
        className="space-y-3 rounded-xl border p-4"
        noValidate
      >
        <div className="space-y-2">
          <Label htmlFor="account-current-password">Kata sandi saat ini</Label>
          <Input
            id="account-current-password"
            type="password"
            className="h-9"
            autoComplete="current-password"
            aria-invalid={Boolean(passwordForm.formState.errors.current_password)}
            {...passwordForm.register('current_password')}
          />
          {passwordForm.formState.errors.current_password && (
            <p className="text-destructive text-sm">
              {passwordForm.formState.errors.current_password.message}
            </p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="account-new-password">Kata sandi baru</Label>
          <Input
            id="account-new-password"
            type="password"
            className="h-9"
            autoComplete="new-password"
            aria-invalid={Boolean(passwordForm.formState.errors.new_password)}
            {...passwordForm.register('new_password')}
          />
          {passwordForm.formState.errors.new_password && (
            <p className="text-destructive text-sm">
              {passwordForm.formState.errors.new_password.message}
            </p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="account-confirm-password">Ulangi kata sandi baru</Label>
          <Input
            id="account-confirm-password"
            type="password"
            className="h-9"
            autoComplete="new-password"
            aria-invalid={Boolean(passwordForm.formState.errors.confirm_password)}
            {...passwordForm.register('confirm_password')}
          />
          {passwordForm.formState.errors.confirm_password && (
            <p className="text-destructive text-sm">
              {passwordForm.formState.errors.confirm_password.message}
            </p>
          )}
        </div>

        {changePassword.error && (
          <p className="text-destructive text-sm" role="alert">
            {changePassword.error instanceof ApiError
              ? changePassword.error.message
              : 'Tidak dapat mengubah kata sandi.'}
          </p>
        )}

        {passwordNotice && (
          <p className="text-sm text-emerald-600 dark:text-emerald-400" role="status">
            {passwordNotice}
          </p>
        )}

        <Button
          type="submit"
          className="h-9 w-full"
          disabled={changePassword.isPending}
        >
          {changePassword.isPending ? 'Menyimpan…' : 'Ubah kata sandi'}
        </Button>
      </form>
    </section>
  )
}
