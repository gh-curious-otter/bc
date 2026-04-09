import { useCallback, useState, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { api } from "../api/client";
import type { Channel } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { AgentPeekPanel } from "../components/AgentPeekPanel";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";
import { ChannelSidebar } from "../components/channels/ChannelSidebar";
import { GatewayFeed } from "../components/channels/GatewayFeed";
import { SubscriptionPanel } from "../components/channels/SubscriptionPanel";

export function Channels() {
  const { channelName: paramChannel } = useParams<{ channelName: string }>();
  const navigate = useNavigate();

  const fetcher = useCallback(() => api.listChannels(), []);
  const { data: channels, loading, error, refresh, timedOut } = usePolling(fetcher, 10000);

  const [selected, setSelected] = useState<string | null>(paramChannel ?? null);
  const [peekAgent, setPeekAgent] = useState<string | null>(null);

  useEffect(() => {
    setSelected(paramChannel ?? null);
  }, [paramChannel]);

  // Auto-select first gateway channel if none selected
  useEffect(() => {
    if (!selected && channels && channels.length > 0) {
      const gwChannel = channels.find((c) =>
        c.name.startsWith("slack:") || c.name.startsWith("telegram:") || c.name.startsWith("discord:")
      );
      if (gwChannel) {
        navigate("/channels/" + gwChannel.name, { replace: true });
      }
    }
  }, [selected, channels, navigate]);

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
          description="The server may be unavailable."
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
  const selectedChannel = channelList.find((c: Channel) => c.name === selected);

  return (
    <div className="flex h-full">
      {/* Left: Gateway sidebar */}
      <ChannelSidebar
        channels={channelList}
        selected={selected}
        onSelect={selectChannel}
      />

      {/* Center: Activity feed */}
      <div className="flex-1 flex flex-col min-w-0">
        {selected ? (
          <GatewayFeed
            channelName={selected}
            channel={selectedChannel}
            onPeekAgent={setPeekAgent}
          />
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <EmptyState
              icon="#"
              title="Select a channel"
              description="Choose a channel from the sidebar to view activity."
            />
          </div>
        )}
      </div>

      {/* Right: Subscription panel */}
      {selected && (
        <SubscriptionPanel channelName={selected} />
      )}

      {/* Agent peek overlay */}
      {peekAgent && (
        <AgentPeekPanel
          agentName={peekAgent}
          onClose={() => setPeekAgent(null)}
        />
      )}
    </div>
  );
}
