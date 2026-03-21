import { useCallback, useState } from 'react';
import { api } from '../api/client';
import type { Role } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import { EmptyState } from '../components/EmptyState';

export function Roles() {
  const fetcher = useCallback(async () => {
    const res = await api.listRoles();
    return Object.entries(res).map(([key, role]) => ({ key, ...role }));
  }, []);
  const { data: roles, loading, error, refresh, timedOut } = usePolling(fetcher, 30000);

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
        <span className="text-sm text-bc-muted">{roles?.length ?? 0} roles</span>
      </div>
      {(roles ?? []).length === 0 ? (
        <EmptyState
          icon="@"
          title="No roles defined"
          description="Define roles in .bc/roles/ to assign capabilities and prompts to agents."
        />
      ) : (
        <div className="grid gap-4">
          {(roles ?? []).map((r) => (
            <RoleCard key={r.key} role={r} />
          ))}
        </div>
      )}
    </div>
  );
}

function Tags({ label, items, color }: { label: string; items: string[]; color: string }) {
  if (!items || items.length === 0) return null;
  return (
    <div className="flex flex-wrap items-center gap-2">
      <span className="text-xs text-bc-muted w-20 shrink-0">{label}</span>
      {items.map((v) => (
        <span key={v} className={`text-xs px-2 py-0.5 rounded ${color}`}>{v}</span>
      ))}
    </div>
  );
}

function MapTags({ label, items, color }: { label: string; items: Record<string, string>; color: string }) {
  const keys = Object.keys(items ?? {});
  if (keys.length === 0) return null;
  return (
    <div className="flex flex-wrap items-center gap-2">
      <span className="text-xs text-bc-muted w-20 shrink-0">{label}</span>
      {keys.map((k) => (
        <span key={k} className={`text-xs px-2 py-0.5 rounded ${color}`}>{k}</span>
      ))}
    </div>
  );
}

function Pre({ label, text }: { label: string; text: string }) {
  if (!text) return null;
  return (
    <div className="space-y-1">
      <span className="text-xs text-bc-muted">{label}</span>
      <pre className="text-xs bg-bc-bg rounded p-2 whitespace-pre-wrap text-bc-fg/80 border border-bc-border">
        {text.trim()}
      </pre>
    </div>
  );
}

function RoleCard({ role }: { role: Role & { key: string } }) {
  const [expanded, setExpanded] = useState(false);
  const hasPrompts = role.PromptCreate || role.PromptStart || role.PromptStop || role.PromptDelete;
  const hasCommands = Object.keys(role.Commands ?? {}).length > 0;
  const hasRules = Object.keys(role.Rules ?? {}).length > 0;
  const hasSkills = Object.keys(role.Skills ?? {}).length > 0;

  return (
    <div className="rounded border border-bc-border bg-bc-surface p-4 space-y-3">
      <div className="flex items-center justify-between cursor-pointer" onClick={() => setExpanded(!expanded)}>
        <div className="flex items-center gap-3">
          <h3 className="font-medium text-lg">{role.Name}</h3>
        </div>
        <div className="flex items-center gap-2">
          {hasPrompts && <span className="text-xs px-2 py-0.5 rounded bg-purple-500/20 text-purple-400">lifecycle</span>}
          {hasCommands && <span className="text-xs px-2 py-0.5 rounded bg-cyan-500/20 text-cyan-400">commands</span>}
          {hasRules && <span className="text-xs px-2 py-0.5 rounded bg-orange-500/20 text-orange-400">rules</span>}
          <span className="text-xs text-bc-muted">{expanded ? '\u25BC' : '\u25B6'}</span>
        </div>
      </div>

      <div className="space-y-1.5">
        <Tags label="mcp" items={role.MCPServers ?? []} color="bg-blue-500/20 text-blue-400" />
        <Tags label="secrets" items={role.Secrets ?? []} color="bg-yellow-500/20 text-yellow-400" />
        <Tags label="plugins" items={role.Plugins ?? []} color="bg-green-500/20 text-green-400" />
        <MapTags label="commands" items={role.Commands} color="bg-cyan-500/20 text-cyan-400" />
        <MapTags label="rules" items={role.Rules} color="bg-orange-500/20 text-orange-400" />
        {hasSkills && <MapTags label="skills" items={role.Skills} color="bg-emerald-500/20 text-emerald-400" />}
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
              <span className="text-xs text-bc-muted">Commands (.claude/commands/)</span>
              {Object.entries(role.Commands).map(([name, content]) => (
                <Pre key={name} label={`/${name}`} text={content} />
              ))}
            </div>
          )}
          {hasRules && (
            <div className="space-y-2">
              <span className="text-xs text-bc-muted">Rules (.claude/rules/)</span>
              {Object.entries(role.Rules).map(([name, content]) => (
                <Pre key={name} label={name} text={content} />
              ))}
            </div>
          )}
          {role.Review && <Pre label="REVIEW.md" text={role.Review} />}
        </div>
      )}
    </div>
  );
}
