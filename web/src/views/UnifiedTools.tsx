import { useCallback, useState } from "react";
import { api } from "../api/client";
import type { Tool, MCPServer } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

interface ToolsData {
  tools: Tool[];
  mcpServers: MCPServer[];
}

type AddFormType = "mcp" | "cli" | null;

function StatusDot({ status }: { status: "connected" | "installed" | "error" | "missing" }) {
  const colors = {
    connected: "bg-bc-success",
    installed: "bg-bc-success",
    error: "bg-bc-warning",
    missing: "bg-bc-error",
  };
  return <span className={`inline-block w-2 h-2 rounded-full ${colors[status]}`} />;
}

function StatusLabel({ status }: { status: "connected" | "installed" | "error" | "missing" }) {
  const labels = {
    connected: "Connected",
    installed: "Installed",
    error: "Error",
    missing: "Not installed",
  };
  const colors = {
    connected: "text-bc-success",
    installed: "text-bc-success",
    error: "text-bc-warning",
    missing: "text-bc-error",
  };
  return <span className={`text-xs ${colors[status]}`}>{labels[status]}</span>;
}

function MCPCard({
  server,
  onToggle,
  onRemove,
}: {
  server: MCPServer;
  onToggle: () => void;
  onRemove: () => void;
}) {
  const [removing, setRemoving] = useState(false);
  const [confirmRemove, setConfirmRemove] = useState(false);
  const status = server.enabled ? "connected" : "error";

  const handleRemove = async () => {
    setRemoving(true);
    try {
      onRemove();
    } finally {
      setRemoving(false);
      setConfirmRemove(false);
    }
  };

  return (
    <div className="rounded border border-bc-border bg-bc-surface p-4 flex items-start gap-3">
      <StatusDot status={status} />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-medium text-sm">{server.name}</span>
          <span className="text-[10px] px-1.5 py-0.5 rounded bg-bc-accent/10 text-bc-accent font-medium">
            MCP
          </span>
          <StatusLabel status={status} />
        </div>
        <div className="mt-1 text-xs text-bc-muted truncate" title={server.url || server.command}>
          {server.transport === "sse"
            ? server.url
            : server.command}
        </div>
        <div className="mt-1 text-[10px] text-bc-muted">
          Transport: {server.transport}
        </div>
      </div>
      <div className="flex items-center gap-2 shrink-0">
        <button
          type="button"
          onClick={onToggle}
          className={`text-xs px-2 py-1 rounded transition-colors ${
            server.enabled
              ? "bg-bc-success/10 text-bc-success hover:bg-bc-success/20"
              : "bg-bc-border text-bc-muted hover:bg-bc-border/80"
          }`}
          aria-label={server.enabled ? "Disable server" : "Enable server"}
        >
          {server.enabled ? "Enabled" : "Disabled"}
        </button>
        {confirmRemove ? (
          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={() => void handleRemove()}
              disabled={removing}
              className="text-xs px-2 py-1 rounded bg-bc-error/20 text-bc-error disabled:opacity-50"
              aria-label="Confirm remove"
            >
              Yes
            </button>
            <button
              type="button"
              onClick={() => setConfirmRemove(false)}
              className="text-xs px-2 py-1 rounded border border-bc-border text-bc-muted"
              aria-label="Cancel remove"
            >
              No
            </button>
          </div>
        ) : (
          <button
            type="button"
            onClick={() => setConfirmRemove(true)}
            className="text-xs px-2 py-1 rounded border border-bc-border text-bc-muted hover:text-bc-error hover:border-bc-error/50 transition-colors"
            aria-label={`Remove ${server.name}`}
          >
            Remove
          </button>
        )}
      </div>
    </div>
  );
}

