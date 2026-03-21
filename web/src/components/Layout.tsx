import { useEffect } from 'react';
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
  { to: '/logs', label: 'Logs', icon: 'L' },
  { to: '/workspace', label: 'Workspace', icon: 'W' },
  { to: '/doctor', label: 'Doctor', icon: '+' },
] as const;

const THEME_LABELS = { dark: 'Dark', light: 'Light', system: 'System' } as const;

export function Layout() {
  const location = useLocation();
  const { mode, toggle } = useTheme();

  // Dynamic page title (#2150)
  useEffect(() => {
    const match = NAV_ITEMS.find((item) =>
      item.to === '/' ? location.pathname === '/' : location.pathname.startsWith(item.to)
    );
    document.title = match ? `${match.label} \u2014 bc` : 'bc';
  }, [location.pathname]);

  return (
    <div className="flex h-screen">
      <nav className="w-48 shrink-0 border-r border-bc-border bg-bc-surface flex flex-col">
        <div className="p-4 border-b border-bc-border">
          <span className="text-lg font-bold text-bc-accent">bc</span>
          <span className="ml-2 text-xs text-bc-muted">v2</span>
        </div>
        <ul className="flex-1 py-2">
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
