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

const PLATFORM_META: Record<string, { label: string; color: string }> = {
  slack: { label: "Slack", color: "text-[#E01E5A]" },
  telegram: { label: "Telegram", color: "text-[#26A5E4]" },
  discord: { label: "Discord", color: "text-[#5865F2]" },
  github: { label: "GitHub", color: "text-[#8B949E]" },
  gmail: { label: "Gmail", color: "text-[#EA4335]" },
};

function getMeta(platform: string) {
  return PLATFORM_META[platform] ?? { label: platform, color: "text-bc-muted" };
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

  // Build subscription count per channel
  const subCountMap = new Map<string, number>();
  for (const sub of allSubs) {
    subCountMap.set(sub.channel, (subCountMap.get(sub.channel) ?? 0) + 1);
  }

  // Build gateway buckets from channels + gateway status
  const gwMap = new Map<string, GatewayStatus>();
  for (const gw of gateways) gwMap.set(gw.platform, gw);

  const bucketMap = new Map<string, Channel[]>();
  for (const ch of channels) {
    const platform = channelPlatform(ch.name);
    if (platform === "internal") continue; // skip bc-native channels
    const list = bucketMap.get(platform) ?? [];
    list.push(ch);
    bucketMap.set(platform, list);
  }

  // Also add empty buckets for configured but channelless gateways
  for (const gw of gateways) {
    if (!bucketMap.has(gw.platform)) {
      bucketMap.set(gw.platform, []);
    }
  }

  const buckets: GatewayBucket[] = [];
  for (const [platform, chs] of bucketMap) {
    const meta = getMeta(platform);
    const gwStatus = gwMap.get(platform);
    buckets.push({
      platform,
      label: meta.label,
      enabled: gwStatus?.enabled ?? false,
      channels: chs,
    });
  }
  // Sort: enabled first, then alphabetically
  buckets.sort((a, b) => {
    if (a.enabled !== b.enabled) return a.enabled ? -1 : 1;
    return a.label.localeCompare(b.label);
  });

  // Supported but not configured gateways
  const configuredPlatforms = new Set(bucketMap.keys());
  const unconfigured = Object.keys(PLATFORM_META).filter(p => !configuredPlatforms.has(p));

  return (
    <nav className="w-60 shrink-0 border-r border-bc-border overflow-auto flex flex-col" aria-label="Channel list">
      <div className="p-3 border-b border-bc-border">
        <h2 className="text-[11px] font-semibold text-bc-muted uppercase tracking-widest">
          Gateways
        </h2>
      </div>

      <div className="flex-1 overflow-auto">
        {buckets.map((bucket) => {
          const meta = getMeta(bucket.platform);
          const isCollapsed = collapsed.has(bucket.platform);
          const gwStatus = gwMap.get(bucket.platform);
          const isConnected = gwStatus?.enabled && (gwStatus?.channels?.length ?? 0) > 0;

          return (
            <div key={bucket.platform}>
              <button
                type="button"
                onClick={() => toggleCollapse(bucket.platform)}
                className="w-full flex items-center gap-2 px-3 pt-3 pb-1 group hover:bg-bc-surface/30 transition-colors"
                aria-expanded={!isCollapsed}
              >
                <span
                  className="text-[10px] text-bc-muted transition-transform duration-150"
                  style={{ transform: isCollapsed ? "rotate(-90deg)" : "rotate(0deg)" }}
                >
                  ▼
                </span>
                {/* Status dot */}
                <span
                  className={`w-1.5 h-1.5 rounded-full shrink-0 ${
                    isConnected ? "bg-bc-success" : bucket.enabled ? "bg-bc-warning" : "bg-bc-muted/40"
                  }`}
                />
                <span className={`text-[10px] font-semibold uppercase tracking-widest ${meta.color}`}>
                  {meta.label}
                </span>
                <span className="text-[9px] text-bc-muted ml-auto">
                  {bucket.channels.length}
                </span>
              </button>

              {!isCollapsed && (
                <>
                  {bucket.channels.length === 0 && (
                    <div className="px-3 py-2 text-[11px] text-bc-muted/60 italic">
                      No channels discovered
                    </div>
                  )}
                  {bucket.channels.map((ch) => (
                    <button
                      key={ch.name}
                      onClick={() => onSelect(ch.name)}
                      className={`w-full text-left px-3 py-1.5 text-[13px] flex items-center gap-2 transition-colors ${
                        selected === ch.name
                          ? "bg-bc-accent/10 text-bc-accent border-l-2 border-l-bc-accent"
                          : "text-bc-text/80 hover:bg-bc-surface/50 border-l-2 border-l-transparent"
                      }`}
                    >
                      <span className="text-bc-muted/50 text-[11px]">#</span>
                      <span className="truncate">{displayName(ch.name)}</span>
                      {(() => {
                        const count = subCountMap.get(ch.name) ?? 0;
                        return count > 0 ? (
                          <span className="ml-auto text-[9px] text-bc-success/60 shrink-0">
                            {count} agent{count !== 1 ? "s" : ""}
                          </span>
                        ) : null;
                      })()}
                    </button>
                  ))}
                </>
              )}
            </div>
          );
        })}

        {/* Unconfigured gateways */}
        {unconfigured.length > 0 && (
          <div className="mt-2 pt-2 border-t border-bc-border/30">
            {unconfigured.map((platform) => {
              const meta = getMeta(platform);
              return (
                <div
                  key={platform}
                  className="flex items-center gap-2 px-3 py-1.5 text-[11px] text-bc-muted/50"
                >
                  <span className="w-1.5 h-1.5 rounded-full bg-bc-muted/20 shrink-0" />
                  <span className="uppercase tracking-widest font-medium">{meta.label}</span>
                  <button
                    type="button"
                    onClick={() => setSetupPlatform(platform)}
                    className="ml-auto text-[10px] text-bc-muted/30 hover:text-bc-accent cursor-pointer transition-colors"
                  >
                    Setup &rarr;
                  </button>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Connect app button */}
      <div className="p-2 border-t border-bc-border">
        <button
          type="button"
          onClick={() => setSetupPlatform("_choose")}
          className="w-full py-1.5 text-[11px] text-bc-muted hover:text-bc-accent border border-bc-border/50 rounded hover:border-bc-accent/30 transition-colors"
        >
          + Connect app
        </button>
      </div>

      {/* Setup wizard modal */}
      {setupPlatform && setupPlatform !== "_choose" && (
        <SetupWizard
          platform={setupPlatform}
          onClose={() => setSetupPlatform(null)}
          onConnected={() => void fetchGateways()}
        />
      )}

      {/* Platform chooser modal */}
      {setupPlatform === "_choose" && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-bc-bg border border-bc-border rounded-lg p-5 max-w-sm w-full mx-4">
            <h2 className="text-[14px] font-semibold text-bc-text mb-4">Connect an app</h2>
            <div className="grid grid-cols-2 gap-2">
              {Object.entries(PLATFORM_META).map(([key, meta]) => (
                <button
                  key={key}
                  type="button"
                  onClick={() => setSetupPlatform(key)}
                  className="p-3 border border-bc-border rounded hover:border-bc-accent/40 hover:bg-bc-surface/30 transition-colors text-left"
                >
                  <span className={`text-[13px] font-medium ${meta.color}`}>{meta.label}</span>
                </button>
              ))}
            </div>
            <button
              type="button"
              onClick={() => setSetupPlatform(null)}
              className="mt-3 w-full py-1.5 text-[11px] text-bc-muted hover:text-bc-text transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </nav>
  );
}
