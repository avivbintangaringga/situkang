import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { api } from '@/lib/api'
import type { CashAdvance } from '@/lib/types'

export type CashAdvanceInput = {
  worker_id?: number
  date: string
  amount: number
  note: string
}

export function cashAdvancesQueryKey(workerId: number) {
  return ['cash-advances', { workerId }] as const
}

export function useCashAdvances(workerId: number) {
  return useQuery({
    queryKey: cashAdvancesQueryKey(workerId),
    queryFn: () =>
      api.get<CashAdvance[]>(`/cash-advances?worker_id=${workerId}`),
    enabled: workerId > 0,
  })
}

function useInvalidate(workerId: number) {
  const queryClient = useQueryClient()
  return () => {
    void queryClient.invalidateQueries({ queryKey: ['cash-advances'] })
    void queryClient.invalidateQueries({ queryKey: cashAdvancesQueryKey(workerId) })
  }
}

export function useCreateCashAdvance(workerId: number) {
  const invalidate = useInvalidate(workerId)

  return useMutation({
    mutationFn: (input: CashAdvanceInput) =>
      api.post<CashAdvance>('/cash-advances', { ...input, worker_id: workerId }),
    onSuccess: () => void invalidate(),
  })
}

export function useUpdateCashAdvance(id: number, workerId: number) {
  const invalidate = useInvalidate(workerId)

  return useMutation({
    mutationFn: (input: CashAdvanceInput) =>
      api.put<CashAdvance>(`/cash-advances/${id}`, {
        date: input.date,
        amount: input.amount,
        note: input.note,
      }),
    onSuccess: () => void invalidate(),
  })
}

export function useDeleteCashAdvance(workerId: number) {
  const invalidate = useInvalidate(workerId)

  return useMutation({
    mutationFn: (id: number) => api.delete<void>(`/cash-advances/${id}`),
    onSuccess: () => void invalidate(),
  })
}
