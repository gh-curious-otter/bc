import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client';
import type { Agent } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { useWebSocket } from '../hooks/useWebSocket';
import { StatusBadge } from '../components/StatusBadge';
import { Table } from '../components/Table';

export function Agents() {
  const fetcher = useCallback(async () => {
    const res = await api.listAgents();
    return res.agents;
  }, []);
  const { data: agents, loading, error, refresh } = usePolling(fetcher, 5000);
  const { subscribe } = useWebSocket();
  const [selected, setSelected] = useState<string | null>(null);

  // Refresh on agent state changes — cleanup prevents listener leak
  useEffect(() => {
    return subscribe('agent.state_changed', () => void refresh());
  }, [subscribe, refresh]);

  const columns = [
    { key: 'name', label: 'Name', render: (a: Agent) => <span className="font-medium">{a.name}</span> },
    { key: 'role', label: 'Role', render: (a: Agent) => <span className="text-bc-muted">{a.role}</span> },
    { key: 'tool', label: 'Tool', render: (a: Agent) => <span className="text-bc-muted">{a.tool}</span> },
    { key: 'state', label: 'Status', render: (a: Agent) => <StatusBadge status={a.state} /> },
    {
      key: 'cost', label: 'Cost', render: (a: Agent) => (
        <span className="text-bc-muted">${a.cost_usd.toFixed(4)}</span>
      ),
    },
  ];

  if (loading && !agents) {
    return <div className="p-6 text-bc-muted">Loading agents...</div>;
  }
  if (error && !agents) {
    return <div className="p-6 text-bc-error">Error: {error}</div>;
  }

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Agents</h1>
        <span className="text-sm text-bc-muted">{agents?.length ?? 0} agents</span>
      </div>

      <div className="rounded border border-bc-border overflow-hidden">
        <Table
          columns={columns}
          data={agents ?? []}
          keyFn={(a) => a.name}
          onRowClick={(a) => setSelected(a.name === selected ? null : a.name)}
          emptyMessage="No agents. Use 'bc agent create' to create one."
        />
      </div>

      {selected && agents && (
        <AgentDetail
          agent={agents.find((a) => a.name === selected)}
          onClose={() => setSelected(null)}
        />
      )}
    </div>
  );
}

function AgentDetail({ agent, onClose }: { agent?: Agent; onClose: () => void }) {
  const [message, setMessage] = useState('');
  const [sending, setSending] = useState(false);

  if (!agent) return null;

  const handleSend = async () => {
    if (!message.trim()) return;
    setSending(true);
    try {
      await api.sendToAgent(agent.name, message);
      setMessage('');
    } finally {
      setSending(false);
    }
  };

  return (
    <div className="rounded border border-bc-border bg-bc-surface p-4 space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="font-medium">{agent.name}</h3>
        <button onClick={onClose} className="text-bc-muted hover:text-bc-text text-sm">close</button>
      </div>
      <div className="grid grid-cols-3 gap-4 text-sm">
        <div><span className="text-bc-muted">Role:</span> {agent.role}</div>
        <div><span className="text-bc-muted">Tool:</span> {agent.tool}</div>
        <div><span className="text-bc-muted">Cost:</span> ${agent.cost_usd.toFixed(4)}</div>
      </div>
      <div className="flex gap-2">
        <input
          type="text"
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') void handleSend(); }}
          placeholder="Send message to agent..."
          className="flex-1 bg-bc-bg border border-bc-border rounded px-3 py-1.5 text-sm focus:outline-none focus:border-bc-accent"
        />
        <button
          onClick={() => void handleSend()}
          disabled={sending || !message.trim()}
          className="px-3 py-1.5 bg-bc-accent text-bc-bg rounded text-sm font-medium disabled:opacity-50"
        >
          Send
        </button>
      </div>
    </div>
  );
}
