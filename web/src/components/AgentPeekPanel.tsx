import { useEffect, useRef, useState } from 'react';
import { StatusBadge } from './StatusBadge';
import type { Agent } from '../api/client';
import { api } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { useCallback } from 'react';

/** Strip ANSI escape codes from terminal output. */
function stripAnsi(text: string): string {
  // Covers CSI sequences, OSC sequences, and simple escape codes
  return text.replace(
    // eslint-disable-next-line no-control-regex
    /\x1b\[[0-9;]*[A-Za-z]|\x1b\][^\x07]*\x07|\x1b[()][AB012]|\x1b[=>]|\x1b\[[?]?[0-9;]*[hlm]/g,
    '',
  );
}

interface AgentPeekPanelProps {
  agentName: string;
  onClose: () => void;
}

export function AgentPeekPanel({ agentName, onClose }: AgentPeekPanelProps) {
  const [output, setOutput] = useState('');
  const outputRef = useRef<HTMLPreElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  // Fetch agent details for header metadata
  const agentFetcher = useCallback(async () => {
    return api.getAgent(agentName);
  }, [agentName]);
  const { data: agent } = usePolling<Agent>(agentFetcher, 5000);

  // Connect to SSE stream for live terminal output
  useEffect(() => {
    setOutput('');

    const es = new EventSource(`/api/agents/${encodeURIComponent(agentName)}/output`);

    // Initial snapshot comes as a plain "message" event (no event: field)
    es.onmessage = (e: MessageEvent) => {
      try {
        const parsed = JSON.parse(e.data as string) as { output?: string };
        if (parsed.output) {
          setOutput((prev) => prev + stripAnsi(parsed.output!));
        }
      } catch {
        // ignore malformed data
      }
    };

    // Incremental updates arrive as named "agent.output" events
    es.addEventListener('agent.output', ((e: MessageEvent) => {
      try {
        const parsed = JSON.parse(e.data as string) as { output?: string };
        if (parsed.output) {
          setOutput((prev) => prev + stripAnsi(parsed.output!));
        }
      } catch {
        // ignore
      }
    }) as EventListener);

    es.onerror = () => {
      // EventSource auto-reconnects; nothing to do
    };

    return () => {
      es.close();
    };
  }, [agentName]);

  // Auto-scroll when output grows, only if near bottom
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;
    const isNearBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 120;
    if (isNearBottom) {
      container.scrollTop = container.scrollHeight;
    }
  }, [output]);

  return (
    <div className="w-[420px] shrink-0 border-l border-bc-border flex flex-col bg-bc-bg">
      {/* Header */}
      <div className="px-4 py-2 border-b border-bc-border bg-bc-surface flex items-center justify-between">
        <div className="flex items-center gap-2 min-w-0">
          <span className="font-medium truncate">{agentName}</span>
          {agent && <StatusBadge status={agent.state} />}
        </div>
        <button
          onClick={onClose}
          className="text-bc-muted hover:text-bc-text text-sm ml-2 shrink-0"
          aria-label="Close peek panel"
        >
          close
        </button>
      </div>

      {/* Agent metadata */}
      {agent && (
        <div className="px-4 py-1.5 border-b border-bc-border/50 text-xs text-bc-muted flex gap-3">
          <span>Role: {agent.role}</span>
          <span>Tool: {agent.tool || '\u2014'}</span>
          {agent.cost_usd != null && <span>Cost: ${agent.cost_usd.toFixed(4)}</span>}
        </div>
      )}

      {/* Live output */}
      <div ref={scrollContainerRef} className="flex-1 overflow-auto">
        <pre
          ref={outputRef}
          className="p-3 text-xs font-mono whitespace-pre-wrap break-words leading-relaxed text-bc-text/80"
        >
          {output || 'Connecting...'}
        </pre>
      </div>
    </div>
  );
}
