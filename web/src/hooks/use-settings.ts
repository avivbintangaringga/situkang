import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { api } from '@/lib/api'
import type { Settings } from '@/lib/types'

export function useSettings() {
  return useQuery({
    queryKey: ['settings'],
    queryFn: () => api.get<Settings>('/settings'),
    staleTime: 5 * 60 * 1000,
  })
}

export function useUpdateSettings() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: { working_days: number[]; attendance_fractions: number[] }) =>
      api.put<Settings>('/settings', input),
    onSuccess: (settings) => {
      queryClient.setQueryData(['settings'], settings)
      void queryClient.invalidateQueries({ queryKey: ['attendance'] })
      void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })
}
