import { useState, useEffect, useCallback } from "react";
import { api } from "../../api/client";
import type { Agent, NotifySubscription } from "../../api/client";
import { getRoleColor } from "./messageUtils";

export function SubscriptionPanel({
  channelName,
}: {
  channelName: string;
}) {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [subscriptions, setSubscriptions] = useState<NotifySubscription[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    try {
      const [agentList, subs] = await Promise.all([
        api.listAgents(),
        api.getChannelSubscriptions(channelName),
      ]);
      setAgents(agentList ?? []);
      setSubscriptions(subs ?? []);
    } catch {
      // keep previous state
    }
  }, [channelName]);

  useEffect(() => {
    void fetchData();
    const interval = setInterval(() => void fetchData(), 8000);
    return () => clearInterval(interval);
  }, [fetchData]);

  const subMap = new Map<string, NotifySubscription>();
  for (const sub of subscriptions) subMap.set(sub.agent, sub);

  const handleSubscribe = async (agentName: string) => {
    setLoading(true);
    try {
      await api.subscribe(channelName, agentName, false);
      await fetchData();
    } catch { /* */ }
    setLoading(false);
  };

  const handleUnsubscribe = async (agentName: string) => {
    setLoading(true);
    try {
      await api.unsubscribe(channelName, agentName);
      await fetchData();
    } catch { /* */ }
    setLoading(false);
  };

  const handleToggleMention = async (agentName: string, current: boolean) => {
    try {
      await api.setMentionOnly(channelName, agentName, !current);
      await fetchData();
    } catch { /* */ }
  };

  // Sort: subscribed first, then by name
  const sorted = [...agents].sort((a, b) => {
    const aSub = subMap.has(a.name) ? 0 : 1;
    const bSub = subMap.has(b.name) ? 0 : 1;
    if (aSub !== bSub) return aSub - bSub;
    return a.name.localeCompare(b.name);
  });

  return (
    <aside className="w-56 shrink-0 border-l border-bc-border overflow-auto flex flex-col">
      <div className="p-3 border-b border-bc-border">
        <h3 className="text-[11px] font-semibold text-bc-muted uppercase tracking-widest">
          Agents
        </h3>
        <p className="text-[10px] text-bc-muted/50 mt-0.5">
          {subscriptions.length} subscribed
        </p>
      </div>

      <div className="flex-1 overflow-auto">
        {sorted.map((agent) => {
          const sub = subMap.get(agent.name);
          const isSubscribed = !!sub;
          const isOnline = agent.state === "running" || agent.state === "working";
          const roleColor = getRoleColor(agent.role);

          return (
            <div
              key={agent.name}
              className={`px-3 py-2 border-b border-bc-border/20 transition-colors ${
                isSubscribed ? "bg-bc-surface/20" : ""
              }`}
            >
              <div className="flex items-center gap-2">
                {/* Online dot */}
                <span
                  className={`w-1.5 h-1.5 rounded-full shrink-0 ${
                    isOnline ? "bg-bc-success" : "bg-bc-muted/30"
                  }`}
                />
                {/* Name */}
                <span className="text-[12px] text-bc-text truncate flex-1 font-medium">
                  {agent.name}
                </span>
                {/* Role badge */}
                <span className={`text-[9px] px-1.5 py-0.5 rounded ${roleColor.bg} ${roleColor.text} font-medium`}>
                  {agent.role}
                </span>
              </div>

              <div className="flex items-center gap-2 mt-1.5 pl-3.5">
                {isSubscribed ? (
                  <>
                    {/* @mention toggle */}
                    <button
                      type="button"
                      onClick={() => handleToggleMention(agent.name, sub.mention_only)}
                      className={`text-[10px] px-1.5 py-0.5 rounded border transition-colors ${
                        sub.mention_only
                          ? "border-bc-accent/40 bg-bc-accent/10 text-bc-accent"
                          : "border-bc-border/40 text-bc-muted hover:border-bc-border"
                      }`}
                      title={sub.mention_only ? "@mention only — click for all messages" : "All messages — click for @mention only"}
                    >
                      @mention {sub.mention_only ? "on" : "off"}
                    </button>
                    {/* Remove */}
                    <button
                      type="button"
                      onClick={() => handleUnsubscribe(agent.name)}
                      disabled={loading}
                      className="text-[10px] text-bc-muted/50 hover:text-bc-error transition-colors ml-auto"
                    >
                      Remove
                    </button>
                  </>
                ) : (
                  <button
                    type="button"
                    onClick={() => handleSubscribe(agent.name)}
                    disabled={loading}
                    className="text-[10px] text-bc-muted/50 hover:text-bc-accent transition-colors"
                  >
                    + Add
                  </button>
                )}
              </div>
            </div>
          );
        })}

        {agents.length === 0 && (
          <div className="p-4 text-center text-[11px] text-bc-muted/40">
            No agents available
          </div>
        )}
      </div>

      {/* Legend */}
      <div className="p-2 border-t border-bc-border text-[9px] text-bc-muted/40 space-y-0.5">
        <div className="flex items-center gap-1.5">
          <span className="w-1.5 h-1.5 rounded-full bg-bc-success" /> online
          <span className="w-1.5 h-1.5 rounded-full bg-bc-muted/30 ml-2" /> offline
        </div>
      </div>
    </aside>
  );
}
