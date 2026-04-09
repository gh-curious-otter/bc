import { useCallback, useEffect, useState } from "react";
import { NavLink, Outlet, useLocation } from "react-router-dom";
import { useTheme, THEME_LABELS } from "../context/ThemeContext";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { CommandPalette } from "./CommandPalette";
import { api } from "../api/client";
import type { Channel, GatewayStatus, NotifySubscription } from "../api/client";
import { channelPlatform } from "./channels/messageUtils";
import { SetupWizard } from "./channels/SetupWizard";

const SIDEBAR_KEY = "bc-sidebar-collapsed";

function NavIcon({ name }: { name: string }) {
  const icons: Record<string, JSX.Element> = {
    dashboard: <path d="M3 3h4v5H3zM9 3h4v3H9zM9 8h4v5H9zM3 10h4v3H3z" />,
    agents: <path d="M8 4a2 2 0 100 4 2 2 0 000-4zM4 12c0-2 2-3 4-3s4 1 4 3" />,
    channels: <path d="M3 5h10M3 8h7M3 11h10" />,
    roles: <path d="M8 3l5 3v4l-5 3-5-3V6z" />,
    tools: <path d="M10 3l3 3-7 7H3v-3z" />,
    cron: <path d="M8 3a5 5 0 100 10A5 5 0 008 3zM8 5v3l2 2" />,
    secrets: <path d="M8 3a2 2 0 00-2 2v2H5v5h6V7H10V5a2 2 0 00-2-2zm0 6a1 1 0 110 2 1 1 0 010-2z" />,
    metrics: <path d="M3 11l3-4 2 2 4-5" strokeLinecap="round" strokeLinejoin="round" />,
    live: <>
      <path d="M2.5 11.5a7.5 7.5 0 0111-10" strokeLinecap="round" />
      <path d="M4.5 10a4.5 4.5 0 016.5-6" strokeLinecap="round" />
      <circle cx="8" cy="8" r="1.5" fill="currentColor" />
    </>,
    workspace: <path d="M3 4h10v8H3zM5 4V3h6v1" />,
    settings: <path d="M8 6a2 2 0 100 4 2 2 0 000-4zM8 2v2M8 12v2M2 8h2M12 8h2M3.8 3.8l1.4 1.4M10.8 10.8l1.4 1.4M3.8 12.2l1.4-1.4M10.8 5.2l1.4-1.4" />,
  };
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      {icons[name] ?? <path d="M4 4h8v8H4z" />}
    </svg>
  );
}

const PLATFORM_META: Record<string, { label: string; color: string }> = {
  slack: { label: "Slack", color: "#E01E5A" },
  telegram: { label: "Telegram", color: "#26A5E4" },
  discord: { label: "Discord", color: "#5865F2" },
  github: { label: "GitHub", color: "#8B949E" },
  gmail: { label: "Gmail", color: "#EA4335" },
};

function getPlatformMeta(p: string) {
  return PLATFORM_META[p] ?? { label: p, color: "#8c7e72" };
}

function displayChannelName(name: string): string {
  const idx = name.indexOf(":");
  return idx > 0 ? name.slice(idx + 1) : name;
}

