import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { api } from '@/lib/api'
import type { RecapDetail, RecapShare } from '@/lib/types'

export function recapShareQueryKey(id: number) {
  return ['recaps', 'share', id] as const
}

export function publicRecapQueryKey(token: string) {
  return ['public-recap', token] as const
}

/** Reads a recap's existing share link. Fails with a 404 when none exists. */
export function useRecapShare(id: number, enabled = true) {
  return useQuery({
    queryKey: recapShareQueryKey(id),
    queryFn: () => api.get<RecapShare>(`/recaps/${id}/share`),
    enabled: enabled && id > 0,
    retry: false,
  })
}

/** Creates the share link, or returns the existing one. */
export function useCreateRecapShare(id: number) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => api.post<RecapShare>(`/recaps/${id}/share`),
    onSuccess: (data) => {
      queryClient.setQueryData(recapShareQueryKey(id), data)
    },
  })
}

export function useRevokeRecapShare(id: number) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => api.delete<void>(`/recaps/${id}/share`),
    onSuccess: () => {
      queryClient.removeQueries({ queryKey: recapShareQueryKey(id) })
    },
  })
}

/** Loads a shared recap without a session. */
export function usePublicRecap(token: string) {
  return useQuery({
    queryKey: publicRecapQueryKey(token),
    queryFn: () => api.get<RecapDetail>(`/public/recaps/${token}`),
    enabled: token.length > 0,
    retry: false,
  })
}
