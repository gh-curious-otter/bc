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
  total_cost: number;
  input_tokens: number;
  output_tokens: number;
  record_count: number;
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
    request<{ messages: ChannelMessage[] }>(`/channels/${name}/history?limit=${limit}`),
  sendToChannel: (name: string, message: string) =>
    request<void>(`/channels/${name}/send`, { method: 'POST', body: JSON.stringify({ message }) }),

  getCostSummary: () => request<CostSummary>('/costs'),
  getCostByAgent: () => request<AgentCostSummary[]>('/costs/agents'),
};
