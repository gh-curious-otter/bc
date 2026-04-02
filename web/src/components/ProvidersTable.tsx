import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import type { ProviderInfo } from "../api/client";
import { EmptyState } from "./EmptyState";

type SortKey = "name" | "status" | "version" | "agent_count" | "total_tokens" | "total_cost_usd";
type SortDir = "asc" | "desc";

function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(0)}K`;
  return String(n);
}

function formatCost(n: number): string {
  if (n === 0) return "$0.00";
  return `$${n.toFixed(2)}`;
}

function statusOrder(p: ProviderInfo): number {
  if (p.installed && p.agent_count > 0) return 0; // active
  if (p.installed) return 1; // idle
  return 2; // not installed
}

function StatusDot({ provider }: { provider: ProviderInfo }) {
  if (!provider.installed) {
    return <span className="inline-flex items-center gap-1.5 text-bc-error"><span className="text-xs">&#10005;</span> N/A</span>;
  }
  if (provider.agent_count > 0) {
    return <span className="inline-flex items-center gap-1.5 text-bc-success"><span className="w-2 h-2 rounded-full bg-bc-success inline-block" /> Active</span>;
  }
  return <span className="inline-flex items-center gap-1.5 text-bc-muted"><span className="w-2 h-2 rounded-full bg-bc-muted inline-block" /> Idle</span>;
}

interface Props {
  providers: ProviderInfo[];
  search: string;
}

export function ProvidersTable({ providers, search }: Props) {
  const [sortKey, setSortKey] = useState<SortKey>("name");
  const [sortDir, setSortDir] = useState<SortDir>("asc");
  const navigate = useNavigate();

  const filtered = useMemo(() => {
    const q = search.toLowerCase().trim();
    if (!q) return providers;
    return providers.filter((p) => p.name.toLowerCase().includes(q));
  }, [providers, search]);

  const sorted = useMemo(() => {
    const arr = [...filtered];
    arr.sort((a, b) => {
      let cmp = 0;
      switch (sortKey) {
        case "name":
          cmp = a.name.localeCompare(b.name);
          break;
        case "status":
          cmp = statusOrder(a) - statusOrder(b);
          break;
        case "version":
          cmp = (a.version || "").localeCompare(b.version || "");
          break;
        case "agent_count":
          cmp = a.agent_count - b.agent_count;
          break;
        case "total_tokens":
          cmp = a.total_tokens - b.total_tokens;
          break;
        case "total_cost_usd":
          cmp = a.total_cost_usd - b.total_cost_usd;
          break;
      }
      return sortDir === "asc" ? cmp : -cmp;
    });
    return arr;
  }, [filtered, sortKey, sortDir]);

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("asc");
    }
  };

  const sortIndicator = (key: SortKey) => {
    if (sortKey !== key) return null;
    return <span className="ml-1 text-bc-accent">{sortDir === "asc" ? "\u25B2" : "\u25BC"}</span>;
  };

  if (sorted.length === 0) {
    return (
      <EmptyState
        icon="*"
        title={search ? "No matching providers" : "No providers"}
        description={search ? "Try a different search term." : "No AI providers configured."}
      />
    );
  }

  const columns: { key: SortKey; label: string; className?: string }[] = [
    { key: "name", label: "Provider" },
    { key: "status", label: "Status" },
    { key: "version", label: "Version" },
    { key: "agent_count", label: "Agents", className: "text-right" },
    { key: "total_tokens", label: "Tokens", className: "text-right" },
    { key: "total_cost_usd", label: "Cost", className: "text-right" },
  ];

  return (
    <div className="rounded border border-bc-border overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-bc-border bg-bc-surface">
            {columns.map((col) => (
              <th
                key={col.key}
                onClick={() => toggleSort(col.key)}
                className={`px-4 py-2 font-medium text-bc-muted cursor-pointer select-none hover:text-bc-text transition-colors text-left ${col.className ?? ""}`}
              >
                {col.label}{sortIndicator(col.key)}
              </th>
            ))}
            <th className="px-4 py-2 font-medium text-bc-muted text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((p) => (
            <tr
              key={p.name}
              onClick={() => navigate(`/tools/${encodeURIComponent(p.name)}`)}
              className="border-b border-bc-border/50 cursor-pointer hover:bg-bc-surface transition-colors"
            >
              <td className="px-4 py-2.5 font-medium">{p.name}</td>
              <td className="px-4 py-2.5 text-xs"><StatusDot provider={p} /></td>
              <td className="px-4 py-2.5 text-xs text-bc-muted font-mono">{p.version || "\u2014"}</td>
              <td className="px-4 py-2.5 text-right tabular-nums">{p.agent_count}</td>
              <td className="px-4 py-2.5 text-right tabular-nums text-bc-muted">{formatTokens(p.total_tokens)}</td>
              <td className="px-4 py-2.5 text-right tabular-nums">{formatCost(p.total_cost_usd)}</td>
              <td className="px-4 py-2.5 text-right">
                <div className="flex items-center justify-end gap-1.5" onClick={(e) => e.stopPropagation()}>
                  {!p.installed && p.install_hint && (
                    <button
                      type="button"
                      onClick={() => navigate(`/tools/${encodeURIComponent(p.name)}`)}
                      className="text-xs px-2 py-0.5 rounded bg-bc-warning/10 text-bc-warning hover:bg-bc-warning/20 transition-colors"
                    >
                      Install
                    </button>
                  )}
                  {p.installed && p.install_hint && (
                    <span className="text-xs px-2 py-0.5 rounded bg-bc-info/10 text-bc-info">
                      Update
                    </span>
                  )}
                  <button
                    type="button"
                    onClick={() => navigate(`/tools/${encodeURIComponent(p.name)}`)}
                    className="text-xs px-1.5 py-0.5 rounded border border-bc-border text-bc-muted hover:text-bc-text hover:border-bc-accent/50 transition-colors"
                    aria-label={`Configure ${p.name}`}
                  >
                    &#9881;
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
