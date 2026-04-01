import { useState, useEffect, useRef } from "react";
import { api } from "../../api/client";
import type { Channel, ChannelMessage } from "../../api/client";
import { useWebSocket } from "../../hooks/useWebSocket";
import { MessageContent } from "../MessageContent";
import { EmptyState } from "../EmptyState";
import { MemberPanel } from "./MemberPanel";
import { gatewayPlatform, formatTimestamp } from "./messageUtils";

/** Classify activity type from message content. */
function activityType(content: string): {
  icon: string;
  color: string;
} {
  if (
    content.includes("[shared a file]") ||
    content.includes("screenshot") ||
    content.includes("Uploaded")
  )
    return { icon: "📎", color: "text-blue-400" };
  if (content.includes("PR #") || content.includes("pull/"))
    return { icon: "⤴", color: "text-purple-400" };
  if (content.includes("merged") || content.includes("Merge"))
    return { icon: "✓", color: "text-green-400" };
  if (content.includes("review") || content.includes("LGTM"))
    return { icon: "◉", color: "text-yellow-400" };
  return { icon: "›", color: "text-bc-muted" };
}

export function GatewayFeed({
  channelName,
  channel,
  agentRoles,
  onPeekAgent,
  onChannelUpdated,
}: {
  channelName: string;
  channel?: Channel;
  agentRoles: Record<string, string>;
  onPeekAgent: (name: string) => void;
  onChannelUpdated: () => void;
}) {
  const [messages, setMessages] = useState<ChannelMessage[]>([]);
  const [showMembers, setShowMembers] = useState(false);
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
      const data = event.data as {
        channel?: string;
        message?: ChannelMessage;
      };
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

  return (
    <div className="flex flex-1 min-w-0">
      <div className="flex flex-col flex-1 min-w-0">
        {/* Header */}
        <div className="px-4 py-3 border-b border-bc-border bg-bc-surface">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <span className="font-medium">#{channelLabel}</span>
              <span className="text-[10px] px-2 py-0.5 rounded-full bg-bc-border/60 text-bc-muted font-medium uppercase tracking-wider">
                {platform} gateway
              </span>
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setShowMembers((p) => !p)}
                className={`px-2 py-1 rounded border text-xs transition-colors focus-visible:ring-1 focus-visible:ring-bc-accent ${
                  showMembers
                    ? "border-bc-accent text-bc-accent bg-bc-accent/10"
                    : "border-bc-border text-bc-muted hover:text-bc-text"
                }`}
                aria-label="Toggle members panel"
                aria-pressed={showMembers}
              >
                Members ({channel?.member_count ?? 0})
              </button>
              <span className="text-xs text-bc-muted">
                {messages.length} event{messages.length !== 1 ? "s" : ""}
              </span>
            </div>
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
                    <span
                      className={`text-xs mt-0.5 w-4 text-center shrink-0 ${activity.color}`}
                    >
                      {activity.icon}
                    </span>
                    <div className="flex-1 min-w-0 text-sm">
                      <span className="inline-flex items-baseline gap-2">
                        <button
                          type="button"
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

        {/* Read-only footer */}
        <div className="px-4 py-2.5 border-t border-bc-border bg-bc-surface/50">
          <p className="text-xs text-bc-muted text-center">
            Gateway channel — activity from{" "}
            {platform ?? "external platform"}. Messages are sent via{" "}
            {platform ?? "the platform"} directly.
          </p>
        </div>
      </div>

      {/* Member panel */}
      {showMembers && channel && (
        <MemberPanel
          channel={channel}
          agentRoles={agentRoles}
          onChannelUpdated={onChannelUpdated}
        />
      )}
    </div>
  );
}
