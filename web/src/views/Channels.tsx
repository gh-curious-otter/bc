import { useCallback, useState, useEffect, useRef } from 'react';
import { api } from '../api/client';
import type { ChannelMessage } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { useWebSocket } from '../hooks/useWebSocket';
import { AgentPeekPanel } from '../components/AgentPeekPanel';

export function Channels() {
  const fetcher = useCallback(async () => {
    const res = await api.listChannels();
    return res;
  }, []);
  const { data: channels, loading, error } = usePolling(fetcher, 10000);
  const [selected, setSelected] = useState<string | null>(null);
  const [peekAgent, setPeekAgent] = useState<string | null>(null);

  if (loading && !channels) {
    return <div className="p-6 text-bc-muted">Loading channels...</div>;
  }
  if (error && !channels) {
    return <div className="p-6 text-bc-error">Error: {error}</div>;
  }

  return (
    <div className="flex h-full">
      <div className="w-56 shrink-0 border-r border-bc-border overflow-auto">
        <div className="p-3 border-b border-bc-border">
          <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">Channels</h2>
        </div>
        {(channels ?? []).map((ch) => (
          <button
            key={ch.name}
            onClick={() => setSelected(ch.name)}
            className={`w-full text-left px-3 py-2 text-sm border-b border-bc-border/30 ${
              selected === ch.name
                ? 'bg-bc-accent/10 text-bc-accent'
                : 'text-bc-text hover:bg-bc-surface'
            }`}
          >
            <span className="font-medium">#{ch.name}</span>
            <span className="ml-2 text-xs text-bc-muted">({ch.member_count})</span>
          </button>
        ))}
      </div>
      <div className="flex-1 flex flex-col min-w-0">
        {selected ? (
          <ChatRoom channelName={selected} onPeekAgent={setPeekAgent} />
        ) : (
          <div className="flex-1 flex items-center justify-center text-bc-muted text-sm">
            Select a channel
          </div>
        )}
      </div>
      {peekAgent && (
        <AgentPeekPanel agentName={peekAgent} onClose={() => setPeekAgent(null)} />
      )}
    </div>
  );
}

function ChatRoom({ channelName, onPeekAgent }: { channelName: string; onPeekAgent: (name: string) => void }) {
  const [messages, setMessages] = useState<ChannelMessage[]>([]);
  const [input, setInput] = useState('');
  const [sending, setSending] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const { subscribe } = useWebSocket();

  // Fetch history on channel change
  useEffect(() => {
    void (async () => {
      try {
        const msgs = await api.getChannelHistory(channelName, 100);
        setMessages(msgs ?? []);
      } catch {
        setMessages([]);
      }
    })();
  }, [channelName]);

  // Live messages via SSE — deduplicate by ID to prevent doubles
  useEffect(() => {
    return subscribe('channel.message', (event) => {
      const data = event.data as { channel?: string; message?: ChannelMessage };
      if (data.channel === channelName && data.message) {
        setMessages((prev) => {
          if (prev.some((m) => m.id === data.message!.id)) return prev;
          return [...prev, data.message!];
        });
      }
    });
  }, [subscribe, channelName]);

  // Auto-scroll only when user is near the bottom
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;
    const isNearBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 100;
    if (isNearBottom) {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages]);

  const handleSend = async () => {
    if (!input.trim()) return;
    setSending(true);
    try {
      await api.sendToChannel(channelName, input);
      setInput('');
      // SSE listener will deliver the message — no refetch needed
    } finally {
      setSending(false);
    }
  };

  return (
    <>
      <div className="px-4 py-2 border-b border-bc-border bg-bc-surface">
        <span className="font-medium">#{channelName}</span>
      </div>
      <div ref={scrollContainerRef} className="flex-1 overflow-auto p-4 space-y-2">
        {messages.map((msg) => (
          <div key={msg.id} className="text-sm">
            <button
              onClick={() => onPeekAgent(msg.sender)}
              className="font-medium text-bc-accent hover:underline cursor-pointer"
              title={`Peek at ${msg.sender}'s terminal`}
            >
              {msg.sender}
            </button>
            <span className="ml-2 text-xs text-bc-muted">
              {new Date(msg.created_at).toLocaleTimeString()}
            </span>
            <p className="mt-0.5">{msg.content}</p>
          </div>
        ))}
        <div ref={bottomRef} />
      </div>
      <div className="p-3 border-t border-bc-border flex gap-2">
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') void handleSend(); }}
          placeholder={`Message #${channelName}...`}
          className="flex-1 bg-bc-bg border border-bc-border rounded px-3 py-1.5 text-sm focus:outline-none focus:border-bc-accent"
        />
        <button
          onClick={() => void handleSend()}
          disabled={sending || !input.trim()}
          className="px-4 py-1.5 bg-bc-accent text-bc-bg rounded text-sm font-medium disabled:opacity-50"
        >
          Send
        </button>
      </div>
    </>
  );
}
