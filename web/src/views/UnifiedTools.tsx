import { useCallback, useState } from "react";
import { api } from "../api/client";
import type { UnifiedTool } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

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
      <span className={`w-2 h-2 rounded-full shrink-0 ${cfg.dot}`} />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">{tool.name}</span>
          <span className={`text-[10px] ${cfg.textColor}`}>{cfg.label}</span>
        </div>
        <p className="text-[10px] text-bc-muted font-mono truncate">{tool.command || "\u2014"}</p>
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

function ToolCard({ tool, onToggle, onRemove }: { tool: UnifiedTool; onToggle: () => void; onRemove: () => void }) {
  const [confirmRemove, setConfirmRemove] = useState(false);
  const [copied, setCopied] = useState<string | null>(null);
  const cfg = getStatusConfig(tool.status);
  const isMCP = tool.type === "mcp";
  const isDisabled = tool.status === "disabled";
  return (
    <div className={`rounded border bg-bc-surface p-4 flex items-start gap-3 ${tool.error ? "border-bc-error/30" : "border-bc-border"}`}>
      <span className={`inline-block w-2 h-2 mt-1.5 rounded-full shrink-0 ${cfg.dot}`} />
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
          {isMCP ? (tool.transport === "sse" ? tool.url : tool.command) || tool.transport : tool.command || "\u2014"}
        </div>
        {tool.version && <span className="text-[10px] text-bc-muted">v{tool.version}</span>}
        {tool.status === "not_installed" && tool.install_cmd && (
          <p className="mt-1 text-[10px] text-bc-muted font-mono">install: <span className="text-bc-text">{tool.install_cmd}</span></p>
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
        <button type="button" onClick={onToggle} aria-label={isDisabled ? `Enable ${tool.name}` : `Disable ${tool.name}`}
          className={`text-xs px-2 py-1 rounded transition-colors ${isDisabled ? "bg-bc-border text-bc-muted hover:bg-bc-border/80" : "bg-bc-success/10 text-bc-success hover:bg-bc-success/20"}`}>
          {isDisabled ? "Enable" : "Enabled"}
        </button>
        {confirmRemove ? (
          <div className="flex items-center gap-1">
            <button type="button" onClick={() => { onRemove(); setConfirmRemove(false); }}
              className="text-xs px-2 py-1 rounded bg-bc-error/20 text-bc-error" aria-label="Confirm remove">Yes</button>
            <button type="button" onClick={() => setConfirmRemove(false)}
              className="text-xs px-2 py-1 rounded border border-bc-border text-bc-muted" aria-label="Cancel remove">No</button>
          </div>
        ) : (
          <button type="button" onClick={() => setConfirmRemove(true)}
            className="text-xs px-2 py-1 rounded border border-bc-border text-bc-muted hover:text-bc-error hover:border-bc-error/50 transition-colors"
            aria-label={`Remove ${tool.name}`}>Remove</button>
        )}
      </div>
    </div>
  );
}

function AddToolForm({ type, onClose, onAdded }: { type: "mcp" | "cli"; onClose: () => void; onAdded: () => void }) {
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
      } else {
        await api.upsertTool({ name: name.trim(), command: command.trim(), install_cmd: installCmd.trim(), enabled: true });
      }
      onAdded();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to add tool");
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

  const handleCheck = async () => {
    setChecking(true);
    try { setCheckedTools(await api.checkUnifiedTools()); }
    catch { /* fall back to polled data */ }
    finally { setChecking(false); }
  };

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

  const allTools = checkedTools ?? tools ?? [];
  const providers = allTools.filter((t) => t.type === "provider");
  const mcpTools = allTools.filter((t) => t.type === "mcp");
  const cliTools = allTools.filter((t) => !["provider", "mcp"].includes(t.type));

  const handleToggle = async (tool: UnifiedTool) => {
    try {
      if (tool.type === "mcp") {
        tool.status === "disabled" ? await api.enableMCP(tool.name) : await api.disableMCP(tool.name);
      } else {
        tool.status === "disabled" || tool.status === "not_installed"
          ? await api.enableTool(tool.name) : await api.disableTool(tool.name);
      }
      setCheckedTools(null);
      refresh();
    } catch { /* silently fail */ }
  };

  const handleRemove = async (tool: UnifiedTool) => {
    try {
      tool.type === "mcp" ? await api.removeMCP(tool.name) : await api.deleteTool(tool.name);
      setCheckedTools(null);
      refresh();
    } catch { /* silently fail */ }
  };

  return (
    <div className="p-6 space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold">Tools</h1>
          <p className="text-xs text-bc-muted mt-0.5">
            {providers.length} Providers &middot; {mcpTools.length} MCP &middot; {cliTools.length} CLI
            {checkedTools && " \u00b7 checked"}
          </p>
        </div>
        <div className="flex items-center gap-2">
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

      {addForm && <AddToolForm type={addForm} onClose={() => setAddForm(null)} onAdded={() => { setCheckedTools(null); refresh(); }} />}

      {/* Providers */}
      <section>
        <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest mb-3">
          Providers ({providers.length}) &mdash; AI model providers
        </h2>
        {providers.length === 0 ? (
          <EmptyState icon="*" title="No providers" description="No AI providers configured." />
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2">
            {providers.map((t) => <ProviderCard key={t.name} tool={t} />)}
          </div>
        )}
      </section>

      {/* MCP Servers */}
      <section>
        <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest mb-3">
          MCP Servers ({mcpTools.length}) &mdash; External tool connections
        </h2>
        {mcpTools.length === 0 ? (
          <EmptyState icon="~" title="No MCP servers" description="Add an MCP server to connect external tools." />
        ) : (
          <div className="space-y-2">
            {mcpTools.map((t) => <ToolCard key={t.name} tool={t} onToggle={() => void handleToggle(t)} onRemove={() => void handleRemove(t)} />)}
          </div>
        )}
      </section>

      {/* CLI Tools */}
      <section>
        <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest mb-3">
          CLI Tools ({cliTools.length}) &mdash; Command-line utilities
        </h2>
        {cliTools.length === 0 ? (
          <EmptyState icon=">" title="No CLI tools" description="Add CLI tools like gh, aws, or wrangler." />
        ) : (
          <div className="space-y-2">
            {cliTools.map((t) => <ToolCard key={t.name} tool={t} onToggle={() => void handleToggle(t)} onRemove={() => void handleRemove(t)} />)}
          </div>
        )}
      </section>
    </div>
  );
}
