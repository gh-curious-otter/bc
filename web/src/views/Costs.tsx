import { useCallback, useEffect, useState } from "react";
import { api } from "../api/client";
import type {
  CostSummary,
  AgentCostSummary,
  ModelCostSummary,
  DailyCost,
  BudgetStatus,
} from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { useWebSocket } from "../hooks/useWebSocket";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

interface CostData {
  summary: CostSummary;
  byAgent: AgentCostSummary[];
  byModel: ModelCostSummary[];
  daily: DailyCost[];
  budgets: BudgetStatus[];
}

function CostCard({
  label,
  value,
  sub,
}: {
  label: string;
  value: string;
  sub?: string;
}) {
  return (
    <div className="rounded border border-bc-border bg-bc-surface p-4 text-center">
      <p className="text-xs text-bc-muted uppercase tracking-wide">{label}</p>
      <p className="mt-1 text-xl font-bold">{value}</p>
      {sub && <p className="mt-0.5 text-xs text-bc-muted">{sub}</p>}
    </div>
  );
}

function progressColor(pct: number): string {
  if (pct >= 80) return "bg-bc-error";
  if (pct >= 50) return "bg-bc-warning";
  return "bg-bc-success";
}

function AgentBreakdown({
  agents,
  total,
}: {
  agents: AgentCostSummary[];
  total: number;
}) {
  if (agents.length === 0) {
    return (
      <div className="text-sm text-bc-muted py-4 text-center">
        No agent cost data yet.
      </div>
    );
  }

  const sorted = [...agents].sort(
    (a, b) => b.total_cost_usd - a.total_cost_usd,
  );
  const maxCost =
    total > 0 ? total : Math.max(...sorted.map((a) => a.total_cost_usd), 1);

  return (
    <div className="space-y-3">
      {sorted.map((agent) => {
        const pct = maxCost > 0 ? (agent.total_cost_usd / maxCost) * 100 : 0;
        return (
          <div key={agent.agent_id}>
            <div className="flex items-center justify-between text-sm mb-1">
              <span className="font-medium truncate mr-2">
                {agent.agent_id}
              </span>
              <span className="text-bc-muted whitespace-nowrap">
                ${agent.total_cost_usd.toFixed(4)}
                <span className="ml-2 text-xs">
                  ({(agent.input_tokens + agent.output_tokens).toLocaleString()}{" "}
                  tokens)
                </span>
              </span>
            </div>
            <div className="h-2 w-full rounded-full bg-bc-border/40 overflow-hidden">
              <div
                className={`h-full rounded-full transition-all ${progressColor(pct)}`}
                style={{ width: `${Math.max(pct, 1)}%` }}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
}

function ModelBreakdown({ models }: { models: ModelCostSummary[] }) {
  if (models.length === 0) {
    return (
      <div className="text-sm text-bc-muted py-4 text-center">
        No model cost data yet.
      </div>
    );
  }

  const sorted = [...models].sort(
    (a, b) => b.total_cost_usd - a.total_cost_usd,
  );
  const totalCost = sorted.reduce((sum, m) => sum + m.total_cost_usd, 0);

  return (
    <div className="space-y-2">
      {sorted.map((model) => {
        const pct =
          totalCost > 0 ? (model.total_cost_usd / totalCost) * 100 : 0;
        return (
          <div key={model.model} className="flex items-center gap-3 text-sm">
            <span className="w-40 truncate font-medium" title={model.model}>
              {model.model || "unknown"}
            </span>
            <div className="flex-1 h-2 rounded-full bg-bc-border/40 overflow-hidden">
              <div
                className="h-full rounded-full bg-blue-500 transition-all"
                style={{ width: `${Math.max(pct, 1)}%` }}
              />
            </div>
            <span className="w-20 text-right text-bc-muted whitespace-nowrap">
              ${model.total_cost_usd.toFixed(4)}
            </span>
            <span className="w-14 text-right text-xs text-bc-muted">
              {pct.toFixed(1)}%
            </span>
          </div>
        );
      })}
    </div>
  );
}

function DailyChart({ daily }: { daily: DailyCost[] }) {
  if (daily.length === 0) {
    return (
      <div className="text-sm text-bc-muted py-4 text-center">
        No daily cost data yet.
      </div>
    );
  }

  const maxCost = Math.max(...daily.map((d) => d.cost_usd), 0.001);

  return (
    <div>
      <div className="flex items-end gap-1" style={{ height: "120px" }}>
        {daily.map((day) => {
          const heightPct = (day.cost_usd / maxCost) * 100;
          const dateStr = day.date.slice(5); // MM-DD
          return (
            <div
              key={day.date}
              className="flex-1 flex flex-col items-center justify-end h-full group relative"
            >
              <div
                className="w-full rounded-t bg-blue-500 hover:bg-blue-400 transition-colors min-h-[2px]"
                style={{ height: `${Math.max(heightPct, 1.5)}%` }}
                title={`${day.date}: $${day.cost_usd.toFixed(4)}`}
              />
              <div className="absolute -top-6 left-1/2 -translate-x-1/2 hidden group-hover:block bg-bc-surface border border-bc-border rounded px-1.5 py-0.5 text-xs whitespace-nowrap z-10 shadow">
                ${day.cost_usd.toFixed(4)}
              </div>
              {daily.length <= 14 && (
                <span className="text-[10px] text-bc-muted mt-1 leading-none">
                  {dateStr}
                </span>
              )}
            </div>
          );
        })}
      </div>
      {daily.length > 14 && (
        <div className="flex justify-between text-[10px] text-bc-muted mt-1">
          <span>{daily[0]?.date}</span>
          <span>{daily[daily.length - 1]?.date}</span>
        </div>
      )}
    </div>
  );
}

type FormStatus =
  | { type: "idle" }
  | { type: "saving" }
  | { type: "success" }
  | { type: "error"; message: string };

function AddBudgetForm({ onCreated }: { onCreated: () => void }) {
  const [scope, setScope] = useState("workspace");
  const [period, setPeriod] = useState("monthly");
  const [limitUsd, setLimitUsd] = useState("");
  const [alertAt, setAlertAt] = useState("");
  const [hardStop, setHardStop] = useState(false);
  const [status, setStatus] = useState<FormStatus>({ type: "idle" });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const limit = parseFloat(limitUsd);
    if (!scope.trim() || isNaN(limit) || limit <= 0) return;

    setStatus({ type: "saving" });
    try {
      const budget: {
        scope: string;
        period: string;
        limit_usd: number;
        alert_at?: number;
        hard_stop?: boolean;
      } = {
        scope: scope.trim(),
        period,
        limit_usd: limit,
        hard_stop: hardStop,
      };
      const alert = parseFloat(alertAt);
      if (!isNaN(alert) && alert > 0) {
        budget.alert_at = alert;
      }
      await api.createCostBudget(budget);
      setScope("workspace");
      setPeriod("monthly");
      setLimitUsd("");
      setAlertAt("");
      setHardStop(false);
      setStatus({ type: "success" });
      onCreated();
      setTimeout(() => setStatus({ type: "idle" }), 2000);
    } catch (err) {
      setStatus({
        type: "error",
        message: err instanceof Error ? err.message : "Failed to create budget",
      });
      setTimeout(() => setStatus({ type: "idle" }), 4000);
    }
  };

  return (
    <form
      onSubmit={handleSubmit}
      className="rounded border border-bc-border bg-bc-surface p-4 space-y-3"
    >
      <h3 className="text-sm font-medium text-bc-muted uppercase tracking-wide">
        Add Budget
      </h3>
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <div className="space-y-1">
          <label className="block text-sm text-bc-text">Scope</label>
          <select
            value={scope}
            onChange={(e) => setScope(e.target.value)}
            className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          >
            <option value="workspace">Workspace</option>
            <option value="agent">Agent</option>
          </select>
        </div>
        <div className="space-y-1">
          <label className="block text-sm text-bc-text">Period</label>
          <select
            value={period}
            onChange={(e) => setPeriod(e.target.value)}
            className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          >
            <option value="daily">Daily</option>
            <option value="weekly">Weekly</option>
            <option value="monthly">Monthly</option>
          </select>
        </div>
        <div className="space-y-1">
          <label className="block text-sm text-bc-text">Limit (USD)</label>
          <input
            type="number"
            step="0.01"
            min="0.01"
            value={limitUsd}
            onChange={(e) => setLimitUsd(e.target.value)}
            placeholder="10.00"
            className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <div className="space-y-1">
          <label className="block text-sm text-bc-text">Alert At (USD)</label>
          <input
            type="number"
            step="0.01"
            min="0"
            value={alertAt}
            onChange={(e) => setAlertAt(e.target.value)}
            placeholder="8.00"
            className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>
        <div className="flex items-end pb-1">
          <label className="flex items-center gap-2 text-sm text-bc-text cursor-pointer">
            <input
              type="checkbox"
              checked={hardStop}
              onChange={(e) => setHardStop(e.target.checked)}
              className="rounded border-bc-border"
            />
            Hard stop at limit
          </label>
        </div>
      </div>
      <div className="flex items-center gap-3">
        <button
          type="submit"
          disabled={
            status.type === "saving" ||
            !scope.trim() ||
            !limitUsd ||
            parseFloat(limitUsd) <= 0
          }
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-offset-1 focus-visible:ring-offset-bc-bg"
        >
          {status.type === "saving" ? "Adding..." : "Add Budget"}
        </button>
        {status.type === "success" && (
          <span className="text-xs text-bc-success">Budget added</span>
        )}
        {status.type === "error" && (
          <span className="text-xs text-bc-error">{status.message}</span>
        )}
      </div>
    </form>
  );
}

function BudgetDeleteButton({
  scope,
  onDeleted,
}: {
  scope: string;
  onDeleted: () => void;
}) {
  const [confirming, setConfirming] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const handleDelete = async () => {
    setDeleting(true);
    try {
      await api.deleteCostBudget(scope);
      onDeleted();
    } catch {
      setDeleting(false);
      setConfirming(false);
    }
  };

  if (confirming) {
    return (
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={handleDelete}
          disabled={deleting}
          className="px-2 py-1 rounded bg-bc-error text-bc-bg text-xs font-medium hover:bg-red-700 disabled:opacity-50 transition-colors focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-offset-1 focus-visible:ring-offset-bc-bg"
          aria-label={`Confirm delete budget ${scope}`}
        >
          {deleting ? "Deleting..." : "Confirm"}
        </button>
        <button
          type="button"
          onClick={() => setConfirming(false)}
          disabled={deleting}
          className="px-2 py-1 rounded border border-bc-border text-bc-muted text-xs hover:text-bc-text transition-colors focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-offset-1 focus-visible:ring-offset-bc-bg"
          aria-label="Cancel delete"
        >
          Cancel
        </button>
      </div>
    );
  }

  return (
    <button
      type="button"
      onClick={() => setConfirming(true)}
      className="px-2 py-1 rounded border border-bc-border text-bc-muted text-xs hover:text-bc-error hover:border-bc-error/50 transition-colors focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-offset-1 focus-visible:ring-offset-bc-bg"
      aria-label={`Delete budget ${scope}`}
    >
      Delete
    </button>
  );
}

function BudgetList({
  budgets,
  onDeleted,
}: {
  budgets: BudgetStatus[];
  onDeleted: () => void;
}) {
  if (budgets.length === 0) {
    return (
      <div className="text-sm text-bc-muted py-4 text-center">
        No budgets configured. Add one using the form above.
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {budgets.map((b) => {
        const usedPct =
          b.limit_usd > 0 ? Math.min((b.alert_at / b.limit_usd) * 100, 100) : 0;
        return (
          <div
            key={b.scope}
            className="rounded border border-bc-border bg-bc-surface p-4"
          >
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-3">
                <span className="font-medium text-sm">{b.scope}</span>
                <span className="text-xs px-2 py-0.5 rounded bg-bc-border/40 text-bc-muted">
                  {b.period}
                </span>
                {b.hard_stop && (
                  <span className="text-xs px-2 py-0.5 rounded bg-bc-error/20 text-bc-error">
                    Hard Stop
                  </span>
                )}
              </div>
              <BudgetDeleteButton scope={b.scope} onDeleted={onDeleted} />
            </div>
            <div className="flex items-center gap-3 text-sm mb-2">
              <span className="text-bc-muted">
                Limit:{" "}
                <span className="text-bc-text">${b.limit_usd.toFixed(2)}</span>
              </span>
              {b.alert_at > 0 && (
                <span className="text-bc-muted">
                  Alert at:{" "}
                  <span className="text-bc-text">${b.alert_at.toFixed(2)}</span>
                </span>
              )}
            </div>
            <div className="h-2 w-full rounded-full bg-bc-border/40 overflow-hidden">
              <div
                className={`h-full rounded-full transition-all ${progressColor(usedPct)}`}
                style={{ width: `${Math.max(usedPct, 1)}%` }}
              />
            </div>
            <div className="flex justify-between text-xs text-bc-muted mt-1">
              <span>Alert threshold: {usedPct.toFixed(0)}% of limit</span>
              <span>${b.limit_usd.toFixed(2)}</span>
            </div>
          </div>
        );
      })}
    </div>
  );
}

export function Costs() {
  const fetcher = useCallback(async (): Promise<CostData> => {
    let summary: CostSummary = {
      input_tokens: 0,
      output_tokens: 0,
      total_tokens: 0,
      total_cost_usd: 0,
      record_count: 0,
    };
    let byAgent: AgentCostSummary[] = [];
    let byModel: ModelCostSummary[] = [];
    let daily: DailyCost[] = [];
    let budgets: BudgetStatus[] = [];

    const results = await Promise.allSettled([
      api.getCostSummary(),
      api.getCostByAgent(),
      api.getCostByModel(),
      api.getCostDaily(14),
      api.getCostBudgets(),
    ]);

    if (results[0].status === "fulfilled") summary = results[0].value;
    if (results[1].status === "fulfilled") byAgent = results[1].value;
    if (results[2].status === "fulfilled") byModel = results[2].value;
    if (results[3].status === "fulfilled") daily = results[3].value;
    if (results[4].status === "fulfilled") budgets = results[4].value;

    return { summary, byAgent, byModel, daily, budgets };
  }, []);

  const { data, loading, error, refresh, timedOut } = usePolling(
    fetcher,
    10000,
  );
  const { subscribe } = useWebSocket();

  // Refresh cost data in real-time via SSE
  useEffect(() => {
    return subscribe("cost.updated", () => void refresh());
  }, [subscribe, refresh]);

  if (loading && !data) {
    return (
      <div className="p-6 space-y-6">
        <div className="h-6 w-20 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="cards" rows={3} />
        <LoadingSkeleton variant="table" rows={3} />
      </div>
    );
  }
  if (timedOut && !data) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Costs took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !data) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load costs"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (!data) return null;

  const activeAgents = data.byAgent.filter((a) => a.record_count > 0).length;

  return (
    <div className="p-6 space-y-6">
      <h1 className="text-xl font-bold">Costs</h1>

      {/* Summary Cards */}
      <div className="grid grid-cols-3 gap-4">
        <CostCard
          label="Total Cost"
          value={`$${(data.summary?.total_cost_usd ?? 0).toFixed(2)}`}
          sub={`${data.summary?.record_count ?? 0} records`}
        />
        <CostCard
          label="Total Tokens"
          value={(data.summary?.total_tokens ?? 0).toLocaleString()}
          sub={`In: ${(data.summary?.input_tokens ?? 0).toLocaleString()} / Out: ${(data.summary?.output_tokens ?? 0).toLocaleString()}`}
        />
        <CostCard
          label="Active Agents"
          value={String(activeAgents)}
          sub={`${data.byAgent.length} total`}
        />
      </div>

      {/* Daily Trend */}
      <section>
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide mb-3">
          Daily Trend (14d)
        </h2>
        <div className="rounded border border-bc-border bg-bc-surface p-4">
          <DailyChart daily={data.daily} />
        </div>
      </section>

      {/* Agent Breakdown */}
      <section>
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide mb-3">
          Cost by Agent
        </h2>
        <div className="rounded border border-bc-border bg-bc-surface p-4">
          <AgentBreakdown
            agents={data.byAgent}
            total={data.summary?.total_cost_usd ?? 0}
          />
        </div>
      </section>

      {/* Model Breakdown */}
      <section>
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide mb-3">
          Cost by Model
        </h2>
        <div className="rounded border border-bc-border bg-bc-surface p-4">
          <ModelBreakdown models={data.byModel} />
        </div>
      </section>

      {/* Budgets */}
      <section>
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide mb-3">
          Budgets
        </h2>
        <AddBudgetForm onCreated={refresh} />
        <div className="mt-4">
          <BudgetList budgets={data.budgets ?? []} onDeleted={refresh} />
        </div>
      </section>
    </div>
  );
}
