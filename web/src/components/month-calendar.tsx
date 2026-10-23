import {
  addMonths,
  eachDayOfInterval,
  endOfMonth,
  endOfWeek,
  format,
  isSameMonth,
  isToday,
  parseISO,
  startOfMonth,
  startOfWeek,
} from 'date-fns'
import { id as localeId } from 'date-fns/locale'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useMemo } from 'react'

import { Button } from '@/components/ui/button'
import { formatLongDate } from '@/lib/format'
import { cn } from '@/lib/utils'

export type CalendarMarker = {
  className: string
  label: string
}

type Props = {
  /** Selected month in YYYY-MM form. */
  month: string
  onMonthChange: (month: string) => void
  /** Markers keyed by YYYY-MM-DD date. */
  markers?: Record<string, CalendarMarker[]>
  /** Optional range tint, using YYYY-MM-DD dates. */
  range?: { start?: string | null; end?: string | null }
  /** Optional selected range tint, using YYYY-MM-DD dates. */
  selection?: { start?: string | null; end?: string | null }
  /** Highlighted day in YYYY-MM-DD form. */
  selectedDate?: string | null
  onSelectDate?: (date: string) => void
  disabled?: boolean
}

const weekdays = ['Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab', 'Min']

function inRange(date: string, range?: Props['range']): boolean {
  if (!range) return false
  const start = range.start ?? null
  const end = range.end ?? null
  if (!start && !end) return false
  if (start && date < start) return false
  if (end && date > end) return false
  return true
}

export function MonthCalendar({
  month,
  onMonthChange,
  markers,
  range,
  selection,
  selectedDate,
  onSelectDate,
  disabled,
}: Props) {
  const monthDate = parseISO(`${month}-01`)
  const days = useMemo(() => {
    const first = startOfMonth(monthDate)
    const last = endOfMonth(monthDate)
    return eachDayOfInterval({
      start: startOfWeek(first, { weekStartsOn: 1 }),
      end: endOfWeek(last, { weekStartsOn: 1 }),
    })
  }, [monthDate])

  function shift(offset: number) {
    onMonthChange(format(addMonths(monthDate, offset), 'yyyy-MM'))
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between gap-2">
        <Button
          type="button"
          variant="outline"
          className="size-11 p-0"
          onClick={() => shift(-1)}
          disabled={disabled}
          aria-label="Bulan sebelumnya"
        >
          <ChevronLeft className="size-5" aria-hidden />
        </Button>
        <p className="text-sm font-semibold capitalize">
          {format(monthDate, 'MMMM yyyy', { locale: localeId })}
        </p>
        <Button
          type="button"
          variant="outline"
          className="size-11 p-0"
          onClick={() => shift(1)}
          disabled={disabled}
          aria-label="Bulan berikutnya"
        >
          <ChevronRight className="size-5" aria-hidden />
        </Button>
      </div>

      <div className="grid grid-cols-7 gap-1">
        {weekdays.map((day) => (
          <div
            key={day}
            className="text-muted-foreground py-1 text-center text-[10px] font-medium"
            aria-hidden
          >
            {day}
          </div>
        ))}

        {days.map((day) => {
          const iso = format(day, 'yyyy-MM-dd')
          const outside = !isSameMonth(day, monthDate)
          const today = isToday(day)
          const selected = selectedDate === iso
          const tinted = inRange(iso, range)
          const inSelection = inRange(iso, selection)
          const selectionEdge = selection?.start === iso || selection?.end === iso
          const dayMarkers = markers?.[iso] ?? []
          const label = `${formatLongDate(iso)}${
            dayMarkers.length > 0
              ? `, ${dayMarkers.map((marker) => marker.label).join(', ')}`
              : ''
          }`

          const content = (
            <>
              <span
                className={cn(
                  'text-xs leading-none',
                  today && !outside && 'font-semibold',
                )}
              >
                {format(day, 'd')}
              </span>
              <span className="flex min-h-2 flex-wrap items-center justify-center gap-0.5">
                {dayMarkers.map((marker, index) => (
                  <span
                    key={`${marker.label}-${index}`}
                    title={marker.label}
                    className={cn('inline-block shrink-0', marker.className)}
                  />
                ))}
              </span>
            </>
          )

          const className = cn(
            'flex min-h-11 flex-col items-center justify-center gap-1 rounded-lg border border-transparent px-0.5 py-1 transition-colors',
            outside
              ? 'text-muted-foreground/45'
              : 'text-foreground hover:bg-muted/60',
            tinted && !outside && 'border-primary/20 bg-primary/5',
            inSelection && !outside && 'border-primary/20 bg-primary/5',
            selectionEdge && !outside && 'border-primary bg-primary/15 font-semibold',
            selected && !outside && 'border-primary bg-primary/15 font-semibold',
            today && !outside && 'border-primary/60 ring-1 ring-primary/30',
            disabled && 'cursor-default opacity-60',
          )

          if (!onSelectDate) {
            return (
              <div key={iso} className={className} title={label} aria-label={label}>
                {content}
              </div>
            )
          }

          return (
            <button
              key={iso}
              type="button"
              onClick={() => onSelectDate(iso)}
              disabled={disabled}
              className={className}
              title={label}
              aria-label={label}
            >
              {content}
            </button>
          )
        })}
      </div>
    </div>
  )
}
