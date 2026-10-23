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

export type ProjectStatus = 'aktif' | 'selesai' | 'ditunda'

export type Project = {
  id: number
  name: string
  start_date: string | null
  end_date: string | null
  status: ProjectStatus
  notes: string
  created_at: string
}

export type ProjectRef = {
  id: number
  name: string
}

export type TimelineProject = {
  id: number
  name: string
  status: ProjectStatus
  start_date: string | null
  end_date: string | null
  tagged_days: string[]
}

export type ProjectTimeline = {
  month: string
  days: number
  projects: TimelineProject[]
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
  projects: ProjectRef[]
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
  projects: ProjectRef[]
}

export type CashAdvance = {
  id: number
  worker_id: number
  date: string
  amount: number
  note: string
  locked: boolean
}

export type RecapStatus = 'draft' | 'final'
export type RecapMode = 'since_monday' | 'all_unsettled' | 'custom'

export type Recap = {
  id: number
  mode: RecapMode
  start_date: string | null
  end_date: string | null
  status: RecapStatus
  created_at: string
  finalized_at: string | null
}

export type RecapAttendanceItem = {
  id: number
  date: string
  quarters: number
  earned: number
}

export type RecapAdvanceItem = {
  id: number
  date: string
  amount: number
  note: string
}

export type RecapExtraFee = {
  id: number
  worker_id: number
  amount: number
  description: string
}

export type RecapWorkerLine = {
  worker_id: number
  name: string
  role: string
  quarters: number
  gross: number
  advances: number
  extras: number
  net: number
  paid: boolean
  warning: number
  attendance: RecapAttendanceItem[]
  cash_advances: RecapAdvanceItem[]
  extra_fees: RecapExtraFee[]
}

export type RecapTotals = {
  gross: number
  advances: number
  extras: number
  net: number
}

export type RecapDetail = {
  recap: Recap
  workers: RecapWorkerLine[]
  totals: RecapTotals
  period_start: string
  period_end: string
}
