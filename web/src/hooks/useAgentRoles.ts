import { useState, useEffect, useRef } from "react";
import { api } from "../api/client";

/** Module-level cache to avoid re-fetching across component remounts. */
let cachedRoleMap: Record<string, string> | null = null;
let cachedStateMap: Record<string, string> | null = null;

/**
 * Fetches agent list once and returns name→role and name→state maps.
 * Cached at module level so it persists across component remounts.
 */
export function useAgentRoles(): {
  roleMap: Record<string, string>;
  stateMap: Record<string, string>;
  loading: boolean;
} {
  const [roleMap, setRoleMap] = useState<Record<string, string>>(
    cachedRoleMap ?? {},
  );
  const [stateMap, setStateMap] = useState<Record<string, string>>(
    cachedStateMap ?? {},
  );
  const [loading, setLoading] = useState(cachedRoleMap === null);
  const fetched = useRef(cachedRoleMap !== null);

  useEffect(() => {
    if (fetched.current) return;
    fetched.current = true;
    void (async () => {
      try {
        const agents = await api.listAgents();
        const roles: Record<string, string> = {};
        const states: Record<string, string> = {};
        for (const agent of agents) {
          roles[agent.name] = agent.role;
          states[agent.name] = agent.state;
        }
        cachedRoleMap = roles;
        cachedStateMap = states;
        setRoleMap(roles);
        setStateMap(states);
      } catch {
        // keep empty maps
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  return { roleMap, stateMap, loading };
}
