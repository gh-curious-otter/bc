import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client';
import type { SettingsConfig } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import { EmptyState } from '../components/EmptyState';

type SaveStatus = { type: 'idle' } | { type: 'saving' } | { type: 'success' } | { type: 'error'; message: string };

function StatusMessage({ status }: { status: SaveStatus }) {
  if (status.type === 'saving') return <span className="text-xs text-bc-muted">Saving...</span>;
  if (status.type === 'success') return <span className="text-xs text-green-400">Saved</span>;
  if (status.type === 'error') return <span className="text-xs text-red-400">{status.message}</span>;
  return null;
}

function SectionCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded border border-bc-border bg-bc-surface p-5 space-y-4">
      <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">{title}</h2>
      {children}
    </div>
  );
}

function useSectionSave(refresh: () => void) {
  const [status, setStatus] = useState<SaveStatus>({ type: 'idle' });

  const save = useCallback(async (patch: Record<string, unknown>) => {
    setStatus({ type: 'saving' });
    try {
      await api.updateSettings(patch);
      setStatus({ type: 'success' });
      refresh();
      setTimeout(() => setStatus({ type: 'idle' }), 2000);
    } catch (err) {
      setStatus({ type: 'error', message: err instanceof Error ? err.message : 'Save failed' });
      setTimeout(() => setStatus({ type: 'idle' }), 4000);
    }
  }, [refresh]);

  return { status, save };
}

// --- Section components ---

function UserSection({ config, refresh }: { config: SettingsConfig; refresh: () => void }) {
  const [nickname, setNickname] = useState(config.User.Nickname);
  const { status, save } = useSectionSave(refresh);

  useEffect(() => { setNickname(config.User.Nickname); }, [config.User.Nickname]);

  return (
    <SectionCard title="User">
      <div className="space-y-2">
        <label className="block text-sm text-bc-text">Nickname</label>
        <input
          type="text"
          value={nickname}
          onChange={(e) => setNickname(e.target.value)}
          placeholder="@username"
          maxLength={15}
          className="w-full max-w-xs px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
        />
        <p className="text-xs text-bc-muted">Must start with @ and be 15 characters or less</p>
      </div>
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={() => save({ user: { Nickname: nickname } })}
          disabled={status.type === 'saving'}
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          Save
        </button>
        <StatusMessage status={status} />
      </div>
    </SectionCard>
  );
}

function UISection({ config, refresh }: { config: SettingsConfig; refresh: () => void }) {
  const [theme, setTheme] = useState(config.TUI.Theme);
  const [mode, setMode] = useState(config.TUI.Mode);
  const { status, save } = useSectionSave(refresh);

  useEffect(() => {
    setTheme(config.TUI.Theme);
    setMode(config.TUI.Mode);
  }, [config.TUI.Theme, config.TUI.Mode]);

  return (
    <SectionCard title="UI">
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="space-y-2">
          <label className="block text-sm text-bc-text">Theme</label>
          <select
            value={theme}
            onChange={(e) => setTheme(e.target.value)}
            className="w-full max-w-xs px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          >
            <option value="dark">Dark</option>
            <option value="light">Light</option>
            <option value="matrix">Matrix</option>
            <option value="synthwave">Synthwave</option>
            <option value="high-contrast">High Contrast</option>
          </select>
        </div>
        <div className="space-y-2">
          <label className="block text-sm text-bc-text">Color Mode</label>
          <select
            value={mode}
            onChange={(e) => setMode(e.target.value)}
            className="w-full max-w-xs px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          >
            <option value="auto">System</option>
            <option value="dark">Dark</option>
            <option value="light">Light</option>
          </select>
        </div>
      </div>
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={() => save({ tui: { Theme: theme, Mode: mode } })}
          disabled={status.type === 'saving'}
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          Save
        </button>
        <StatusMessage status={status} />
      </div>
    </SectionCard>
  );
}

function RuntimeSection({ config, refresh }: { config: SettingsConfig; refresh: () => void }) {
  const [backend, setBackend] = useState(config.Runtime.Backend);
  const { status, save } = useSectionSave(refresh);

  useEffect(() => { setBackend(config.Runtime.Backend); }, [config.Runtime.Backend]);

  return (
    <SectionCard title="Runtime">
      <div className="space-y-2">
        <label className="block text-sm text-bc-text">Backend</label>
        <select
          value={backend}
          onChange={(e) => setBackend(e.target.value)}
          className="w-full max-w-xs px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
        >
          <option value="tmux">tmux</option>
          <option value="docker">Docker</option>
        </select>
      </div>
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={() => save({ runtime: { Backend: backend } })}
          disabled={status.type === 'saving'}
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          Save
        </button>
        <StatusMessage status={status} />
      </div>
    </SectionCard>
  );
}

