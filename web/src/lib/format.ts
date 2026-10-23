/**
 * Rupiah and number formatting. Money is always an integer number of rupiah;
 * the UI only adds separators for display.
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
