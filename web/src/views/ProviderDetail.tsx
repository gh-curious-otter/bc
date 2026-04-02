import { useCallback, useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { api } from "../api/client";
import type {
  ProviderDetailResponse,
  ProviderCommand,
  ProviderMCPServer,
} from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";
import { StatusBadge } from "../components/StatusBadge";
import { ToastContainer, useToast } from "../components/Toast";

/* ──────────────────────── Helpers ──────────────────────── */

function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(0)}K`;
  return String(n);
}

function formatCost(n: number): string {
  if (n === 0) return "$0.00";
  if (n < 0.01) return `$${n.toFixed(4)}`;
  return `$${n.toFixed(2)}`;
}

const inputCls =
  "w-full px-2.5 py-1.5 text-sm rounded border border-bc-border bg-bc-bg text-bc-text focus:outline-none focus:ring-1 focus:ring-bc-accent";

/* ──────────────────────── Section: Header ──────────────────────── */

function ProviderHeader({
  provider,
  onInstall,
  onUpdate,
  installing,
  updating,
}: {
  provider: ProviderDetailResponse;
  onInstall: () => void;
  onUpdate: () => void;
  installing: boolean;
  updating: boolean;
}) {
  return (
    <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div className="flex items-center gap-3">
        <Link
          to="/tools"
          className="text-bc-muted hover:text-bc-text text-sm shrink-0"
        >
          &larr; Tools
        </Link>
        <h1 className="text-xl font-bold">{provider.name}</h1>
        {provider.version && (
          <span className="px-2 py-0.5 rounded text-xs font-mono bg-bc-surface border border-bc-border text-bc-muted">
            v{provider.version}
          </span>
        )}
        <StatusBadge
          status={
            provider.installed
              ? provider.agent_count > 0
                ? "running"
                : "stopped"
              : "error"
          }
        />
      </div>
      <div className="flex items-center gap-2">
        {!provider.installed && provider.install_hint && (
          <button
            type="button"
            onClick={onInstall}
            disabled={installing}
            className="px-3 py-1.5 text-sm rounded bg-bc-warning/10 text-bc-warning hover:bg-bc-warning/20 transition-colors disabled:opacity-50"
          >
            {installing ? "Installing..." : "Install"}
          </button>
        )}
        {provider.installed && provider.install_hint && (
          <button
            type="button"
            onClick={onUpdate}
            disabled={updating}
            className="px-3 py-1.5 text-sm rounded bg-bc-info/10 text-bc-info hover:bg-bc-info/20 transition-colors disabled:opacity-50"
          >
            {updating ? "Checking..." : "Check for Update"}
          </button>
        )}
        <span
          className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded text-xs font-medium ${
            provider.enabled
              ? "bg-bc-success/10 text-bc-success"
              : "bg-bc-muted/10 text-bc-muted"
          }`}
        >
          <span
            className={`w-2 h-2 rounded-full ${provider.enabled ? "bg-bc-success" : "bg-bc-muted"}`}
          />
          {provider.enabled ? "Enabled" : "Disabled"}
        </span>
      </div>
    </div>
  );
}

/* ──────────────────────── Section: Configuration ──────────────────────── */

