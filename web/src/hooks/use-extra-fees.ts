import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { api } from '@/lib/api'
import type { ExtraFee } from '@/lib/types'

export type ExtraFeeInput = {
  worker_id?: number
  date: string
  amount: number
  description: string
}

export function extraFeesQueryKey(workerId: number) {
  return ['extra-fees', { workerId }] as const
}

export function useExtraFees(workerId: number) {
  return useQuery({
    queryKey: extraFeesQueryKey(workerId),
    queryFn: () => api.get<ExtraFee[]>(`/extra-fees?worker_id=${workerId}`),
    enabled: workerId > 0,
  })
}

function useInvalidate(workerId: number) {
  const queryClient = useQueryClient()
  return () => {
    void queryClient.invalidateQueries({ queryKey: ['extra-fees'] })
    void queryClient.invalidateQueries({ queryKey: extraFeesQueryKey(workerId) })
    void queryClient.invalidateQueries({ queryKey: ['recaps'] })
  }
}

export function useCreateExtraFee(workerId: number) {
  const invalidate = useInvalidate(workerId)

  return useMutation({
    mutationFn: (input: ExtraFeeInput) =>
      api.post<ExtraFee>('/extra-fees', { ...input, worker_id: workerId }),
    onSuccess: () => void invalidate(),
  })
}

export function useUpdateExtraFee(id: number, workerId: number) {
  const invalidate = useInvalidate(workerId)

  return useMutation({
    mutationFn: (input: ExtraFeeInput) =>
      api.put<ExtraFee>(`/extra-fees/${id}`, {
        date: input.date,
        amount: input.amount,
        description: input.description,
      }),
    onSuccess: () => void invalidate(),
  })
}

export function useDeleteExtraFee(workerId: number) {
  const invalidate = useInvalidate(workerId)

  return useMutation({
    mutationFn: (id: number) => api.delete<void>(`/extra-fees/${id}`),
    onSuccess: () => void invalidate(),
  })
}
