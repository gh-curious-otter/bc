import { useCallback, useState } from "react";
import { api } from "../api/client";
import type { Role } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

type FormStatus =
  | { type: "idle" }
  | { type: "saving" }
  | { type: "success" }
  | { type: "error"; message: string };

function CreateRoleForm({ onCreated }: { onCreated: () => void }) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [prompt, setPrompt] = useState("");
  const [parentRoles, setParentRoles] = useState("");
  const [mcpServers, setMcpServers] = useState("");
  const [status, setStatus] = useState<FormStatus>({ type: "idle" });
  const [expanded, setExpanded] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const trimmedName = name.trim();
    if (!trimmedName) return;

    setStatus({ type: "saving" });
    try {
      await api.createRole({
        name: trimmedName,
        description: description.trim() || undefined,
        prompt: prompt.trim() || undefined,
        parent_roles: parentRoles
          .split(",")
          .map((s) => s.trim())
          .filter(Boolean),
        mcp_servers: mcpServers
          .split(",")
          .map((s) => s.trim())
          .filter(Boolean),
      });
      setName("");
      setDescription("");
      setPrompt("");
      setParentRoles("");
      setMcpServers("");
      setStatus({ type: "success" });
      setExpanded(false);
      onCreated();
      setTimeout(() => setStatus({ type: "idle" }), 2000);
    } catch (err) {
      setStatus({
        type: "error",
        message: err instanceof Error ? err.message : "Failed to create role",
      });
      setTimeout(() => setStatus({ type: "idle" }), 4000);
    }
  };

  if (!expanded) {
    return (
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={() => setExpanded(true)}
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 transition-opacity focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-offset-1 focus-visible:ring-offset-bc-bg"
          aria-label="Create new role"
        >
          + New Role
        </button>
        {status.type === "success" && (
          <span className="text-xs text-bc-success">Role created</span>
        )}
      </div>
    );
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="rounded border border-bc-border bg-bc-surface p-4 space-y-3"
    >
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">
          Create Role
        </h2>
        <button
          type="button"
          onClick={() => setExpanded(false)}
          className="text-xs text-bc-muted hover:text-bc-text transition-colors focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-offset-1 focus-visible:ring-offset-bc-bg rounded"
          aria-label="Cancel creating role"
        >
          Cancel
        </button>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div className="space-y-1">
          <label className="block text-sm text-bc-text" htmlFor="role-name">
            Name
          </label>
          <input
            id="role-name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="my-role"
            className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>
        <div className="space-y-1">
          <label className="block text-sm text-bc-text" htmlFor="role-desc">
            Description
          </label>
          <input
            id="role-desc"
            type="text"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Short description"
            className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>
        <div className="space-y-1">
          <label className="block text-sm text-bc-text" htmlFor="role-parents">
            Parent Roles
          </label>
          <input
            id="role-parents"
            type="text"
            value={parentRoles}
            onChange={(e) => setParentRoles(e.target.value)}
            placeholder="base, feature-dev"
            className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>
        <div className="space-y-1">
          <label className="block text-sm text-bc-text" htmlFor="role-mcp">
            MCP Servers
          </label>
          <input
            id="role-mcp"
            type="text"
            value={mcpServers}
            onChange={(e) => setMcpServers(e.target.value)}
            placeholder="bc, github"
            className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>
      </div>
      <div className="space-y-1">
        <label className="block text-sm text-bc-text" htmlFor="role-prompt">
          Prompt
        </label>
        <textarea
          id="role-prompt"
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          placeholder="# Role Name&#10;&#10;Role instructions in Markdown..."
          rows={4}
          className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm font-mono focus:outline-none focus:ring-2 focus:ring-bc-accent resize-y"
        />
      </div>
      <div className="flex items-center gap-3">
        <button
          type="submit"
          disabled={status.type === "saving" || !name.trim()}
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-offset-1 focus-visible:ring-offset-bc-bg"
        >
          {status.type === "saving" ? "Creating..." : "Create Role"}
        </button>
        {status.type === "success" && (
          <span className="text-xs text-bc-success">Role created</span>
        )}
        {status.type === "error" && (
          <span className="text-xs text-bc-error">{status.message}</span>
        )}
      </div>
    </form>
  );
}

function DeleteButton({
  name,
  onDeleted,
}: {
  name: string;
  onDeleted: () => void;
}) {
  const [confirming, setConfirming] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const handleDelete = async () => {
    setDeleting(true);
    try {
      await api.deleteRole(name);
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
          aria-label={`Confirm delete role ${name}`}
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
      aria-label={`Delete role ${name}`}
    >
      Delete
    </button>
  );
}

export function Roles() {
  const fetcher = useCallback(async () => {
    const res = await api.listRoles();
    return Object.entries(res).map(([key, role]) => ({ key, ...role }));
  }, []);
  const {
    data: roles,
    loading,
    error,
    refresh,
    timedOut,
  } = usePolling(fetcher, 30000);

  if (loading && !roles) {
    return (
      <div className="p-6 space-y-4">
        <div className="h-6 w-20 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="text" rows={6} />
      </div>
    );
  }
  if (timedOut && !roles) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Roles took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !roles) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load roles"
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
        <h1 className="text-xl font-bold">Roles</h1>
        <span className="text-sm text-bc-muted">
          {roles?.length ?? 0} roles
        </span>
      </div>

      <CreateRoleForm onCreated={refresh} />

      {(roles ?? []).length === 0 ? (
        <EmptyState
          icon="@"
          title="No roles defined"
          description="Define roles to assign capabilities and prompts to agents."
        />
      ) : (
        <div className="grid gap-4">
          {(roles ?? []).map((r) => (
            <RoleCard key={r.key} role={r} onDeleted={refresh} />
          ))}
        </div>
      )}
    </div>
  );
}

