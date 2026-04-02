import { useCallback, useMemo, useState } from "react";
import { api } from "../api/client";
import type { UnifiedTool } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";
import { ToastContainer, useToast } from "../components/Toast";
import type { ToastLevel } from "../components/Toast";

type AddFormType = "mcp" | "cli" | null;

const STATUS_CONFIG: Record<string, { dot: string; label: string; textColor: string }> = {
  connected:     { dot: "bg-bc-success", label: "Connected",     textColor: "text-bc-success" },
  configured:    { dot: "bg-bc-success", label: "Configured",    textColor: "text-bc-success" },
  installed:     { dot: "bg-bc-success", label: "Installed",     textColor: "text-bc-success" },
  disabled:      { dot: "bg-bc-muted",   label: "Disabled",      textColor: "text-bc-muted" },
  not_installed: { dot: "bg-bc-error",   label: "Not installed", textColor: "text-bc-error" },
  error:         { dot: "bg-bc-error",   label: "Error",         textColor: "text-bc-error" },
  unknown:       { dot: "bg-bc-muted",   label: "Unknown",       textColor: "text-bc-muted" },
};

const inputCls = "w-full px-2 py-1.5 text-sm rounded border border-bc-border bg-bc-bg text-bc-text focus:outline-none focus:ring-1 focus:ring-bc-accent";

function getStatusConfig(s: string) { return STATUS_CONFIG[s] ?? STATUS_CONFIG.unknown!; }

function copyToClipboard(cmd: string, setCopied: (v: string | null) => void, label: string) {
  navigator.clipboard.writeText(cmd).then(() => {
    setCopied(label);
    setTimeout(() => setCopied(null), 2000);
  }).catch(() => {});
}

function ProviderCard({ tool }: { tool: UnifiedTool }) {
  const [copied, setCopied] = useState<string | null>(null);
  const cfg = getStatusConfig(tool.status);
  return (
    <div className="rounded border border-bc-border bg-bc-surface p-3 flex items-center gap-3">
      <span className={`w-2 h-2 rounded-full shrink-0 ${cfg.dot}`} aria-label={`Status: ${cfg.label}`} />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">{tool.name}</span>
          <span className={`text-[10px] ${cfg.textColor}`}>{cfg.label}</span>
        </div>
        <p className="text-[10px] text-bc-muted font-mono truncate" title={tool.command || ""}>{tool.command || "\u2014"}</p>
        {tool.version && <span className="text-[10px] text-bc-muted">v{tool.version}</span>}
        {tool.error && <p className="text-[10px] text-bc-error">{tool.error}</p>}
      </div>
      <div className="flex items-center gap-1.5 shrink-0">
        {copied && <span className="text-[10px] text-bc-success animate-pulse">{copied} copied</span>}
        {tool.status === "not_installed" && tool.install_cmd && (
          <button type="button" onClick={() => copyToClipboard(tool.install_cmd!, setCopied, "Install cmd")}
            className="text-[10px] px-2 py-0.5 rounded bg-bc-accent/10 text-bc-accent hover:bg-bc-accent/20 transition-colors">
            Install
          </button>
        )}
        {tool.status === "installed" && tool.upgrade_cmd && (
          <button type="button" onClick={() => copyToClipboard(tool.upgrade_cmd!, setCopied, "Update cmd")}
            className="text-[10px] px-2 py-0.5 rounded bg-bc-info/10 text-bc-info hover:bg-bc-info/20 transition-colors">
            Update
          </button>
        )}
      </div>
    </div>
  );
}

