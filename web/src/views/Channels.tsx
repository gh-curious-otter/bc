import { useCallback, useState, useEffect, useRef } from 'react';
import { api } from '../api/client';
import type { ChannelMessage } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { useWebSocket } from '../hooks/useWebSocket';
import { AgentPeekPanel } from '../components/AgentPeekPanel';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import { EmptyState } from '../components/EmptyState';

export function Channels() {
  const fetcher = useCallback(async () => {
    const res = await api.listChannels();
    return res;
  }, []);
  const { data: channels, loading, error, refresh, timedOut } = usePolling(fetcher, 10000);
  const [selected, setSelected] = useState<string | null>(null);
  const [peekAgent, setPeekAgent] = useState<string | null>(null);

  if (loading && !channels) {
    return (
      <div className="p-6 space-y-4">
        <div className="h-6 w-28 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="text" rows={5} />
      </div>
    );
  }
  if (timedOut && !channels) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Channels took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !channels) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load channels"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }

  return (
    <div className="flex h-full">
      <div className="w-56 shrink-0 border-r border-bc-border overflow-auto">
        <div className="p-3 border-b border-bc-border">
          <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">Channels</h2>
        </div>
        {(channels ?? []).length === 0 ? (
          <div className="p-4">
            <EmptyState
              icon="#"
              title="No channels"
              description="Channels are created automatically when agents communicate."
            />
          </div>
        ) : (
          (channels ?? []).map((ch) => (
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
          ))
        )}
      </div>
      <div className="flex-1 flex flex-col min-w-0">
        {selected ? (
          <ChatRoom channelName={selected} onPeekAgent={setPeekAgent} />
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <EmptyState
              icon="#"
              title="Select a channel"
              description="Choose a channel from the sidebar to view messages."
            />
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
        {messages.length === 0 && (
          <div className="flex-1 flex items-center justify-center py-8">
            <EmptyState
              icon="..."
              title="No messages yet"
              description={`Be the first to send a message in #${channelName}.`}
            />
          </div>
        )}
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