function Tags({
  label,
  items,
  color,
}: {
  label: string;
  items: string[];
  color: string;
}) {
  if (!items || items.length === 0) return null;
  return (
    <div className="flex flex-wrap items-center gap-2">
      <span className="text-xs text-bc-muted w-20 shrink-0">{label}</span>
      {items.map((v) => (
        <span key={v} className={`text-xs px-2 py-0.5 rounded ${color}`}>
          {v}
        </span>
      ))}
    </div>
  );
}

function MapTags({
  label,
  items,
  color,
}: {
  label: string;
  items: Record<string, string>;
  color: string;
}) {
  const keys = Object.keys(items ?? {});
  if (keys.length === 0) return null;
  return (
    <div className="flex flex-wrap items-center gap-2">
      <span className="text-xs text-bc-muted w-20 shrink-0">{label}</span>
      {keys.map((k) => (
        <span key={k} className={`text-xs px-2 py-0.5 rounded ${color}`}>
          {k}
        </span>
      ))}
    </div>
  );
}

function Pre({ label, text }: { label: string; text: string }) {
  if (!text) return null;
  return (
    <div className="space-y-1">
      <span className="text-xs text-bc-muted">{label}</span>
      <pre className="text-xs bg-bc-bg rounded p-2 whitespace-pre-wrap text-bc-text/80 border border-bc-border">
        {text.trim()}
      </pre>
    </div>
  );
}

function RoleCard({
  role,
  onDeleted,
}: {
  role: Role & { key: string };
  onDeleted: () => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const hasPrompts =
    role.PromptCreate ||
    role.PromptStart ||
    role.PromptStop ||
    role.PromptDelete;
  const hasCommands = Object.keys(role.Commands ?? {}).length > 0;
  const hasRules = Object.keys(role.Rules ?? {}).length > 0;
  const hasSkills = Object.keys(role.Skills ?? {}).length > 0;

  return (
    <div className="rounded border border-bc-border bg-bc-surface p-4 space-y-3">
      <div
        className="flex items-center justify-between cursor-pointer focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-offset-1 focus-visible:ring-offset-bc-bg rounded"
        onClick={() => setExpanded(!expanded)}
        role="button"
        tabIndex={0}
        aria-expanded={expanded}
        aria-label={`Toggle details for role ${role.Name}`}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            setExpanded(!expanded);
          }
        }}
      >
        <div className="flex items-center gap-3">
          <h3 className="font-medium text-lg">{role.Name}</h3>
        </div>
        <div className="flex items-center gap-2">
          {hasPrompts && (
            <span className="text-xs px-2 py-0.5 rounded bg-purple-500/20 text-purple-400">
              lifecycle
            </span>
          )}
          {hasCommands && (
            <span className="text-xs px-2 py-0.5 rounded bg-cyan-500/20 text-cyan-400">
              commands
            </span>
          )}
          {hasRules && (
            <span className="text-xs px-2 py-0.5 rounded bg-orange-500/20 text-orange-400">
              rules
            </span>
          )}
          <span className="text-xs text-bc-muted">
            {expanded ? "\u25BC" : "\u25B6"}
          </span>
        </div>
      </div>

      <div className="space-y-1.5">
        <Tags
          label="mcp"
          items={role.MCPServers ?? []}
          color="bg-blue-500/20 text-blue-400"
        />
        <Tags
          label="secrets"
          items={role.Secrets ?? []}
          color="bg-yellow-500/20 text-yellow-400"
        />
        <Tags
          label="plugins"
          items={role.Plugins ?? []}
          color="bg-green-500/20 text-green-400"
        />
        <MapTags
          label="commands"
          items={role.Commands}
          color="bg-cyan-500/20 text-cyan-400"
        />
        <MapTags
          label="rules"
          items={role.Rules}
          color="bg-orange-500/20 text-orange-400"
        />
        {hasSkills && (
          <MapTags
            label="skills"
            items={role.Skills}
            color="bg-emerald-500/20 text-emerald-400"
          />
        )}
      </div>

      {expanded && (
        <div className="space-y-3 pt-2 border-t border-bc-border">
          <Pre label="Role Prompt (CLAUDE.md)" text={role.Prompt} />
          {hasPrompts && (
            <div className="grid grid-cols-2 gap-3">
              <Pre label="on create" text={role.PromptCreate} />
              <Pre label="on start" text={role.PromptStart} />
              <Pre label="on stop" text={role.PromptStop} />
              <Pre label="on delete" text={role.PromptDelete} />
            </div>
          )}
          {hasCommands && (
            <div className="space-y-2">
              <span className="text-xs text-bc-muted">
                Commands (.claude/commands/)
              </span>
              {Object.entries(role.Commands).map(([name, content]) => (
                <Pre key={name} label={`/${name}`} text={content} />
              ))}
            </div>
          )}
          {hasRules && (
            <div className="space-y-2">
              <span className="text-xs text-bc-muted">
                Rules (.claude/rules/)
              </span>
              {Object.entries(role.Rules).map(([name, content]) => (
                <Pre key={name} label={name} text={content} />
              ))}
            </div>
          )}
          {role.Review && <Pre label="REVIEW.md" text={role.Review} />}
          <div className="pt-2">
            <DeleteButton name={role.key} onDeleted={onDeleted} />
          </div>
        </div>
      )}
    </div>
  );
}
