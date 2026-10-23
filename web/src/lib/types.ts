export type User = {
  id: number
  username: string
}

export type Worker = {
  id: number
  name: string
  role: string
  daily_fee: number
  active: boolean
  created_at: string
}

export type AttendanceDayRow = {
  worker_id: number
  name: string
  role: string
  daily_fee: number
  quarters: number
  snapshot: number
  earned: number
  locked: boolean
}

export type AttendanceDay = {
  date: string
  workers: AttendanceDayRow[]
  total_earned: number
  present: number
}

export type AttendanceWeekEntry = {
  id: number
  date: string
  quarters: number
  earned: number
  locked: boolean
}

export type CashAdvance = {
  id: number
  worker_id: number
  date: string
  amount: number
  note: string
  locked: boolean
}
