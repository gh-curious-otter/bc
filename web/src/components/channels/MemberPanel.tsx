import { useState, useEffect } from "react";
import { api } from "../../api/client";
import type { Channel } from "../../api/client";
import { AgentAvatar, RoleBadge } from "./AgentAvatar";

function AgentStatusDot({ state }: { state?: string }) {
  const s = state ?? "unknown";
  const colors: Record<string, string> = {
    working: "bg-bc-success",
    running: "bg-bc-success",
    idle: "bg-bc-warning",
    waiting: "bg-bc-warning",
    stopped: "bg-bc-muted",
    error: "bg-bc-error",
    stuck: "bg-bc-error",
  };
  const labels: Record<string, string> = {
    working: "Online",
    running: "Online",
    idle: "Idle",
    waiting: "Waiting",
    stopped: "Offline",
    error: "Error",
    stuck: "Stuck",
  };
  return (
    <span
      className={`w-2 h-2 rounded-full shrink-0 ${colors[s] ?? "bg-bc-muted"}`}
      title={labels[s] ?? s}
    />
  );
}

export function MemberPanel({
  channel,
  agentRoles,
  agentStates,
  onChannelUpdated,
}: {
  channel: Channel;
  agentRoles: Record<string, string>;
  agentStates?: Record<string, string>;
  onChannelUpdated: () => void;
}) {
  const [addingMember, setAddingMember] = useState(false);
  const [agents, setAgents] = useState<string[]>([]);
  const [removingMember, setRemovingMember] = useState<string | null>(null);

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
      await api.addChannelMember(channel.name, agentName);
      onChannelUpdated();
    } catch {
      // silently fail
    }
    setAddingMember(false);
  };

  const handleRemoveMember = async (agentName: string) => {
    setRemovingMember(agentName);
    try {
      await api.removeChannelMember(channel.name, agentName);
      onChannelUpdated();
    } catch {
      // silently fail
    } finally {
      setRemovingMember(null);
    }
  };

  return (
    <div className="w-56 shrink-0 border-l border-bc-border overflow-auto bg-bc-surface/30">
      <div className="p-3 border-b border-bc-border">
        <h3 className="text-xs font-semibold text-bc-muted uppercase tracking-wider">
          Members ({channel.member_count})
        </h3>
        {agentStates && (
          <p className="text-[10px] text-bc-muted mt-0.5">
            {(channel.members ?? []).filter((m) => {
              const s = agentStates[m];
              return s === "working" || s === "running";
            }).length}/{channel.member_count} online
          </p>
        )}
      </div>
      <div className="p-2 space-y-1">
        {(channel.members ?? []).map((m) => (
          <div
            key={m}
            className="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-bc-surface/50 transition-colors group"
          >
            <AgentAvatar name={m} role={agentRoles[m]} size="sm" />
            <div className="flex-1 min-w-0 flex items-center gap-1.5">
              <AgentStatusDot state={agentStates?.[m]} />
              <span className="text-sm text-bc-text truncate">{m}</span>
            </div>
            <RoleBadge role={agentRoles[m]} />
            <button
              type="button"
              onClick={() => void handleRemoveMember(m)}
              disabled={removingMember === m}
              className="text-xs text-bc-muted hover:text-bc-error opacity-0 group-hover:opacity-100 transition-opacity disabled:opacity-50 focus-visible:opacity-100 focus-visible:ring-1 focus-visible:ring-bc-error rounded px-1"
              aria-label={`Remove ${m} from channel`}
              title={`Remove ${m}`}
            >
              ✕
            </button>
          </div>
        ))}
      </div>
      <div className="p-2 border-t border-bc-border">
        {addingMember ? (
          <select
            className="w-full text-xs px-2 py-1.5 rounded border border-bc-border bg-bc-bg text-bc-text focus:outline-none focus:border-bc-accent"
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
              .filter((a) => !(channel.members ?? []).includes(a))
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
            className="w-full text-xs px-2 py-1.5 rounded border border-dashed border-bc-border text-bc-muted hover:text-bc-accent hover:border-bc-accent transition-colors focus-visible:ring-1 focus-visible:ring-bc-accent"
            aria-label="Add member to channel"
          >
            + Add Member
          </button>
        )}
      </div>
    </div>
  );
}
