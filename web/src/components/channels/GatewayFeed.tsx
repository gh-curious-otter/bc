import { useState, useEffect, useRef, useCallback } from "react";
import { api } from "../../api/client";
import type { Channel, ChannelMessage, DeliveryEntry, NotifySubscription } from "../../api/client";
import { useWebSocket } from "../../hooks/useWebSocket";
import { MessageContent } from "../MessageContent";
import { EmptyState } from "../EmptyState";
import { gatewayPlatform, formatTimestamp, groupMessages } from "./messageUtils";

function activityIcon(content: string): { icon: string; color: string } {
  if (content.includes("[shared a file]") || content.includes("screenshot"))
    return { icon: "\u{1F4CE}", color: "text-blue-400" };
  if (content.includes("PR #") || content.includes("pull/"))
    return { icon: "\u2934", color: "text-purple-400" };
  if (content.includes("merged") || content.includes("Merge"))
    return { icon: "\u2713", color: "text-green-400" };
  return { icon: "\u203A", color: "text-bc-muted/50" };
}

export function GatewayFeed({
  channelName,
  channel,
  onPeekAgent,
}: {
  channelName: string;
  channel?: Channel;
  onPeekAgent: (name: string) => void;
}) {
  const [messages, setMessages] = useState<ChannelMessage[]>([]);
  const [deliveries, setDeliveries] = useState<DeliveryEntry[]>([]);
  const [subscriptions, setSubscriptions] = useState<NotifySubscription[]>([]);
  const bottomRef = useRef<HTMLDivElement>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const { subscribe } = useWebSocket();

  const platform = gatewayPlatform(channelName);
  const channelLabel = channelName.includes(":") ? channelName.split(":").slice(1).join(":") : channelName;

  // Fetch history + delivery log + subscriptions
  const fetchAll = useCallback(async () => {
    try {
      const [msgs, activity, subs] = await Promise.all([
        api.getChannelHistory(channelName, 200),
        api.getChannelActivity(channelName, 100).catch(() => []),
        api.getChannelSubscriptions(channelName).catch(() => []),
      ]);
      setMessages(msgs ?? []);
      setDeliveries(activity ?? []);
      setSubscriptions(subs ?? []);
    } catch {
      setMessages([]);
    }
  }, [channelName]);

  useEffect(() => {
    void fetchAll();
  }, [fetchAll]);

  // Live updates via WebSocket
  useEffect(() => {
    const unsub1 = subscribe("channel.message", (event) => {
      const data = event.data as { channel?: string; message?: ChannelMessage };
      if (data.channel === channelName && data.message) {
        const msg = { ...data.message, created_at: data.message.created_at || new Date().toISOString() };
        setMessages((prev) => {
          if (prev.some((m) => m.id === msg.id)) return prev;
          return [...prev, msg];
        });
      }
    });
    const unsub2 = subscribe("gateway.message", (event) => {
      const data = event.data as { channel?: string };
      if (data.channel === channelName) {
        // Refresh delivery log on gateway activity
        void api.getChannelActivity(channelName, 100).then(d => setDeliveries(d ?? [])).catch(() => {});
      }
    });
    return () => { unsub1(); unsub2(); };
  }, [subscribe, channelName]);

  // Auto-scroll
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  // Build delivery map: messageId-ish -> agents delivered to
  // Since delivery log uses preview text, we match by approximate time
  const deliveryByPreview = new Map<string, DeliveryEntry[]>();
  for (const d of deliveries) {
    const key = d.preview ?? "";
    const list = deliveryByPreview.get(key) ?? [];
    list.push(d);
    deliveryByPreview.set(key, list);
  }

  const subAgents = new Set(subscriptions.map(s => s.agent));
  const groups = groupMessages(messages);

  return (
    <>
      {/* Header */}
      <div className="px-4 py-3 border-b border-bc-border bg-bc-surface/30">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="text-[15px] font-medium text-bc-text">#{channelLabel}</span>
            {platform && (
              <span className="text-[9px] px-2 py-0.5 rounded-full bg-bc-border/40 text-bc-muted font-medium uppercase tracking-wider">
                {platform}
              </span>
            )}
          </div>
          <div className="flex items-center gap-3 text-[11px] text-bc-muted">
            <span>{messages.length} messages</span>
            {subscriptions.length > 0 && (
              <span className="text-bc-success">{subscriptions.length} agents</span>
            )}
          </div>
        </div>
        {channel?.description && (
          <p className="text-[11px] text-bc-muted/60 mt-1">{channel.description}</p>
        )}
      </div>

      {/* Messages */}
      <div className="relative flex-1">
        <div ref={scrollRef} className="absolute inset-0 overflow-auto px-4 py-2">
          {messages.length === 0 && (
            <div className="flex items-center justify-center py-12">
              <EmptyState
                icon="\u21C4"
                title="No activity yet"
                description={`Messages from ${platform ?? "this channel"} will appear here.`}
              />
            </div>
          )}

          {groups.map((group, gi) => (
            <div key={group.messages[0]?.id ?? gi} className="mb-3">
              {/* Sender header */}
              <div className="flex items-baseline gap-2 mb-0.5">
                <button
                  type="button"
                  onClick={() => onPeekAgent(group.sender)}
                  className="text-[12px] font-semibold text-bc-accent hover:underline cursor-pointer"
                >
                  {group.sender}
                </button>
                <span className="text-[10px] text-bc-muted/50">
                  {formatTimestamp(group.timestamp)}
                </span>
              </div>

              {/* Messages in group */}
              {group.messages.map((msg) => {
                const icon = activityIcon(msg.content);
                // Find delivery entries matching this message's content preview
                const preview = msg.content.slice(0, 120);
                const msgDeliveries = deliveryByPreview.get(preview) ?? [];
                const delivered = msgDeliveries.filter(d => d.status === "delivered");
                const failed = msgDeliveries.filter(d => d.status === "failed");

                return (
                  <div key={msg.id} className="group">
                    <div className="flex items-start gap-2 py-0.5 rounded hover:bg-bc-surface/30 transition-colors px-1 -mx-1">
                      <span className={`text-[10px] mt-0.5 w-4 text-center shrink-0 ${icon.color}`}>
                        {icon.icon}
                      </span>
                      <div className="flex-1 min-w-0">
                        <p className="text-[12px] text-bc-text/80 whitespace-pre-wrap break-words leading-relaxed">
                          <MessageContent content={msg.content} />
                        </p>
                      </div>
                    </div>

                    {/* Delivery badges */}
                    {(delivered.length > 0 || failed.length > 0) && (
                      <div className="flex items-center gap-1.5 ml-6 mt-0.5 mb-1">
                        {delivered.length > 0 && (
                          <span className="text-[9px] text-bc-success/70 flex items-center gap-1">
                            <span className="opacity-50">&rarr;</span>
                            delivered to {delivered.map(d => d.agent).join(", ")}
                          </span>
                        )}
                        {failed.length > 0 && (
                          <span className="text-[9px] text-bc-error/70 flex items-center gap-1">
                            <span className="opacity-50">&times;</span>
                            failed: {failed.map(d => d.agent).join(", ")}
                          </span>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          ))}
          <div ref={bottomRef} />
        </div>
      </div>

      {/* Footer */}
      <div className="px-4 py-2 border-t border-bc-border bg-bc-surface/20">
        <p className="text-[10px] text-bc-muted/40 text-center">
          {platform ? (
            <>Activity from {platform}. Agents respond via MCP.</>
          ) : (
            <>Agents communicate via MCP tools.</>
          )}
          {subAgents.size > 0 && (
            <> &middot; {subAgents.size} agent{subAgents.size !== 1 ? "s" : ""} subscribed</>
          )}
        </p>
      </div>
    </>
  );
}
