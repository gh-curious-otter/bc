import { useState, useEffect, useRef, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { api } from "../../api/client";
import type {
  Agent,
  Channel,
  ChannelMessage,
  DeliveryEntry,
  NotifySubscription,
} from "../../api/client";
import { useWebSocket } from "../../hooks/useWebSocket";
import { MessageContent } from "../MessageContent";
import { EmptyState } from "../EmptyState";
import {
  gatewayPlatform,
  formatTimestamp,
  groupMessages,
  agentColor,
  getRoleColor,
  dateKey,
  formatDayLabel,
} from "./messageUtils";

/* ── Platform colors ─────────────────────────────────────────── */

const PLATFORM_ACCENT: Record<string, string> = {
  slack: "#E01E5A",
  telegram: "#26A5E4",
  discord: "#5865F2",
  github: "#8B949E",
};

/* ── Animation variants ──────────────────────────────────────── */

const groupIn = {
  initial: { opacity: 0, y: -6 },
  animate: { opacity: 1, y: 0 },
  exit: { opacity: 0, scale: 0.98 },
};

/* ── Component ───────────────────────────────────────────────── */

export function GatewayFeed({
  channelName,
  channel,
  onPeekAgent,
}: {
  channelName: string;
  channel?: Channel;
  onPeekAgent: (name: string) => void;
}) {
  const PAGE_SIZE = 30;
  const [messages, setMessages] = useState<ChannelMessage[]>([]);
  const [deliveries, setDeliveries] = useState<DeliveryEntry[]>([]);
  const [subscriptions, setSubscriptions] = useState<NotifySubscription[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [agentLoading, setAgentLoading] = useState(false);
  const [showAgents, setShowAgents] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const agentsPopoverRef = useRef<HTMLDivElement>(null);
  const { subscribe } = useWebSocket();

  const platform = gatewayPlatform(channelName);
  const platformColor = PLATFORM_ACCENT[platform ?? ""] ?? "var(--bc-accent)";
  const channelLabel = channelName.includes(":")
    ? channelName.split(":").slice(1).join(":")
    : channelName;

  /* ── Data fetching ─────────────────────────────────────────── */

  const fetchInitial = useCallback(async () => {
    try {
      const [msgs, activity, subs] = await Promise.all([
        api.getChannelHistory(channelName, PAGE_SIZE),
        api.getChannelActivity(channelName, 100).catch(() => []),
        api.getChannelSubscriptions(channelName).catch(() => []),
      ]);
      const m = msgs ?? [];
      setMessages(m);
      setHasMore(m.length >= PAGE_SIZE);
      setDeliveries(activity ?? []);
      setSubscriptions(subs ?? []);
    } catch {
      setMessages([]);
    }
  }, [channelName]);

  const fetchAgents = useCallback(async () => {
    try {
      const [agentList, subs] = await Promise.all([
        api.listAgents(),
        api.getChannelSubscriptions(channelName).catch(() => []),
      ]);
      setAgents(agentList ?? []);
      setSubscriptions(subs ?? []);
    } catch { /* keep previous */ }
  }, [channelName]);

  useEffect(() => {
    if (!showAgents) return;
    void fetchAgents();
    const interval = setInterval(() => void fetchAgents(), 8000);
    return () => clearInterval(interval);
  }, [showAgents, fetchAgents]);

  // Close popover on outside click
  useEffect(() => {
    if (!showAgents) return;
    const handleClick = (e: MouseEvent) => {
      if (agentsPopoverRef.current && !agentsPopoverRef.current.contains(e.target as Node)) {
        setShowAgents(false);
      }
    };
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [showAgents]);

  const handleSubscribe = async (agentName: string) => {
    setAgentLoading(true);
    try {
      await api.subscribe(channelName, agentName, false);
      await fetchAgents();
    } catch { /* */ }
    setAgentLoading(false);
  };

  const handleUnsubscribe = async (agentName: string) => {
    setAgentLoading(true);
    try {
      await api.unsubscribe(channelName, agentName);
      await fetchAgents();
    } catch { /* */ }
    setAgentLoading(false);
  };

  const handleToggleMention = async (agentName: string, current: boolean) => {
    try {
      await api.setMentionOnly(channelName, agentName, !current);
      await fetchAgents();
    } catch { /* */ }
  };

  useEffect(() => {
    void fetchInitial();
  }, [fetchInitial]);

  // Load more older messages when scrolling to bottom
  const loadMore = useCallback(async () => {
    if (loadingMore || !hasMore || messages.length === 0) return;
    setLoadingMore(true);
    try {
      const oldestId = messages[messages.length - 1]?.id;
      const older = await api.getChannelHistory(channelName, PAGE_SIZE, oldestId);
      if (!older || older.length === 0) {
        setHasMore(false);
      } else {
        setMessages((prev) => {
          const ids = new Set(prev.map((m) => m.id));
          const newMsgs = older.filter((m) => !ids.has(m.id));
          return [...prev, ...newMsgs];
        });
        setHasMore(older.length >= PAGE_SIZE);
      }
    } catch { /* */ }
    setLoadingMore(false);
  }, [channelName, messages, loadingMore, hasMore]);

  // IntersectionObserver for infinite scroll
  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) {
          void loadMore();
        }
      },
      { rootMargin: "200px" },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [loadMore]);

  /* ── Live WebSocket updates ────────────────────────────────── */

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

  /* ── Delivery matching ─────────────────────────────────────── */

  const deliveryByPreview = new Map<string, DeliveryEntry[]>();
  for (const d of deliveries) {
    const key = d.preview ?? "";
    const list = deliveryByPreview.get(key) ?? [];
    list.push(d);
    deliveryByPreview.set(key, list);
  }

  const subAgents = new Set(subscriptions.map((s) => s.agent));

  const subMap = new Map<string, NotifySubscription>();
  for (const sub of subscriptions) subMap.set(sub.agent, sub);
  const subscribedAgents = agents.filter((a) => subMap.has(a.name));
  const availableAgents = agents
    .filter((a) => !subMap.has(a.name))
    .sort((a, b) => a.name.localeCompare(b.name));

  /* ── Message grouping (newest first) ───────────────────────── */

  const reversed = [...messages].reverse();
  const groups = groupMessages(reversed);

  // Day separators
  let lastDateKey = "";

  return (
    <div className="flex flex-col h-full">
      {/* ── Header ─────────────────────────────────────────────── */}
      <div className="shrink-0 px-5 py-3.5 border-b border-bc-border/60">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            {/* Platform color bar */}
            <div
              className="w-0.5 h-5 rounded-full"
              style={{ backgroundColor: platformColor }}
            />
            <h1 className="text-[15px] font-semibold text-bc-text tracking-tight">
              {channelLabel}
            </h1>
            {platform && (
              <span
                className="text-[9px] font-semibold uppercase tracking-[0.08em] px-1.5 py-0.5 rounded"
                style={{
                  color: platformColor,
                  backgroundColor: `${platformColor}12`,
                }}
              >
                {platform}
              </span>
            )}
          </div>

          <div className="flex items-center gap-4 text-[11px]">
            <span className="text-bc-muted/60 tabular-nums">
              {messages.length}
            </span>

            {/* Agents popover trigger */}
            <div className="relative" ref={agentsPopoverRef}>
              <button
                type="button"
                onClick={() => setShowAgents((v) => !v)}
                className={`flex items-center gap-1.5 px-2 py-0.5 rounded-md border transition-all duration-150 text-[11px] ${
                  showAgents
                    ? "border-bc-accent/40 bg-bc-accent/8 text-bc-accent"
                    : subAgents.size > 0
                    ? "border-bc-success/25 bg-bc-success/5 text-bc-success/70 hover:border-bc-success/40"
                    : "border-bc-border/30 text-bc-muted/50 hover:border-bc-border/50 hover:text-bc-muted"
                }`}
              >
                {subAgents.size > 0 && (
                  <span className="w-1 h-1 rounded-full bg-bc-success animate-pulse" />
                )}
                {subAgents.size} agent{subAgents.size !== 1 ? "s" : ""}
              </button>

              {/* Agents popover */}
              {showAgents && (
                <div className="absolute right-0 top-full mt-1.5 w-64 bg-bc-bg border border-bc-border/50 rounded-lg shadow-xl z-20 max-h-[400px] overflow-auto"
                  style={{ scrollbarWidth: "thin", scrollbarColor: "rgba(255,255,255,0.04) transparent" }}
                >
                  {/* Popover header */}
                  <div className="px-3 py-2.5 border-b border-bc-border/30">
                    <h3 className="text-[11px] font-bold text-bc-muted/70 uppercase tracking-[0.12em]">
                      Agents
                    </h3>
                  </div>

                  <AnimatePresence>
                    {/* Subscribed agents */}
                    {subscribedAgents.length > 0 && (
                      <div>
                        <div className="px-3 pt-2.5 pb-1">
                          <div className="flex items-center gap-1.5">
                            <span className="w-1 h-1 rounded-full bg-bc-success" />
                            <span className="text-[9px] font-bold text-bc-success/70 uppercase tracking-[0.1em]">
                              Listening ({subscribedAgents.length})
                            </span>
                          </div>
                        </div>
                        {subscribedAgents.map((agent) => {
                          const sub = subMap.get(agent.name);
                          const isOnline = agent.state === "running" || agent.state === "working";
                          const roleColor = getRoleColor(agent.role);
                          const nameColor = agentColor(agent.name);
                          return (
                            <motion.div
                              key={agent.name}
                              layout
                              initial={{ opacity: 0, x: 8 }}
                              animate={{ opacity: 1, x: 0 }}
                              exit={{ opacity: 0, x: -8 }}
                              transition={{ duration: 0.12 }}
                              className="px-3 py-2 hover:bg-bc-surface/30 transition-colors duration-100"
                            >
                              <div className="flex items-center gap-2">
                                <span
                                  className="w-5 h-5 rounded-md flex items-center justify-center text-[9px] font-bold shrink-0"
                                  style={{ backgroundColor: `${nameColor}12`, color: nameColor }}
                                >
                                  {agent.name.charAt(0).toUpperCase()}
                                </span>
                                <div className="flex-1 min-w-0 flex items-center gap-1.5">
                                  <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${isOnline ? "bg-bc-success" : "bg-bc-muted/20"}`} />
                                  <span className="text-[12px] text-bc-text/90 truncate font-medium">{agent.name}</span>
                                </div>
                                <span className={`text-[8px] px-1.5 py-0.5 rounded-md ${roleColor.bg} ${roleColor.text} font-semibold uppercase tracking-wider shrink-0`}>
                                  {agent.role}
                                </span>
                              </div>
                              <div className="flex items-center gap-1.5 mt-1.5 ml-7">
                                <button
                                  type="button"
                                  onClick={() => handleToggleMention(agent.name, sub?.mention_only ?? false)}
                                  className={`text-[9px] px-2 py-0.5 rounded-md border transition-all duration-150 ${
                                    sub?.mention_only
                                      ? "border-bc-accent/30 bg-bc-accent/8 text-bc-accent"
                                      : "border-bc-border/30 text-bc-muted/50 hover:border-bc-border/50 hover:text-bc-muted"
                                  }`}
                                >
                                  {sub?.mention_only ? "@ mentions" : "all msgs"}
                                </button>
                                <button
                                  type="button"
                                  onClick={() => handleUnsubscribe(agent.name)}
                                  disabled={agentLoading}
                                  className="text-[9px] text-bc-muted/25 hover:text-bc-error/60 transition-colors ml-auto"
                                >
                                  remove
                                </button>
                              </div>
                            </motion.div>
                          );
                        })}
                      </div>
                    )}

                    {/* Divider */}
                    {subscribedAgents.length > 0 && availableAgents.length > 0 && (
                      <div className="mx-3 my-2 border-t border-bc-border/15" />
                    )}

                    {/* Available agents */}
                    {availableAgents.length > 0 && (
                      <div>
                        <div className="px-3 pt-2 pb-1">
                          <span className="text-[9px] font-bold text-bc-muted/30 uppercase tracking-[0.1em]">
                            Available ({availableAgents.length})
                          </span>
                        </div>
                        {availableAgents.map((agent) => {
                          const isOnline = agent.state === "running" || agent.state === "working";
                          const roleColor = getRoleColor(agent.role);
                          const nameColor = agentColor(agent.name);
                          return (
                            <motion.div
                              key={agent.name}
                              layout
                              initial={{ opacity: 0, x: 8 }}
                              animate={{ opacity: 1, x: 0 }}
                              exit={{ opacity: 0, x: -8 }}
                              transition={{ duration: 0.12 }}
                              className="px-3 py-2 hover:bg-bc-surface/15 transition-colors duration-100"
                            >
                              <div className="flex items-center gap-2">
                                <span
                                  className="w-5 h-5 rounded-md flex items-center justify-center text-[9px] font-bold shrink-0"
                                  style={{ backgroundColor: `${nameColor}12`, color: nameColor }}
                                >
                                  {agent.name.charAt(0).toUpperCase()}
                                </span>
                                <div className="flex-1 min-w-0 flex items-center gap-1.5">
                                  <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${isOnline ? "bg-bc-success" : "bg-bc-muted/20"}`} />
                                  <span className="text-[12px] text-bc-text/90 truncate font-medium">{agent.name}</span>
                                </div>
                                <span className={`text-[8px] px-1.5 py-0.5 rounded-md ${roleColor.bg} ${roleColor.text} font-semibold uppercase tracking-wider shrink-0`}>
                                  {agent.role}
                                </span>
                              </div>
                              <div className="flex items-center gap-1.5 mt-1.5 ml-7">
                                <button
                                  type="button"
                                  onClick={() => handleSubscribe(agent.name)}
                                  disabled={agentLoading}
                                  className="text-[9px] text-bc-muted/30 hover:text-bc-accent transition-colors"
                                >
                                  + subscribe
                                </button>
                              </div>
                            </motion.div>
                          );
                        })}
                      </div>
                    )}
                  </AnimatePresence>

                  {agents.length === 0 && (
                    <div className="p-6 text-center text-[11px] text-bc-muted/25">
                      No agents
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
        {channel?.description && (
          <p className="text-[11px] text-bc-muted/40 mt-1 ml-3">
            {channel.description}
          </p>
        )}
      </div>

      {/* ── Message stream ─────────────────────────────────────── */}
      <div className="flex-1 relative">
        <div
          ref={scrollRef}
          className="absolute inset-0 overflow-auto"
          style={{
            scrollbarWidth: "thin",
            scrollbarColor: "rgba(255,255,255,0.06) transparent",
          }}
        >
          <div className="px-5 py-3">
            {messages.length === 0 && (
              <div className="flex items-center justify-center py-20">
                <EmptyState
                  icon="\u21C4"
                  title="Waiting for messages"
                  description={`Activity from ${platform ?? "this channel"} will stream here in real-time.`}
                />
              </div>
            )}

            <AnimatePresence initial={false}>
              {groups.map((group, gi) => {
                const dk = dateKey(group.timestamp);
                const showDateSep = dk !== lastDateKey;
                lastDateKey = dk;

                return (
                  <motion.div
                    key={group.messages[0]?.id ?? gi}
                    variants={groupIn}
                    initial="initial"
                    animate="animate"
                    exit="exit"
                    transition={{ duration: 0.12, ease: [0.25, 0.1, 0.25, 1] }}
                  >
                    {/* Date separator */}
                    {showDateSep && (
                      <div className="flex items-center gap-3 my-4">
                        <div className="flex-1 h-px bg-bc-border/30" />
                        <span className="text-[10px] font-medium text-bc-muted/40 tracking-wide">
                          {formatDayLabel(group.timestamp)}
                        </span>
                        <div className="flex-1 h-px bg-bc-border/30" />
                      </div>
                    )}

                    {/* Message group */}
                    <div className="mb-3 group/block">
                      {/* Sender line */}
                      <div className="flex items-center gap-2 mb-0.5 pl-1">
                        {/* Agent initial */}
                        <span
                          className="w-5 h-5 rounded-md flex items-center justify-center text-[10px] font-bold shrink-0"
                          style={{
                            backgroundColor: `${agentColor(group.sender)}15`,
                            color: agentColor(group.sender),
                          }}
                        >
                          {group.sender.charAt(0).toUpperCase()}
                        </span>
                        <button
                          type="button"
                          onClick={() => onPeekAgent(group.sender)}
                          className="text-[13px] font-semibold hover:underline cursor-pointer decoration-1 underline-offset-2"
                          style={{ color: agentColor(group.sender) }}
                        >
                          {group.sender}
                        </button>
                        <span className="text-[10px] text-bc-muted/30 tabular-nums">
                          {formatTimestamp(group.timestamp)}
                        </span>
                      </div>

                      {/* Messages */}
                      {group.messages.map((msg) => {
                        const preview = msg.content.slice(0, 120);
                        const msgDeliveries =
                          deliveryByPreview.get(preview) ?? [];
                        const delivered = msgDeliveries.filter(
                          (d) => d.status === "delivered",
                        );
                        const failed = msgDeliveries.filter(
                          (d) => d.status === "failed",
                        );
                        const hasDelivery =
                          delivered.length > 0 || failed.length > 0;

                        return (
                          <div key={msg.id} className="group/msg relative">
                            <div className="py-[3px] pl-8 pr-3 rounded-md transition-colors duration-100 hover:bg-bc-surface/40">
                              <div className="text-[13px] text-bc-text/80 whitespace-pre-wrap break-words leading-[1.65] [word-break:break-word]">
                                <MessageContent content={msg.content} />
                              </div>
                            </div>

                            {/* Delivery badges — shown on hover */}
                            {hasDelivery && (
                              <div className="pl-8 pb-1 opacity-0 group-hover/msg:opacity-100 transition-opacity duration-150">
                                <div className="flex items-center gap-3 text-[9px]">
                                  {delivered.length > 0 && (
                                    <span className="flex items-center gap-1 text-bc-success/50">
                                      <svg
                                        width="10"
                                        height="10"
                                        viewBox="0 0 10 10"
                                        fill="none"
                                      >
                                        <path
                                          d="M2 5.5L4 7.5L8 3"
                                          stroke="currentColor"
                                          strokeWidth="1.2"
                                          strokeLinecap="round"
                                          strokeLinejoin="round"
                                        />
                                      </svg>
                                      {delivered.map((d) => d.agent).join(", ")}
                                    </span>
                                  )}
                                  {failed.length > 0 && (
                                    <span className="flex items-center gap-1 text-bc-error/50">
                                      <svg
                                        width="10"
                                        height="10"
                                        viewBox="0 0 10 10"
                                        fill="none"
                                      >
                                        <path
                                          d="M3 3L7 7M7 3L3 7"
                                          stroke="currentColor"
                                          strokeWidth="1.2"
                                          strokeLinecap="round"
                                        />
                                      </svg>
                                      {failed.map((d) => d.agent).join(", ")}
                                    </span>
                                  )}
                                </div>
                              </div>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  </motion.div>
                );
              })}
            </AnimatePresence>

            {/* Load more sentinel */}
            {hasMore && (
              <div ref={sentinelRef} className="py-4 text-center">
                {loadingMore ? (
                  <span className="text-[10px] text-bc-muted/30">Loading older messages...</span>
                ) : (
                  <span className="text-[10px] text-bc-muted/20">Scroll for more</span>
                )}
              </div>
            )}
            {!hasMore && messages.length > 0 && (
              <div className="py-4 text-center text-[10px] text-bc-muted/15">
                Beginning of channel history
              </div>
            )}
          </div>
        </div>
      </div>

      {/* ── Footer ─────────────────────────────────────────────── */}
      <div className="shrink-0 px-5 py-2 border-t border-bc-border/30">
        <div className="flex items-center justify-between text-[10px] text-bc-muted/30">
          <span>
            {platform
              ? `${platform} gateway`
              : "bc channel"}{" "}
            · agents respond via MCP
          </span>
          {subAgents.size > 0 && (
            <span>
              {subAgents.size} agent{subAgents.size !== 1 ? "s" : ""} subscribed
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
