import { format, isValid, parse, parseISO } from 'date-fns'
import { id as localeId } from 'date-fns/locale'
import { CalendarDays } from 'lucide-react'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { formatDate } from '@/lib/format'
import { cn } from '@/lib/utils'

// Bound the year dropdown to a sensible range without reading the clock during
// render.
const maxYear = new Date().getFullYear() + 3

type Props = {
  value?: string | null
  onChange: (value: string) => void
  onClear?: () => void
  placeholder?: string
  id?: string
  disabled?: boolean
  className?: string
}

function parseManual(value: string): Date | null {
  const trimmed = value.trim()
  if (!trimmed) return null

  const iso = parseISO(trimmed)
  if (/^\d{4}-\d{2}-\d{2}$/.test(trimmed) && isValid(iso)) {
    return iso
  }

  const local = parse(trimmed, 'd/M/yyyy', new Date())
  return isValid(local) ? local : null
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
  const [manual, setManual] = useState('')
  const [invalid, setInvalid] = useState(false)
  const selected = value ? parseISO(value) : undefined

  function applyManual() {
    const parsed = parseManual(manual)
    if (!parsed) {
      setInvalid(true)
      return
    }
    onChange(format(parsed, 'yyyy-MM-dd'))
    setInvalid(false)
    setOpen(false)
  }

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next)
        if (next) {
          setManual(value ?? '')
          setInvalid(false)
        }
      }}
    >
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
        <div className="space-y-1 border-b p-2">
          <Input
            className="h-8"
            inputMode="numeric"
            autoComplete="off"
            placeholder="YYYY-MM-DD atau DD/MM/YYYY"
            aria-label="Ketik tanggal"
            aria-invalid={invalid}
            value={manual}
            onChange={(event) => {
              setManual(event.target.value)
              setInvalid(false)
            }}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                event.preventDefault()
                applyManual()
              }
            }}
          />
          {invalid ? (
            <p className="text-destructive px-1 text-xs">Format tanggal tidak valid.</p>
          ) : (
            <p className="text-muted-foreground px-1 text-xs">Tekan Enter untuk memakai tanggal.</p>
          )}
        </div>

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
          captionLayout="dropdown"
          startMonth={new Date(2000, 0)}
          endMonth={new Date(maxYear, 11)}
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
