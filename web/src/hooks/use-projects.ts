import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { api } from '@/lib/api'
import type { Project, ProjectStatus, ProjectTimeline } from '@/lib/types'

export type ProjectInput = {
  name: string
  status: ProjectStatus
  start_date: string
  end_date: string
  notes: string
}

function invalidateProjectQueries(queryClient: ReturnType<typeof useQueryClient>) {
  void queryClient.invalidateQueries({ queryKey: ['projects'] })
  void queryClient.invalidateQueries({ queryKey: ['attendance'] })
  void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
}

export function useProjects() {
  return useQuery({
    queryKey: ['projects', 'list'],
    queryFn: () => api.get<Project[]>('/projects'),
  })
}

export function useProjectTimeline(month: string) {
  return useQuery({
    queryKey: ['projects', 'timeline', month],
    queryFn: () => api.get<ProjectTimeline>(`/projects/timeline?month=${month}`),
  })
}

export function useCreateProject() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: ProjectInput) => api.post<Project>('/projects', input),
    onSuccess: () => invalidateProjectQueries(queryClient),
  })
}

export function useUpdateProject(id: number) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: ProjectInput) => api.put<Project>(`/projects/${id}`, input),
    onSuccess: () => invalidateProjectQueries(queryClient),
  })
}

export function useDeleteProject() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => api.delete<void>(`/projects/${id}`),
    onSuccess: () => invalidateProjectQueries(queryClient),
  })
}