function ConfigPanel({
  provider,
  onSave,
}: {
  provider: ProviderDetailResponse;
  onSave: (config: Record<string, string>) => Promise<void>;
}) {
  const [command, setCommand] = useState(provider.config?.command ?? provider.command ?? "");
  const [saving, setSaving] = useState(false);
  const [dirty, setDirty] = useState(false);

  useEffect(() => {
    setCommand(provider.config?.command ?? provider.command ?? "");
    setDirty(false);
  }, [provider.config, provider.command]);

  const handleSave = async () => {
    setSaving(true);
    try {
      await onSave({ command });
      setDirty(false);
    } finally {
      setSaving(false);
    }
  };

  const handleReset = () => {
    setCommand(provider.binary ?? provider.name);
    setDirty(true);
  };

  return (
    <section>
      <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest mb-3">
        Configuration
      </h2>
      <div className="rounded border border-bc-border bg-bc-surface p-4 space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="text-xs text-bc-muted block mb-1">Binary</label>
            <div className="text-sm font-mono text-bc-text/80 px-2.5 py-1.5 rounded bg-bc-bg border border-bc-border/50">
              {provider.binary || "\u2014"}
            </div>
          </div>
          <div>
            <label className="text-xs text-bc-muted block mb-1">Command</label>
            <input
              type="text"
              value={command}
              onChange={(e) => {
                setCommand(e.target.value);
                setDirty(true);
              }}
              className={inputCls}
            />
          </div>
          <div>
            <label className="text-xs text-bc-muted block mb-1">Description</label>
            <div className="text-sm text-bc-text/80 px-2.5 py-1.5 rounded bg-bc-bg border border-bc-border/50">
              {provider.description || "\u2014"}
            </div>
          </div>
          <div>
            <label className="text-xs text-bc-muted block mb-1">Install Hint</label>
            <div className="text-sm font-mono text-bc-text/80 px-2.5 py-1.5 rounded bg-bc-bg border border-bc-border/50 truncate">
              {provider.install_hint || "\u2014"}
            </div>
          </div>
        </div>
        {provider.config?.default === "true" && (
          <div className="text-xs text-bc-accent">
            This is the default provider.
          </div>
        )}
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => void handleSave()}
            disabled={saving || !dirty}
            className="px-3 py-1.5 text-sm rounded bg-bc-accent text-bc-bg font-medium disabled:opacity-50 transition-colors"
          >
            {saving ? "Saving..." : "Save Configuration"}
          </button>
          <button
            type="button"
            onClick={handleReset}
            disabled={saving}
            className="px-3 py-1.5 text-sm rounded border border-bc-border text-bc-muted hover:text-bc-text transition-colors disabled:opacity-50"
          >
            Reset to Default
          </button>
        </div>
      </div>
    </section>
  );
}

/* ──────────────────────── Section: MCP Servers ──────────────────────── */

type MCPHealthStatus = "connected" | "error" | "unknown";

