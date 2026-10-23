import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { api } from '@/lib/api'
import type { Recap, RecapDetail, RecapExtraFee, RecapMode } from '@/lib/types'

export type CreateRecapInput = {
  mode: RecapMode
  start_date?: string
  end_date?: string
}

export function recapQueryKey(id: number) {
  return ['recaps', 'detail', id] as const
}

export function useRecaps() {
  return useQuery({
    queryKey: ['recaps', 'list'],
    queryFn: () => api.get<Recap[]>('/recaps'),
  })
}

export function useRecap(id: number) {
  return useQuery({
    queryKey: recapQueryKey(id),
    queryFn: () => api.get<RecapDetail>(`/recaps/${id}`),
    enabled: id > 0,
  })
}

function useInvalidateRecaps(id?: number) {
  const queryClient = useQueryClient()
  return () => {
    void queryClient.invalidateQueries({ queryKey: ['recaps', 'list'] })
    if (id) void queryClient.invalidateQueries({ queryKey: recapQueryKey(id) })
    void queryClient.invalidateQueries({ queryKey: ['attendance'] })
    void queryClient.invalidateQueries({ queryKey: ['cash-advances'] })
    void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
  }
}

export function useCreateRecap() {
  const invalidate = useInvalidateRecaps()

  return useMutation({
    mutationFn: (input: CreateRecapInput) => api.post<Recap>('/recaps', input),
    onSuccess: () => void invalidate(),
  })
}

export function useDeleteRecap(id: number) {
  const invalidate = useInvalidateRecaps(id)

  return useMutation({
    mutationFn: () => api.delete<void>(`/recaps/${id}`),
    onSuccess: () => void invalidate(),
  })
}

export function useSetRecapPaid(id: number) {
  const invalidate = useInvalidateRecaps(id)

  return useMutation({
    mutationFn: ({ workerId, paid }: { workerId: number; paid: boolean }) =>
      api.patch<void>(`/recaps/${id}/paid`, { worker_id: workerId, paid }),
    onSuccess: () => void invalidate(),
  })
}

export function useFinalizeRecap(id: number) {
  const invalidate = useInvalidateRecaps(id)

  return useMutation({
    mutationFn: () => api.post<void>(`/recaps/${id}/finalize`),
    onSuccess: () => void invalidate(),
  })
}

export function useReopenRecap(id: number) {
  const invalidate = useInvalidateRecaps(id)

  return useMutation({
    mutationFn: () => api.post<void>(`/recaps/${id}/reopen`),
    onSuccess: () => void invalidate(),
  })
}

export function useAddExtraFee(id: number) {
  const invalidate = useInvalidateRecaps(id)

  return useMutation({
    mutationFn: (input: { worker_id: number; amount: number; description: string }) =>
      api.post<RecapExtraFee>(`/recaps/${id}/extra-fees`, input),
    onSuccess: () => void invalidate(),
  })
}

export function useUpdateExtraFee(id: number) {
  const invalidate = useInvalidateRecaps(id)

  return useMutation({
    mutationFn: ({
      feeId,
      amount,
      description,
    }: {
      feeId: number
      amount: number
      description: string
    }) =>
      api.patch<void>(`/recaps/${id}/extra-fees/${feeId}`, { amount, description }),
    onSuccess: () => void invalidate(),
  })
}

export function useDeleteExtraFee(id: number) {
  const invalidate = useInvalidateRecaps(id)

  return useMutation({
    mutationFn: (feeId: number) =>
      api.delete<void>(`/recaps/${id}/extra-fees/${feeId}`),
    onSuccess: () => void invalidate(),
  })
}
