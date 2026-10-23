import { format, parseISO } from 'date-fns'
import { id as localeId } from 'date-fns/locale'

/**
 * Rupiah, number, and date formatting. Money is always an integer number of
 * rupiah; the UI only adds separators for display.
 */

export function digitsOnly(value: string): string {
  return value.replace(/\D/g, '')
}

/** Turns "1250000" into "1.250.000". */
export function formatThousands(digits: string): string {
  const clean = digitsOnly(digits).replace(/^0+(?=\d)/, '')
  return clean.replace(/\B(?=(\d{3})+(?!\d))/g, '.')
}

/** Formats an integer rupiah amount as "Rp 1.250.000". */
export function formatRupiah(value: number): string {
  const safe = Number.isFinite(value) ? Math.trunc(value) : 0
  const sign = safe < 0 ? '-' : ''
  return `${sign}Rp ${formatThousands(String(Math.abs(safe)))}`
}

/** Formats a YYYY-MM-DD date as "Sen, 12 Jan 2026". */
export function formatDate(value: string): string {
  return format(parseISO(value), 'EEE, d MMM yyyy', { locale: localeId })
}

/** Formats a YYYY-MM-DD date as "Sabtu, 10 Oktober 2026". */
export function formatLongDate(value: string): string {
  return format(parseISO(value), 'EEEE, d MMMM yyyy', { locale: localeId })
}

/** Formats a YYYY-MM-DD date as "12 Jan". */
export function formatDateShort(value: string): string {
  return format(parseISO(value), 'd MMM', { locale: localeId })
}

/** Formats an RFC3339 timestamp as "12 Jan 2026". */
export function formatTimestampDate(value: string): string {
  return format(parseISO(value), 'd MMM yyyy', { locale: localeId })
}

/** Today in the device timezone, as YYYY-MM-DD. */
export function todayISO(): string {
  return format(new Date(), 'yyyy-MM-dd')
}

/** Turns a quarters value into its Indonesian day label. */
export function quartersLabel(quarters: number): string {
  switch (quarters) {
    case 1:
      return '¼ hari'
    case 2:
      return '½ hari'
    case 3:
      return '¾ hari'
    case 4:
      return 'Penuh'
    default:
      return 'Absen'
  }
}

/** Formats a quarters total as Indonesian days: 22 -> "5,5". */
export function formatDays(quarters: number): string {
  const days = quarters / 4
  const text = Number.isInteger(days) ? String(days) : days.toFixed(2).replace(/0$/, '')
  return text.replace('.', ',')
}