function ToolCard({ tool, onToggle, onRemove, toggling, removing }: { tool: UnifiedTool; onToggle: () => void; onRemove: () => void; toggling: boolean; removing: boolean }) {
  const [confirmRemove, setConfirmRemove] = useState(false);
  const [copied, setCopied] = useState<string | null>(null);
  const cfg = getStatusConfig(tool.status);
  const isMCP = tool.type === "mcp";
  const isDisabled = tool.status === "disabled";
  const displayText = isMCP ? (tool.transport === "sse" ? tool.url : tool.command) || tool.transport : tool.command || "\u2014";
  return (
    <div className={`rounded border bg-bc-surface p-4 flex items-start gap-3 ${tool.error ? "border-bc-error/30" : "border-bc-border"}`}>
      <span className={`inline-block w-2 h-2 mt-1.5 rounded-full shrink-0 ${cfg.dot}`} aria-label={`Status: ${cfg.label}`} />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="font-medium text-sm">{tool.name}</span>
          {isMCP && tool.transport && (
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-bc-accent/10 text-bc-accent font-medium uppercase">{tool.transport}</span>
          )}
          {tool.required && <span className="text-[10px] px-1.5 py-0.5 rounded bg-bc-border text-bc-muted">required</span>}
          <span className={`text-xs ${cfg.textColor}`}>{cfg.label}</span>
        </div>
        <div className="mt-1 text-xs text-bc-muted font-mono truncate" title={tool.url || tool.command || ""}>
          {displayText}
        </div>
        {tool.version && <span className="text-[10px] text-bc-muted">v{tool.version}</span>}
        {tool.status === "not_installed" && tool.install_cmd && (
          <p className="mt-1 text-[10px] text-bc-muted font-mono truncate" title={tool.install_cmd}>install: <span className="text-bc-text">{tool.install_cmd}</span></p>
        )}
        {tool.error && <p className="mt-1 text-[10px] text-bc-error">{tool.error}</p>}
      </div>
      <div className="flex items-center gap-2 shrink-0">
        {copied && <span className="text-[10px] text-bc-success animate-pulse">{copied} copied</span>}
        {tool.status === "not_installed" && tool.install_cmd && (
          <button type="button" onClick={() => copyToClipboard(tool.install_cmd!, setCopied, "Install cmd")}
            className="text-xs px-2 py-1 rounded bg-bc-warning/10 text-bc-warning hover:bg-bc-warning/20 transition-colors"
            aria-label={`Install ${tool.name}`}>Install</button>
        )}
        {tool.status === "installed" && tool.upgrade_cmd && (
          <button type="button" onClick={() => copyToClipboard(tool.upgrade_cmd!, setCopied, "Update cmd")}
            className="text-xs px-2 py-1 rounded bg-bc-info/10 text-bc-info hover:bg-bc-info/20 transition-colors"
            aria-label={`Update ${tool.name}`}>Update</button>
        )}
        <button type="button" onClick={onToggle} disabled={toggling}
          role="switch" aria-checked={!isDisabled}
          aria-label={isDisabled ? `Enable ${tool.name}` : `Disable ${tool.name}`}
          className={`text-xs px-2 py-1 rounded transition-colors focus-visible:ring-2 focus-visible:ring-bc-accent disabled:opacity-50 ${isDisabled ? "bg-bc-border text-bc-muted hover:bg-bc-border/80" : "bg-bc-success/10 text-bc-success hover:bg-bc-success/20"}`}>
          {toggling ? "..." : isDisabled ? "Enable" : "Enabled"}
        </button>
        {confirmRemove ? (
          <div className="flex items-center gap-1">
            <span className="text-xs text-bc-error whitespace-nowrap">Remove &lsquo;{tool.name}&rsquo;?</span>
            <button type="button" onClick={() => { onRemove(); setConfirmRemove(false); }} disabled={removing}
              className="text-xs px-2 py-1 rounded bg-bc-error/20 text-bc-error focus-visible:ring-2 focus-visible:ring-bc-error disabled:opacity-50" aria-label="Confirm remove">
              {removing ? "..." : "Yes"}
            </button>
            <button type="button" onClick={() => setConfirmRemove(false)} disabled={removing}
              className="text-xs px-2 py-1 rounded border border-bc-border text-bc-muted focus-visible:ring-2 focus-visible:ring-bc-accent" aria-label="Cancel remove">No</button>
          </div>
        ) : (
          <button type="button" onClick={() => setConfirmRemove(true)}
            className="text-xs px-2 py-1 rounded border border-bc-border text-bc-muted hover:text-bc-error hover:border-bc-error/50 transition-colors focus-visible:ring-2 focus-visible:ring-bc-accent"
            aria-label={`Remove ${tool.name}`}>Remove</button>
        )}
      </div>
    </div>
  );
}

