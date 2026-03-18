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

export const api = {
  listAgents: () => request<Agent[]>('/agents'),
  getAgent: (name: string) => request<Agent>(`/agents/${name}`),
  startAgent: (name: string) => request<Agent>(`/agents/${name}/start`, { method: 'POST' }),
  stopAgent: (name: string) => request<void>(`/agents/${name}/stop`, { method: 'POST' }),
  sendToAgent: (name: string, message: string) =>
    request<void>(`/agents/${name}/send`, { method: 'POST', body: JSON.stringify({ message }) }),

  listChannels: () => request<Channel[]>('/channels'),
  getChannelHistory: (name: string, limit = 50) =>
    request<ChannelMessage[]>(`/channels/${name}/history?limit=${limit}`),
  sendToChannel: (name: string, message: string, sender = 'web') =>
    request<ChannelMessage>(`/channels/${name}/messages`, { method: 'POST', body: JSON.stringify({ sender, content: message }) }),

  getCostSummary: () => request<CostSummary>('/costs'),
  getCostByAgent: () => request<AgentCostSummary[]>('/costs/agents'),

  listRoles: () => request<Record<string, Role>>('/workspace/roles'),
  listTools: () => request<Tool[]>('/tools'),
  listMCP: () => request<MCPServer[]>('/mcp'),
  getLogs: (tail = 50) => request<EventLogEntry[]>(`/logs?tail=${tail}`),
  getDoctor: () => request<DoctorReport>('/doctor'),

  listCron: () => request<CronJob[]>('/cron'),
  listSecrets: () => request<Secret[]>('/secrets'),
  getWorkspaceStatus: () => request<Record<string, unknown>>('/workspace/status'),
};