function CLIToolCard({
  tool,
  onToggle,
  onRemove,
}: {
  tool: Tool;
  onToggle: () => void;
  onRemove: () => void;
}) {
  const [removing, setRemoving] = useState(false);
  const [confirmRemove, setConfirmRemove] = useState(false);
  const status = tool.enabled ? "installed" : "missing";

  const handleRemove = async () => {
    setRemoving(true);
    try {
      onRemove();
    } finally {
      setRemoving(false);
      setConfirmRemove(false);
    }
  };

  return (
    <div className="rounded border border-bc-border bg-bc-surface p-4 flex items-start gap-3">
      <StatusDot status={status} />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-medium text-sm">{tool.name}</span>
          <span className="text-[10px] px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-400 font-medium">
            CLI
          </span>
          {tool.builtin && (
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-bc-border text-bc-muted">
              built-in
            </span>
          )}
          <StatusLabel status={status} />
        </div>
        <div className="mt-1 text-xs text-bc-muted font-mono truncate" title={tool.command}>
          {tool.command}
        </div>
        {tool.install_cmd && (
          <div className="mt-0.5 text-[10px] text-bc-muted">
            Install: <code className="bg-bc-bg px-1 rounded">{tool.install_cmd}</code>
          </div>
        )}
      </div>
      <div className="flex items-center gap-2 shrink-0">
        <button
          type="button"
          onClick={onToggle}
          className={`text-xs px-2 py-1 rounded transition-colors ${
            tool.enabled
              ? "bg-bc-success/10 text-bc-success hover:bg-bc-success/20"
              : "bg-bc-border text-bc-muted hover:bg-bc-border/80"
          }`}
          aria-label={tool.enabled ? "Disable tool" : "Enable tool"}
        >
          {tool.enabled ? "Enabled" : "Disabled"}
        </button>
        {!tool.builtin && (
          confirmRemove ? (
            <div className="flex items-center gap-1">
              <button
                type="button"
                onClick={() => void handleRemove()}
                disabled={removing}
                className="text-xs px-2 py-1 rounded bg-bc-error/20 text-bc-error disabled:opacity-50"
                aria-label="Confirm remove"
              >
                Yes
              </button>
              <button
                type="button"
                onClick={() => setConfirmRemove(false)}
                className="text-xs px-2 py-1 rounded border border-bc-border text-bc-muted"
                aria-label="Cancel remove"
              >
                No
              </button>
            </div>
          ) : (
            <button
              type="button"
              onClick={() => setConfirmRemove(true)}
              className="text-xs px-2 py-1 rounded border border-bc-border text-bc-muted hover:text-bc-error hover:border-bc-error/50 transition-colors"
              aria-label={`Remove ${tool.name}`}
            >
              Remove
            </button>
          )
        )}
      </div>
    </div>
  );
}

