import { format, parseISO } from 'date-fns'
import { id as localeId } from 'date-fns/locale'
import { CalendarDays } from 'lucide-react'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { formatDate } from '@/lib/format'
import { cn } from '@/lib/utils'

type Props = {
  value?: string | null
  onChange: (value: string) => void
  onClear?: () => void
  placeholder?: string
  id?: string
  disabled?: boolean
  className?: string
}

export function DatePicker({
  value,
  onChange,
  onClear,
  placeholder = 'Pilih tanggal',
  id,
  disabled,
  className,
}: Props) {
  const [open, setOpen] = useState(false)
  const selected = value ? parseISO(value) : undefined

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            id={id}
            type="button"
            variant="outline"
            disabled={disabled}
            className={cn(
              'h-9 w-full justify-start gap-2 rounded-full px-3 text-left font-normal',
              className,
            )}
          >
            <CalendarDays className="size-4 shrink-0" aria-hidden />
            <span className={cn('truncate', !value && 'text-muted-foreground')}>
              {value ? formatDate(value) : placeholder}
            </span>
          </Button>
        }
      />
      <PopoverContent className="w-auto p-0" align="start">
        <Calendar
          mode="single"
          selected={selected}
          onSelect={(date) => {
            if (!date) return
            onChange(format(date, 'yyyy-MM-dd'))
            setOpen(false)
          }}
          locale={localeId}
          weekStartsOn={1}
          autoFocus
        />
        {value && onClear && (
          <div className="border-t p-2">
            <Button
              type="button"
              variant="ghost"
              className="h-8 w-full rounded-full text-xs"
              onClick={() => {
                onClear()
                setOpen(false)
              }}
            >
              Kosongkan tanggal
            </Button>
          </div>
        )}
      </PopoverContent>
    </Popover>
  )
}
