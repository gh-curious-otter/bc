const BASE = '/api';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  });
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`);
  }
  return res.json() as Promise<T>;
}

export interface Agent {
  name: string;
  role: string;
  tool: string;
  state: string;
  cost_usd: number;
  started_at: string;
  created_at: string;
  updated_at: string;
  stopped_at?: string;
  task?: string;
  team?: string;
  session?: string;
  session_id?: string;
  parent_id?: string;
  children?: string[];
  total_tokens?: number;
  mcp_servers?: string[];
}

export interface Channel {
  name: string;
  description: string;
  members: string[];
  member_count: number;
}

export interface ChannelMessage {
  id: number;
  sender: string;
  content: string;
  created_at: string;
}

export interface CostSummary {
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
  total_cost_usd: number;
  record_count: number;
}

export interface AgentCostSummary {
  agent_id: string;
  total_cost_usd: number;
  input_tokens: number;
  output_tokens: number;
  record_count: number;
}

export interface ModelCostSummary {
  model: string;
  total_cost_usd: number;
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
  record_count: number;
}

export interface DailyCost {
  date: string;
  cost_usd: number;
  total_tokens: number;
  record_count: number;
  input_tokens: number;
  output_tokens: number;
}

export interface Budget {
  scope: string;
  period: string;
  limit_usd: number;
  alert_at: number;
  hard_stop: boolean;
  id: number;
  updated_at: string;
}

export interface BudgetStatus {
  budget: Budget;
  current_spend: number;
  remaining: number;
  percent_used: number;
  is_over_budget: boolean;
  is_near_limit: boolean;
}

// ResolvedRole — BFS-resolved role with inherited fields merged
export interface Role {
  Name: string;
  Prompt: string;
  MCPServers: string[];
  Secrets: string[];
  Plugins: string[];
  PromptCreate: string;
  PromptStart: string;
  PromptStop: string;
  PromptDelete: string;
  Commands: Record<string, string>;
  Skills: Record<string, string>;
  Agents: Record<string, string>;
  Rules: Record<string, string>;
  Settings: Record<string, unknown>;
  Review: string;
}

export interface Tool {
  name: string;
  command: string;
  install_cmd: string;
  builtin: boolean;
  enabled: boolean;
}

export interface MCPServer {
  name: string;
  transport: string;
  command: string;
  url: string;
  enabled: boolean;
}

export interface EventLogEntry {
  id: number;
  type: string;
  agent: string;
  message: string;
  created_at: string;
}

export interface DoctorCategory {
  Name: string;
  Items: { Name: string; Message: string; Fix: string; Severity: number }[];
}

export interface DoctorReport {
  Categories: DoctorCategory[];
}

export interface CronJob {
  name: string;
  schedule: string;
  agent_name: string;
  prompt: string;
  command: string;
  enabled: boolean;
  run_count: number;
  last_run: string | null;
  next_run: string | null;
  created_at: string;
}

export interface Secret {
  name: string;
  description: string;
  backend: string;
  created_at: string;
}

export interface SystemStats {
  hostname: string;
  os: string;
  arch: string;
  cpus: number;
  cpu_usage_percent: number;
  memory_total_bytes: number;
  memory_used_bytes: number;
  memory_usage_percent: number;
  disk_total_bytes: number;
  disk_used_bytes: number;
  disk_usage_percent: number;
  go_version: string;
  uptime_seconds: number;
  goroutines: number;
}

export interface WorkspaceInfo {
  name: string;
  nickname: string;
  agent_count: number;
  running_count: number;
  is_healthy: boolean;
}

export interface StatsSummary {
  agents_total: number;
  agents_running: number;
  agents_stopped: number;
  channels_total: number;
  messages_total: number;
  total_cost_usd: number;
  roles_total: number;
  tools_total: number;
  uptime_seconds: number;
}

export interface ChannelTopSender {
  sender: string;
  count: number;
}

export interface ChannelStats {
  name: string;
  message_count: number;
  member_count: number;
  last_activity: string;
  top_senders: ChannelTopSender[];
}

export interface AgentStatsRecord {
  collected_at: string;
  agent_name: string;
  cpu_pct: number;
  mem_used_mb: number;
  mem_limit_mb: number;
  net_rx_mb: number;
  net_tx_mb: number;
  block_read_mb: number;
  block_write_mb: number;
}

export interface Team {
  name: string;
  description: string;
  lead: string;
  members: string[];
  created_at: string;
  updated_at: string;
}

export interface SettingsConfig {
  User: { Nickname: string };
  TUI: { Theme: string; Mode: string };
  Runtime: { Backend: string; Docker?: { Image: string; Network: string; ExtraMounts: string[]; CPUs: number; MemoryMB: number } };
  Providers: {
    Default: string;
    Claude?: { Command: string; Enabled: boolean; Env?: Record<string, string> };
    Gemini?: { Command: string; Enabled: boolean; Env?: Record<string, string> };
    Cursor?: { Command: string; Enabled: boolean; Env?: Record<string, string> };
    Codex?: { Command: string; Enabled: boolean; Env?: Record<string, string> };
    OpenCode?: { Command: string; Enabled: boolean; Env?: Record<string, string> };
    OpenClaw?: { Command: string; Enabled: boolean; Env?: Record<string, string> };
    Aider?: { Command: string; Enabled: boolean; Env?: Record<string, string> };
  };
  Workspace: { Name: string; Path: string; Version: number };
  Logs: { Path: string; MaxBytes: number };
  Env: Record<string, string>;
  Performance: Record<string, number>;
  Services: {
    GitHub?: { Command: string; Enabled: boolean };
    GitLab?: { Command: string; Enabled: boolean };
    Jira?: { Command: string; Enabled: boolean };
  };
  Roster: { Agents: { Name: string; Role: string; Tool: string; Runtime: string }[] };
}

export const api = {
  listAgents: () => request<Agent[]>('/agents'),
  getAgent: (name: string) => request<Agent>(`/agents/${encodeURIComponent(name)}`),
  getAgentPeek: (name: string, lines = 50) =>
    request<{ output: string }>(`/agents/${encodeURIComponent(name)}/peek?${new URLSearchParams({ lines: String(lines) })}`),
  startAgent: (name: string) => request<Agent>(`/agents/${encodeURIComponent(name)}/start`, { method: 'POST' }),
  stopAgent: (name: string) => request<void>(`/agents/${encodeURIComponent(name)}/stop`, { method: 'POST' }),
  sendToAgent: (name: string, message: string) =>
    request<void>(`/agents/${encodeURIComponent(name)}/send`, { method: 'POST', body: JSON.stringify({ message }) }),
  getAgentStats: (name: string, limit = 20) =>
    request<AgentStatsRecord[]>(`/agents/${encodeURIComponent(name)}/stats?${new URLSearchParams({ limit: String(limit) })}`),

  listChannels: () => request<Channel[]>('/channels'),
  getChannelHistory: (name: string, limit = 50) =>
    request<ChannelMessage[]>(`/channels/${encodeURIComponent(name)}/history?${new URLSearchParams({ limit: String(limit) })}`),
  sendToChannel: (name: string, message: string, sender = 'web') =>
    request<ChannelMessage>(`/channels/${encodeURIComponent(name)}/messages`, { method: 'POST', body: JSON.stringify({ sender, content: message }) }),

  getCostSummary: () => request<CostSummary>('/costs'),
  getCostByAgent: () => request<AgentCostSummary[]>('/costs/agents'),
  getCostByModel: () => request<ModelCostSummary[]>('/costs/models'),
  getCostDaily: (days = 14) => request<DailyCost[]>(`/costs/daily?days=${days}`),
  getCostBudgets: () => request<Budget[]>('/costs/budgets'),
  getCostBudgetStatus: (scope: string) => request<BudgetStatus>(`/costs/budgets/${encodeURIComponent(scope)}`),
  setCostBudget: (budget: { scope: string; period: string; limit_usd: number; alert_at: number; hard_stop: boolean }) =>
    request<Budget>('/costs/budgets', { method: 'POST', body: JSON.stringify(budget) }),
  deleteCostBudget: (scope: string) =>
    request<void>(`/costs/budgets/${encodeURIComponent(scope)}`, { method: 'DELETE' }),

  listRoles: () => request<Record<string, Role>>('/workspace/roles'),
  getRole: (name: string) => request<Role>(`/roles/${encodeURIComponent(name)}`),
  createRole: (role: Partial<Role> & { Name: string }) =>
    request<Role>(`/roles`, { method: 'POST', body: JSON.stringify(role) }),
  updateRole: (name: string, role: Partial<Role>) =>
    request<Role>(`/roles/${encodeURIComponent(name)}`, { method: 'PUT', body: JSON.stringify(role) }),
  deleteRole: (name: string) =>
    request<void>(`/roles/${encodeURIComponent(name)}`, { method: 'DELETE' }),
  listTools: () => request<Tool[]>('/tools'),
  listMCP: () => request<MCPServer[]>('/mcp'),
  getLogs: (tail = 50) => request<EventLogEntry[]>(`/logs?${new URLSearchParams({ tail: String(tail) })}`),
  getAgentLogs: (agent: string, tail = 50) => request<EventLogEntry[]>(`/logs/${encodeURIComponent(agent)}?${new URLSearchParams({ tail: String(tail) })}`),
  getDoctor: () => request<DoctorReport>('/doctor'),

  listCron: () => request<CronJob[]>('/cron'),
  listSecrets: () => request<Secret[]>('/secrets'),
  getWorkspace: () => request<WorkspaceInfo>('/workspace'),
  getWorkspaceStatus: () => request<Record<string, unknown>>('/workspace/status'),

  getStatsSystem: () => request<SystemStats>('/stats/system'),
  getStatsSummary: () => request<StatsSummary>('/stats/summary'),
  getStatsChannels: () => request<ChannelStats[]>('/stats/channels'),

  listTeams: () => request<Team[]>('/teams'),
  getTeam: (name: string) => request<Team>(`/teams/${encodeURIComponent(name)}`),

  getSettings: () => request<SettingsConfig>('/settings'),
  updateSettings: (patch: Record<string, unknown>) =>
    request<SettingsConfig>('/settings', { method: 'PUT', body: JSON.stringify(patch) }),
};
