import { useState, useEffect, useCallback } from "react";
import { api } from "../../api/client";
import type { Agent, NotifySubscription } from "../../api/client";
import { getRoleColor } from "./messageUtils";

function AgentRow({
  agent,
  sub,
  loading,
  onSubscribe,
  onUnsubscribe,
  onToggleMention,
}: {
  agent: Agent;
  sub?: NotifySubscription;
  loading: boolean;
  onSubscribe: () => void;
  onUnsubscribe: () => void;
  onToggleMention: () => void;
}) {
  const isOnline = agent.state === "running" || agent.state === "working";
  const roleColor = getRoleColor(agent.role);

  return (
    <div className="px-3 py-2 hover:bg-bc-surface/30 transition-colors">
      <div className="flex items-center gap-2">
        <span
          className={`w-1.5 h-1.5 rounded-full shrink-0 ${
            isOnline ? "bg-bc-success" : "bg-bc-muted/30"
          }`}
        />
        <span className="text-[12px] text-bc-text truncate flex-1 font-medium">
          {agent.name}
        </span>
        <span
          className={`text-[9px] px-1.5 py-0.5 rounded ${roleColor.bg} ${roleColor.text} font-medium`}
        >
          {agent.role}
        </span>
      </div>

      {sub ? (
        <div className="flex items-center gap-2 mt-1.5 pl-3.5">
          <button
            type="button"
            onClick={onToggleMention}
            className={`text-[10px] px-1.5 py-0.5 rounded border transition-colors ${
              sub.mention_only
                ? "border-bc-accent/40 bg-bc-accent/10 text-bc-accent"
                : "border-bc-border/40 text-bc-muted hover:border-bc-border"
            }`}
            title={
              sub.mention_only
                ? "@mention only — click for all messages"
                : "All messages — click for @mention only"
            }
          >
            @ {sub.mention_only ? "mentions only" : "all messages"}
          </button>
          <button
            type="button"
            onClick={onUnsubscribe}
            disabled={loading}
            className="text-[10px] text-bc-muted/40 hover:text-bc-error transition-colors ml-auto"
          >
            Remove
          </button>
        </div>
      ) : (
        <div className="pl-3.5 mt-1">
          <button
            type="button"
            onClick={onSubscribe}
            disabled={loading}
            className="text-[10px] text-bc-muted/40 hover:text-bc-accent transition-colors"
          >
            + Subscribe
          </button>
        </div>
      )}
    </div>
  );
}

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
    } catch {
      /* */
    }
    setLoading(false);
  };

  const handleUnsubscribe = async (agentName: string) => {
    setLoading(true);
    try {
      await api.unsubscribe(channelName, agentName);
      await fetchData();
    } catch {
      /* */
    }
    setLoading(false);
  };

  const handleToggleMention = async (agentName: string, current: boolean) => {
    try {
      await api.setMentionOnly(channelName, agentName, !current);
      await fetchData();
    } catch {
      /* */
    }
  };

  const subscribedAgents = agents.filter((a) => subMap.has(a.name));
  const availableAgents = agents
    .filter((a) => !subMap.has(a.name))
    .sort((a, b) => a.name.localeCompare(b.name));

  return (
    <aside className="w-56 shrink-0 border-l border-bc-border overflow-auto flex flex-col">
      {/* Header */}
      <div className="p-3 border-b border-bc-border">
        <h3 className="text-[11px] font-semibold text-bc-muted uppercase tracking-widest">
          Agents
        </h3>
      </div>

      <div className="flex-1 overflow-auto">
        {/* Subscribed section */}
        {subscribedAgents.length > 0 && (
          <div>
            <div className="px-3 pt-2.5 pb-1">
              <span className="text-[9px] font-semibold text-bc-success uppercase tracking-widest">
                Subscribed ({subscribedAgents.length})
              </span>
            </div>
            {subscribedAgents.map((agent) => (
              <AgentRow
                key={agent.name}
                agent={agent}
                sub={subMap.get(agent.name)}
                loading={loading}
                onSubscribe={() => handleSubscribe(agent.name)}
                onUnsubscribe={() => handleUnsubscribe(agent.name)}
                onToggleMention={() =>
                  handleToggleMention(
                    agent.name,
                    subMap.get(agent.name)?.mention_only ?? false,
                  )
                }
              />
            ))}
          </div>
        )}

        {/* Divider */}
        {subscribedAgents.length > 0 && availableAgents.length > 0 && (
          <div className="mx-3 my-1 border-t border-bc-border/30" />
        )}

        {/* Available section */}
        {availableAgents.length > 0 && (
          <div>
            <div className="px-3 pt-2.5 pb-1">
              <span className="text-[9px] font-semibold text-bc-muted/50 uppercase tracking-widest">
                Available ({availableAgents.length})
              </span>
            </div>
            {availableAgents.map((agent) => (
              <AgentRow
                key={agent.name}
                agent={agent}
                loading={loading}
                onSubscribe={() => handleSubscribe(agent.name)}
                onUnsubscribe={() => handleUnsubscribe(agent.name)}
                onToggleMention={() => {}}
              />
            ))}
          </div>
        )}

        {agents.length === 0 && (
          <div className="p-4 text-center text-[11px] text-bc-muted/40">
            No agents available
          </div>
        )}
      </div>

      {/* Legend */}
      <div className="p-2 border-t border-bc-border text-[9px] text-bc-muted/30 flex items-center gap-3">
        <span className="flex items-center gap-1">
          <span className="w-1.5 h-1.5 rounded-full bg-bc-success" /> online
        </span>
        <span className="flex items-center gap-1">
          <span className="w-1.5 h-1.5 rounded-full bg-bc-muted/30" /> offline
        </span>
      </div>
    </aside>
  );
}
