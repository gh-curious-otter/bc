import { useState, useEffect, useRef } from "react";
import { api } from "../api/client";

/** Module-level cache to avoid re-fetching across component remounts. */
let cachedRoleMap: Record<string, string> | null = null;

/**
 * Fetches agent list once and returns a name→role map.
 * Cached at module level so it persists across component remounts.
 */
export function useAgentRoles(): {
  roleMap: Record<string, string>;
  loading: boolean;
} {
  const [roleMap, setRoleMap] = useState<Record<string, string>>(
    cachedRoleMap ?? {},
  );
  const [loading, setLoading] = useState(cachedRoleMap === null);
  const fetched = useRef(cachedRoleMap !== null);

  useEffect(() => {
    if (fetched.current) return;
    fetched.current = true;
    void (async () => {
      try {
        const agents = await api.listAgents();
        const map: Record<string, string> = {};
        for (const agent of agents) {
          map[agent.name] = agent.role;
        }
        cachedRoleMap = map;
        setRoleMap(map);
      } catch {
        // keep empty map
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  return { roleMap, loading };
}
