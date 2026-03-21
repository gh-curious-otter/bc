import { useCallback, useState, useEffect, useRef } from 'react';
import { api } from '../api/client';
import type { ChannelMessage } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { useWebSocket } from '../hooks/useWebSocket';
import { AgentPeekPanel } from '../components/AgentPeekPanel';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import { EmptyState } from '../components/EmptyState';
import { MessageContent } from '../components/MessageContent';

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

/** Group consecutive messages from the same sender. */
interface MessageGroup {
  sender: string;
  timestamp: string;
  messages: ChannelMessage[];
}

function groupMessages(msgs: ChannelMessage[]): MessageGroup[] {
  const groups: MessageGroup[] = [];
  for (const msg of msgs) {
    const last = groups[groups.length - 1];
    if (last && last.sender === msg.sender) {
      last.messages.push(msg);
    } else {
      groups.push({
        sender: msg.sender,
        timestamp: msg.created_at,
        messages: [msg],
      });
    }
  }
  return groups;
}

function formatTimestamp(iso: string): string {
  const d = new Date(iso);
  const now = new Date();
  const isToday =
    d.getFullYear() === now.getFullYear() &&
    d.getMonth() === now.getMonth() &&
    d.getDate() === now.getDate();
  if (isToday) {
    return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
  }
  return d.toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function ChatRoom({ channelName, onPeekAgent }: { channelName: string; onPeekAgent: (name: string) => void }) {
  const [messages, setMessages] = useState<ChannelMessage[]>([]);
  const [input, setInput] = useState('');
  const [sending, setSending] = useState(false);
  const [isNearBottom, setIsNearBottom] = useState(true);
  const bottomRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const { subscribe } = useWebSocket();
  // Track whether the initial fetch for the current channel has completed
  // so we can force-scroll to bottom on channel switch.
  const channelLoadedRef = useRef<string | null>(null);

  // Fetch full history on channel change
  useEffect(() => {
    channelLoadedRef.current = null;
    setMessages([]);
    void (async () => {
      try {
        const msgs = await api.getChannelHistory(channelName, 500);
        setMessages(msgs ?? []);
      } catch {
        setMessages([]);
      }
      channelLoadedRef.current = channelName;
    })();
  }, [channelName]);

  // Live messages via SSE -- deduplicate by ID
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

  // Track scroll position
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;
    const handleScroll = () => {
      const nearBottom =
        container.scrollHeight - container.scrollTop - container.clientHeight < 100;
      setIsNearBottom(nearBottom);
    };
    container.addEventListener('scroll', handleScroll, { passive: true });
    return () => container.removeEventListener('scroll', handleScroll);
  }, []);

  // Auto-scroll: always on channel switch load, otherwise only when near bottom
  useEffect(() => {
    if (messages.length === 0) return;
    const justLoaded = channelLoadedRef.current === channelName;
    if (justLoaded || isNearBottom) {
      bottomRef.current?.scrollIntoView({ behavior: justLoaded ? 'auto' : 'smooth' });
    }
    // After the initial load scroll, clear the flag so subsequent messages
    // use the near-bottom heuristic only.
    if (justLoaded) {
      channelLoadedRef.current = null;
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [messages]);

  const scrollToBottom = () => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  const handleSend = async () => {
    if (!input.trim()) return;
    setSending(true);
    try {
      await api.sendToChannel(channelName, input);
      setInput('');
    } finally {
      setSending(false);
    }
  };

  const groups = groupMessages(messages);

  return (
    <>
      <div className="px-4 py-2 border-b border-bc-border bg-bc-surface">
        <span className="font-medium">#{channelName}</span>
        <span className="ml-2 text-xs text-bc-muted">
          {messages.length} message{messages.length !== 1 ? 's' : ''}
        </span>
      </div>
      <div className="relative flex-1">
        <div ref={scrollContainerRef} className="absolute inset-0 overflow-auto p-4 space-y-3">
          {messages.length === 0 && (
            <div className="flex-1 flex items-center justify-center py-8">
              <EmptyState
                icon="..."
                title="No messages yet"
                description={`Be the first to send a message in #${channelName}.`}
              />
            </div>
          )}
          {groups.map((group) => {
            const firstMsg = group.messages[0]!;
            return (
            <div key={firstMsg.id} className="text-sm">
              <div className="flex items-baseline gap-2">
                <button
                  onClick={() => onPeekAgent(group.sender)}
                  className="font-medium text-bc-accent hover:underline cursor-pointer"
                  title={`Peek at ${group.sender}'s terminal`}
                >
                  {group.sender}
                </button>
                <span className="text-xs text-bc-muted">{formatTimestamp(group.timestamp)}</span>
              </div>
              {group.messages.map((msg) => (
                <p key={msg.id} className="mt-0.5 pl-0 whitespace-pre-wrap break-words">
                  <MessageContent content={msg.content} />
                </p>
              ))}
            </div>
            );
          })}
          <div ref={bottomRef} />
        </div>
        {!isNearBottom && messages.length > 0 && (
          <button
            onClick={scrollToBottom}
            className="absolute bottom-4 right-4 bg-bc-accent text-bc-bg rounded-full px-3 py-1.5 text-xs font-medium shadow-lg hover:opacity-90 transition-opacity"
          >
            Jump to bottom
          </button>
        )}
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
