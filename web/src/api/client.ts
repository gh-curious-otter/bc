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
  total_cost_usd: number;
  total_tokens: number;
  input_tokens?: number;
  output_tokens?: number;
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
  runtime_backend?: string;
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

export interface BudgetStatus {
  scope: string;
  period: string;
  limit_usd: number;
  alert_at: number;
  hard_stop: boolean;
  id: number;
  updated_at: string;
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
    request<AgentStatsRecord[]>(`/agents/${encodeURIComponent(name)}/stats?limit=${limit}`),

  listChannels: () => request<Channel[]>('/channels'),
  getChannelHistory: (name: string, limit = 50) =>
    request<ChannelMessage[]>(`/channels/${encodeURIComponent(name)}/history?${new URLSearchParams({ limit: String(limit) })}`),
  sendToChannel: (name: string, message: string, sender = 'web') =>
    request<ChannelMessage>(`/channels/${encodeURIComponent(name)}/messages`, { method: 'POST', body: JSON.stringify({ sender, content: message }) }),

  getCostSummary: () => request<CostSummary>('/costs'),
  getCostByAgent: () => request<AgentCostSummary[]>('/costs/agents'),
  getCostByModel: () => request<ModelCostSummary[]>('/costs/models'),
  getCostDaily: (days = 14) => request<DailyCost[]>(`/costs/daily?days=${days}`),
  getCostBudgets: () => request<BudgetStatus[]>('/costs/budgets'),

  listRoles: () => request<Record<string, Role>>('/workspace/roles'),
  listTools: () => request<Tool[]>('/tools'),
  listMCP: () => request<MCPServer[]>('/mcp'),
  getLogs: (tail = 50) => request<EventLogEntry[]>(`/logs?${new URLSearchParams({ tail: String(tail) })}`),
  getDoctor: () => request<DoctorReport>('/doctor'),

  listCron: () => request<CronJob[]>('/cron'),
  listSecrets: () => request<Secret[]>('/secrets'),
  getWorkspace: () => request<WorkspaceInfo>('/workspace'),
  getWorkspaceStatus: () => request<Record<string, unknown>>('/workspace/status'),

  getStatsSystem: () => request<SystemStats>('/stats/system'),
  getStatsSummary: () => request<StatsSummary>('/stats/summary'),
  getStatsChannels: () => request<ChannelStats[]>('/stats/channels'),
};
