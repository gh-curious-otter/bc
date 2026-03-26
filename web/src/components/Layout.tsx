import { useCallback, useEffect, useState } from "react";
import { NavLink, Outlet, useLocation } from "react-router-dom";
import { useTheme } from "../context/ThemeContext";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { CommandPalette } from "./CommandPalette";

const SIDEBAR_KEY = "bc-sidebar-collapsed";

const NAV_ITEMS = [
  { to: "/", label: "Dashboard", icon: "~" },
  { to: "/activity", label: "Activity", icon: "▶" },
  { to: "/agents", label: "Agents", icon: "A" },
  { to: "/channels", label: "Channels", icon: "C" },
  { to: "/costs", label: "Costs", icon: "$" },
  { to: "/roles", label: "Roles", icon: "R" },
  { to: "/tools", label: "Tools", icon: "T" },
  { to: "/mcp", label: "MCP", icon: "M" },
  { to: "/cron", label: "Cron", icon: "@" },
  { to: "/secrets", label: "Secrets", icon: "#" },
  { to: "/stats", label: "Stats", icon: "S" },
  { to: "/logs", label: "Logs", icon: "L" },
  { to: "/workspace", label: "Workspace", icon: "W" },
  { to: "/doctor", label: "Doctor", icon: "+" },
  { to: "/settings", label: "Settings", icon: "\u2699" },
] as const;

const THEME_LABELS = {
  dark: "Dark",
  light: "Light",
  system: "System",
} as const;

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

export function Layout() {
  const location = useLocation();
  const { mode, toggle } = useTheme();
  const isMobile = useMediaQuery("(max-width: 767px)");

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
      item.to === "/"
        ? location.pathname === "/"
        : location.pathname.startsWith(item.to),
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
          <div className="flex items-center overflow-hidden">
            <span className="text-lg font-bold text-bc-accent">bc</span>
            {(!collapsed || isMobile) && (
              <span className="ml-2 text-xs text-bc-muted">v2</span>
            )}
          </div>
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
          {NAV_ITEMS.map(({ to, label, icon }) => (
            <li key={to}>
              <NavLink
                to={to}
                end={to === "/"}
                title={collapsed && !isMobile ? label : undefined}
                className={({ isActive }) =>
                  `flex items-center gap-2 ${collapsed && !isMobile ? "justify-center px-2" : "px-4"} py-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-inset ${
                    isActive
                      ? "text-bc-accent bg-bc-bg font-medium"
                      : "text-bc-muted hover:text-bc-text hover:bg-bc-bg/50"
                  }`
                }
              >
                <span className="w-5 text-center font-mono text-xs shrink-0">
                  {icon}
                </span>
                {(!collapsed || isMobile) && label}
              </NavLink>
            </li>
          ))}
        </ul>
        <div
          className={`p-3 border-t border-bc-border text-xs text-bc-muted flex items-center ${collapsed && !isMobile ? "justify-center" : "justify-between"}`}
        >
          {(!collapsed || isMobile) && (
            <span>
              <kbd className="text-bc-text">?</kbd> help
            </span>
          )}
          <button
            type="button"
            onClick={toggle}
            className="px-2 py-1 rounded border border-bc-border text-bc-muted hover:text-bc-text hover:border-bc-accent transition-colors"
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
