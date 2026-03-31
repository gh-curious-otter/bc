import { useCallback, useEffect, useRef, useState, Fragment } from "react";
import { api } from "../api/client";
import type { Agent } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { useWebSocket } from "../hooks/useWebSocket";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

interface LogEntry {
  id?: number;
  type: string;
  agent?: string;
  message?: string;
  created_at?: string;
}

export function Logs() {
  const [agentFilter, setAgentFilter] = useState("");
  const [agents, setAgents] = useState<Agent[]>([]);
  const [streamedLogs, setStreamedLogs] = useState<LogEntry[]>([]);
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());

  const toggleRow = (key: string) => {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const formatJSON = (msg: string): string => {
    try {
      return JSON.stringify(JSON.parse(msg), null, 2);
    } catch {
      return msg;
    }
  };

  const summarize = (msg: string): string => {
    try {
      const obj = JSON.parse(msg);
      const parts: string[] = [];
      if (obj.event) parts.push(obj.event);
      if (obj.tool_name) parts.push(obj.tool_name);
      if (obj.task) parts.push(obj.task);
      else if (obj.command) parts.push(obj.command);
      return parts.join(" - ") || msg.slice(0, 80);
    } catch {
      return msg.slice(0, 80);
    }
  };
  const { subscribe } = useWebSocket();
  const bottomRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);

  useEffect(() => {
    api
      .listAgents()
      .then(setAgents)
      .catch(() => {});
  }, []);

  // Subscribe to real-time hook events via SSE
  useEffect(() => {
    return subscribe("agent.hook", (event) => {
      const d = event.data;
      const entry: LogEntry = {
        type: `hook.${(d.event as string) || "unknown"}`,
        agent: d.agent as string,
        message: JSON.stringify(d),
        created_at: event.timestamp || new Date().toISOString(),
      };
      // Filter if agent filter is active
      if (agentFilter && entry.agent !== agentFilter) return;
      setStreamedLogs((prev) => [...prev.slice(-499), entry]);
    });
  }, [subscribe, agentFilter]);

  // Auto-scroll to bottom when new events arrive
  useEffect(() => {
    if (autoScroll) {
      bottomRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [streamedLogs, autoScroll]);

  // Clear streamed logs when filter changes
  useEffect(() => {
    setStreamedLogs([]);
  }, [agentFilter]);

  const fetcher = useCallback(() => {
    if (agentFilter) {
      return api.getAgentLogs(agentFilter, 100);
    }
    return api.getLogs(100);
  }, [agentFilter]);

  const {
    data: polledLogs,
    loading,
    error,
    refresh,
    timedOut,
  } = usePolling(fetcher, 10000); // Slower polling since we have SSE

  // Merge polled logs with streamed logs, dedup by timestamp+type+agent
  const allLogs = [...(polledLogs || []), ...streamedLogs];

  if (loading && !polledLogs && streamedLogs.length === 0) {
    return (
      <div className="p-6 space-y-4">
        <div className="h-6 w-28 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="table" rows={6} />
      </div>
    );
  }
  if (timedOut && !polledLogs) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Logs took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !polledLogs) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load logs"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <h1 className="text-xl font-bold">Event Log</h1>
          <select
            value={agentFilter}
            onChange={(e) => setAgentFilter(e.target.value)}
            className="text-sm rounded border border-bc-border bg-bc-surface px-2 py-1 text-bc-fg focus:outline-none focus:ring-1 focus:ring-bc-accent"
          >
            <option value="">All agents</option>
            {agents.map((a) => (
              <option key={a.name} value={a.name}>
                {a.name}
              </option>
            ))}
          </select>
          <label className="flex items-center gap-1.5 text-xs text-bc-muted cursor-pointer select-none">
            <input
              type="checkbox"
              checked={autoScroll}
              onChange={(e) => setAutoScroll(e.target.checked)}
              className="rounded"
            />
            Auto-scroll
          </label>
        </div>
        <span className="text-sm text-bc-muted">
          {allLogs.length} events
          {streamedLogs.length > 0 && (
            <span className="ml-2 text-bc-success">
              +{streamedLogs.length} live
            </span>
          )}
        </span>
      </div>

      {allLogs.length === 0 ? (
        <EmptyState
          icon="[]"
          title="No events recorded yet"
          description="Events will appear here in real-time as agents work."
        />
      ) : (
        <div className="rounded border border-bc-border overflow-hidden">
          <div className="overflow-auto max-h-[70vh]">
            <table className="w-full text-sm">
              <thead className="sticky top-0 bg-bc-surface">
                <tr className="border-b border-bc-border text-left">
                  <th className="px-4 py-2 font-medium text-bc-muted">Time</th>
                  <th className="px-4 py-2 font-medium text-bc-muted">Type</th>
                  <th className="px-4 py-2 font-medium text-bc-muted">Agent</th>
                  <th className="px-4 py-2 font-medium text-bc-muted">
                    Message
                  </th>
                </tr>
              </thead>
              <tbody>
                {allLogs.map((entry, i) => {
                  const rowKey = String(entry.id || `stream-${i}`);
                  const isExpanded = expandedRows.has(rowKey);
                  const isJSON = entry.message?.startsWith("{");
                  return (
                    <Fragment key={rowKey}>
                      <tr
                        className="border-b border-bc-border/50 cursor-pointer hover:bg-bc-surface/50"
                        onClick={() => entry.message && toggleRow(rowKey)}
                      >
                        <td className="px-4 py-2 text-bc-muted whitespace-nowrap">
                          {entry.created_at
                            ? new Date(entry.created_at).toLocaleTimeString()
                            : "\u2014"}
                        </td>
                        <td className="px-4 py-2">
                          <span className="text-xs px-2 py-0.5 rounded bg-bc-border text-bc-muted">
                            {entry.type}
                          </span>
                        </td>
                        <td className="px-4 py-2 font-medium">
                          {entry.agent || "\u2014"}
                        </td>
                        <td className="px-4 py-2 text-bc-muted text-xs">
                          <span className="flex items-center gap-1">
                            {isJSON && (
                              <span className="text-bc-muted/50 text-[10px]">
                                {isExpanded ? "\u25BC" : "\u25B6"}
                              </span>
                            )}
                            {isJSON ? summarize(entry.message!) : (entry.message || "\u2014")}
                          </span>
                        </td>
                      </tr>
                      {isExpanded && entry.message && (
                        <tr className="border-b border-bc-border/50">
                          <td colSpan={4} className="px-4 py-2 bg-bc-surface">
                            <pre className="text-xs font-mono text-bc-muted whitespace-pre-wrap overflow-x-auto max-h-64 overflow-y-auto">
                              {formatJSON(entry.message)}
                            </pre>
                          </td>
                        </tr>
                      )}
                    </Fragment>
                  );
                })}
              </tbody>
            </table>
            <div ref={bottomRef} />
          </div>
        </div>
      )}
    </div>
  );
}
