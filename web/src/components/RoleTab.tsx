import { useCallback, useState } from 'react';
import { api } from '../api/client';
import type { Role } from '../api/client';
import { usePolling } from '../hooks/usePolling';

const PROMPT_COLLAPSE_THRESHOLD = 200;

function Badge({ text, color }: { text: string; color: string }) {
  return (
    <span className={`inline-block text-xs px-2 py-0.5 rounded ${color}`}>
      {text}
    </span>
  );
}

function ExpandableText({ text, label }: { text: string; label?: string }) {
  const long = text.length > PROMPT_COLLAPSE_THRESHOLD;
  const [expanded, setExpanded] = useState(!long);

  return (
    <div className="space-y-1">
      {label && <span className="text-xs text-bc-muted">{label}</span>}
      <pre className="text-xs bg-bc-bg rounded p-2 whitespace-pre-wrap text-bc-fg/80 border border-bc-border">
        {expanded ? text.trim() : text.slice(0, PROMPT_COLLAPSE_THRESHOLD).trim() + '...'}
      </pre>
      {long && (
        <button
          onClick={() => setExpanded(!expanded)}
          className="text-xs text-bc-accent hover:underline"
        >
          {expanded ? 'Collapse' : 'Show full prompt'}
        </button>
      )}
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="space-y-2">
      <h3 className="text-sm font-medium text-bc-muted uppercase tracking-wide">{title}</h3>
      {children}
    </div>
  );
}

function RuleItem({ name, content }: { name: string; content: string }) {
  const [expanded, setExpanded] = useState(false);
  return (
    <div className="rounded border border-bc-border bg-bc-bg p-2 space-y-1">
      <div
        className="flex items-center justify-between cursor-pointer"
        onClick={() => setExpanded(!expanded)}
      >
        <span className="text-xs font-medium">{name}</span>
        <span className="text-xs text-bc-muted">{expanded ? '\u25BC' : '\u25B6'}</span>
      </div>
      {expanded && (
        <pre className="text-xs whitespace-pre-wrap text-bc-fg/80 pt-1 border-t border-bc-border">
          {content.trim()}
        </pre>
      )}
    </div>
  );
}

export function RoleTab({ roleName }: { roleName: string }) {
  const fetcher = useCallback(async () => {
    const roles = await api.listRoles();
    return roles[roleName] ?? null;
  }, [roleName]);

  const { data: role, loading, error } = usePolling<Role | null>(fetcher, 30000);

  if (loading && role === undefined) {
    return <div className="text-sm text-bc-muted">Loading role...</div>;
  }

  if (error) {
    return <div className="text-sm text-bc-error">Failed to load role: {error}</div>;
  }

  if (!role) {
    return (
      <div className="rounded border border-bc-border bg-bc-surface p-4 text-sm text-bc-muted">
        Role not found: <span className="font-medium text-bc-text">{roleName}</span>
      </div>
    );
  }

  const mcpServers = role.MCPServers ?? [];
  const secrets = role.Secrets ?? [];
  const commands = role.Commands ?? {};
  const rules = role.Rules ?? {};
  const commandEntries = Object.entries(commands);
  const ruleEntries = Object.entries(rules);

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-bold">{role.Name}</h2>

      {/* Prompt */}
      {role.Prompt && (
        <Section title="Prompt">
          <ExpandableText text={role.Prompt} />
        </Section>
      )}

      {/* MCP Servers */}
      {mcpServers.length > 0 && (
        <Section title="MCP Servers">
          <div className="flex flex-wrap gap-2">
            {mcpServers.map((s) => (
              <Badge key={s} text={s} color="bg-blue-500/20 text-blue-400" />
            ))}
          </div>
        </Section>
      )}

      {/* Secrets */}
      {secrets.length > 0 && (
        <Section title="Secrets">
          <div className="flex flex-wrap gap-2">
            {secrets.map((s) => (
              <Badge key={s} text={s} color="bg-yellow-500/20 text-yellow-400" />
            ))}
          </div>
        </Section>
      )}

      {/* Commands */}
      {commandEntries.length > 0 && (
        <Section title="Commands">
          <div className="rounded border border-bc-border overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-bc-border bg-bc-bg">
                  <th className="text-left px-3 py-1.5 text-xs text-bc-muted font-medium">Name</th>
                  <th className="text-left px-3 py-1.5 text-xs text-bc-muted font-medium">Description</th>
                </tr>
              </thead>
              <tbody>
                {commandEntries.map(([name, desc]) => (
                  <tr key={name} className="border-b border-bc-border/30 last:border-0">
                    <td className="px-3 py-1.5 font-mono text-xs text-bc-accent">/{name}</td>
                    <td className="px-3 py-1.5 text-xs text-bc-fg/80 whitespace-pre-wrap">{desc || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Section>
      )}

      {/* Rules */}
      {ruleEntries.length > 0 && (
        <Section title="Rules">
          <div className="space-y-2">
            {ruleEntries.map(([name, content]) => (
              <RuleItem key={name} name={name} content={content} />
            ))}
          </div>
        </Section>
      )}
    </div>
  );
}
