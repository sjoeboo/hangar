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

export interface PRFullInfo {
  number: number
  title: string
  state: string
  is_draft?: boolean
  url: string
  repo?: string
  head_branch?: string
  base_branch?: string
  author?: string
  review_decision?: string
  comment_count?: number
  checks_passed?: number
  checks_failed?: number
  checks_pending?: number
  has_checks?: boolean
  created_at: string
  updated_at: string
  source?: string
  session_id?: string
}

export interface PRDashboard {
  all: PRFullInfo[]
  mine: PRFullInfo[]
  review_requested: PRFullInfo[]
  sessions: Record<string, PRFullInfo>
}

export interface PRComment {
  id: number
  author: string
  body: string
  created_at: string
  path?: string
  line?: number
}

export interface PRReview {
  author: string
  state: string
  body: string
  created_at: string
  comments: PRComment[]
}

export interface PRFileChange {
  path: string
  additions: number
  deletions: number
  status: string
}

export interface PRDetail extends PRFullInfo {
  body?: string
  mergeability?: string
  comments: PRComment[]
  reviews: PRReview[]
  files: PRFileChange[]
  diff_content?: string
}

export interface Session {
  id: string
  title: string
  project_path: string
  group_path: string
  session_type?: string
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