function ChannelNavTree() {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [gateways, setGateways] = useState<GatewayStatus[]>([]);
  const [subs, setSubs] = useState<NotifySubscription[]>([]);
  const [expandedGw, setExpandedGw] = useState<Set<string>>(new Set(["slack", "telegram", "discord"]));
  const [setupPlatform, setSetupPlatform] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    try {
      const [chs, gws, subList] = await Promise.all([
        api.listChannels().catch(() => [] as Channel[]),
        api.listGateways().catch(() => [] as GatewayStatus[]),
        api.listSubscriptions().catch(() => [] as NotifySubscription[]),
      ]);
      setChannels(chs ?? []);
      setGateways(gws ?? []);
      setSubs(subList ?? []);
    } catch { /* */ }
  }, []);

  useEffect(() => {
    void fetchData();
    const interval = setInterval(() => void fetchData(), 12000);
    return () => clearInterval(interval);
  }, [fetchData]);

  const toggleGw = (p: string) => {
    setExpandedGw((prev) => {
      const next = new Set(prev);
      if (next.has(p)) next.delete(p); else next.add(p);
      return next;
    });
  };

  // Subscription counts per channel
  const subCountMap = new Map<string, number>();
  for (const sub of subs) {
    subCountMap.set(sub.channel, (subCountMap.get(sub.channel) ?? 0) + 1);
  }

  // Build gateway buckets
  const gwMap = new Map<string, GatewayStatus>();
  for (const gw of gateways) gwMap.set(gw.platform, gw);

  const bucketMap = new Map<string, Channel[]>();
  for (const ch of channels) {
    const p = channelPlatform(ch.name);
    if (p === "internal") continue;
    const list = bucketMap.get(p) ?? [];
    list.push(ch);
    bucketMap.set(p, list);
  }
  for (const gw of gateways) {
    if (!bucketMap.has(gw.platform)) bucketMap.set(gw.platform, []);
  }

  const configuredPlatforms = new Set(bucketMap.keys());
  const unconfigured = Object.keys(PLATFORM_META).filter((p) => !configuredPlatforms.has(p));

  return (
    <div className="pb-1">
      {[...bucketMap.entries()].map(([platform, chs]) => {
        const meta = getPlatformMeta(platform);
        const gwStatus = gwMap.get(platform);
        const isConnected = (gwStatus?.enabled && (gwStatus?.channels?.length ?? 0) > 0) || chs.length > 0;
        const isExpanded = expandedGw.has(platform);

        return (
          <div key={platform} className="mb-0.5">
            <button
              type="button"
              onClick={() => toggleGw(platform)}
              className="w-full flex items-center gap-1.5 pl-4 pr-3 py-1 hover:bg-bc-bg/60 transition-colors"
            >
              <svg
                width="7" height="7" viewBox="0 0 8 8"
                className={`text-bc-muted/30 transition-transform duration-150 shrink-0 ${isExpanded ? "" : "-rotate-90"}`}
              >
                <path d="M1.5 2L4 5L6.5 2" stroke="currentColor" strokeWidth="1.2" fill="none" strokeLinecap="round" />
              </svg>
              <span
                className="w-1.5 h-1.5 rounded-full shrink-0"
                style={{
                  backgroundColor: isConnected
                    ? "#22c55e"
                    : gwStatus?.enabled
                    ? "#fb923c"
                    : "rgba(140,126,114,0.15)",
                }}
              />
              <span className="text-[9px] font-bold uppercase tracking-[0.08em] truncate" style={{ color: meta.color }}>
                {meta.label}
              </span>
              <span className="text-[8px] text-bc-muted/20 ml-auto tabular-nums shrink-0">{chs.length}</span>
            </button>

            {isExpanded && (
              <div className="pb-0.5">
                {chs.length === 0 && (
                  <div className="pl-8 pr-3 py-0.5 text-[9px] text-bc-muted/20 italic">No channels</div>
                )}
                {chs.map((ch) => {
                  const count = subCountMap.get(ch.name) ?? 0;
                  return (
                    <NavLink
                      key={ch.name}
                      to={"/channels/" + ch.name}
                      className={({ isActive }) =>
                        `w-full text-left pl-8 pr-3 py-[4px] text-[11px] flex items-center gap-1 transition-all duration-100 ${
                          isActive
                            ? "bg-bc-surface/50 text-bc-text font-medium"
                            : "text-bc-muted/50 hover:text-bc-text/80 hover:bg-bc-bg/60"
                        }`
                      }
                      style={({ isActive }) => ({
                        borderLeft: isActive ? `2px solid ${meta.color}` : "2px solid transparent",
                      })}
                    >
                      <span className="text-bc-muted/20 text-[9px] shrink-0">#</span>
                      <span className="truncate">{displayChannelName(ch.name)}</span>
                      {count > 0 && (
                        <span className="ml-auto text-[8px] text-bc-success/40 tabular-nums shrink-0">{count}</span>
                      )}
                    </NavLink>
                  );
                })}
              </div>
            )}
          </div>
        );
      })}

      {/* Unconfigured platforms */}
      {unconfigured.length > 0 && (
        <div className="pt-1 mt-0.5 border-t border-bc-border/10 mx-3">
          {unconfigured.map((p) => (
            <button
              key={p}
              type="button"
              onClick={() => setSetupPlatform(p)}
              className="w-full flex items-center gap-1.5 py-0.5 text-[9px] text-bc-muted/20 hover:text-bc-muted/40 transition-colors"
            >
              <span className="w-1.5 h-1.5 rounded-full bg-bc-muted/10 shrink-0" />
              <span className="uppercase tracking-[0.08em] font-medium truncate">{getPlatformMeta(p).label}</span>
              <span className="ml-auto opacity-50 shrink-0">+</span>
            </button>
          ))}
        </div>
      )}

      {/* Connect app */}
      <div className="px-3 pt-1 pb-0.5">
        <button
          type="button"
          onClick={() => setSetupPlatform("_choose")}
          className="w-full py-1 text-[9px] font-medium text-bc-muted/25 hover:text-bc-accent border border-bc-border/15 rounded-md hover:border-bc-accent/20 transition-all"
        >
          + Connect app
        </button>
      </div>

      {/* Setup wizard */}
      {setupPlatform && setupPlatform !== "_choose" && (
        <SetupWizard
          platform={setupPlatform}
          onClose={() => setSetupPlatform(null)}
          onConnected={() => void fetchData()}
        />
      )}
      {setupPlatform === "_choose" && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
          <div className="bg-bc-bg border border-bc-border/50 rounded-xl p-5 max-w-sm w-full mx-4 shadow-2xl">
            <h2 className="text-[14px] font-semibold text-bc-text mb-4">Connect an app</h2>
            <div className="grid grid-cols-2 gap-2">
              {Object.entries(PLATFORM_META).map(([key, meta]) => (
                <button
                  key={key}
                  type="button"
                  onClick={() => setSetupPlatform(key)}
                  className="p-3 border border-bc-border/30 rounded-lg hover:border-bc-border/50 hover:bg-bc-surface/20 transition-all text-left group"
                >
                  <div
                    className="w-6 h-6 rounded flex items-center justify-center text-[11px] font-bold mb-1.5"
                    style={{ backgroundColor: `${meta.color}12`, color: meta.color }}
                  >
                    {meta.label.charAt(0)}
                  </div>
                  <span className="text-[11px] text-bc-muted/50 group-hover:text-bc-text">{meta.label}</span>
                </button>
              ))}
            </div>
            <button
              type="button"
              onClick={() => setSetupPlatform(null)}
              className="mt-3 w-full py-1.5 text-[10px] text-bc-muted/30 hover:text-bc-text transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

const MAIN_NAV_ITEMS = [
  { to: "/live", label: "Live", icon: "live" },
  { to: "/agents", label: "Agents", icon: "agents" },
  { to: "/channels", label: "Channels", icon: "channels" },
  { to: "/roles", label: "Roles", icon: "roles" },
  { to: "/tools", label: "Tools", icon: "tools" },
  { to: "/cron", label: "Cron", icon: "cron" },
  { to: "/secrets", label: "Secrets", icon: "secrets" },
  { to: "/stats", label: "Metrics", icon: "metrics" },
] as const;

const UTIL_NAV_ITEMS = [
  { to: "/workspace", label: "Workspace", icon: "workspace" },
  { to: "/settings", label: "Settings", icon: "settings" },
] as const;

const NAV_ITEMS = [...MAIN_NAV_ITEMS, ...UTIL_NAV_ITEMS];

function readCollapsed(): boolean {
  try {
    return localStorage.getItem(SIDEBAR_KEY) === "true";
  } catch {
    return false;
  }
}

function writeCollapsed(v: boolean) {
  try {
    localStorage.setItem(SIDEBAR_KEY, String(v));
  } catch {
    // storage unavailable
  }
}

function NavList({
  items,
  collapsed,
  isMobile,
  channelsExpanded,
}: {
  items: ReadonlyArray<{ to: string; label: string; icon: string }>;
  collapsed: boolean;
  isMobile: boolean;
  channelsExpanded?: boolean;
}) {
  const isIconOnly = collapsed && !isMobile;
  const showTree = !isIconOnly && channelsExpanded;

  return (
    <>
      {items.map(({ to, label, icon }) => {
        const isChannels = label === "Channels";
        return (
          <li key={to}>
            <NavLink
              to={to}
              end={!isChannels}
              title={isIconOnly ? label : undefined}
              className={({ isActive }) =>
                `relative flex items-center gap-3 ${isIconOnly ? "justify-center px-2" : "px-4"} py-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-inset transition-colors ${
                  isActive
                    ? "text-bc-accent bg-bc-bg font-medium border-l-[3px] border-bc-accent"
                    : "text-bc-muted hover:text-bc-text hover:bg-bc-bg/50 border-l-[3px] border-transparent"
                }`
              }
            >
              <span className="shrink-0 flex items-center justify-center w-5">
                <NavIcon name={icon} />
              </span>
              {(!collapsed || isMobile) && <span>{label}</span>}
              {label === "Live" && (
                <span className="w-1.5 h-1.5 rounded-full bg-red-500 animate-pulse" />
              )}
            </NavLink>
            {isChannels && showTree && <ChannelNavTree />}
          </li>
        );
      })}
    </>
  );
}

export function Layout() {
  const location = useLocation();
  const { mode, toggle } = useTheme();
  const isMobile = useMediaQuery("(max-width: 767px)");

  // Fetch user name for sidebar header
  const [userName, setUserName] = useState("");
  useEffect(() => {
    fetch("/api/settings").then(r => r.json()).then(d => {
      setUserName(d?.user?.name || "");
    }).catch(() => {});
  }, []);

  // Mobile overlay sidebar (open/close)
  const [mobileOpen, setMobileOpen] = useState(false);

  // Desktop collapsed sidebar (icons only)
  const [collapsed, setCollapsed] = useState(readCollapsed);

  const toggleCollapsed = useCallback(() => {
    setCollapsed((prev) => {
      const next = !prev;
      writeCollapsed(next);
      return next;
    });
  }, []);

  // Auto-collapse on small screens
  useEffect(() => {
    if (isMobile) {
      setCollapsed(true);
    }
  }, [isMobile]);

  // Dynamic page title (#2150)
  useEffect(() => {
    const match = NAV_ITEMS.find((item) =>
      location.pathname.startsWith(item.to),
    );
    document.title = match ? `${match.label} \u2014 bc` : "bc";
  }, [location.pathname]);

  // Close mobile sidebar on route change
  useEffect(() => {
    setMobileOpen(false);
  }, [location.pathname]);

  // Determine effective sidebar width class
  const sidebarWidth = collapsed && !isMobile ? "w-14" : "w-48";

  return (
    <div className="flex h-screen">
      {/* Mobile hamburger button */}
      <button
        type="button"
        onClick={() => setMobileOpen(true)}
        className="fixed top-3 left-3 z-40 md:hidden p-2 rounded border border-bc-border bg-bc-surface text-bc-muted hover:text-bc-text transition-colors"
        aria-label="Open navigation"
      >
        <svg
          width="20"
          height="20"
          viewBox="0 0 20 20"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
        >
          <path d="M3 5h14M3 10h14M3 15h14" />
        </svg>
      </button>

      {/* Overlay for mobile sidebar */}
      {mobileOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 md:hidden"
          onClick={() => setMobileOpen(false)}
        />
      )}

      {/* Sidebar */}
      <nav
        className={`fixed inset-y-0 left-0 z-50 ${sidebarWidth} shrink-0 border-r border-bc-border bg-bc-surface flex flex-col transition-all duration-200 md:relative md:translate-x-0 ${
          isMobile
            ? mobileOpen
              ? "translate-x-0 w-48"
              : "-translate-x-full"
            : ""
        }`}
      >
        <div className="p-4 border-b border-bc-border flex items-center justify-between">
          {(!collapsed || isMobile) ? (
            <div className="flex items-center gap-2 overflow-hidden">
              <span className="w-7 h-7 rounded-full bg-bc-accent/20 text-bc-accent flex items-center justify-center text-xs font-bold shrink-0">
                {(userName || "U")[0]!.toUpperCase()}
              </span>
              <div className="min-w-0">
                <p className="text-sm font-medium text-bc-text truncate">{userName || "User"}</p>
                <p className="text-[10px] text-bc-muted">workspace</p>
              </div>
            </div>
          ) : (
            <span className="w-7 h-7 rounded-full bg-bc-accent/20 text-bc-accent flex items-center justify-center text-xs font-bold shrink-0">
              {(userName || "U")[0]!.toUpperCase()}
            </span>
          )}
          {/* Close button on mobile */}
          {isMobile && (
            <button
              type="button"
              onClick={() => setMobileOpen(false)}
              className="p-1 rounded text-bc-muted hover:text-bc-text transition-colors"
              aria-label="Close navigation"
            >
              <svg
                width="16"
                height="16"
                viewBox="0 0 16 16"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
              >
                <path d="M4 4l8 8M12 4l-8 8" />
              </svg>
            </button>
          )}
          {/* Collapse toggle on desktop */}
          {!isMobile && (
            <button
              type="button"
              onClick={toggleCollapsed}
              className="p-1 rounded text-bc-muted hover:text-bc-text transition-colors"
              aria-label={
                collapsed ? "Expand navigation" : "Collapse navigation"
              }
            >
              <svg
                width="16"
                height="16"
                viewBox="0 0 16 16"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
              >
                {collapsed ? (
                  <path d="M6 4l4 4-4 4" />
                ) : (
                  <path d="M10 4l-4 4 4 4" />
                )}
              </svg>
            </button>
          )}
        </div>
        <ul className="flex-1 py-2 overflow-y-auto">
          <NavList
            items={MAIN_NAV_ITEMS}
            collapsed={collapsed}
            isMobile={isMobile}
            channelsExpanded={location.pathname.startsWith("/channels")}
          />
          <li className={`my-2 ${collapsed && !isMobile ? "mx-2" : "mx-4"}`}>
            <div className="border-t border-bc-border" />
          </li>
          <NavList items={UTIL_NAV_ITEMS} collapsed={collapsed} isMobile={isMobile} />
        </ul>
        <div className="p-3 border-t border-bc-border flex items-center justify-center">
          <button
            type="button"
            onClick={toggle}
            className="px-2 py-1 rounded border border-bc-border text-xs text-bc-muted hover:text-bc-text hover:border-bc-accent transition-colors"
            title={`Theme: ${THEME_LABELS[mode]}`}
          >
            {collapsed && !isMobile
              ? THEME_LABELS[mode][0]
              : THEME_LABELS[mode]}
          </button>
        </div>
      </nav>
      <main className="flex-1 overflow-auto bg-bc-bg">
        <Outlet />
      </main>
      <CommandPalette />
    </div>
  );
}
