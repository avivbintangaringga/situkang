import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { api } from '@/lib/api'
import type { Worker } from '@/lib/types'

export type WorkerInput = {
  name: string
  role: string
  daily_fee: number
  active?: boolean
}

export function workersQueryKey(search: string, includeInactive: boolean) {
  return ['workers', { search, includeInactive }] as const
}

export function useWorkers(search: string, includeInactive: boolean) {
  return useQuery({
    queryKey: workersQueryKey(search, includeInactive),
    queryFn: () => {
      const params = new URLSearchParams()
      const term = search.trim()
      if (term) params.set('q', term)
      if (includeInactive) params.set('include_inactive', '1')

      const query = params.toString()
      return api.get<Worker[]>(`/workers${query ? `?${query}` : ''}`)
    },
  })
}

export function useWorker(id: number) {
  return useQuery({
    queryKey: ['workers', 'detail', id],
    queryFn: () => api.get<Worker>(`/workers/${id}`),
    enabled: id > 0,
  })
}

function useInvalidateWorkers() {
  const queryClient = useQueryClient()
  return () => queryClient.invalidateQueries({ queryKey: ['workers'] })
}

export function useCreateWorker() {
  const invalidate = useInvalidateWorkers()

  return useMutation({
    mutationFn: (input: WorkerInput) => api.post<Worker>('/workers', input),
    onSuccess: () => void invalidate(),
  })
}

export function useUpdateWorker(id: number) {
  const invalidate = useInvalidateWorkers()

  return useMutation({
    mutationFn: (input: WorkerInput) => api.put<Worker>(`/workers/${id}`, input),
    onSuccess: () => void invalidate(),
  })
}

export function useDeleteWorker() {
  const invalidate = useInvalidateWorkers()

  return useMutation({
    mutationFn: (id: number) => api.delete<void>(`/workers/${id}`),
    onSuccess: () => void invalidate(),
  })
}
