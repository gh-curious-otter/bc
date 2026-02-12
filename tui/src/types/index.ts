/**
 * TypeScript types for bc CLI JSON responses
 * Auto-generated from bc CLI output structures
 */

// Agent states matching pkg/agent/agent.go
export type AgentState =
  | 'idle'
  | 'starting'
  | 'working'
  | 'done'
  | 'stuck'
  | 'error'
  | 'stopped';

// Agent roles matching pkg/agent/agent.go
export type AgentRole =
  | 'root'
  | 'product-manager'
  | 'manager'
  | 'tech-lead'
  | 'engineer';

// Agent memory info
export interface AgentMemory {
  loaded_at: string;
  role_prompt: string;
}

// Agent from bc status --json
export interface Agent {
  id: string;
  name: string;
  role: AgentRole;
  state: AgentState;
  task: string;
  session: string;
  tool?: string;
  workspace: string;
  worktree_dir: string;
  memory_dir: string;
  started_at: string;
  updated_at: string;
  memory?: AgentMemory;
}

// Response from bc status --json
export interface StatusResponse {
  workspace: string;
  total: number;
  active: number;
  working: number;
  agents: Agent[];
}

// Channel types
export interface Channel {
  name: string;
  members: string[];
  created_at?: string;
}

export interface ChannelMessage {
  sender: string;
  message: string;
  time: string;
}

export interface ChannelHistory {
  channel: string;
  messages: ChannelMessage[];
}

// Response from bc channel list --json
export interface ChannelsResponse {
  channels: Channel[];
}

// Cost types
export interface CostRecord {
  agent_id: string;
  team_id: string;
  model: string;
  input_tokens: number;
  output_tokens: number;
  cost_usd: number;
  timestamp: string;
}

export interface CostSummary {
  total_cost: number;
  total_input_tokens: number;
  total_output_tokens: number;
  by_agent: Record<string, number>;
  by_team: Record<string, number>;
  by_model: Record<string, number>;
}

// Generic bc command result
export interface BcResult<T> {
  data: T | null;
  error: string | null;
  loading: boolean;
}

// Event types for real-time updates
export interface BcEvent {
  type: string;
  timestamp: string;
  agent: string;
  message: string;
  data?: Record<string, unknown>;
}
