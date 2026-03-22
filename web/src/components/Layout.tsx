import { useEffect, useState } from 'react';
import { NavLink, Outlet, useLocation } from 'react-router-dom';
import { useTheme } from '../context/ThemeContext';

const NAV_ITEMS = [
  { to: '/', label: 'Dashboard', icon: '~' },
  { to: '/agents', label: 'Agents', icon: 'A' },
  { to: '/channels', label: 'Channels', icon: 'C' },
  { to: '/costs', label: 'Costs', icon: '$' },
  { to: '/roles', label: 'Roles', icon: 'R' },
  { to: '/tools', label: 'Tools', icon: 'T' },
  { to: '/mcp', label: 'MCP', icon: 'M' },
  { to: '/cron', label: 'Cron', icon: '@' },
  { to: '/secrets', label: 'Secrets', icon: '#' },
  { to: '/stats', label: 'Stats', icon: 'S' },
  { to: '/logs', label: 'Logs', icon: 'L' },
  { to: '/workspace', label: 'Workspace', icon: 'W' },
  { to: '/daemons', label: 'Daemons', icon: 'D' },
  { to: '/doctor', label: 'Doctor', icon: '+' },
  { to: '/settings', label: 'Settings', icon: '\u2699' },
] as const;

const THEME_LABELS = { dark: 'Dark', light: 'Light', system: 'System' } as const;

export function Layout() {
  const location = useLocation();
  const { mode, toggle } = useTheme();
  const [sidebarOpen, setSidebarOpen] = useState(false);

  // Dynamic page title (#2150)
  useEffect(() => {
    const match = NAV_ITEMS.find((item) =>
      item.to === '/' ? location.pathname === '/' : location.pathname.startsWith(item.to)
    );
    document.title = match ? `${match.label} \u2014 bc` : 'bc';
  }, [location.pathname]);

  // Close sidebar on route change (mobile)
  useEffect(() => {
    setSidebarOpen(false);
  }, [location.pathname]);

  return (
    <div className="flex h-screen">
      {/* Mobile hamburger button */}
      <button
        type="button"
        onClick={() => setSidebarOpen(true)}
        className="fixed top-3 left-3 z-40 md:hidden p-2 rounded border border-bc-border bg-bc-surface text-bc-muted hover:text-bc-text transition-colors"
        aria-label="Open navigation"
      >
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="2">
          <path d="M3 5h14M3 10h14M3 15h14" />
        </svg>
      </button>

      {/* Overlay for mobile sidebar */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 md:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <nav
        className={`fixed inset-y-0 left-0 z-50 w-48 shrink-0 border-r border-bc-border bg-bc-surface flex flex-col transform transition-transform duration-200 md:relative md:translate-x-0 ${
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <div className="p-4 border-b border-bc-border flex items-center justify-between">
          <div>
            <span className="text-lg font-bold text-bc-accent">bc</span>
            <span className="ml-2 text-xs text-bc-muted">v2</span>
          </div>
          <button
            type="button"
            onClick={() => setSidebarOpen(false)}
            className="md:hidden p-1 rounded text-bc-muted hover:text-bc-text transition-colors"
            aria-label="Close navigation"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M4 4l8 8M12 4l-8 8" />
            </svg>
          </button>
        </div>
        <ul className="flex-1 py-2 overflow-y-auto">
          {NAV_ITEMS.map(({ to, label, icon }) => (
            <li key={to}>
              <NavLink
                to={to}
                end={to === '/'}
                className={({ isActive }) =>
                  `flex items-center gap-2 px-4 py-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-inset ${
                    isActive
                      ? 'text-bc-accent bg-bc-bg font-medium'
                      : 'text-bc-muted hover:text-bc-text hover:bg-bc-bg/50'
                  }`
                }
              >
                <span className="w-5 text-center font-mono text-xs">{icon}</span>
                {label}
              </NavLink>
            </li>
          ))}
        </ul>
        <div className="p-3 border-t border-bc-border text-xs text-bc-muted flex items-center justify-between">
          <span><kbd className="text-bc-text">?</kbd> help</span>
          <button
            type="button"
            onClick={toggle}
            className="px-2 py-1 rounded border border-bc-border text-bc-muted hover:text-bc-text hover:border-bc-accent transition-colors"
            title={`Theme: ${THEME_LABELS[mode]}`}
          >
            {THEME_LABELS[mode]}
          </button>
        </div>
      </nav>
      <main className="flex-1 overflow-auto bg-bc-bg">
        <Outlet />
      </main>
    </div>
  );
}
