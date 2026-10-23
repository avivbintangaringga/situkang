import { Moon, Sun } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { useTheme } from '@/hooks/use-theme'

export function ThemeToggle() {
  const { theme, toggle } = useTheme()
  const dark = theme === 'dark'

  return (
    <Button
      variant="ghost"
      className="size-9 p-0"
      onClick={toggle}
      aria-label={dark ? 'Mode terang' : 'Mode gelap'}
      title={dark ? 'Mode terang' : 'Mode gelap'}
    >
      {dark ? (
        <Sun className="size-5" aria-hidden />
      ) : (
        <Moon className="size-5" aria-hidden />
      )}
    </Button>
  )
}
