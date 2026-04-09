import { useState, useEffect, useRef, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
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

const messageVariants = {
  initial: { opacity: 0, y: -8 },
  animate: { opacity: 1, y: 0 },
  exit: { opacity: 0, y: -4 },
};

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
  const scrollRef = useRef<HTMLDivElement>(null);
  const { subscribe } = useWebSocket();

  const platform = gatewayPlatform(channelName);
  const channelLabel = channelName.includes(":")
    ? channelName.split(":").slice(1).join(":")
    : channelName;

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
    const unsub2 = subscribe("gateway.message", (event) => {
      const data = event.data as { channel?: string };
      if (data.channel === channelName) {
        void api
          .getChannelActivity(channelName, 100)
          .then((d) => setDeliveries(d ?? []))
          .catch(() => {});
      }
    });
    return () => {
      unsub1();
      unsub2();
    };
  }, [subscribe, channelName]);

  // Build delivery map by preview text
  const deliveryByPreview = new Map<string, DeliveryEntry[]>();
  for (const d of deliveries) {
    const key = d.preview ?? "";
    const list = deliveryByPreview.get(key) ?? [];
    list.push(d);
    deliveryByPreview.set(key, list);
  }

  const subAgents = new Set(subscriptions.map((s) => s.agent));

  // Reverse messages: newest first (stream style)
  const reversed = [...messages].reverse();
  const groups = groupMessages(reversed);

  return (
    <>
      {/* Header */}
      <div className="px-5 py-3 border-b border-bc-border bg-bc-surface/30">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="text-[15px] font-semibold text-bc-text">
              #{channelLabel}
            </span>
            {platform && (
              <span className="text-[9px] px-2 py-0.5 rounded-full bg-bc-border/40 text-bc-muted font-medium uppercase tracking-wider">
                {platform}
              </span>
            )}
          </div>
          <div className="flex items-center gap-3 text-[11px] text-bc-muted">
            <span>
              {messages.length} message{messages.length !== 1 ? "s" : ""}
            </span>
            {subscriptions.length > 0 && (
              <span className="text-bc-success">
                {subscriptions.length} agent
                {subscriptions.length !== 1 ? "s" : ""}
              </span>
            )}
          </div>
        </div>
        {channel?.description && (
          <p className="text-[11px] text-bc-muted/60 mt-1">
            {channel.description}
          </p>
        )}
      </div>

      {/* Stream — newest first */}
      <div className="relative flex-1">
        <div ref={scrollRef} className="absolute inset-0 overflow-auto px-5 py-3">
          {messages.length === 0 && (
            <div className="flex items-center justify-center py-16">
              <EmptyState
                icon="\u21C4"
                title="No activity yet"
                description={`Messages from ${platform ?? "this channel"} will appear here.`}
              />
            </div>
          )}

          <AnimatePresence initial={false}>
            {groups.map((group, gi) => (
              <motion.div
                key={group.messages[0]?.id ?? gi}
                variants={messageVariants}
                initial="initial"
                animate="animate"
                exit="exit"
                transition={{ duration: 0.15, ease: "easeOut" }}
                className="mb-4"
              >
                {/* Sender header */}
                <div className="flex items-baseline gap-2 mb-1">
                  <button
                    type="button"
                    onClick={() => onPeekAgent(group.sender)}
                    className="text-[13px] font-semibold text-bc-accent hover:underline cursor-pointer"
                  >
                    {group.sender}
                  </button>
                  <span className="text-[10px] text-bc-muted/40">
                    {formatTimestamp(group.timestamp)}
                  </span>
                </div>

                {/* Messages in group */}
                {group.messages.map((msg) => {
                  const icon = activityIcon(msg.content);
                  const preview = msg.content.slice(0, 120);
                  const msgDeliveries = deliveryByPreview.get(preview) ?? [];
                  const delivered = msgDeliveries.filter(
                    (d) => d.status === "delivered",
                  );
                  const failed = msgDeliveries.filter(
                    (d) => d.status === "failed",
                  );

                  return (
                    <div key={msg.id} className="group">
                      <div className="flex items-start gap-2 py-0.5 rounded-md hover:bg-bc-surface/30 transition-colors px-2 -mx-2">
                        <span
                          className={`text-[10px] mt-1 w-4 text-center shrink-0 ${icon.color}`}
                        >
                          {icon.icon}
                        </span>
                        <div className="flex-1 min-w-0">
                          <p className="text-[12.5px] text-bc-text/85 whitespace-pre-wrap break-words leading-[1.6]">
                            <MessageContent content={msg.content} />
                          </p>
                        </div>
                      </div>

                      {/* Delivery badges */}
                      {(delivered.length > 0 || failed.length > 0) && (
                        <motion.div
                          initial={{ opacity: 0, height: 0 }}
                          animate={{ opacity: 1, height: "auto" }}
                          transition={{ duration: 0.2 }}
                          className="flex items-center gap-2 ml-8 mt-0.5 mb-1"
                        >
                          {delivered.length > 0 && (
                            <span className="text-[9px] text-bc-success/60 flex items-center gap-1">
                              <span className="w-1 h-1 rounded-full bg-bc-success/60" />
                              delivered to{" "}
                              {delivered.map((d) => d.agent).join(", ")}
                            </span>
                          )}
                          {failed.length > 0 && (
                            <span className="text-[9px] text-bc-error/60 flex items-center gap-1">
                              <span className="w-1 h-1 rounded-full bg-bc-error/60" />
                              failed: {failed.map((d) => d.agent).join(", ")}
                            </span>
                          )}
                        </motion.div>
                      )}
                    </div>
                  );
                })}
              </motion.div>
            ))}
          </AnimatePresence>
        </div>
      </div>

      {/* Footer */}
      <div className="px-5 py-2 border-t border-bc-border bg-bc-surface/20">
        <p className="text-[10px] text-bc-muted/40 text-center">
          {platform ? (
            <>Activity from {platform}. Agents respond via MCP.</>
          ) : (
            <>Agents communicate via MCP tools.</>
          )}
          {subAgents.size > 0 && (
            <>
              {" "}
              · {subAgents.size} agent{subAgents.size !== 1 ? "s" : ""}{" "}
              subscribed
            </>
          )}
        </p>
      </div>
    </>
  );
}
