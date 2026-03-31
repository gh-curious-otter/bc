import { useCallback, useState } from "react";
import { api } from "../api/client";
import type { SettingsConfig, StatsSummary } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

interface WorkspaceData {
  status: Record<string, unknown>;
  settings: SettingsConfig | null;
  stats: StatsSummary | null;
}

const fmtUptime = (s: number) => {
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  return [d && `${d}d`, h && `${h}h`, `${m}m`].filter(Boolean).join(" ");
};

function InfoCard({
  label,
  value,
  sub,
}: {
  label: string;
  value: string;
  sub?: string;
}) {
  return (
    <div className="rounded border border-bc-border bg-bc-surface p-4">
      <p className="text-xs text-bc-muted uppercase tracking-wide">{label}</p>
      <p className="mt-1 text-lg font-bold truncate">{value}</p>
      {sub && (
        <p className="mt-0.5 text-xs text-bc-muted truncate" title={sub}>
          {sub}
        </p>
      )}
    </div>
  );
}

function Section({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section>
      <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest mb-3">
        {title}
      </h2>
      {children}
    </section>
  );
}

function ConfigRow({
  label,
  value,
}: {
  label: string;
  value: string | number | boolean | undefined;
}) {
  if (value === undefined) return null;
  return (
    <div className="flex items-center justify-between py-1.5 border-b border-bc-border/30 last:border-b-0">
      <span className="text-xs text-bc-muted">{label}</span>
      <span className="text-xs font-mono text-bc-text truncate ml-4 max-w-[60%] text-right">
        {String(value)}
      </span>
    </div>
  );
}

function GatewayBadge({
  name,
  enabled,
}: {
  name: string;
  enabled: boolean;
}) {
  return (
    <div
      className={`flex items-center gap-2 px-3 py-2 rounded border ${
        enabled
          ? "border-bc-success/30 bg-bc-success/5"
          : "border-bc-border bg-bc-surface"
      }`}
    >
      <span
        className={`w-2 h-2 rounded-full ${enabled ? "bg-bc-success" : "bg-bc-muted"}`}
      />
      <span className="text-sm font-medium capitalize">{name}</span>
      <span
        className={`text-[10px] ml-auto ${enabled ? "text-bc-success" : "text-bc-muted"}`}
      >
        {enabled ? "Connected" : "Disabled"}
      </span>
    </div>
  );
}

