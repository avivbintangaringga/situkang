import { Navigate, Outlet, useLocation } from 'react-router'

import { Button } from '@/components/ui/button'
import { useMe } from '@/hooks/use-auth'

export function RequireAuth() {
  const { data, isPending, isError, error, refetch } = useMe()
  const location = useLocation()

  if (isPending) {
    return (
      <div className="bg-background text-muted-foreground flex min-h-dvh items-center justify-center text-sm">
        Memuat…
      </div>
    )
  }

  if (isError) {
    return (
      <div className="bg-background flex min-h-dvh flex-col items-center justify-center gap-4 px-6 text-center">
        <p className="text-sm">
          {error instanceof Error ? error.message : 'Tidak dapat menghubungi server.'}
        </p>
        <Button variant="outline" className="h-9 px-6" onClick={() => void refetch()}>
          Coba lagi
        </Button>
      </div>
    )
  }

  if (!data) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }

  return <Outlet />
}
