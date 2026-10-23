import { Building2, ClipboardCheck, Home, Receipt, Settings, Users } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { Link, NavLink, Outlet, useNavigate } from 'react-router'

import { ThemeToggle } from '@/components/theme-toggle'
import { Button } from '@/components/ui/button'
import { useLogout, useMe } from '@/hooks/use-auth'
import { cn } from '@/lib/utils'

type NavItem = {
  to: string
  label: string
  icon: LucideIcon
  end?: boolean
}

// Absensi, Rekap, Proyek, and Pekerja make up the five-item navigation.
const navItems: NavItem[] = [
  { to: '/', label: 'Beranda', icon: Home, end: true },
  { to: '/attendance', label: 'Absensi', icon: ClipboardCheck },
  { to: '/recaps', label: 'Rekap', icon: Receipt },
  { to: '/projects', label: 'Proyek', icon: Building2 },
  { to: '/workers', label: 'Pekerja', icon: Users },
]

function navLinkClass(compact: boolean) {
  return ({ isActive }: { isActive: boolean }) =>
    cn(
      'flex items-center gap-3 rounded-full px-3 text-sm font-medium transition-colors',
      compact
        ? 'h-14 flex-1 flex-col justify-center gap-1 rounded-none text-xs'
        : 'h-9',
      isActive
        ? 'bg-muted text-foreground'
        : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground',
    )
}

export function AppShell() {
  const { data: user } = useMe()
  const logout = useLogout()
  const navigate = useNavigate()

  function onLogout() {
    logout.mutate(undefined, {
      onSettled: () => navigate('/login', { replace: true }),
    })
  }

  return (
    <div className="min-h-dvh md:flex">
      <aside className="bg-background hidden md:flex md:w-56 md:shrink-0 md:flex-col md:border-r">
        <div className="flex h-14 items-center gap-2 px-4">
          <img src="/logo.svg" alt="" className="size-7" />
          <span className="text-lg font-semibold">SiTukang</span>
        </div>
        <nav className="flex flex-1 flex-col gap-1 p-2">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={navLinkClass(false)}
            >
              <item.icon className="size-5" aria-hidden />
              {item.label}
            </NavLink>
          ))}
        </nav>
      </aside>

      <div className="flex min-h-dvh flex-1 flex-col">
        <header className="bg-background/90 sticky top-0 z-30 border-b shadow-sm backdrop-blur">
          <div className="mx-auto flex h-14 w-full max-w-3xl items-center justify-between gap-3 px-4">
            <span className="flex min-w-0 items-center gap-2 md:hidden">
              <img src="/logo.svg" alt="" className="size-7" />
              <span className="truncate text-lg font-semibold">SiTukang</span>
            </span>
            <div className="hidden md:block" />
            <div className="flex min-w-0 items-center gap-2">
              <span className="text-muted-foreground hidden max-w-[10rem] truncate text-xs sm:inline">
                {user?.username}
              </span>
              <ThemeToggle />
              <Button
                variant="ghost"
                className="size-9 p-0"
                aria-label="Pengaturan"
                render={<Link to="/settings" />}
              >
                <Settings className="size-5" aria-hidden />
              </Button>
              <Button
                variant="ghost"
                className="h-9 px-3"
                onClick={onLogout}
                disabled={logout.isPending}
              >
                {logout.isPending ? 'Keluar…' : 'Keluar'}
              </Button>
            </div>
          </div>
        </header>

        <main className="mx-auto w-full max-w-3xl flex-1 px-4 py-4 pb-24 md:pb-8">
          <Outlet />
        </main>

        <nav className="bg-background/95 fixed inset-x-0 bottom-0 z-40 flex border-t pb-[env(safe-area-inset-bottom)] shadow-[0_-1px_12px_rgb(0_0_0_/_0.06)] backdrop-blur md:hidden">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={navLinkClass(true)}
            >
              <item.icon className="size-5" aria-hidden />
              {item.label}
            </NavLink>
          ))}
        </nav>
      </div>
    </div>
  )
}
