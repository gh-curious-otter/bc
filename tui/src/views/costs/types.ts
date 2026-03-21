export type SortMode = 'cost' | 'name' | 'percent';

export interface AgentEntry {
  name: string;
  cost: number;
  percent: number;
}
