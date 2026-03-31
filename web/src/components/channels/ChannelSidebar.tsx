import { useState } from "react";
import type { Channel } from "../../api/client";
import { channelPlatform } from "./messageUtils";
import { EmptyState } from "../EmptyState";

interface PlatformBucket {
  key: string;
  label: string;
  channels: Channel[];
}

const PLATFORM_CONFIG: Record<string, { label: string; badgeClass: string }> = {
  internal: { label: "Channels", badgeClass: "" },
  slack: { label: "Slack", badgeClass: "bg-bc-accent/10 text-bc-accent" },
  telegram: { label: "Telegram", badgeClass: "bg-blue-500/10 text-blue-400" },
  discord: { label: "Discord", badgeClass: "bg-indigo-500/10 text-indigo-400" },
};

function getPlatformConfig(key: string) {
  return PLATFORM_CONFIG[key] ?? { label: key, badgeClass: "bg-bc-border/60 text-bc-muted" };
}

/** Strip platform prefix from channel name for display. */
function displayName(name: string): string {
  const colonIdx = name.indexOf(":");
  if (colonIdx > 0 && ["slack", "telegram", "discord"].includes(name.slice(0, colonIdx))) {
    return name.slice(colonIdx + 1);
  }
  return name;
}

export function ChannelSidebar({
  channels,
  selected,
  onSelect,
}: {
  channels: Channel[];
  selected: string | null;
  onSelect: (name: string) => void;
}) {
  const [collapsed, setCollapsed] = useState<Set<string>>(() => {
    try {
      const stored = localStorage.getItem("bc-channels-collapsed");
      return stored ? new Set(JSON.parse(stored) as string[]) : new Set<string>();
    } catch {
      return new Set<string>();
    }
  });

  const toggleCollapse = (key: string) => {
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      try {
        localStorage.setItem("bc-channels-collapsed", JSON.stringify([...next]));
      } catch {
        // ignore
      }
      return next;
    });
  };

  // Group channels by platform
  const bucketMap = new Map<string, Channel[]>();
  for (const ch of channels) {
    const platform = channelPlatform(ch.name);
    const list = bucketMap.get(platform) ?? [];
    list.push(ch);
    bucketMap.set(platform, list);
  }

  // Sort buckets: internal first, then alphabetically
  const buckets: PlatformBucket[] = [];
  const internalChannels = bucketMap.get("internal");
  if (internalChannels && internalChannels.length > 0) {
    buckets.push({ key: "internal", label: "Channels", channels: internalChannels });
  }
  for (const [key, chs] of bucketMap) {
    if (key !== "internal") {
      const config = getPlatformConfig(key);
      buckets.push({ key, label: config.label, channels: chs });
    }
  }

  if (channels.length === 0) {
    return (
      <div className="w-64 shrink-0 border-r border-bc-border overflow-auto">
        <div className="p-3 border-b border-bc-border">
          <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">
            Channels
          </h2>
        </div>
        <div className="p-4">
          <EmptyState
            icon="#"
            title="No channels"
            description="Channels are created automatically when agents communicate."
          />
        </div>
      </div>
    );
  }

  return (
    <nav className="w-64 shrink-0 border-r border-bc-border overflow-auto" aria-label="Channel list">
      <div className="p-3 border-b border-bc-border">
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">
          Channels
        </h2>
      </div>
      {buckets.map((bucket) => {
        const config = getPlatformConfig(bucket.key);
        const isCollapsed = collapsed.has(bucket.key);
        return (
          <div key={bucket.key}>
            <button
              type="button"
              onClick={() => toggleCollapse(bucket.key)}
              className="w-full flex items-center gap-2 px-3 pt-3 pb-1 group"
              aria-expanded={!isCollapsed}
            >
              <span className="text-[10px] text-bc-muted transition-transform duration-150"
                style={{ transform: isCollapsed ? "rotate(-90deg)" : "rotate(0deg)" }}
              >
                ▼
              </span>
              <span className="text-[10px] font-semibold text-bc-muted uppercase tracking-widest">
                {config.label}
              </span>
              {bucket.key !== "internal" && (
                <span className={`text-[9px] px-1.5 py-0.5 rounded ${config.badgeClass}`}>
                  {bucket.channels.length}
                </span>
              )}
            </button>
            {!isCollapsed &&
              bucket.channels.map((ch) => (
                <button
                  key={ch.name}
                  onClick={() => onSelect(ch.name)}
                  className={`w-full text-left px-3 py-2 text-sm border-b border-bc-border/30 flex items-center gap-2 transition-colors ${
                    selected === ch.name
                      ? "bg-bc-accent/10 text-bc-accent border-l-2 border-l-bc-accent"
                      : "text-bc-text hover:bg-bc-surface border-l-2 border-l-transparent"
                  }`}
                >
                  <span className="font-medium truncate">#{displayName(ch.name)}</span>
                  <span className="ml-auto text-xs text-bc-muted shrink-0">
                    {ch.member_count}
                  </span>
                </button>
              ))}
          </div>
        );
      })}
    </nav>
  );
}