function AddToolForm({
  type,
  onClose,
  onAdded,
}: {
  type: "mcp" | "cli";
  onClose: () => void;
  onAdded: () => void;
}) {
  const [name, setName] = useState("");
  const [command, setCommand] = useState("");
  const [url, setUrl] = useState("");
  const [transport, setTransport] = useState<"stdio" | "sse">("sse");
  const [installCmd, setInstallCmd] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    if (!name.trim()) return;
    setSubmitting(true);
    setError(null);
    try {
      if (type === "mcp") {
        await api.registerMCP({
          name: name.trim(),
          transport,
          command: transport === "stdio" ? command.trim() : "",
          url: transport === "sse" ? url.trim() : "",
          enabled: true,
        });
      } else {
        // CLI tool registration — uses the tools API
        // For now, CLI tools are managed via role config
        throw new Error("CLI tool registration coming soon — configure via role settings");
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
        <h3 className="text-sm font-medium">
          Add {type === "mcp" ? "MCP Server" : "CLI Tool"}
        </h3>
        <button
          type="button"
          onClick={onClose}
          className="text-xs text-bc-muted hover:text-bc-text"
        >
          Cancel
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div>
          <label className="text-xs text-bc-muted block mb-1">Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={type === "mcp" ? "playwright" : "gh"}
            className="w-full px-2 py-1.5 text-sm rounded border border-bc-border bg-bc-bg text-bc-text focus:outline-none focus:ring-1 focus:ring-bc-accent"
          />
        </div>

        {type === "mcp" ? (
          <>
            <div>
              <label className="text-xs text-bc-muted block mb-1">Transport</label>
              <select
                value={transport}
                onChange={(e) => setTransport(e.target.value as "stdio" | "sse")}
                className="w-full px-2 py-1.5 text-sm rounded border border-bc-border bg-bc-bg text-bc-text focus:outline-none focus:ring-1 focus:ring-bc-accent"
              >
                <option value="sse">SSE</option>
                <option value="stdio">stdio</option>
              </select>
            </div>
            {transport === "sse" ? (
              <div className="md:col-span-2">
                <label className="text-xs text-bc-muted block mb-1">URL</label>
                <input
                  type="text"
                  value={url}
                  onChange={(e) => setUrl(e.target.value)}
                  placeholder="http://localhost:3000/sse"
                  className="w-full px-2 py-1.5 text-sm rounded border border-bc-border bg-bc-bg text-bc-text focus:outline-none focus:ring-1 focus:ring-bc-accent"
                />
              </div>
            ) : (
              <div className="md:col-span-2">
                <label className="text-xs text-bc-muted block mb-1">Command</label>
                <input
                  type="text"
                  value={command}
                  onChange={(e) => setCommand(e.target.value)}
                  placeholder="npx @playwright/mcp"
                  className="w-full px-2 py-1.5 text-sm rounded border border-bc-border bg-bc-bg text-bc-text focus:outline-none focus:ring-1 focus:ring-bc-accent"
                />
              </div>
            )}
          </>
        ) : (
          <>
            <div>
              <label className="text-xs text-bc-muted block mb-1">Command</label>
              <input
                type="text"
                value={command}
                onChange={(e) => setCommand(e.target.value)}
                placeholder="gh"
                className="w-full px-2 py-1.5 text-sm rounded border border-bc-border bg-bc-bg text-bc-text focus:outline-none focus:ring-1 focus:ring-bc-accent"
              />
            </div>
            <div className="md:col-span-2">
              <label className="text-xs text-bc-muted block mb-1">Install Command (optional)</label>
              <input
                type="text"
                value={installCmd}
                onChange={(e) => setInstallCmd(e.target.value)}
                placeholder="apt-get install -y gh"
                className="w-full px-2 py-1.5 text-sm rounded border border-bc-border bg-bc-bg text-bc-text focus:outline-none focus:ring-1 focus:ring-bc-accent"
              />
            </div>
          </>
        )}
      </div>

      {error && <p className="text-xs text-bc-error">{error}</p>}

      <button
        type="button"
        onClick={() => void handleSubmit()}
        disabled={submitting || !name.trim()}
        className="px-3 py-1.5 text-sm rounded bg-bc-accent text-bc-bg font-medium disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-bc-accent"
      >
        {submitting ? "Adding..." : `Add ${type === "mcp" ? "MCP Server" : "CLI Tool"}`}
      </button>
    </div>
  );
}

export function UnifiedTools() {
  const fetcher = useCallback(async (): Promise<ToolsData> => {
    const [r0, r1] = await Promise.allSettled([
      api.listTools(),
      api.listMCP(),
    ]);
    return {
      tools: r0.status === "fulfilled" ? r0.value : [],
      mcpServers: r1.status === "fulfilled" ? r1.value : [],
    };
  }, []);

  const { data, loading, error, refresh, timedOut } = usePolling(fetcher, 10000);
  const [addForm, setAddForm] = useState<AddFormType>(null);

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
        <EmptyState icon="!" title="Tools timed out" actionLabel="Retry" onAction={refresh} />
      </div>
    );
  }
  if (error && !data) {
    return (
      <div className="p-6">
        <EmptyState icon="!" title="Failed to load tools" description={error} actionLabel="Retry" onAction={refresh} />
      </div>
    );
  }

  const mcpServers = data?.mcpServers ?? [];
  const cliTools = data?.tools ?? [];

  const handleToggleMCP = async (name: string, enabled: boolean) => {
    try {
      await enabled ? api.disableMCP(name) : api.enableMCP(name);
      refresh();
    } catch {
      // silently fail
    }
  };

  const handleRemoveMCP = async (name: string) => {
    try {
      await api.removeMCP(name);
      refresh();
    } catch {
      // silently fail
    }
  };

  const handleToggleTool = async (name: string, enabled: boolean) => {
    try {
      await enabled ? api.disableTool(name) : api.enableTool(name);
      refresh();
    } catch {
      // silently fail
    }
  };

  const handleRemoveTool = async (name: string) => {
    try {
      await api.deleteTool(name);
      refresh();
    } catch {
      // silently fail
    }
  };

  return (
    <div className="p-6 space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold">Tools</h1>
          <p className="text-xs text-bc-muted mt-0.5">
            {mcpServers.length} MCP servers · {cliTools.length} CLI tools
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setAddForm(addForm === "mcp" ? null : "mcp")}
            className="px-3 py-1.5 text-sm rounded bg-bc-accent/10 text-bc-accent hover:bg-bc-accent/20 transition-colors focus-visible:ring-2 focus-visible:ring-bc-accent"
          >
            + MCP Server
          </button>
          <button
            type="button"
            onClick={() => setAddForm(addForm === "cli" ? null : "cli")}
            className="px-3 py-1.5 text-sm rounded bg-blue-500/10 text-blue-400 hover:bg-blue-500/20 transition-colors focus-visible:ring-2 focus-visible:ring-bc-accent"
          >
            + CLI Tool
          </button>
        </div>
      </div>

      {addForm && (
        <AddToolForm
          type={addForm}
          onClose={() => setAddForm(null)}
          onAdded={refresh}
        />
      )}

      {/* MCP Servers */}
      <section>
        <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest mb-3">
          MCP Servers ({mcpServers.length})
        </h2>
        {mcpServers.length === 0 ? (
          <EmptyState
            icon="🔌"
            title="No MCP servers"
            description="Add an MCP server to connect external tools."
          />
        ) : (
          <div className="space-y-2">
            {mcpServers.map((s) => (
              <MCPCard
                key={s.name}
                server={s}
                onToggle={() => void handleToggleMCP(s.name, s.enabled)}
                onRemove={() => void handleRemoveMCP(s.name)}
              />
            ))}
          </div>
        )}
      </section>

      {/* CLI Tools */}
      <section>
        <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest mb-3">
          CLI Tools ({cliTools.length})
        </h2>
        {cliTools.length === 0 ? (
          <EmptyState
            icon="⌨"
            title="No CLI tools"
            description="Add CLI tools like gh, aws, or wrangler."
          />
        ) : (
          <div className="space-y-2">
            {cliTools.map((t) => (
              <CLIToolCard
                key={t.name}
                tool={t}
                onToggle={() => void handleToggleTool(t.name, t.enabled)}
                onRemove={() => void handleRemoveTool(t.name)}
              />
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
