import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { meQueryKey } from '@/hooks/use-auth'
import { api } from '@/lib/api'
import type { Account, User } from '@/lib/types'

export const usersQueryKey = ['users'] as const

type CreateUserValues = {
  username: string
  password: string
}

type ChangePasswordValues = {
  current_password: string
  new_password: string
}

export function useUsers() {
  return useQuery({
    queryKey: usersQueryKey,
    queryFn: () => api.get<Account[]>('/users'),
  })
}

export function useCreateUser() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (values: CreateUserValues) => api.post<User>('/users', values),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: usersQueryKey })
    },
  })
}

export function useUpdateUsername() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (username: string) =>
      api.patch<User>('/auth/username', { username }),
    onSuccess: (user) => {
      queryClient.setQueryData(meQueryKey, user)
      void queryClient.invalidateQueries({ queryKey: usersQueryKey })
    },
  })
}

export function useChangePassword() {
  return useMutation({
    mutationFn: (values: ChangePasswordValues) =>
      api.patch<void>('/auth/password', values),
  })
}
