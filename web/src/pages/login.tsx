import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { Navigate, useLocation, useNavigate } from 'react-router'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useLogin, useMe } from '@/hooks/use-auth'
import { ApiError } from '@/lib/api'

const loginSchema = z.object({
  username: z.string().trim().min(1, 'Nama pengguna wajib diisi.'),
  password: z.string().min(1, 'Kata sandi wajib diisi.'),
})

type LoginForm = z.infer<typeof loginSchema>

export function LoginPage() {
  const { data: user, isPending } = useMe()
  const login = useLogin()
  const navigate = useNavigate()
  const location = useLocation()

  const form = useForm<LoginForm>({
    resolver: zodResolver(loginSchema),
    defaultValues: { username: '', password: '' },
  })

  if (!isPending && user) {
    return <Navigate to="/" replace />
  }

  const from = (location.state as { from?: string } | null)?.from ?? '/'

  function onSubmit(values: LoginForm) {
    login.mutate(values, {
      onSuccess: () => navigate(from, { replace: true }),
    })
  }

  const errorMessage =
    login.error instanceof ApiError
      ? login.error.message
      : login.error
        ? 'Tidak dapat menghubungi server.'
        : null

  return (
    <main className="bg-background flex min-h-dvh flex-col items-center justify-center px-4 py-10">
      <div className="w-full max-w-sm space-y-6">
        <div className="space-y-1 text-center">
          <h1 className="text-2xl font-semibold tracking-tight">SiTukang</h1>
          <p className="text-muted-foreground text-sm">Sistem Informasi Tukang</p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Masuk</CardTitle>
            <CardDescription>Gunakan akun admin atau pengawas.</CardDescription>
          </CardHeader>
          <form onSubmit={form.handleSubmit(onSubmit)} noValidate>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="username">Nama pengguna</Label>
                <Input
                  id="username"
                  className="h-11"
                  autoComplete="username"
                  autoCapitalize="none"
                  spellCheck={false}
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
                <Label htmlFor="password">Kata sandi</Label>
                <Input
                  id="password"
                  type="password"
                  className="h-11"
                  autoComplete="current-password"
                  aria-invalid={Boolean(form.formState.errors.password)}
                  {...form.register('password')}
                />
                {form.formState.errors.password && (
                  <p className="text-destructive text-sm">
                    {form.formState.errors.password.message}
                  </p>
                )}
              </div>

              {errorMessage && (
                <p className="text-destructive text-sm" role="alert">
                  {errorMessage}
                </p>
              )}
            </CardContent>

            <CardFooter className="pt-4">
              <Button type="submit" className="h-11 w-full" disabled={login.isPending}>
                {login.isPending ? 'Memproses…' : 'Masuk'}
              </Button>
            </CardFooter>
          </form>
        </Card>
      </div>
    </main>
  )
}
