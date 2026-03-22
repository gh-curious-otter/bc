const BASE = "/api";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
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

export interface CronLogEntry {
  id: number;
  job_name: string;
  status: string;
  output: string;
  error: string;
  started_at: string;
  finished_at: string;
  duration_ms: number;
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

export interface SettingsConfig {
  User: { Nickname: string };
  TUI: { Theme: string; Mode: string };
  Runtime: {
    Backend: string;
    Docker?: {
      Image: string;
      Network: string;
      ExtraMounts: string[];
      CPUs: number;
      MemoryMB: number;
    };
  };
  Providers: {
    Default: string;
    Claude?: {
      Command: string;
      Enabled: boolean;
      Env?: Record<string, string>;
    };
    Gemini?: {
      Command: string;
      Enabled: boolean;
      Env?: Record<string, string>;
    };
    Cursor?: {
      Command: string;
      Enabled: boolean;
      Env?: Record<string, string>;
    };
    Codex?: { Command: string; Enabled: boolean; Env?: Record<string, string> };
    OpenCode?: {
      Command: string;
      Enabled: boolean;
      Env?: Record<string, string>;
    };
    OpenClaw?: {
      Command: string;
      Enabled: boolean;
      Env?: Record<string, string>;
    };
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
  Roster: {
    Agents: { Name: string; Role: string; Tool: string; Runtime: string }[];
  };
}

export interface Daemon {
  name: string;
  runtime: string;
  cmd: string;
  image: string;
  container_id: string;
  restart: string;
  status: string;
  ports: string[];
  volumes: string[];
  env: string[];
  pid: number;
  created_at: string;
  started_at: string;
  stopped_at: string | null;
}

export const api = {
  listAgents: () => request<Agent[]>("/agents"),
  getAgent: (name: string) =>
    request<Agent>(`/agents/${encodeURIComponent(name)}`),
  getAgentPeek: (name: string, lines = 50) =>
    request<{ output: string }>(
      `/agents/${encodeURIComponent(name)}/peek?${new URLSearchParams({ lines: String(lines) })}`,
    ),
  startAgent: (name: string) =>
    request<Agent>(`/agents/${encodeURIComponent(name)}/start`, {
      method: "POST",
    }),
  stopAgent: (name: string) =>
    request<void>(`/agents/${encodeURIComponent(name)}/stop`, {
      method: "POST",
    }),
  createAgent: (opts: {
    name?: string;
    role: string;
    tool?: string;
    runtime?: string;
  }) =>
    request<Agent>("/agents", {
      method: "POST",
      body: JSON.stringify(opts),
    }),
  generateAgentName: () => request<{ name: string }>("/agents/generate-name"),
  deleteAgent: (name: string, force = false) =>
    request<void>(
      `/agents/${encodeURIComponent(name)}${force ? "?force=true" : ""}`,
      { method: "DELETE" },
    ),
  sendToAgent: (name: string, message: string) =>
    request<void>(`/agents/${encodeURIComponent(name)}/send`, {
      method: "POST",
      body: JSON.stringify({ message }),
    }),
  getAgentStats: (name: string, limit = 20) =>
    request<AgentStatsRecord[]>(
      `/agents/${encodeURIComponent(name)}/stats?${new URLSearchParams({ limit: String(limit) })}`,
    ),

  listChannels: () => request<Channel[]>("/channels"),
  getChannelHistory: (name: string, limit = 50) =>
    request<ChannelMessage[]>(
      `/channels/${encodeURIComponent(name)}/history?${new URLSearchParams({ limit: String(limit) })}`,
    ),
  sendToChannel: (name: string, message: string, sender = "web") =>
    request<ChannelMessage>(`/channels/${encodeURIComponent(name)}/messages`, {
      method: "POST",
      body: JSON.stringify({ sender, content: message }),
    }),

  getCostSummary: () => request<CostSummary>("/costs"),
  getCostByAgent: () => request<AgentCostSummary[]>("/costs/agents"),
  getCostByModel: () => request<ModelCostSummary[]>("/costs/models"),
  getCostDaily: (days = 14) =>
    request<DailyCost[]>(`/costs/daily?days=${days}`),
  getCostBudgets: () => request<BudgetStatus[]>("/costs/budgets"),

  listRoles: () => request<Record<string, Role>>("/workspace/roles"),
  listTools: () => request<Tool[]>("/tools"),
  enableTool: (name: string) =>
    request<{ enabled: boolean }>(`/tools/${encodeURIComponent(name)}/enable`, {
      method: "POST",
    }),
  disableTool: (name: string) =>
    request<{ enabled: boolean }>(
      `/tools/${encodeURIComponent(name)}/disable`,
      { method: "POST" },
    ),
  deleteTool: (name: string) =>
    request<void>(`/tools/${encodeURIComponent(name)}`, { method: "DELETE" }),
  listMCP: () => request<MCPServer[]>("/mcp"),
  registerMCP: (server: Omit<MCPServer, "enabled"> & { enabled?: boolean }) =>
    request<MCPServer>("/mcp", {
      method: "POST",
      body: JSON.stringify(server),
    }),
  removeMCP: (name: string) =>
    request<void>(`/mcp/${encodeURIComponent(name)}`, { method: "DELETE" }),
  enableMCP: (name: string) =>
    request<void>(`/mcp/${encodeURIComponent(name)}/enable`, {
      method: "POST",
    }),
  disableMCP: (name: string) =>
    request<void>(`/mcp/${encodeURIComponent(name)}/disable`, {
      method: "POST",
    }),
  getLogs: (tail = 50) =>
    request<EventLogEntry[]>(
      `/logs?${new URLSearchParams({ tail: String(tail) })}`,
    ),
  getAgentLogs: (agent: string, tail = 50) =>
    request<EventLogEntry[]>(
      `/logs?${new URLSearchParams({ tail: String(tail), agent })}`,
    ),
  getDoctor: () => request<DoctorReport>("/doctor"),

  listCron: () => request<CronJob[]>("/cron"),
  createCron: (job: { name: string; schedule: string; command: string }) =>
    request<CronJob>("/cron", { method: "POST", body: JSON.stringify(job) }),
  runCron: (name: string) =>
    request<void>(`/cron/${encodeURIComponent(name)}/run`, { method: "POST" }),
  enableCron: (name: string) =>
    request<void>(`/cron/${encodeURIComponent(name)}/enable`, {
      method: "POST",
    }),
  disableCron: (name: string) =>
    request<void>(`/cron/${encodeURIComponent(name)}/disable`, {
      method: "POST",
    }),
  deleteCron: (name: string) =>
    request<void>(`/cron/${encodeURIComponent(name)}`, { method: "DELETE" }),
  getCronLogs: (name: string) =>
    request<CronLogEntry[]>(`/cron/${encodeURIComponent(name)}/logs`),
  listSecrets: () => request<Secret[]>("/secrets"),
  createSecret: (name: string, value: string, description?: string) =>
    request<Secret>("/secrets", {
      method: "POST",
      body: JSON.stringify({ name, value, description: description ?? "" }),
    }),
  deleteSecret: (name: string) =>
    request<void>(`/secrets/${encodeURIComponent(name)}`, { method: "DELETE" }),
  getWorkspace: () => request<WorkspaceInfo>("/workspace"),
  getWorkspaceStatus: () =>
    request<Record<string, unknown>>("/workspace/status"),

  getStatsSystem: () => request<SystemStats>("/stats/system"),
  getStatsSummary: () => request<StatsSummary>("/stats/summary"),
  getStatsChannels: () => request<ChannelStats[]>("/stats/channels"),

  getSettings: () => request<SettingsConfig>("/settings"),
  updateSettings: (patch: Record<string, unknown>) =>
    request<SettingsConfig>("/settings", {
      method: "PUT",
      body: JSON.stringify(patch),
    }),

  listDaemons: () => request<Daemon[]>("/daemons"),
  stopDaemon: (name: string) =>
    request<{ status: string }>(`/daemons/${encodeURIComponent(name)}/stop`, {
      method: "POST",
    }),
  restartDaemon: (name: string) =>
    request<Daemon>(`/daemons/${encodeURIComponent(name)}/restart`, {
      method: "POST",
    }),
  removeDaemon: (name: string) =>
    request<void>(`/daemons/${encodeURIComponent(name)}`, { method: "DELETE" }),
};
