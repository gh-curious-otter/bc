import { useMemo, useCallback } from "react";
import { Virtuoso } from "react-virtuoso";
import type { ChannelMessage } from "../../api/client";
import { MessageContent } from "../MessageContent";
import { AgentAvatar, RoleBadge } from "./AgentAvatar";
import {
  groupMessages,
  formatTimestamp,
  formatDayLabel,
  dateKey,
} from "./messageUtils";
import type { MessageGroup } from "./messageUtils";
import { EmptyState } from "../EmptyState";

type ListItem =
  | { type: "separator"; date: string; label: string }
  | { type: "group"; group: MessageGroup };

export function MessageList({
  messages,
  channelName,
  agentRoles,
  onPeekAgent,
  atBottomChange,
}: {
  messages: ChannelMessage[];
  channelName: string;
  agentRoles: Record<string, string>;
  onPeekAgent: (name: string) => void;
  atBottomChange?: (atBottom: boolean) => void;
}) {
  const items = useMemo(() => {
    const groups = groupMessages(messages);
    const result: ListItem[] = [];
    let lastDate = "";
    for (const group of groups) {
      const day = dateKey(group.timestamp);
      if (day !== lastDate) {
        result.push({
          type: "separator",
          date: day,
          label: formatDayLabel(group.timestamp),
        });
        lastDate = day;
      }
      result.push({ type: "group", group });
    }
    return result;
  }, [messages]);

  const renderItem = useCallback(
    (_index: number, item: ListItem) => {
      if (item.type === "separator") {
        return (
          <div className="flex items-center gap-3 py-3" role="separator">
            <div className="flex-1 h-px bg-bc-border" />
            <time className="text-[10px] text-bc-muted font-medium uppercase tracking-wider">
              {item.label}
            </time>
            <div className="flex-1 h-px bg-bc-border" />
          </div>
        );
      }

      const { group } = item;
      const firstMsg = group.messages[0];
      if (!firstMsg) return null;
      const role = agentRoles[group.sender];

      return (
        <div className="flex gap-3 py-1.5 px-1 hover:bg-bc-surface/30 rounded transition-colors" role="listitem">
          <AgentAvatar name={group.sender} role={role} />
          <div className="flex-1 min-w-0">
            <div className="flex items-baseline gap-2">
              <button
                type="button"
                onClick={() => onPeekAgent(group.sender)}
                className="font-medium text-sm text-bc-text hover:text-bc-accent hover:underline cursor-pointer focus-visible:ring-1 focus-visible:ring-bc-accent rounded"
                title={`Peek at ${group.sender}'s terminal`}
              >
                {group.sender}
              </button>
              <RoleBadge role={role} />
              <time className="text-xs text-bc-muted">
                {formatTimestamp(group.timestamp)}
              </time>
            </div>
            {group.messages.map((msg) => (
              <p
                key={msg.id}
                className="mt-0.5 text-sm whitespace-pre-wrap break-words text-bc-text"
              >
                <MessageContent content={msg.content} />
              </p>
            ))}
          </div>
        </div>
      );
    },
    [agentRoles, onPeekAgent],
  );

  if (messages.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <EmptyState
          icon="..."
          title="No messages yet"
          description={`Be the first to send a message in #${channelName}.`}
        />
      </div>
    );
  }

  return (
    <Virtuoso
      data={items}
      itemContent={renderItem}
      followOutput="smooth"
      initialTopMostItemIndex={items.length > 0 ? items.length - 1 : 0}
      atBottomStateChange={atBottomChange}
      className="flex-1"
      style={{ height: "100%" }}
      increaseViewportBy={200}
    />
  );
}
