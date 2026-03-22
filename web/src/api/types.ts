/** WebSocket event types from bcd */
export type WSEventType =
  | "agent.created"
  | "agent.started"
  | "agent.stopped"
  | "agent.deleted"
  | "agent.state_changed"
  | "agent.output"
  | "channel.message"
  | "cost.updated"
  | "cost.budget_alert"
  | "team.updated"
  | "connected";

export interface WSEvent {
  type: WSEventType;
  data: Record<string, unknown>;
  timestamp: string;
}
