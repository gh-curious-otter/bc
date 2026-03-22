import { useCallback, useEffect } from "react";
import { api } from "../api/client";
import type {
  CostSummary,
  AgentCostSummary,
  ModelCostSummary,
  DailyCost,
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
  if (pct >= 80) return "bg-red-500";
  if (pct >= 50) return "bg-yellow-500";
  return "bg-emerald-500";
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

    const results = await Promise.allSettled([
      api.getCostSummary(),
      api.getCostByAgent(),
      api.getCostByModel(),
      api.getCostDaily(14),
    ]);

    if (results[0].status === "fulfilled") summary = results[0].value;
    if (results[1].status === "fulfilled") byAgent = results[1].value;
    if (results[2].status === "fulfilled") byModel = results[2].value;
    if (results[3].status === "fulfilled") daily = results[3].value;

    return { summary, byAgent, byModel, daily };
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
    </div>
  );
}
