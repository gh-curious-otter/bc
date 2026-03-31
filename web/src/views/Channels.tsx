import { useCallback, useState, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { api } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { useAgentRoles } from "../hooks/useAgentRoles";
import { AgentPeekPanel } from "../components/AgentPeekPanel";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";
import { ChannelSidebar } from "../components/channels/ChannelSidebar";
import { ChatRoom } from "../components/channels/ChatRoom";
import { GatewayFeed } from "../components/channels/GatewayFeed";
import { isGatewayChannel } from "../components/channels/messageUtils";

export function Channels() {
  const { channelName: paramChannel } = useParams<{ channelName: string }>();
  const navigate = useNavigate();
  const fetcher = useCallback(async () => {
    const res = await api.listChannels();
    return res;
  }, []);
  const {
    data: channels,
    loading,
    error,
    refresh,
    timedOut,
  } = usePolling(fetcher, 10000);
  const { roleMap } = useAgentRoles();
  const [selected, setSelected] = useState<string | null>(paramChannel ?? null);
  const [peekAgent, setPeekAgent] = useState<string | null>(null);

  useEffect(() => {
    setSelected(paramChannel ?? null);
  }, [paramChannel]);

  const selectChannel = (name: string) => {
    navigate("/channels/" + name);
  };

  if (loading && !channels) {
    return (
      <div className="p-6 space-y-4">
        <div className="h-6 w-28 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="text" rows={5} />
      </div>
    );
  }
  if (timedOut && !channels) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Channels took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !channels) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load channels"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }

  const channelList = channels ?? [];
  const selectedChannel = channelList.find((c) => c.name === selected);

  return (
    <div className="flex h-full">
      <ChannelSidebar
        channels={channelList}
        selected={selected}
        onSelect={selectChannel}
      />
      <div className="flex-1 flex flex-col min-w-0">
        {selected ? (
          isGatewayChannel(selected) ? (
            <GatewayFeed
              channelName={selected}
              channel={selectedChannel}
              onPeekAgent={setPeekAgent}
            />
          ) : (
            <ChatRoom
              channelName={selected}
              channel={selectedChannel}
              agentRoles={roleMap}
              onPeekAgent={setPeekAgent}
              onChannelUpdated={refresh}
            />
          )
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <EmptyState
              icon="#"
              title="Select a channel"
              description="Choose a channel from the sidebar to view messages."
            />
          </div>
        )}
      </div>
      {peekAgent && (
        <AgentPeekPanel
          agentName={peekAgent}
          onClose={() => setPeekAgent(null)}
        />
      )}
    </div>
  );
}