function MCPHealthBadge({ status, error }: { status: MCPHealthStatus; error?: string }) {
  const styles: Record<MCPHealthStatus, { bg: string; text: string; label: string }> = {
    connected: { bg: "bg-bc-success/15", text: "text-bc-success", label: "Connected" },
    error:     { bg: "bg-bc-error/15",   text: "text-bc-error",   label: "Error" },
    unknown:   { bg: "bg-bc-warning/15", text: "text-bc-warning", label: "Unknown" },
  };
  const s = styles[status];
  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs font-medium ${s.bg} ${s.text}`}
      title={error || undefined}
    >
      <span className={`w-1.5 h-1.5 rounded-full ${status === "connected" ? "bg-bc-success" : status === "error" ? "bg-bc-error" : "bg-bc-warning"}`} />
      {s.label}
    </span>
  );
}

function resolveMCPHealth(server: ProviderMCPServer, healthMap: Record<string, { status: string; error?: string }>): { status: MCPHealthStatus; error?: string } {
  // Check health map from unified check first
  const checked = healthMap[server.name];
  if (checked) {
    if (checked.status === "ok" || checked.status === "active" || checked.status === "connected") {
      return { status: "connected" };
    }
    return { status: "error", error: checked.error || checked.status };
  }
  // Fall back to server's own status field — but never trust "connected"
  // without a confirmed health check (only healthMap above provides that).
  if (server.status) {
    const s = server.status.toLowerCase();
    if (s === "error" || s === "failed") return { status: "error", error: server.error };
    // Any other status without a health check is treated as unknown
    return { status: "unknown" };
  }
  // Default: no confirmed health data — always unknown
  return { status: "unknown" };
}

function MCPSection({
  providerName,
  servers,
  onRefresh,
  onToast,
}: {
  providerName: string;
  servers: ProviderMCPServer[];
  onRefresh: () => void;
  onToast: (level: "success" | "error" | "info", msg: string) => void;
}) {
  const [showAdd, setShowAdd] = useState(false);
  const [mcpName, setMcpName] = useState("");
  const [mcpTransport, setMcpTransport] = useState<"stdio" | "sse">("stdio");
  const [mcpValue, setMcpValue] = useState("");
  const [adding, setAdding] = useState(false);
  const [checking, setChecking] = useState(false);
  const [healthMap, setHealthMap] = useState<Record<string, { status: string; error?: string }>>({});

  const handleAdd = async () => {
    if (!mcpName.trim() || !mcpValue.trim()) return;
    setAdding(true);
    try {
      await api.addProviderMCP(providerName, {
        name: mcpName.trim(),
        transport: mcpTransport,
        ...(mcpTransport === "sse" ? { url: mcpValue.trim() } : { command: mcpValue.trim() }),
      });
      onToast("success", `MCP '${mcpName.trim()}' added`);
      setMcpName("");
      setMcpValue("");
      setShowAdd(false);
      onRefresh();
    } catch (err) {
      onToast("error", err instanceof Error ? err.message : "Failed to add MCP");
    } finally {
      setAdding(false);
    }
  };

  const handleCheckAll = async () => {
    setChecking(true);
    try {
      const tools = await api.checkUnifiedTools();
      const mcpTools = tools.filter((t) => t.type === "mcp");
      const newMap: Record<string, { status: string; error?: string }> = {};
      for (const t of mcpTools) {
        newMap[t.name] = { status: t.status, error: t.error };
      }
      setHealthMap(newMap);
      const errors = mcpTools.filter((t) => t.status !== "ok" && t.status !== "active" && t.status !== "connected");
      if (errors.length === 0) {
        onToast("success", `All ${mcpTools.length} MCP server(s) healthy`);
      } else {
        onToast("error", `${errors.length} of ${mcpTools.length} MCP server(s) have issues`);
      }
    } catch (err) {
      onToast("error", err instanceof Error ? err.message : "Health check failed");
    } finally {
      setChecking(false);
    }
  };

  return (
    <section>
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest">
          MCP Servers ({servers.length})
        </h2>
        <div className="flex items-center gap-2">
          {servers.length > 0 && (
            <button
              type="button"
              onClick={() => void handleCheckAll()}
              disabled={checking}
              className="text-xs px-2 py-1 rounded bg-bc-accent/10 text-bc-accent hover:bg-bc-accent/20 transition-colors disabled:opacity-50"
            >
              {checking ? "Checking..." : "Check All MCPs"}
            </button>
          )}
          <button
            type="button"
            onClick={() => setShowAdd(!showAdd)}
            className="text-xs px-2 py-1 rounded bg-bc-info/10 text-bc-info hover:bg-bc-info/20 transition-colors"
          >
            {showAdd ? "Cancel" : "+ Add MCP"}
          </button>
        </div>
      </div>

      {showAdd && (
        <div className="rounded border border-bc-accent bg-bc-surface p-4 space-y-3 mb-4">
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <div>
              <label className="text-xs text-bc-muted block mb-1">Name</label>
              <input
                type="text"
                value={mcpName}
                onChange={(e) => setMcpName(e.target.value)}
                placeholder="my-mcp"
                className={inputCls}
              />
            </div>
            <div>
              <label className="text-xs text-bc-muted block mb-1">Transport</label>
              <select
                value={mcpTransport}
                onChange={(e) => setMcpTransport(e.target.value as "stdio" | "sse")}
                className={inputCls}
              >
                <option value="stdio">stdio</option>
                <option value="sse">SSE</option>
              </select>
            </div>
            <div>
              <label className="text-xs text-bc-muted block mb-1">
                {mcpTransport === "sse" ? "URL" : "Command"}
              </label>
              <input
                type="text"
                value={mcpValue}
                onChange={(e) => setMcpValue(e.target.value)}
                placeholder={mcpTransport === "sse" ? "http://localhost:3000/sse" : "npx my-mcp"}
                className={inputCls}
              />
            </div>
          </div>
          <button
            type="button"
            onClick={() => void handleAdd()}
            disabled={adding || !mcpName.trim() || !mcpValue.trim()}
            className="px-3 py-1.5 text-sm rounded bg-bc-accent text-bc-bg font-medium disabled:opacity-50"
          >
            {adding ? "Adding..." : "Add MCP Server"}
          </button>
        </div>
      )}

      {servers.length === 0 ? (
        <EmptyState
          icon="~"
          title="No MCP servers"
          description={`No MCP servers configured for ${providerName}.`}
        />
      ) : (
        <div className="rounded border border-bc-border overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-bc-border bg-bc-surface text-[11px] text-bc-muted uppercase tracking-wider">
                <th className="px-4 py-2 font-medium text-left">Name</th>
                <th className="px-4 py-2 font-medium text-left">Transport</th>
                <th className="px-4 py-2 font-medium text-left">URL / Command</th>
                <th className="px-4 py-2 font-medium text-left">Status</th>
              </tr>
            </thead>
            <tbody>
              {servers.map((s) => (
                <tr key={s.name} className="border-b border-bc-border/50 hover:bg-bc-surface/50 transition-colors">
                  <td className="px-4 py-2.5 font-medium">{s.name}</td>
                  <td className="px-4 py-2.5">
                    <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-bc-surface border border-bc-border">
                      {s.transport}
                    </span>
                  </td>
                  <td className="px-4 py-2.5 font-mono text-xs text-bc-muted truncate max-w-xs">
                    {s.url || s.command || "\u2014"}
                  </td>
                  <td className="px-4 py-2.5">
                    {(() => {
                      const h = resolveMCPHealth(s, healthMap);
                      return <MCPHealthBadge status={h.status} error={h.error} />;
                    })()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

/* ──────────────────────── Section: Agents ──────────────────────── */

function AgentsSection({
  agents,
}: {
  agents: ProviderDetailResponse["agents"];
}) {
  return (
    <section>
      <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest mb-3">
        Agents Using This Provider ({agents.length})
      </h2>
      {agents.length === 0 ? (
        <EmptyState
          icon="*"
          title="No agents"
          description="No agents are currently using this provider."
        />
      ) : (
        <div className="rounded border border-bc-border overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-bc-border bg-bc-surface text-[11px] text-bc-muted uppercase tracking-wider">
                <th className="px-4 py-2 font-medium text-left">Agent</th>
                <th className="px-4 py-2 font-medium text-left">Role</th>
                <th className="px-4 py-2 font-medium text-left">Status</th>
              </tr>
            </thead>
            <tbody>
              {agents.map((a) => (
                <tr key={a.name} className="border-b border-bc-border/50 hover:bg-bc-surface/50 transition-colors">
                  <td className="px-4 py-2.5">
                    <Link
                      to={`/agents/${encodeURIComponent(a.name)}`}
                      className="font-medium text-bc-accent hover:underline"
                    >
                      {a.name}
                    </Link>
                  </td>
                  <td className="px-4 py-2.5">
                    <span className="px-2 py-0.5 rounded text-xs font-medium bg-bc-accent/20 text-bc-accent">
                      {a.role}
                    </span>
                  </td>
                  <td className="px-4 py-2.5">
                    <StatusBadge status={a.state} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

/* ──────────────────────── Section: Commands ──────────────────────── */

function CommandsSection({ commands }: { commands: ProviderCommand[] }) {
  return (
    <section>
      <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest mb-3">
        Available Commands ({commands.length})
      </h2>
      {commands.length === 0 ? (
        <EmptyState
          icon=">"
          title="No commands"
          description="No commands available for this provider."
        />
      ) : (
        <div className="rounded border border-bc-border overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-bc-border bg-bc-surface text-[11px] text-bc-muted uppercase tracking-wider">
                <th className="px-4 py-2 font-medium text-left">Command</th>
                <th className="px-4 py-2 font-medium text-left">Description</th>
                <th className="px-4 py-2 font-medium text-left">Usage</th>
              </tr>
            </thead>
            <tbody>
              {commands.map((c) => (
                <tr key={c.name} className="border-b border-bc-border/50 hover:bg-bc-surface/50 transition-colors">
                  <td className="px-4 py-2.5 font-medium">{c.name}</td>
                  <td className="px-4 py-2.5 text-bc-muted">{c.description}</td>
                  <td className="px-4 py-2.5 font-mono text-xs text-bc-text/80">
                    {c.command}
                    {c.args && <span className="text-bc-muted ml-1">{c.args}</span>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

/* ──────────────────────── Section: Cost Breakdown ──────────────────────── */

function CostSection({
  provider,
}: {
  provider: ProviderDetailResponse;
}) {
  const models = provider.cost_by_model ?? [];
  return (
    <section>
      <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest mb-3">
        Cost Breakdown
      </h2>
      <div className="rounded border border-bc-border bg-bc-surface p-4 space-y-4">
        {/* Summary row */}
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
          <div>
            <span className="text-xs text-bc-muted block">Total Cost</span>
            <span className="text-lg font-bold">{formatCost(provider.total_cost_usd)}</span>
          </div>
          <div>
            <span className="text-xs text-bc-muted block">Total Tokens</span>
            <span className="text-lg font-bold">{formatTokens(provider.total_tokens)}</span>
          </div>
          <div>
            <span className="text-xs text-bc-muted block">Models Used</span>
            <span className="text-lg font-bold">{models.length}</span>
          </div>
        </div>

        {/* Per-model table */}
        {models.length > 0 && (
          <div className="rounded border border-bc-border overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-bc-border bg-bc-bg text-[11px] text-bc-muted uppercase tracking-wider">
                  <th className="px-4 py-2 font-medium text-left">Model</th>
                  <th className="px-4 py-2 font-medium text-right">Tokens</th>
                  <th className="px-4 py-2 font-medium text-right">Cost</th>
                </tr>
              </thead>
              <tbody>
                {models.map((m) => (
                  <tr key={m.model} className="border-b border-bc-border/50">
                    <td className="px-4 py-2 font-mono text-xs">{m.model}</td>
                    <td className="px-4 py-2 text-right tabular-nums text-bc-muted">
                      {formatTokens(m.total_tokens)}
                    </td>
                    <td className="px-4 py-2 text-right tabular-nums">
                      {formatCost(m.total_cost_usd)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </section>
  );
}

/* ──────────────────────── Main Component ──────────────────────── */

export function ProviderDetail() {
  const { provider: providerName } = useParams<{ provider: string }>();
  const { toasts, addToast, dismiss } = useToast();
  const [installing, setInstalling] = useState(false);
  const [updating, setUpdating] = useState(false);

  // Fetch provider detail
  const detailFetcher = useCallback(async () => {
    if (!providerName) throw new Error("No provider name");
    return api.getProvider(providerName);
  }, [providerName]);
  const {
    data: provider,
    loading,
    error,
    refresh,
  } = usePolling<ProviderDetailResponse>(detailFetcher, 10000);

  // Fetch commands
  const cmdFetcher = useCallback(async () => {
    if (!providerName) throw new Error("No provider name");
    return api.getProviderCommands(providerName);
  }, [providerName]);
  const { data: commands } = usePolling<ProviderCommand[]>(cmdFetcher, 30000);

  // Fetch MCP servers
  const mcpFetcher = useCallback(async () => {
    if (!providerName) throw new Error("No provider name");
    return api.getProviderMCPs(providerName);
  }, [providerName]);
  const { data: mcpServers, refresh: refreshMCPs } = usePolling<ProviderMCPServer[]>(mcpFetcher, 15000);

  const handleInstall = async () => {
    if (!providerName) return;
    setInstalling(true);
    try {
      const result = await api.installProvider(providerName);
      addToast("info", `Install: ${result.install_cmd}`);
      refresh();
    } catch (err) {
      addToast("error", err instanceof Error ? err.message : "Install failed");
    } finally {
      setInstalling(false);
    }
  };

  const handleUpdate = async () => {
    if (!providerName) return;
    setUpdating(true);
    try {
      const result = await api.checkProviderUpdate(providerName);
      if (result.update_available) {
        addToast("info", `Update available: ${result.latest_version} (current: ${result.current_version})`);
      } else {
        addToast("success", `Already on latest version: ${result.current_version}`);
      }
    } catch (err) {
      addToast("error", err instanceof Error ? err.message : "Update check failed");
    } finally {
      setUpdating(false);
    }
  };

  const handleSaveConfig = async (config: Record<string, string>) => {
    if (!providerName) return;
    try {
      await api.updateProviderConfig(providerName, config);
      addToast("success", "Configuration saved");
      refresh();
    } catch (err) {
      addToast("error", err instanceof Error ? err.message : "Failed to save config");
      throw err;
    }
  };

  if (loading && !provider) {
    return (
      <div className="p-6 space-y-6">
        <div className="h-6 w-48 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="cards" rows={3} />
      </div>
    );
  }

  if (error && !provider) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load provider"
          description={error}
          actionLabel="Back to Tools"
          onAction={() => window.history.back()}
        />
      </div>
    );
  }

  if (!provider) return null;

  return (
    <div className="p-6 space-y-8">
      <ProviderHeader
        provider={provider}
        onInstall={() => void handleInstall()}
        onUpdate={() => void handleUpdate()}
        installing={installing}
        updating={updating}
      />

      <ConfigPanel provider={provider} onSave={handleSaveConfig} />

      <MCPSection
        providerName={provider.name}
        servers={mcpServers ?? []}
        onRefresh={refreshMCPs}
        onToast={addToast}
      />

      <AgentsSection agents={provider.agents ?? []} />

      <CommandsSection commands={commands ?? []} />

      <CostSection provider={provider} />

      <ToastContainer toasts={toasts} onDismiss={dismiss} />
    </div>
  );
}