const PROVIDER_NAMES = ['claude', 'gemini', 'cursor', 'codex', 'opencode', 'openclaw', 'aider'] as const;
type ProviderName = typeof PROVIDER_NAMES[number];

type ProviderKey = 'Claude' | 'Gemini' | 'Cursor' | 'Codex' | 'OpenCode' | 'OpenClaw' | 'Aider';

function providerKey(name: ProviderName): ProviderKey {
  const map: Record<ProviderName, ProviderKey> = {
    claude: 'Claude', gemini: 'Gemini', cursor: 'Cursor',
    codex: 'Codex', opencode: 'OpenCode', openclaw: 'OpenClaw', aider: 'Aider',
  };
  return map[name];
}

function ProvidersSection({ config, refresh }: { config: SettingsConfig; refresh: () => void }) {
  const [defaultProvider, setDefaultProvider] = useState(config.Providers.Default);
  const { status, save } = useSectionSave(refresh);

  useEffect(() => { setDefaultProvider(config.Providers.Default); }, [config.Providers.Default]);

  const enabledProviders = PROVIDER_NAMES.filter((name) => {
    const cfg = config.Providers[providerKey(name)];
    return cfg && cfg.Enabled;
  });

  const allDefined = PROVIDER_NAMES.filter((name) => {
    return config.Providers[providerKey(name)] != null;
  });

  return (
    <SectionCard title="Providers">
      <div className="space-y-2">
        <label className="block text-sm text-bc-text">Default Provider</label>
        <select
          value={defaultProvider}
          onChange={(e) => setDefaultProvider(e.target.value)}
          className="w-full max-w-xs px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
        >
          {allDefined.map((name) => (
            <option key={name} value={name}>{name}</option>
          ))}
        </select>
      </div>
      <div className="space-y-2">
        <label className="block text-sm text-bc-text">Enabled Providers</label>
        <div className="flex flex-wrap gap-2">
          {enabledProviders.length === 0 ? (
            <span className="text-sm text-bc-muted">No providers enabled</span>
          ) : (
            enabledProviders.map((name) => (
              <span
                key={name}
                className="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-bc-bg border border-bc-border text-bc-text"
              >
                {name}
              </span>
            ))
          )}
        </div>
      </div>
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={() => save({ providers: { Default: defaultProvider } })}
          disabled={status.type === 'saving'}
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          Save
        </button>
        <StatusMessage status={status} />
      </div>
    </SectionCard>
  );
}

function MCPSection(_props: { refresh: () => void }) {
  const mcpFetcher = useCallback(() => api.listMCP(), []);
  const { data: servers } = usePolling(mcpFetcher, 30000);

  if (!servers || servers.length === 0) {
    return (
      <SectionCard title="MCP Servers">
        <p className="text-sm text-bc-muted">No MCP servers configured</p>
      </SectionCard>
    );
  }

  return (
    <SectionCard title="MCP Servers">
      <div className="space-y-2">
        {servers.map((server) => (
          <div
            key={server.name}
            className="flex items-center justify-between px-4 py-2 rounded border border-bc-border bg-bc-bg"
          >
            <div className="flex items-center gap-3">
              <span className="text-sm font-medium text-bc-text">{server.name}</span>
              <span className="text-xs text-bc-muted">{server.transport}</span>
            </div>
            <span
              className={`text-xs px-2 py-0.5 rounded-full ${
                server.enabled
                  ? 'bg-green-400/10 text-green-400'
                  : 'bg-red-400/10 text-red-400'
              }`}
            >
              {server.enabled ? 'Enabled' : 'Disabled'}
            </span>
          </div>
        ))}
      </div>
      <p className="text-xs text-bc-muted">MCP server configuration is managed via config.toml</p>
    </SectionCard>
  );
}

// --- Main component ---

export function Settings() {
  const fetcher = useCallback(() => api.getSettings(), []);
  const { data: config, loading, error, refresh, timedOut } = usePolling(fetcher, 30000);

  if (loading && !config) {
    return (
      <div className="p-6 space-y-6">
        <div className="h-6 w-32 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="cards" rows={4} />
      </div>
    );
  }
  if (timedOut && !config) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Settings took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !config) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load settings"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (!config) return null;

  return (
    <div className="p-6 space-y-6">
      <h1 className="text-xl font-bold">Settings</h1>

      <UserSection config={config} refresh={refresh} />
      <UISection config={config} refresh={refresh} />
      <RuntimeSection config={config} refresh={refresh} />
      <ProvidersSection config={config} refresh={refresh} />
      <MCPSection refresh={refresh} />
    </div>
  );
}