function AddToolForm({ type, onClose, onAdded, onToast }: { type: "mcp" | "cli"; onClose: () => void; onAdded: () => void; onToast: (level: ToastLevel, text: string) => void }) {
  const [name, setName] = useState("");
  const [command, setCommand] = useState("");
  const [url, setUrl] = useState("");
  const [transport, setTransport] = useState<"stdio" | "sse">("sse");
  const [installCmd, setInstallCmd] = useState("");
  const [envText, setEnvText] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    if (!name.trim()) return;
    setSubmitting(true);
    setError(null);
    try {
      if (type === "mcp") {
        const env: Record<string, string> = {};
        for (const line of envText.split("\n")) {
          const eq = line.indexOf("=");
          if (eq > 0) env[line.slice(0, eq).trim()] = line.slice(eq + 1).trim();
        }
        await api.registerMCP({
          name: name.trim(), transport,
          command: transport === "stdio" ? command.trim() : "",
          url: transport === "sse" ? url.trim() : "",
          env: Object.keys(env).length > 0 ? env : undefined, enabled: true,
        });
        onToast("success", `MCP server '${name.trim()}' added`);
      } else {
        await api.upsertTool({ name: name.trim(), command: command.trim(), install_cmd: installCmd.trim(), enabled: true });
        onToast("info", `Tool '${name.trim()}' added \u2014 configuration updated`);
      }
      onAdded();
      onClose();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to add tool";
      setError(msg);
      onToast("error", msg);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="rounded border border-bc-accent bg-bc-surface p-4 space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">Add {type === "mcp" ? "MCP Server" : "CLI Tool"}</h3>
        <button type="button" onClick={onClose} className="text-xs text-bc-muted hover:text-bc-text">Cancel</button>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div>
          <label className="text-xs text-bc-muted block mb-1">Name</label>
          <input type="text" value={name} onChange={(e) => setName(e.target.value)}
            placeholder={type === "mcp" ? "playwright" : "gh"} className={inputCls} />
        </div>
        {type === "mcp" ? (
          <>
            <div>
              <label className="text-xs text-bc-muted block mb-1">Transport</label>
              <select value={transport} onChange={(e) => setTransport(e.target.value as "stdio" | "sse")} className={inputCls}>
                <option value="sse">SSE</option>
                <option value="stdio">stdio</option>
              </select>
            </div>
            <div className="md:col-span-2">
              <label className="text-xs text-bc-muted block mb-1">{transport === "sse" ? "URL" : "Command"}</label>
              <input type="text" className={inputCls}
                value={transport === "sse" ? url : command}
                onChange={(e) => transport === "sse" ? setUrl(e.target.value) : setCommand(e.target.value)}
                placeholder={transport === "sse" ? "http://localhost:3000/sse" : "npx @playwright/mcp"} />
            </div>
            <div className="md:col-span-2">
              <label className="text-xs text-bc-muted block mb-1">Environment Variables (one per line, KEY=VALUE)</label>
              <textarea value={envText} onChange={(e) => setEnvText(e.target.value)}
                placeholder={"API_KEY=${secret:MY_KEY}\nDEBUG=true"} rows={2}
                className={`${inputCls} resize-none font-mono`} />
            </div>
          </>
        ) : (
          <>
            <div>
              <label className="text-xs text-bc-muted block mb-1">Command</label>
              <input type="text" value={command} onChange={(e) => setCommand(e.target.value)} placeholder="gh" className={inputCls} />
            </div>
            <div className="md:col-span-2">
              <label className="text-xs text-bc-muted block mb-1">Install Command (optional)</label>
              <input type="text" value={installCmd} onChange={(e) => setInstallCmd(e.target.value)}
                placeholder="apt-get install -y gh" className={inputCls} />
            </div>
          </>
        )}
      </div>
      {error && <p className="text-xs text-bc-error">{error}</p>}
      <button type="button" onClick={() => void handleSubmit()} disabled={submitting || !name.trim()}
        className="px-3 py-1.5 text-sm rounded bg-bc-accent text-bc-bg font-medium disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-bc-accent">
        {submitting ? "Adding..." : `Add ${type === "mcp" ? "MCP Server" : "CLI Tool"}`}
      </button>
    </div>
  );
}

