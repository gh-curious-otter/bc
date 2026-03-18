/** WebSocket event types from bcd */
export type WSEventType =
  | 'agent.state_changed'
  | 'agent.output'
  | 'channel.message'
  | 'cost.updated'
  | 'cost.budget_alert';

export interface WSEvent {
  type: WSEventType;
  data: Record<string, unknown>;
  timestamp: string;
}
