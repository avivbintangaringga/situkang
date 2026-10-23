import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { api } from '@/lib/api'
import type { AttendanceDay, AttendanceWeekEntry } from '@/lib/types'

export type DayItem = {
  worker_id: number
  quarters: number
  project_ids: number[]
}

export function attendanceDayQueryKey(date: string | null) {
  return ['attendance', 'day', date ?? 'today'] as const
}

export function useAttendanceDay(date: string | null) {
  return useQuery({
    queryKey: attendanceDayQueryKey(date),
    queryFn: () => {
      const query = date ? `?date=${date}` : ''
      return api.get<AttendanceDay>(`/attendances/day${query}`)
    },
    staleTime: 30 * 1000,
  })
}

export function useSaveDay() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ date, items }: { date: string; items: DayItem[] }) =>
      api.put<AttendanceDay>(`/attendances/day/${date}`, { items }),
    onSuccess: (day) => {
      queryClient.setQueryData(attendanceDayQueryKey(day.date), day)
      // The screen may be reading the day under the "today" key, so refresh
      // every attendance query instead of only the dated one.
      void queryClient.invalidateQueries({ queryKey: ['attendance'] })
      void queryClient.invalidateQueries({ queryKey: ['workers'] })
      void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })
}

export function useWorkerWeek(workerId: number, from: string, to: string) {
  return useQuery({
    queryKey: ['attendance', 'week', workerId, from, to],
    queryFn: () =>
      api.get<AttendanceWeekEntry[]>(
        `/attendances?worker_id=${workerId}&from=${from}&to=${to}`,
      ),
    enabled: workerId > 0,
  })
}