export function Workspace() {
  const fetcher = useCallback(async (): Promise<WorkspaceData> => {
    const [r0, r1, r2] = await Promise.allSettled([
      api.getWorkspaceStatus(),
      api.getSettings(),
      api.getStatsSummary(),
    ]);
    return {
      status: r0.status === "fulfilled" ? r0.value : {},
      settings: r1.status === "fulfilled" ? r1.value : null,
      stats: r2.status === "fulfilled" ? r2.value : null,
    };
  }, []);

  const {
    data,
    loading,
    error,
    refresh,
    timedOut,
  } = usePolling(fetcher, 10000);

  const [actionBusy, setActionBusy] = useState<"up" | "down" | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const handleUp = async () => {
    setActionBusy("up");
    setActionError(null);
    try {
      await api.workspaceUp();
      refresh();
    } catch (err) {
      setActionError(
        err instanceof Error ? err.message : "Failed to start workspace",
      );
    } finally {
      setActionBusy(null);
    }
  };

  const handleDown = async () => {
    setActionBusy("down");
    setActionError(null);
    try {
      await api.workspaceDown();
      refresh();
    } catch (err) {
      setActionError(
        err instanceof Error ? err.message : "Failed to stop workspace",
      );
    } finally {
      setActionBusy(null);
    }
  };

  if (loading && !data) {
    return (
      <div className="p-6 space-y-6">
        <div className="h-6 w-32 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="cards" rows={4} />
      </div>
    );
  }
  if (timedOut && !data) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Workspace took too long to load"
          description="The server may be unavailable."
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
          title="Failed to load workspace"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (!data) return null;

  const { status, settings, stats } = data;
  const wsName = String(status.name ?? "unknown");
  const nickname = String(status.nickname ?? "");
  const isHealthy = status.is_healthy === true;

  return (
    <div className="p-6 space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold">{wsName}</h1>
          {nickname && (
            <p className="text-sm text-bc-muted mt-0.5">{nickname}</p>
          )}
        </div>
        <div className="flex items-center gap-3">
          <span
            className={`flex items-center gap-1.5 text-xs ${isHealthy ? "text-bc-success" : "text-bc-error"}`}
          >
            <span
              className={`w-2 h-2 rounded-full ${isHealthy ? "bg-bc-success" : "bg-bc-error"}`}
            />
            {isHealthy ? "Healthy" : "Unhealthy"}
          </span>
          <button
            type="button"
            onClick={handleUp}
            disabled={actionBusy !== null}
            className="px-3 py-1.5 text-sm rounded bg-bc-success/20 text-bc-success hover:bg-bc-success/30 disabled:opacity-50 transition-colors focus-visible:ring-2 focus-visible:ring-bc-success"
            aria-label="Start workspace"
          >
            {actionBusy === "up" ? "Starting..." : "Start"}
          </button>
          <button
            type="button"
            onClick={handleDown}
            disabled={actionBusy !== null}
            className="px-3 py-1.5 text-sm rounded bg-bc-error/20 text-bc-error hover:bg-bc-error/30 disabled:opacity-50 transition-colors focus-visible:ring-2 focus-visible:ring-bc-error"
            aria-label="Stop all agents"
          >
            {actionBusy === "down" ? "Stopping..." : "Stop All"}
          </button>
        </div>
      </div>

      {actionError && <p className="text-xs text-bc-error">{actionError}</p>}

      {/* Overview stats */}
      {stats && (
        <Section title="Overview">
          <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-3">
            <InfoCard
              label="Agents"
              value={String(stats.agents_total)}
              sub={`${stats.agents_running} running`}
            />
            <InfoCard
              label="Channels"
              value={String(stats.channels_total)}
              sub={`${stats.messages_total.toLocaleString()} messages`}
            />
            <InfoCard
              label="Cost"
              value={`$${stats.total_cost_usd.toFixed(2)}`}
            />
            <InfoCard
              label="Roles"
              value={String(stats.roles_total)}
            />
            <InfoCard
              label="Tools"
              value={String(stats.tools_total)}
            />
            <InfoCard
              label="Uptime"
              value={fmtUptime(stats.uptime_seconds)}
            />
          </div>
        </Section>
      )}

      {/* Configuration */}
      {settings && (
        <>
          <Section title="Server">
            <div className="rounded border border-bc-border bg-bc-surface p-4">
              <ConfigRow label="Host" value={settings.server.host} />
              <ConfigRow label="Port" value={settings.server.port} />
              <ConfigRow label="Config Version" value={settings.version} />
              <ConfigRow label="CORS Origin" value={settings.server.cors_origin} />
            </div>
          </Section>

          <Section title="Runtime">
            <div className="rounded border border-bc-border bg-bc-surface p-4">
              <ConfigRow label="Default Runtime" value={settings.runtime.default} />
              <ConfigRow label="Docker Image" value={settings.runtime.docker.image} />
              <ConfigRow label="Docker Network" value={settings.runtime.docker.network} />
              <ConfigRow label="CPU Limit" value={`${settings.runtime.docker.cpus} cores`} />
              <ConfigRow label="Memory Limit" value={`${settings.runtime.docker.memory_mb} MB`} />
              <ConfigRow label="Default Provider" value={settings.providers.default} />
            </div>
          </Section>

          <Section title="Storage">
            <div className="rounded border border-bc-border bg-bc-surface p-4">
              <ConfigRow label="Backend" value={settings.storage.default} />
              {settings.storage.default === "sqlite" && (
                <ConfigRow label="SQLite Path" value={settings.storage.sqlite.path} />
              )}
              {settings.storage.default === "sql" && (
                <>
                  <ConfigRow label="Host" value={settings.storage.sql.host} />
                  <ConfigRow label="Port" value={settings.storage.sql.port} />
                  <ConfigRow label="Database" value={settings.storage.sql.database} />
                </>
              )}
              <ConfigRow label="Log Path" value={settings.logs.path} />
            </div>
          </Section>

          <Section title="Gateways">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
              <GatewayBadge
                name="Slack"
                enabled={settings.gateways.slack?.enabled ?? false}
              />
              <GatewayBadge
                name="Telegram"
                enabled={settings.gateways.telegram?.enabled ?? false}
              />
              <GatewayBadge
                name="Discord"
                enabled={settings.gateways.discord?.enabled ?? false}
              />
            </div>
          </Section>
        </>
      )}
    </div>
  );
}
