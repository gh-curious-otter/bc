import { useCallback, useState, useEffect, useRef } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { api } from "../api/client";
import type { Channel, ChannelMessage } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { useWebSocket } from "../hooks/useWebSocket";
import { AgentPeekPanel } from "../components/AgentPeekPanel";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";
import { MessageContent } from "../components/MessageContent";

/** Gateway channels are bridges to external platforms — read-only activity feeds. */
const GATEWAY_PREFIXES = ["slack:", "telegram:", "discord:"];
function isGatewayChannel(name: string): boolean {
  return GATEWAY_PREFIXES.some((p) => name.startsWith(p));
}

/** Extract platform name from gateway channel for display. */
function gatewayPlatform(name: string): string | null {
  for (const p of GATEWAY_PREFIXES) {
    if (name.startsWith(p)) return p.slice(0, -1);
  }
  return null;
}

export function Channels() {
  const { channelName: paramChannel } = useParams<{ channelName: string }>();
  const navigate = useNavigate();
  const fetcher = useCallback(async () => {
    const res = await api.listChannels();
    return res;
  }, []);
  const {
    data: channels,
    loading,
    error,
    refresh,
    timedOut,
  } = usePolling(fetcher, 10000);
  const [selected, setSelected] = useState<string | null>(paramChannel ?? null);
  const [peekAgent, setPeekAgent] = useState<string | null>(null);

  // Sync selected state when URL param changes
  useEffect(() => {
    setSelected(paramChannel ?? null);
  }, [paramChannel]);

  const selectChannel = (name: string) => {
    navigate("/channels/" + name);
  };

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
          <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">
            Channels
          </h2>
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
          <>
            {/* bc-native channels */}
            {(channels ?? []).filter((ch) => !isGatewayChannel(ch.name)).length > 0 && (
              <div className="px-3 pt-3 pb-1">
                <span className="text-[10px] font-semibold text-bc-muted uppercase tracking-widest">Channels</span>
              </div>
            )}
            {(channels ?? [])
              .filter((ch) => !isGatewayChannel(ch.name))
              .map((ch) => (
                <button
                  key={ch.name}
                  onClick={() => selectChannel(ch.name)}
                  className={`w-full text-left px-3 py-2 text-sm border-b border-bc-border/30 ${
                    selected === ch.name
                      ? "bg-bc-accent/10 text-bc-accent"
                      : "text-bc-text hover:bg-bc-surface"
                  }`}
                >
                  <span className="font-medium">#{ch.name}</span>
                  <span className="ml-2 text-xs text-bc-muted">
                    ({ch.member_count})
                  </span>
                </button>
              ))}
            {/* Gateway channels */}
            {(channels ?? []).filter((ch) => isGatewayChannel(ch.name)).length > 0 && (
              <div className="px-3 pt-4 pb-1">
                <span className="text-[10px] font-semibold text-bc-muted uppercase tracking-widest">Gateways</span>
              </div>
            )}
            {(channels ?? [])
              .filter((ch) => isGatewayChannel(ch.name))
              .map((ch) => {
                const platform = gatewayPlatform(ch.name);
                return (
                  <button
                    key={ch.name}
                    onClick={() => selectChannel(ch.name)}
                    className={`w-full text-left px-3 py-2 text-sm border-b border-bc-border/30 flex items-center gap-2 ${
                      selected === ch.name
                        ? "bg-bc-accent/10 text-bc-accent"
                        : "text-bc-text hover:bg-bc-surface"
                    }`}
                  >
                    <span className="font-medium">#{ch.name}</span>
                    <span className="ml-auto text-[10px] px-1.5 py-0.5 rounded bg-bc-border/60 text-bc-muted">
                      {platform}
                    </span>
                  </button>
                );
              })}
          </>
        )}
      </div>
      <div className="flex-1 flex flex-col min-w-0">
        {selected ? (
          isGatewayChannel(selected) ? (
            <GatewayFeed
              channelName={selected}
              channel={(channels ?? []).find((c) => c.name === selected)}
              onPeekAgent={setPeekAgent}
            />
          ) : (
            <ChatRoom
              channelName={selected}
              channel={(channels ?? []).find((c) => c.name === selected)}
              onPeekAgent={setPeekAgent}
              onChannelUpdated={refresh}
            />
          )
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
        <AgentPeekPanel
          agentName={peekAgent}
          onClose={() => setPeekAgent(null)}
        />
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
    return d.toLocaleTimeString(undefined, {
      hour: "2-digit",
      minute: "2-digit",
    });
  }
  return d.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function ChatRoom({
  channelName,
  channel,
  onPeekAgent,
  onChannelUpdated,
}: {
  channelName: string;
  channel?: Channel;
  onPeekAgent: (name: string) => void;
  onChannelUpdated: () => void;
}) {
  const [messages, setMessages] = useState<ChannelMessage[]>([]);
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const [isNearBottom, setIsNearBottom] = useState(true);
  const [senderName, setSenderName] = useState("web");
  const [showMembers, setShowMembers] = useState(false);
  const [addingMember, setAddingMember] = useState(false);
  const [agents, setAgents] = useState<string[]>([]);
  const [editingDesc, setEditingDesc] = useState(false);
  const [descDraft, setDescDraft] = useState("");
  const [savingDesc, setSavingDesc] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const { subscribe } = useWebSocket();

  // Fetch agents list for the add-member dropdown
  useEffect(() => {
    if (!addingMember) return;
    void (async () => {
      try {
        const list = await api.listAgents();
        setAgents(list.map((a) => a.name));
      } catch {
        setAgents([]);
      }
    })();
  }, [addingMember]);

  const handleAddMember = async (agentName: string) => {
    try {
      await api.addChannelMember(channelName, agentName);
      onChannelUpdated();
    } catch {
      // silently fail
    }
    setAddingMember(false);
  };

  const handleSaveDescription = async () => {
    setSavingDesc(true);
    try {
      await api.updateChannel(channelName, { description: descDraft });
      onChannelUpdated();
      setEditingDesc(false);
    } catch {
      // keep editing open
    } finally {
      setSavingDesc(false);
    }
  };

  // Fetch workspace nickname once to use as sender identity
  useEffect(() => {
    void (async () => {
      try {
        const ws = await api.getWorkspace();
        setSenderName(ws.nickname || ws.name || "web");
      } catch {
        // keep default 'web'
      }
    })();
  }, []);
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
    return subscribe("channel.message", (event) => {
      const data = event.data as { channel?: string; message?: ChannelMessage };
      if (data.channel === channelName && data.message) {
        const msg = {
          ...data.message,
          created_at: data.message.created_at || new Date().toISOString(),
        };
        setMessages((prev) => {
          if (prev.some((m) => m.id === msg.id)) return prev;
          return [...prev, msg];
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
        container.scrollHeight - container.scrollTop - container.clientHeight <
        100;
      setIsNearBottom(nearBottom);
    };
    container.addEventListener("scroll", handleScroll, { passive: true });
    return () => container.removeEventListener("scroll", handleScroll);
  }, []);

  // Auto-scroll: always on channel switch load, otherwise only when near bottom
  useEffect(() => {
    if (messages.length === 0) return;
    const justLoaded = channelLoadedRef.current === channelName;
    if (justLoaded || isNearBottom) {
      bottomRef.current?.scrollIntoView({
        behavior: justLoaded ? "auto" : "smooth",
      });
    }
    // After the initial load scroll, clear the flag so subsequent messages
    // use the near-bottom heuristic only.
    if (justLoaded) {
      channelLoadedRef.current = null;
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [messages]);

  const scrollToBottom = () => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  const autoGrow = useCallback(() => {
    const ta = textareaRef.current;
    if (!ta) return;
    ta.style.height = "auto";
    // Cap at ~4 rows (6rem = 96px)
    ta.style.height = Math.min(ta.scrollHeight, 96) + "px";
  }, []);

  const handleSend = async () => {
    if (!input.trim()) return;
    const content = input;
    setSending(true);
    setInput("");
    // Reset textarea height after clearing
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
    }
    try {
      await api.sendToChannel(channelName, content, senderName);
      // Optimistically add the message to local state so it appears immediately
      // without waiting for SSE delivery
      setMessages((prev) => [
        ...prev,
        {
          id: Date.now(),
          sender: senderName,
          content,
          created_at: new Date().toISOString(),
          channel: channelName,
          type: "text",
        } as ChannelMessage,
      ]);
    } catch {
      // Restore input on failure
      setInput(content);
    } finally {
      setSending(false);
    }
  };

  const groups = groupMessages(messages);

  return (
    <>
      <div className="px-4 py-2 border-b border-bc-border bg-bc-surface space-y-1">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="font-medium">#{channelName}</span>
            <span className="text-xs text-bc-muted">
              {messages.length} message{messages.length !== 1 ? "s" : ""}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setShowMembers((p) => !p)}
              className="px-2 py-1 rounded border border-bc-border text-xs text-bc-muted hover:text-bc-text transition-colors"
              aria-label="Toggle members panel"
            >
              Members ({channel?.member_count ?? 0})
            </button>
            <button
              type="button"
              onClick={() => {
                setDescDraft(channel?.description ?? "");
                setEditingDesc(true);
              }}
              className="px-2 py-1 rounded border border-bc-border text-xs text-bc-muted hover:text-bc-text transition-colors"
              aria-label="Edit channel description"
            >
              Edit
            </button>
          </div>
        </div>
        {editingDesc && (
          <div className="flex items-center gap-2">
            <input
              type="text"
              value={descDraft}
              onChange={(e) => setDescDraft(e.target.value)}
              placeholder="Channel description"
              className="flex-1 px-2 py-1 rounded border border-bc-border bg-bc-bg text-sm text-bc-text focus:outline-none focus:border-bc-accent"
              onKeyDown={(e) => {
                if (e.key === "Enter") void handleSaveDescription();
                if (e.key === "Escape") setEditingDesc(false);
              }}
              aria-label="Channel description"
            />
            <button
              type="button"
              onClick={() => void handleSaveDescription()}
              disabled={savingDesc}
              className="px-2 py-1 rounded bg-bc-accent text-bc-bg text-xs font-medium disabled:opacity-50"
            >
              {savingDesc ? "Saving..." : "Save"}
            </button>
            <button
              type="button"
              onClick={() => setEditingDesc(false)}
              className="px-2 py-1 rounded border border-bc-border text-xs text-bc-muted hover:text-bc-text"
            >
              Cancel
            </button>
          </div>
        )}
        {!editingDesc && channel?.description && (
          <p className="text-xs text-bc-muted">{channel.description}</p>
        )}
        {showMembers && (
          <div className="flex flex-wrap items-center gap-2 pt-1">
            {(channel?.members ?? []).map((m) => (
              <span
                key={m}
                className="text-xs px-2 py-0.5 rounded bg-bc-accent/10 text-bc-accent"
              >
                {m}
              </span>
            ))}
            {addingMember ? (
              <select
                className="text-xs px-2 py-1 rounded border border-bc-border bg-bc-bg text-bc-text focus:outline-none"
                onChange={(e) => {
                  if (e.target.value) void handleAddMember(e.target.value);
                }}
                defaultValue=""
                aria-label="Select agent to add"
              >
                <option value="" disabled>
                  Select agent...
                </option>
                {agents
                  .filter((a) => !(channel?.members ?? []).includes(a))
                  .map((a) => (
                    <option key={a} value={a}>
                      {a}
                    </option>
                  ))}
              </select>
            ) : (
              <button
                type="button"
                onClick={() => setAddingMember(true)}
                className="text-xs px-2 py-0.5 rounded border border-dashed border-bc-border text-bc-muted hover:text-bc-accent hover:border-bc-accent transition-colors"
                aria-label="Add member to channel"
              >
                + Add Member
              </button>
            )}
          </div>
        )}
      </div>
      <div className="relative flex-1">
        <div
          ref={scrollContainerRef}
          className="absolute inset-0 overflow-auto p-4 space-y-3"
        >
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
                  <span className="text-xs text-bc-muted">
                    {formatTimestamp(group.timestamp)}
                  </span>
                </div>
                {group.messages.map((msg) => (
                  <p
                    key={msg.id}
                    className="mt-0.5 pl-0 whitespace-pre-wrap break-words"
                  >
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
      <div className="p-3 border-t border-bc-border flex gap-2 items-end">
        <textarea
          ref={textareaRef}
          rows={1}
          value={input}
          onChange={(e) => {
            setInput(e.target.value);
            autoGrow();
          }}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              void handleSend();
            }
            if (e.key === "Escape") {
              setInput("");
              if (textareaRef.current) {
                textareaRef.current.style.height = "auto";
                textareaRef.current.blur();
              }
            }
          }}
          placeholder={`Message #${channelName}...`}
          className="flex-1 bg-bc-bg border border-bc-border rounded px-3 py-1.5 text-sm focus:outline-none focus:border-bc-accent resize-none"
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

/** Read-only activity feed for gateway channels (slack:*, telegram:*, discord:*). */
function GatewayFeed({
  channelName,
  channel,
  onPeekAgent,
}: {
  channelName: string;
  channel?: Channel;
  onPeekAgent: (name: string) => void;
}) {
  const [messages, setMessages] = useState<ChannelMessage[]>([]);
  const bottomRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const { subscribe } = useWebSocket();

  const platform = gatewayPlatform(channelName);
  const channelLabel = channelName.split(":").slice(1).join(":");

  // Fetch history
  useEffect(() => {
    setMessages([]);
    void (async () => {
      try {
        const msgs = await api.getChannelHistory(channelName, 500);
        setMessages(msgs ?? []);
      } catch {
        setMessages([]);
      }
    })();
  }, [channelName]);

  // Live updates
  useEffect(() => {
    return subscribe("channel.message", (event) => {
      const data = event.data as { channel?: string; message?: ChannelMessage };
      if (data.channel === channelName && data.message) {
        const msg = {
          ...data.message,
          created_at: data.message.created_at || new Date().toISOString(),
        };
        setMessages((prev) => {
          if (prev.some((m) => m.id === msg.id)) return prev;
          return [...prev, msg];
        });
      }
    });
  }, [subscribe, channelName]);

  // Auto-scroll on new messages
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  /** Classify activity type from message content. */
  const activityType = (content: string): { icon: string; label: string; color: string } => {
    if (content.includes("[shared a file]") || content.includes("screenshot") || content.includes("Uploaded"))
      return { icon: "📎", label: "file", color: "text-blue-400" };
    if (content.includes("PR #") || content.includes("pull/"))
      return { icon: "⤴", label: "pr", color: "text-purple-400" };
    if (content.includes("merged") || content.includes("Merge"))
      return { icon: "✓", label: "merged", color: "text-green-400" };
    if (content.includes("review") || content.includes("LGTM"))
      return { icon: "◉", label: "review", color: "text-yellow-400" };
    return { icon: "›", label: "message", color: "text-bc-muted" };
  };

  return (
    <>
      {/* Header */}
      <div className="px-4 py-3 border-b border-bc-border bg-bc-surface">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="font-medium">#{channelLabel}</span>
            <span className="text-[10px] px-2 py-0.5 rounded-full bg-bc-border/60 text-bc-muted font-medium uppercase tracking-wider">
              {platform} gateway
            </span>
          </div>
          <span className="text-xs text-bc-muted">
            {messages.length} event{messages.length !== 1 ? "s" : ""}
          </span>
        </div>
        {channel?.description && (
          <p className="text-xs text-bc-muted mt-1">{channel.description}</p>
        )}
      </div>

      {/* Activity feed */}
      <div className="relative flex-1">
        <div
          ref={scrollContainerRef}
          className="absolute inset-0 overflow-auto p-4"
        >
          {messages.length === 0 && (
            <div className="flex items-center justify-center py-8">
              <EmptyState
                icon="⇄"
                title="No gateway activity"
                description={`Messages from ${platform ?? "the external platform"} will appear here.`}
              />
            </div>
          )}
          <div className="space-y-1">
            {messages.map((msg) => {
              const activity = activityType(msg.content);
              return (
                <div
                  key={msg.id}
                  className="flex items-start gap-3 py-1.5 px-2 rounded hover:bg-bc-surface/50 transition-colors group"
                >
                  <span className={`text-xs mt-0.5 w-4 text-center shrink-0 ${activity.color}`}>
                    {activity.icon}
                  </span>
                  <div className="flex-1 min-w-0 text-sm">
                    <span className="inline-flex items-baseline gap-2">
                      <button
                        onClick={() => onPeekAgent(msg.sender)}
                        className="font-medium text-bc-accent hover:underline cursor-pointer text-xs"
                        title={`Peek at ${msg.sender}`}
                      >
                        {msg.sender}
                      </button>
                      <span className="text-[11px] text-bc-muted">
                        {formatTimestamp(msg.created_at)}
                      </span>
                    </span>
                    <p className="text-xs text-bc-text/80 mt-0.5 whitespace-pre-wrap break-words line-clamp-3 group-hover:line-clamp-none">
                      <MessageContent content={msg.content} />
                    </p>
                  </div>
                </div>
              );
            })}
          </div>
          <div ref={bottomRef} />
        </div>
      </div>

      {/* Read-only footer — no input */}
      <div className="px-4 py-2.5 border-t border-bc-border bg-bc-surface/50">
        <p className="text-xs text-bc-muted text-center">
          Gateway channel — activity from {platform ?? "external platform"}. Messages are sent via {platform ?? "the platform"} directly.
        </p>
      </div>
    </>
  );
}