export function UnifiedTools() {
  const fetcher = useCallback(() => api.listUnifiedTools(), []);
  const { data: tools, loading, error, refresh, timedOut } = usePolling(fetcher, 10000);
  const [addForm, setAddForm] = useState<AddFormType>(null);
  const [checking, setChecking] = useState(false);
  const [checkedTools, setCheckedTools] = useState<UnifiedTool[] | null>(null);
  const [optimisticToggles, setOptimisticToggles] = useState<Map<string, string>>(new Map());
  const [togglingSet, setTogglingSet] = useState<Set<string>>(new Set());
  const [removingSet, setRemovingSet] = useState<Set<string>>(new Set());
  const [search, setSearch] = useState("");
  const { toasts, addToast, dismiss } = useToast();

  const handleCheck = async () => {
    setChecking(true);
    try {
      const checked = await api.checkUnifiedTools();
      // Merge health status into existing tools instead of replacing the list.
      // This prevents losing tools when health check can't reach URLs (e.g. host.docker.internal from browser).
      const statusMap = new Map(checked.map((t) => [t.name, t.status]));
      setCheckedTools((tools ?? []).map((t) => ({
        ...t,
        status: statusMap.get(t.name) ?? t.status,
        health_status: statusMap.has(t.name) ? "checked" : undefined,
      })));
      addToast("success", "Health check complete");
    } catch {
      addToast("error", "Health check failed");
    }
    finally { setChecking(false); }
  };

  // Hooks MUST be called before any early returns (React rules of hooks).
  const allTools = useMemo(() => {
    const source = checkedTools ?? tools ?? [];
    const seen = new Set<string>();
    const deduped: UnifiedTool[] = [];
    for (const t of source) {
      if (seen.has(t.name)) continue;
      seen.add(t.name);
      const optimistic = optimisticToggles.get(t.name);
      deduped.push(optimistic ? { ...t, status: optimistic } : t);
    }
    return deduped;
  }, [checkedTools, tools, optimisticToggles]);

  const searchLower = search.toLowerCase().trim();

  const { providers, mcpTools, cliTools, filteredProviders, filteredMcp, filteredCli } = useMemo(() => {
    const matchesSearch = (t: UnifiedTool) => !searchLower || t.name.toLowerCase().includes(searchLower);
    const prov = allTools.filter((t) => t.type === "provider");
    const mcp = allTools.filter((t) => t.type === "mcp");
    const cli = allTools.filter((t) => !["provider", "mcp"].includes(t.type));
    return {
      providers: prov, mcpTools: mcp, cliTools: cli,
      filteredProviders: prov.filter(matchesSearch),
      filteredMcp: mcp.filter(matchesSearch),
      filteredCli: cli.filter(matchesSearch),
    };
  }, [allTools, searchLower]);

  // Early returns for loading/error states (after all hooks).
  if (loading && !tools) {
    return (
      <div className="p-6 space-y-6">
        <div className="h-6 w-32 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="cards" rows={4} />
      </div>
    );
  }
  if (timedOut && !tools) {
    return <div className="p-6"><EmptyState icon="!" title="Tools timed out" actionLabel="Retry" onAction={refresh} /></div>;
  }
  if (error && !tools) {
    return <div className="p-6"><EmptyState icon="!" title="Failed to load tools" description={error} actionLabel="Retry" onAction={refresh} /></div>;
  }

  const totalCount = allTools.length;
  const matchCount = filteredProviders.length + filteredMcp.length + filteredCli.length;

  const handleToggle = async (tool: UnifiedTool) => {
    const wasDisabled = tool.status === "disabled" || tool.status === "not_installed";
    const newStatus = wasDisabled ? (tool.type === "mcp" ? "configured" : "installed") : "disabled";
    const oldStatus = tool.status;

    // Optimistic update
    setOptimisticToggles((prev) => new Map(prev).set(tool.name, newStatus));
    setTogglingSet((prev) => new Set(prev).add(tool.name));

    try {
      if (tool.type === "mcp") {
        wasDisabled ? await api.enableMCP(tool.name) : await api.disableMCP(tool.name);
      } else {
        wasDisabled ? await api.enableTool(tool.name) : await api.disableTool(tool.name);
      }
      addToast("success", `${tool.name} ${wasDisabled ? "enabled" : "disabled"}`);
      setCheckedTools(null);
      refresh();
    } catch (err) {
      // Revert optimistic update
      setOptimisticToggles((prev) => new Map(prev).set(tool.name, oldStatus));
      const msg = err instanceof Error ? err.message : `Failed to toggle ${tool.name}`;
      addToast("error", msg);
    } finally {
      setTogglingSet((prev) => { const next = new Set(prev); next.delete(tool.name); return next; });
      // Clear optimistic state after server data arrives
      setTimeout(() => {
        setOptimisticToggles((prev) => {
          const next = new Map(prev);
          next.delete(tool.name);
          return next;
        });
      }, 1500);
    }
  };

  const handleRemove = async (tool: UnifiedTool) => {
    setRemovingSet((prev) => new Set(prev).add(tool.name));
    try {
      tool.type === "mcp" ? await api.removeMCP(tool.name) : await api.deleteTool(tool.name);
      addToast("success", `${tool.name} removed`);
      setCheckedTools(null);
      refresh();
    } catch (err) {
      const msg = err instanceof Error ? err.message : `Failed to remove ${tool.name}`;
      addToast("error", msg);
    } finally {
      setRemovingSet((prev) => { const next = new Set(prev); next.delete(tool.name); return next; });
    }
  };

  return (
    <div className="p-6 space-y-8">
      <div className="flex items-center justify-between">
        <div className="shrink-0 pl-2 sm:pl-0">
          <h1 className="text-xl font-bold">Tools</h1>
          <p className="text-xs text-bc-muted mt-0.5 hidden sm:block">
            {searchLower
              ? `${matchCount} of ${totalCount} tools`
              : <>{providers.length} Providers &middot; {mcpTools.length} MCP &middot; {cliTools.length} CLI{checkedTools && " \u00b7 checked"}</>
            }
          </p>
        </div>
        <div className="flex items-center gap-2">
          <div className="relative">
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search tools..."
              className="w-40 sm:w-52 px-2 py-1.5 pr-7 text-sm rounded border border-bc-border bg-bc-bg text-bc-text placeholder:text-bc-muted focus:outline-none focus:ring-1 focus:ring-bc-accent"
            />
            {search && (
              <button
                type="button"
                onClick={() => setSearch("")}
                className="absolute right-1.5 top-1/2 -translate-y-1/2 text-bc-muted hover:text-bc-text text-sm leading-none px-1"
                aria-label="Clear search"
              >
                &times;
              </button>
            )}
          </div>
          <button type="button" onClick={() => void handleCheck()} disabled={checking}
            className="px-3 py-1.5 text-sm rounded border border-bc-border text-bc-muted hover:text-bc-text transition-colors disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-bc-accent">
            {checking ? "Checking..." : "Health Check"}
          </button>
          <button type="button" onClick={() => setAddForm(addForm === "mcp" ? null : "mcp")}
            className="px-3 py-1.5 text-sm rounded bg-bc-accent/10 text-bc-accent hover:bg-bc-accent/20 transition-colors focus-visible:ring-2 focus-visible:ring-bc-accent">
            + MCP Server
          </button>
          <button type="button" onClick={() => setAddForm(addForm === "cli" ? null : "cli")}
            className="px-3 py-1.5 text-sm rounded bg-bc-info/10 text-bc-info hover:bg-bc-info/20 transition-colors focus-visible:ring-2 focus-visible:ring-bc-accent">
            + CLI Tool
          </button>
        </div>
      </div>

      {addForm && <AddToolForm type={addForm} onClose={() => setAddForm(null)} onAdded={() => { setCheckedTools(null); refresh(); }} onToast={addToast} />}

      {/* Providers */}
      <section>
        <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest mb-3">
          Providers ({filteredProviders.length}{searchLower ? `/${providers.length}` : ""}) &mdash; AI model providers
        </h2>
        {filteredProviders.length === 0 ? (
          <EmptyState icon="*" title={searchLower ? "No matching providers" : "No providers"} description={searchLower ? "Try a different search term." : "No AI providers configured."} />
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2">
            {filteredProviders.map((t) => <ProviderCard key={t.name} tool={t} />)}
          </div>
        )}
      </section>

      {/* MCP Servers */}
      <section>
        <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest mb-3">
          MCP Servers ({filteredMcp.length}{searchLower ? `/${mcpTools.length}` : ""}) &mdash; External tool connections
        </h2>
        {filteredMcp.length === 0 ? (
          <EmptyState icon="~" title={searchLower ? "No matching MCP servers" : "No MCP servers"} description={searchLower ? "Try a different search term." : "Add an MCP server to connect external tools."} />
        ) : (
          <div className="space-y-2">
            {filteredMcp.map((t) => <ToolCard key={t.name} tool={t} onToggle={() => void handleToggle(t)} onRemove={() => void handleRemove(t)} toggling={togglingSet.has(t.name)} removing={removingSet.has(t.name)} />)}
          </div>
        )}
      </section>

      {/* CLI Tools */}
      <section>
        <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest mb-3">
          CLI Tools ({filteredCli.length}{searchLower ? `/${cliTools.length}` : ""}) &mdash; Command-line utilities
        </h2>
        {filteredCli.length === 0 ? (
          <EmptyState icon=">" title={searchLower ? "No matching CLI tools" : "No CLI tools"} description={searchLower ? "Try a different search term." : "Add CLI tools like gh, aws, or wrangler."} />
        ) : (
          <div className="space-y-2">
            {filteredCli.map((t) => <ToolCard key={t.name} tool={t} onToggle={() => void handleToggle(t)} onRemove={() => void handleRemove(t)} toggling={togglingSet.has(t.name)} removing={removingSet.has(t.name)} />)}
          </div>
        )}
      </section>

      <ToastContainer toasts={toasts} onDismiss={dismiss} />
    </div>
  );
}
