import { useCallback, useEffect, useRef, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { api } from '../api/client';
import type { Agent } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { useWebSocket } from '../hooks/useWebSocket';
import { StatusBadge } from '../components/StatusBadge';

/** Strip ANSI escape sequences from a string. */
function stripAnsi(s: string): string {
  // eslint-disable-next-line no-control-regex
  return s.replace(/\x1b(?:\[[0-9;]*[A-Za-z]|\].*?(?:\x07|\x1b\\)|\([A-B0-2])/g, '');
}

function RoleBadge({ role }: { role: string }) {
  return (
    <span className="inline-block px-2 py-0.5 rounded text-xs font-medium bg-bc-accent/20 text-bc-accent">
      {role}
    </span>
  );
}

function MetadataRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-start gap-2 py-1.5 border-b border-bc-border/30 last:border-0">
      <span className="text-bc-muted text-sm w-32 shrink-0">{label}</span>
      <span className="text-sm break-all">{value ?? '—'}</span>
    </div>
  );
}

function formatTime(t?: string): string {
  if (!t) return '—';
  try {
    const d = new Date(t);
    if (isNaN(d.getTime())) return '—';
    return d.toLocaleString();
  } catch {
    return '—';
  }
}

export function AgentDetail() {
  const { name } = useParams<{ name: string }>();
  const [outputLines, setOutputLines] = useState<string[]>([]);
  const [message, setMessage] = useState('');
  const [sending, setSending] = useState(false);
  const outputRef = useRef<HTMLPreElement>(null);
  const { subscribe } = useWebSocket();

  const agentFetcher = useCallback(async () => {
    if (!name) throw new Error('No agent name');
    return api.getAgent(name);
  }, [name]);

  const { data: agent, loading, error, refresh } = usePolling<Agent>(agentFetcher, 3000);

  // Fetch initial output via peek, then stream via SSE
  useEffect(() => {
    if (!name) return;
    api.getAgentPeek(name, 100).then(({ output }) => {
      if (output) {
        setOutputLines(stripAnsi(output).split('\n'));
      }
    }).catch(() => {
      // Peek may fail for stopped agents — ignore
    });
  }, [name]);

  // Stream live output via SSE
  useEffect(() => {
    if (!name) return;

    const es = new EventSource(`/api/agents/${encodeURIComponent(name)}/output`);

    es.onmessage = (e: MessageEvent) => {
      try {
        const parsed = JSON.parse(e.data as string) as { output: string };
        if (parsed.output) {
          const newLines = stripAnsi(parsed.output).split('\n');
          setOutputLines((prev) => [...prev, ...newLines].slice(-500));
        }
      } catch {
        // ignore malformed events
      }
    };

    es.addEventListener('agent.output', ((e: MessageEvent) => {
      try {
        const parsed = JSON.parse(e.data as string) as { output: string };
        if (parsed.output) {
          const newLines = stripAnsi(parsed.output).split('\n');
          setOutputLines((prev) => [...prev, ...newLines].slice(-500));
        }
      } catch {
        // ignore
      }
    }) as EventListener);

    es.onerror = () => {
      // SSE reconnects automatically; no action needed
    };

    return () => {
      es.close();
    };
  }, [name]);

  // Auto-scroll output to bottom
  useEffect(() => {
    if (outputRef.current) {
      outputRef.current.scrollTop = outputRef.current.scrollHeight;
    }
  }, [outputLines]);

  // Refresh on agent state changes
  useEffect(() => {
    return subscribe('agent.state_changed', () => void refresh());
  }, [subscribe, refresh]);

  const handleSend = async () => {
    if (!name || !message.trim()) return;
    setSending(true);
    try {
      await api.sendToAgent(name, message);
      setMessage('');
    } finally {
      setSending(false);
    }
  };

  if (loading && !agent) {
    return <div className="p-6 text-bc-muted">Loading agent...</div>;
  }
  if (error && !agent) {
    return (
      <div className="p-6 space-y-2">
        <div className="text-bc-error">Error: {error}</div>
        <Link to="/agents" className="text-sm text-bc-accent hover:underline">Back to agents</Link>
      </div>
    );
  }
  if (!agent) return null;

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center gap-4">
        <Link to="/agents" className="text-bc-muted hover:text-bc-text text-sm">
          &larr; Agents
        </Link>
        <h1 className="text-xl font-bold">{agent.name}</h1>
        <RoleBadge role={agent.role} />
        <StatusBadge status={agent.state} />
      </div>

      {/* Task */}
      {agent.task && (
        <div className="rounded border border-bc-border bg-bc-surface p-3">
          <span className="text-xs text-bc-muted uppercase tracking-wide">Current Task</span>
          <p className="mt-1 text-sm">{agent.task}</p>
        </div>
      )}

      {/* Live Output */}
      <div className="space-y-2">
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">Live Output</h2>
        <pre
          ref={outputRef}
          className="rounded-lg border border-bc-border/50 bg-[#0a0a0f] p-4 text-xs leading-relaxed overflow-y-auto max-h-[50vh] md:max-h-[70vh] whitespace-pre-wrap text-bc-text/90 shadow-inner"
          style={{ fontFamily: "'Space Mono', ui-monospace, SFMono-Regular, Menlo, Consolas, monospace" }}
        >
          {outputLines.length > 0
            ? outputLines.join('\n')
            : <span className="text-bc-muted italic">No output yet. Agent may be idle or stopped.</span>}
        </pre>
      </div>

      {/* Send Message */}
      <div className="flex gap-2">
        <input
          type="text"
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') void handleSend(); }}
          placeholder="Send message to agent..."
          className="flex-1 min-w-0 bg-bc-bg border border-bc-border rounded px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent transition-colors duration-150"
        />
        <button
          onClick={() => void handleSend()}
          disabled={sending || !message.trim()}
          className="px-3 py-1.5 bg-bc-accent text-bc-bg rounded text-sm font-medium disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-bc-accent transition-colors duration-150"
        >
          Send
        </button>
      </div>

      {/* Metadata */}
      <div className="space-y-2">
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">Agent Details</h2>
        <div className="rounded border border-bc-border bg-bc-surface p-4">
          <MetadataRow label="Role" value={agent.role} />
          <MetadataRow label="Tool" value={agent.tool || '—'} />
          <MetadataRow label="State" value={<StatusBadge status={agent.state} />} />
          <MetadataRow label="Team" value={agent.team || '—'} />
          <MetadataRow label="Session" value={agent.session || '—'} />
          <MetadataRow label="Parent" value={agent.parent_id || '—'} />
          <MetadataRow
            label="Children"
            value={agent.children && agent.children.length > 0 ? agent.children.join(', ') : '—'}
          />
          <MetadataRow label="Created" value={formatTime(agent.created_at)} />
          <MetadataRow label="Started" value={formatTime(agent.started_at)} />
          <MetadataRow label="Updated" value={formatTime(agent.updated_at)} />
          <MetadataRow label="Stopped" value={formatTime(agent.stopped_at)} />
        </div>
      </div>
    </div>
  );
}
