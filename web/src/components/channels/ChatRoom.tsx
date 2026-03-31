import { useState, useEffect } from "react";
import { api } from "../../api/client";
import type { Channel, ChannelMessage } from "../../api/client";
import { useWebSocket } from "../../hooks/useWebSocket";
import { ChatHeader } from "./ChatHeader";
import { MessageList } from "./MessageList";
import { MessageComposer } from "./MessageComposer";
import { MemberPanel } from "./MemberPanel";

export function ChatRoom({
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
  const [senderName, setSenderName] = useState("web");
  const [showMembers, setShowMembers] = useState(false);
  const [isAtBottom, setIsAtBottom] = useState(true);
  const { subscribe } = useWebSocket();

  // Fetch workspace nickname once to use as sender identity
  useEffect(() => {
    void (async () => {
      try {
        const ws = await api.getWorkspace();
        setSenderName(ws.nickname || ws.name || "web");
      } catch {
        // keep default
      }
    })();
  }, []);

  // Fetch full history on channel change
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

  // Live messages via SSE — deduplicate by ID
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

  const handleSend = async (content: string) => {
    await api.sendToChannel(channelName, content, senderName);
    // Optimistic update
    setMessages((prev) => [
      ...prev,
      {
        id: Date.now(),
        sender: senderName,
        content,
        created_at: new Date().toISOString(),
      },
    ]);
  };

  const handleDescriptionSave = async (description: string) => {
    await api.updateChannel(channelName, { description });
    onChannelUpdated();
  };

  return (
    <div className="flex flex-1 min-w-0">
      <div className="flex flex-col flex-1 min-w-0">
        <ChatHeader
          channelName={channelName}
          channel={channel}
          messageCount={messages.length}
          showMembers={showMembers}
          onToggleMembers={() => setShowMembers((p) => !p)}
          onDescriptionSave={handleDescriptionSave}
        />
        <div className="flex-1 relative" role="log" aria-live="polite" aria-label={`Messages in #${channelName}`}>
          <div className="absolute inset-0 p-4">
            <MessageList
              messages={messages}
              channelName={channelName}
              agentRoles={agentRoles}
              onPeekAgent={onPeekAgent}
              atBottomChange={setIsAtBottom}
            />
          </div>
          {!isAtBottom && messages.length > 0 && (
            <button
              type="button"
              onClick={() => {
                // Virtuoso handles scroll via followOutput, trigger by adding a dummy state change
                setIsAtBottom(true);
              }}
              className="absolute bottom-4 right-4 bg-bc-accent text-bc-bg rounded-full px-3 py-1.5 text-xs font-medium shadow-lg hover:opacity-90 transition-opacity z-10"
            >
              Jump to bottom
            </button>
          )}
        </div>
        <MessageComposer
          channelName={channelName}
          onSend={handleSend}
        />
      </div>
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
