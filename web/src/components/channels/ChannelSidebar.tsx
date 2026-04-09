import { useState, useEffect, useCallback } from "react";
import type { Channel, GatewayStatus, NotifySubscription } from "../../api/client";
import { api } from "../../api/client";
import { channelPlatform } from "./messageUtils";
import { SetupWizard } from "./SetupWizard";

interface GatewayBucket {
  platform: string;
  label: string;
  enabled: boolean;
  channels: Channel[];
}

const PLATFORM_META: Record<string, { label: string; color: string; icon: string }> = {
  slack: { label: "Slack", color: "#E01E5A", icon: "S" },
  telegram: { label: "Telegram", color: "#26A5E4", icon: "T" },
  discord: { label: "Discord", color: "#5865F2", icon: "D" },
  github: { label: "GitHub", color: "#8B949E", icon: "G" },
  gmail: { label: "Gmail", color: "#EA4335", icon: "M" },
};

function getMeta(platform: string) {
  return PLATFORM_META[platform] ?? { label: platform, color: "#8c7e72", icon: "?" };
}

function displayName(name: string): string {
  const idx = name.indexOf(":");
  if (idx > 0) return name.slice(idx + 1);
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
  const [gateways, setGateways] = useState<GatewayStatus[]>([]);
  const [allSubs, setAllSubs] = useState<NotifySubscription[]>([]);
  const [setupPlatform, setSetupPlatform] = useState<string | null>(null);
  const [collapsed, setCollapsed] = useState<Set<string>>(() => {
    try {
      const stored = localStorage.getItem("bc-channels-collapsed");
      return stored ? new Set(JSON.parse(stored) as string[]) : new Set<string>();
    } catch {
      return new Set<string>();
    }
  });

  const fetchGateways = useCallback(async () => {
    try {
      const [gw, subs] = await Promise.all([
        api.listGateways(),
        api.listSubscriptions().catch(() => []),
      ]);
      setGateways(gw ?? []);
      setAllSubs(subs ?? []);
    } catch {
      // keep empty
    }
  }, []);

  useEffect(() => {
    void fetchGateways();
    const interval = setInterval(() => void fetchGateways(), 15000);
    return () => clearInterval(interval);
  }, [fetchGateways]);

  const toggleCollapse = (key: string) => {
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      try { localStorage.setItem("bc-channels-collapsed", JSON.stringify([...next])); } catch { /* */ }
      return next;
    });
  };

  // Subscription count map
  const subCountMap = new Map<string, number>();
  for (const sub of allSubs) {
    subCountMap.set(sub.channel, (subCountMap.get(sub.channel) ?? 0) + 1);
  }

  // Build gateway buckets
  const gwMap = new Map<string, GatewayStatus>();
  for (const gw of gateways) gwMap.set(gw.platform, gw);

  const bucketMap = new Map<string, Channel[]>();
  for (const ch of channels) {
    const platform = channelPlatform(ch.name);
    if (platform === "internal") continue;
    const list = bucketMap.get(platform) ?? [];
    list.push(ch);
    bucketMap.set(platform, list);
  }

  for (const gw of gateways) {
    if (!bucketMap.has(gw.platform)) {
      bucketMap.set(gw.platform, []);
    }
  }

  const buckets: GatewayBucket[] = [];
  for (const [platform, chs] of bucketMap) {
    const gwStatus = gwMap.get(platform);
    buckets.push({
      platform,
      label: getMeta(platform).label,
      enabled: gwStatus?.enabled ?? false,
      channels: chs,
    });
  }
  buckets.sort((a, b) => {
    if (a.enabled !== b.enabled) return a.enabled ? -1 : 1;
    return a.label.localeCompare(b.label);
  });

  const configuredPlatforms = new Set(bucketMap.keys());
  const unconfigured = Object.keys(PLATFORM_META).filter(p => !configuredPlatforms.has(p));

  return (
    <nav
      className="w-60 shrink-0 border-r border-bc-border/50 flex flex-col bg-bc-bg"
      style={{ scrollbarWidth: "thin", scrollbarColor: "rgba(255,255,255,0.04) transparent" }}
      aria-label="Channel list"
    >
      {/* Header */}
      <div className="px-4 py-3 border-b border-bc-border/30">
        <h2 className="text-[11px] font-bold text-bc-muted/70 uppercase tracking-[0.12em]">
          Gateways
        </h2>
      </div>

      {/* Channel list */}
      <div className="flex-1 overflow-auto py-1">
        {buckets.map((bucket) => {
          const meta = getMeta(bucket.platform);
          const isCollapsed = collapsed.has(bucket.platform);
          const gwStatus = gwMap.get(bucket.platform);
          const isConnected = gwStatus?.enabled && (gwStatus?.channels?.length ?? 0) > 0;

          return (
            <div key={bucket.platform} className="mb-0.5">
              {/* Platform header */}
              <button
                type="button"
                onClick={() => toggleCollapse(bucket.platform)}
                className="w-full flex items-center gap-2 px-4 py-1.5 hover:bg-bc-surface/20 transition-colors duration-100"
                aria-expanded={!isCollapsed}
              >
                <svg
                  width="8" height="8" viewBox="0 0 8 8"
                  className={`text-bc-muted/40 transition-transform duration-150 ${isCollapsed ? "-rotate-90" : ""}`}
                >
                  <path d="M1.5 2L4 5L6.5 2" stroke="currentColor" strokeWidth="1.2" fill="none" strokeLinecap="round" />
                </svg>
                <span
                  className="w-1.5 h-1.5 rounded-full shrink-0"
                  style={{
                    backgroundColor: isConnected ? "#22c55e" : bucket.enabled ? "#fb923c" : "rgba(140,126,114,0.25)",
                  }}
                />
                <span
                  className="text-[10px] font-bold uppercase tracking-[0.1em]"
                  style={{ color: meta.color }}
                >
                  {meta.label}
                </span>
                <span className="text-[9px] text-bc-muted/30 ml-auto tabular-nums">
                  {bucket.channels.length}
                </span>
              </button>

              {/* Channels */}
              {!isCollapsed && (
                <div className="pb-1">
                  {bucket.channels.length === 0 && (
                    <div className="px-4 py-1 text-[10px] text-bc-muted/25 italic pl-9">
                      No channels discovered
                    </div>
                  )}
                  {bucket.channels.map((ch) => {
                    const isActive = selected === ch.name;
                    const count = subCountMap.get(ch.name) ?? 0;
                    return (
                      <button
                        key={ch.name}
                        onClick={() => onSelect(ch.name)}
                        className={`w-full text-left px-3 py-[5px] text-[13px] flex items-center gap-1.5 transition-all duration-100 rounded-md mx-1 ${
                          isActive
                            ? "bg-bc-surface text-bc-text"
                            : "text-bc-muted/70 hover:text-bc-text/90 hover:bg-bc-surface/30"
                        }`}
                        style={{
                          width: "calc(100% - 8px)",
                          borderLeft: isActive ? `2px solid ${meta.color}` : "2px solid transparent",
                        }}
                      >
                        <span className="text-bc-muted/30 text-[11px]">#</span>
                        <span className={`truncate ${isActive ? "font-medium" : ""}`}>
                          {displayName(ch.name)}
                        </span>
                        {count > 0 && (
                          <span className="ml-auto text-[9px] text-bc-success/50 tabular-nums shrink-0">
                            {count}
                          </span>
                        )}
                      </button>
                    );
                  })}
                </div>
              )}
            </div>
          );
        })}

        {/* Unconfigured gateways */}
        {unconfigured.length > 0 && (
          <div className="pt-2 mt-1 border-t border-bc-border/20 mx-4">
            {unconfigured.map((platform) => {
              const meta = getMeta(platform);
              return (
                <button
                  key={platform}
                  type="button"
                  onClick={() => setSetupPlatform(platform)}
                  className="w-full flex items-center gap-2 py-1.5 text-[10px] text-bc-muted/25 hover:text-bc-muted/50 transition-colors"
                >
                  <span className="w-1.5 h-1.5 rounded-full bg-bc-muted/15 shrink-0" />
                  <span className="uppercase tracking-[0.1em] font-medium">{meta.label}</span>
                  <span className="ml-auto opacity-50">Setup &rarr;</span>
                </button>
              );
            })}
          </div>
        )}
      </div>

      {/* Connect button */}
      <div className="p-3 border-t border-bc-border/30">
        <button
          type="button"
          onClick={() => setSetupPlatform("_choose")}
          className="w-full py-2 text-[11px] font-medium text-bc-muted/40 hover:text-bc-accent border border-bc-border/30 rounded-lg hover:border-bc-accent/20 hover:bg-bc-accent/5 transition-all duration-150"
        >
          + Connect app
        </button>
      </div>

      {/* Setup wizard */}
      {setupPlatform && setupPlatform !== "_choose" && (
        <SetupWizard
          platform={setupPlatform}
          onClose={() => setSetupPlatform(null)}
          onConnected={() => void fetchGateways()}
        />
      )}

      {/* Platform chooser */}
      {setupPlatform === "_choose" && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
          <div className="bg-bc-bg border border-bc-border/60 rounded-xl p-5 max-w-sm w-full mx-4 shadow-2xl">
            <h2 className="text-[14px] font-semibold text-bc-text mb-4">Connect an app</h2>
            <div className="grid grid-cols-2 gap-2">
              {Object.entries(PLATFORM_META).map(([key, meta]) => (
                <button
                  key={key}
                  type="button"
                  onClick={() => setSetupPlatform(key)}
                  className="p-3 border border-bc-border/30 rounded-lg hover:border-bc-border/60 hover:bg-bc-surface/30 transition-all duration-150 text-left group"
                >
                  <div
                    className="w-7 h-7 rounded-md flex items-center justify-center text-[12px] font-bold mb-2"
                    style={{ backgroundColor: `${meta.color}15`, color: meta.color }}
                  >
                    {meta.icon}
                  </div>
                  <span className="text-[12px] font-medium text-bc-text/80 group-hover:text-bc-text">
                    {meta.label}
                  </span>
                </button>
              ))}
            </div>
            <button
              type="button"
              onClick={() => setSetupPlatform(null)}
              className="mt-4 w-full py-2 text-[11px] text-bc-muted/40 hover:text-bc-text transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </nav>
  );
}
