import type { Session, SessionOutputResponse, Project, Todo, CreateSessionRequest, CreateTodoRequest, PRDashboard, PRDetail } from './types'

const getBaseURL = (): string => {
  // In dev, Vite proxy handles /api → localhost:47437
  // In production (embedded), we're served from the same origin
  return ''
}

const BASE = getBaseURL()

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    ...init,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error((err as { error?: string }).error || `HTTP ${res.status}`)
  }
  if (res.status === 204 || res.headers.get('Content-Length') === '0') {
    return undefined as T
  }
  return res.json() as Promise<T>
}

export const api = {
  getSessions: () => apiFetch<Session[]>('/api/v1/sessions'),
  getSession: (id: string) => apiFetch<Session>(`/api/v1/sessions/${id}`),
  getSessionOutput: (id: string, width?: number) =>
    apiFetch<SessionOutputResponse>(`/api/v1/sessions/${id}/output${width ? `?width=${width}` : ''}`),
  sendMessage: (id: string, message: string) =>
    apiFetch<{ status: string }>(`/api/v1/sessions/${id}/send`, {
      method: 'POST',
      body: JSON.stringify({ message }),
    }),
  sendRaw: (id: string, data: string) =>
    apiFetch<{ status: string }>(`/api/v1/sessions/${id}/send`, {
      method: 'POST',
      body: JSON.stringify({ message: data, raw: true }),
    }),
  createSession: (req: CreateSessionRequest) =>
    apiFetch<Session>('/api/v1/sessions', {
      method: 'POST',
      body: JSON.stringify(req),
    }),
  stopSession: (id: string) =>
    apiFetch<{ status: string }>(`/api/v1/sessions/${id}/stop`, { method: 'POST' }),
  deleteSession: (id: string) =>
    apiFetch<void>(`/api/v1/sessions/${id}`, { method: 'DELETE' }),
  restartSession: (id: string) =>
    apiFetch<Session>(`/api/v1/sessions/${id}/restart`, { method: 'POST' }),
  getProjects: () => apiFetch<Project[]>('/api/v1/projects'),
  createProject: (req: { name: string; base_dir: string; base_branch?: string }) =>
    apiFetch<Project>('/api/v1/projects', {
      method: 'POST',
      body: JSON.stringify(req),
    }),
  deleteProject: (name: string) =>
    apiFetch<void>(`/api/v1/projects/${encodeURIComponent(name)}`, { method: 'DELETE' }),
  getTodos: (projectPath: string) =>
    apiFetch<Todo[]>(`/api/v1/todos?project=${encodeURIComponent(projectPath)}`),
  createTodo: (req: CreateTodoRequest) =>
    apiFetch<Todo>('/api/v1/todos', {
      method: 'POST',
      body: JSON.stringify(req),
    }),
  updateTodo: (id: string, req: { status?: string; title?: string; description?: string }) =>
    apiFetch<Todo>(`/api/v1/todos/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(req),
    }),
  deleteTodo: (id: string) =>
    apiFetch<void>(`/api/v1/todos/${id}`, { method: 'DELETE' }),
  getPRDashboard: () =>
    apiFetch<PRDashboard>('/api/v1/prs'),
  getPRDetail: (repo: string, number: number) =>
    apiFetch<PRDetail>(`/api/v1/prs/detail?repo=${encodeURIComponent(repo)}&number=${number}`),
  submitPRReview: (repo: string, number: number, action: string, body?: string) =>
    apiFetch<void>(`/api/v1/prs/review?repo=${encodeURIComponent(repo)}&number=${number}`, {
      method: 'POST',
      body: JSON.stringify({ action, body }),
    }),
  addPRComment: (repo: string, number: number, commentBody: string, path?: string, line?: number) =>
    apiFetch<void>(`/api/v1/prs/comment?repo=${encodeURIComponent(repo)}&number=${number}`, {
      method: 'POST',
      body: JSON.stringify({ body: commentBody, path, line }),
    }),
  changePRState: (repo: string, number: number, action: string) =>
    apiFetch<void>(`/api/v1/prs/state?repo=${encodeURIComponent(repo)}&number=${number}`, {
      method: 'POST',
      body: JSON.stringify({ action }),
    }),
}
