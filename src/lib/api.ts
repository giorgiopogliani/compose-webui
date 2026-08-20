import type { ActionResponse, ProjectsResponse } from './types'

export async function fetchProjects(): Promise<ProjectsResponse> {
  const res = await fetch('/api/projects')
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.json()
}

export async function runAction(
  action: 'up' | 'restart',
  project: string,
  service?: string,
): Promise<ActionResponse> {
  const base = service
    ? `/api/projects/${encodeURIComponent(project)}/services/${encodeURIComponent(service)}`
    : `/api/projects/${encodeURIComponent(project)}`
  const res = await fetch(`${base}/${action}`, { method: 'POST' })
  const body = (await res.json().catch(() => null)) as ActionResponse | null
  if (!body) return { ok: false, message: `HTTP ${res.status}` }
  return body
}