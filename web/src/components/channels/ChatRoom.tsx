import { useState, useEffect, useCallback, useRef } from "react";
import { api } from "../../api/client";
import type { Channel, ChannelMessage } from "../../api/client";
import { useWebSocket } from "../../hooks/useWebSocket";
import { ChatHeader } from "./ChatHeader";
import { MessageList } from "./MessageList";
import { MessageComposer } from "./MessageComposer";
import { MemberPanel } from "./MemberPanel";

const INITIAL_LOAD = 30;
const PAGE_SIZE = 30;

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
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const { subscribe } = useWebSocket();
  const channelRef = useRef(channelName);

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

  // Fetch initial messages (most recent) on channel change
  useEffect(() => {
    channelRef.current = channelName;
    setMessages([]);
    setHasMore(true);
    void (async () => {
      try {
        const msgs = await api.getChannelHistory(channelName, INITIAL_LOAD);
        if (channelRef.current !== channelName) return;
        const sorted = (msgs ?? []).sort((a, b) => a.id - b.id);
        setMessages(sorted);
        setHasMore(sorted.length >= INITIAL_LOAD);
      } catch {
        setMessages([]);
      }
    })();
  }, [channelName]);

  // Load older messages when user scrolls to top
  const loadMore = useCallback(async () => {
    if (loadingMore || !hasMore) return;
    const firstMsg = messages[0];
    if (!firstMsg) return;
    setLoadingMore(true);
    try {
      const older = await api.getChannelHistory(
        channelName,
        PAGE_SIZE,
        firstMsg.id,
      );
      if (channelRef.current !== channelName) return;
      const sorted = (older ?? []).sort((a, b) => a.id - b.id);
      if (sorted.length < PAGE_SIZE) {
        setHasMore(false);
      }
      if (sorted.length > 0) {
        setMessages((prev) => {
          // Deduplicate
          const existingIds = new Set(prev.map((m) => m.id));
          const newMsgs = sorted.filter((m) => !existingIds.has(m.id));
          return [...newMsgs, ...prev];
        });
      }
    } catch {
      // silently fail
    } finally {
      setLoadingMore(false);
    }
  }, [channelName, loadingMore, hasMore, messages]);

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

  const handleFileUpload = async (file: File) => {
    const attachment = await api.uploadFile(file, channelName, senderName);
    // Send a message referencing the uploaded file
    const isImage = file.type.startsWith("image/");
    const content = isImage
      ? `[file:${attachment.id}]`
      : `📎 [${file.name}](${api.getFileUrl(attachment.id)})`;
    await api.sendToChannel(channelName, content, senderName);
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
              onLoadMore={loadMore}
              hasMore={hasMore}
              loadingMore={loadingMore}
            />
          </div>
          {!isAtBottom && messages.length > 0 && (
            <button
              type="button"
              onClick={() => {
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
          onFileUpload={handleFileUpload}
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
