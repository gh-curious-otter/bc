import { useMemo } from 'react';
import type { Agent } from '../types';

/** State counts for header summary */
export interface StateCounts {
  working: number;
  idle: number;
  stuck: number;
  error: number;
  stopped: number;
}

/** Role group with agents and stats (#1346) */
export interface RoleGroup {
  role: string;
  agents: Agent[];
  working: number;
  idle: number;
  stuck: number;
}

/** Count agents by state for header summary */
export function countAgentStates(agents: Agent[]): StateCounts {
  const counts: StateCounts = { working: 0, idle: 0, stuck: 0, error: 0, stopped: 0 };
  for (const agent of agents) {
    if (agent.state === 'working' || agent.state === 'starting') {
      counts.working++;
    } else if (agent.state === 'idle' || agent.state === 'done') {
      counts.idle++;
    } else if (agent.state === 'stuck') {
      counts.stuck++;
    } else if (agent.state === 'error') {
      counts.error++;
    } else {
      // stopped or other states
      counts.stopped++;
    }
  }
  return counts;
}

/** Group agents by role for grouped view (#1346) */
export function groupAgentsByRole(agents: Agent[]): RoleGroup[] {
  const groups = new Map<string, Agent[]>();

  for (const agent of agents) {
    const role = agent.role;
    const existing = groups.get(role) ?? [];
    existing.push(agent);
    groups.set(role, existing);
  }

  // Convert to array and calculate stats
  const result: RoleGroup[] = [];
  for (const [role, roleAgents] of groups) {
    const counts = countAgentStates(roleAgents);
    result.push({
      role,
      agents: roleAgents,
      working: counts.working,
      idle: counts.idle,
      stuck: counts.stuck,
    });
  }

  // Sort by role name (engineers first, then alphabetically)
  return result.sort((a, b) => {
    if (a.role === 'engineer') return -1;
    if (b.role === 'engineer') return 1;
    return a.role.localeCompare(b.role);
  });
}

/**
 * Normalize task status by replacing cooking metaphors with clearer terms.
 * Issue #970 - Replace cooking terminology from Claude Code status line.
 */
export function normalizeTask(task: string | undefined): string {
  if (!task) return '-';
  // #1364 Issue 3: Normalize cooking/quirky terms to clear status verbs
  const replacements: [string, string][] = [
    ['Sautéed', 'Working'],
    ['Sauteed', 'Working'], // ASCII fallback
    ['Brewed', 'Done'],
    ['Cooked', 'Processed'],
    ['Cogitated', 'Thinking'],
    ['Marinated', 'Idle'],
    ['Frolicking', 'Active'],
    ['Grooving', 'Active'],
  ];
  for (const [old, replacement] of replacements) {
    if (task.includes(old)) {
      return task.replace(old, replacement);
    }
  }
  return task;
}

/**
 * Abbreviate role names for compact display (#1364)
 * product-manager → PM, tech-lead → TL, engineer → Eng
 */
export function abbreviateRole(role: string): string {
  const abbreviations: Record<string, string> = {
    'product-manager': 'PM',
    'tech-lead': 'TL',
    engineer: 'Eng',
    manager: 'Mgr',
    root: 'Root',
  };
  return abbreviations[role] ?? role;
}

/** Item types for grouped view navigation (#1346) */
export type GroupedItem =
  | { type: 'header'; role: string; group: RoleGroup }
  | { type: 'agent'; agent: Agent; role: string };

/**
 * Hook for agent grouping and filtering logic.
 * Extracts grouping concerns from AgentsView (#1592).
 */
export function useAgentGroups(
  agents: Agent[],
  searchQuery: string,
  groupedView: boolean,
  collapsedRoles: Set<string>
) {
  // Filter agents by search query
  const agentList = useMemo(() => {
    if (!searchQuery) return agents;
    const query = searchQuery.toLowerCase();
    return agents.filter(
      (agent) =>
        agent.name.toLowerCase().includes(query) ||
        agent.role.toLowerCase().includes(query) ||
        agent.state.toLowerCase().includes(query)
    );
  }, [agents, searchQuery]);

  // Calculate state counts for header summary (#1331)
  const stateCounts = useMemo(() => countAgentStates(agentList), [agentList]);

  // Group agents by role for grouped view
  const roleGroups = useMemo(() => groupAgentsByRole(agentList), [agentList]);

  // Build flat list of visible items for navigation in grouped view
  const visibleItems = useMemo((): GroupedItem[] => {
    if (!groupedView) {
      // Return agents wrapped as GroupedItem for consistent typing
      return agentList.map((agent) => ({ type: 'agent' as const, agent, role: agent.role }));
    }

    const items: GroupedItem[] = [];
    for (const group of roleGroups) {
      items.push({ type: 'header', role: group.role, group });
      if (!collapsedRoles.has(group.role)) {
        for (const agent of group.agents) {
          items.push({ type: 'agent', agent, role: group.role });
        }
      }
    }
    return items;
  }, [groupedView, roleGroups, collapsedRoles, agentList]);

  return {
    agentList,
    stateCounts,
    roleGroups,
    visibleItems,
  };
}
