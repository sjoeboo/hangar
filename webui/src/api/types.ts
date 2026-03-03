export interface PRInfo {
  number: number
  title: string
  state: string // OPEN, DRAFT, MERGED, CLOSED
  url: string
  checks_passed?: number
  checks_failed?: number
  checks_pending?: number
  has_checks?: boolean
}

export interface Session {
  id: string
  title: string
  project_path: string
  group_path: string
  tool: string
  status: 'running' | 'waiting' | 'idle' | 'starting' | 'stopped' | 'unknown'
  worktree_branch?: string
  latest_prompt?: string
  created_at: string
  last_accessed_at?: string
  parent_id?: string
  pr?: PRInfo
}

export interface Project {
  name: string
  base_dir: string
  base_branch?: string
  order?: number
}

export interface Todo {
  id: string
  project_path: string
  title: string
  description?: string
  status: 'todo' | 'in_progress' | 'done'
  session_id?: string
  order: number
  created_at: string
  updated_at: string
}

export interface SessionOutputResponse {
  session_id: string
  output: string
  lines: number
}

export interface CreateSessionRequest {
  title: string
  path: string
  tool?: string
  group?: string
  message?: string
  worktree?: boolean
  branch?: string
  skip_permissions?: boolean
}

export interface CreateTodoRequest {
  project_path: string
  title: string
  description?: string
}

export interface WsMessage {
  type: string
  data?: unknown
}

export interface WsHelloData {
  version: string
  sessions: number
}

export interface WsSessionOutputData {
  session_id: string
  output: string
}
