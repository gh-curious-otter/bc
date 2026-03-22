import { useCallback, useEffect, useState } from "react";
import { api } from "../api/client";
import type { SettingsConfig } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

type SaveStatus =
  | { type: "idle" }
  | { type: "saving" }
  | { type: "success" }
  | { type: "error"; message: string };

function StatusMessage({ status }: { status: SaveStatus }) {
  if (status.type === "saving")
    return <span className="text-xs text-bc-muted">Saving...</span>;
  if (status.type === "success")
    return <span className="text-xs text-green-400">Saved</span>;
  if (status.type === "error")
    return <span className="text-xs text-red-400">{status.message}</span>;
  return null;
}

function SectionCard({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded border border-bc-border bg-bc-surface p-5 space-y-4">
      <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">
        {title}
      </h2>
      {children}
    </div>
  );
}

function useSectionSave(refresh: () => void) {
  const [status, setStatus] = useState<SaveStatus>({ type: "idle" });

  const save = useCallback(
    async (patch: Record<string, unknown>) => {
      setStatus({ type: "saving" });
      try {
        await api.updateSettings(patch);
        setStatus({ type: "success" });
        refresh();
        setTimeout(() => setStatus({ type: "idle" }), 2000);
      } catch (err) {
        setStatus({
          type: "error",
          message: err instanceof Error ? err.message : "Save failed",
        });
        setTimeout(() => setStatus({ type: "idle" }), 4000);
      }
    },
    [refresh],
  );

  return { status, save };
}

// --- Section components ---

function UserSection({
  config,
  refresh,
}: {
  config: SettingsConfig;
  refresh: () => void;
}) {
  const [nickname, setNickname] = useState(config.User.Nickname);
  const { status, save } = useSectionSave(refresh);

  useEffect(() => {
    setNickname(config.User.Nickname);
  }, [config.User.Nickname]);

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
        <p className="text-xs text-bc-muted">
          Must start with @ and be 15 characters or less
        </p>
      </div>
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={() => save({ user: { Nickname: nickname } })}
          disabled={status.type === "saving"}
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          Save
        </button>
        <StatusMessage status={status} />
      </div>
    </SectionCard>
  );
}

function UISection({
  config,
  refresh,
}: {
  config: SettingsConfig;
  refresh: () => void;
}) {
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
          disabled={status.type === "saving"}
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          Save
        </button>
        <StatusMessage status={status} />
      </div>
    </SectionCard>
  );
}

function RuntimeSection({
  config,
  refresh,
}: {
  config: SettingsConfig;
  refresh: () => void;
}) {
  const [backend, setBackend] = useState(config.Runtime.Backend);
  const { status, save } = useSectionSave(refresh);

  useEffect(() => {
    setBackend(config.Runtime.Backend);
  }, [config.Runtime.Backend]);

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
          disabled={status.type === "saving"}
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          Save
        </button>
        <StatusMessage status={status} />
      </div>
    </SectionCard>
  );
}

const PROVIDER_NAMES = [
  "claude",
  "gemini",
  "cursor",
  "codex",
  "opencode",
  "openclaw",
  "aider",
] as const;
type ProviderName = (typeof PROVIDER_NAMES)[number];

type ProviderKey =
  | "Claude"
  | "Gemini"
  | "Cursor"
  | "Codex"
  | "OpenCode"
  | "OpenClaw"
  | "Aider";

function providerKey(name: ProviderName): ProviderKey {
  const map: Record<ProviderName, ProviderKey> = {
    claude: "Claude",
    gemini: "Gemini",
    cursor: "Cursor",
    codex: "Codex",
    opencode: "OpenCode",
    openclaw: "OpenClaw",
    aider: "Aider",
  };
  return map[name];
}

function ProvidersSection({
  config,
  refresh,
}: {
  config: SettingsConfig;
  refresh: () => void;
}) {
  const [defaultProvider, setDefaultProvider] = useState(
    config.Providers.Default,
  );
  const { status, save } = useSectionSave(refresh);

  useEffect(() => {
    setDefaultProvider(config.Providers.Default);
  }, [config.Providers.Default]);

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
            <option key={name} value={name}>
              {name}
            </option>
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
          disabled={status.type === "saving"}
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
              <span className="text-sm font-medium text-bc-text">
                {server.name}
              </span>
              <span className="text-xs text-bc-muted">{server.transport}</span>
            </div>
            <span
              className={`text-xs px-2 py-0.5 rounded-full ${
                server.enabled
                  ? "bg-green-400/10 text-green-400"
                  : "bg-red-400/10 text-red-400"
              }`}
            >
              {server.enabled ? "Enabled" : "Disabled"}
            </span>
          </div>
        ))}
      </div>
      <p className="text-xs text-bc-muted">
        MCP server configuration is managed via config.toml
      </p>
    </SectionCard>
  );
}

// --- New section components ---

function EnvSection({
  config,
  refresh,
}: {
  config: SettingsConfig;
  refresh: () => void;
}) {
  const [entries, setEntries] = useState<[string, string][]>(
    Object.entries(config.Env ?? {}),
  );
  const { status, save } = useSectionSave(refresh);

  useEffect(() => {
    setEntries(Object.entries(config.Env ?? {}));
  }, [config.Env]);

  const updateKey = (index: number, key: string) => {
    const next = [...entries] as [string, string][];
    const existing = next[index];
    if (existing) next[index] = [key, existing[1]];
    setEntries(next);
  };

  const updateValue = (index: number, value: string) => {
    const next = [...entries] as [string, string][];
    const existing = next[index];
    if (existing) next[index] = [existing[0], value];
    setEntries(next);
  };

  const addRow = () => setEntries([...entries, ["", ""]]);

  const removeRow = (index: number) =>
    setEntries(entries.filter((_, i) => i !== index));

  const handleSave = () => {
    const env: Record<string, string> = {};
    for (const [k, v] of entries) {
      if (k.trim()) env[k.trim()] = v;
    }
    save({ env: env });
  };

  return (
    <SectionCard title="Environment Variables">
      <div className="space-y-2">
        {entries.map(([key, value], i) => (
          <div key={i} className="flex items-center gap-2">
            <input
              type="text"
              value={key}
              onChange={(e) => updateKey(i, e.target.value)}
              placeholder="KEY"
              className="flex-1 max-w-xs px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
              aria-label={`Environment variable key ${i + 1}`}
            />
            <input
              type="text"
              value={value}
              onChange={(e) => updateValue(i, e.target.value)}
              placeholder="value"
              className="flex-1 px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
              aria-label={`Environment variable value ${i + 1}`}
            />
            <button
              type="button"
              onClick={() => removeRow(i)}
              className="px-2 py-2 rounded text-sm text-red-400 hover:bg-red-400/10 transition-colors"
              aria-label={`Remove environment variable ${i + 1}`}
            >
              Remove
            </button>
          </div>
        ))}
      </div>
      <button
        type="button"
        onClick={addRow}
        className="text-sm text-bc-accent hover:underline"
      >
        + Add Variable
      </button>
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={handleSave}
          disabled={status.type === "saving"}
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          Save
        </button>
        <StatusMessage status={status} />
      </div>
    </SectionCard>
  );
}

function LogsSection({
  config,
  refresh,
}: {
  config: SettingsConfig;
  refresh: () => void;
}) {
  const [path, setPath] = useState(config.Logs?.Path ?? "");
  const [maxBytes, setMaxBytes] = useState(config.Logs?.MaxBytes ?? 0);
  const { status, save } = useSectionSave(refresh);

  useEffect(() => {
    setPath(config.Logs?.Path ?? "");
    setMaxBytes(config.Logs?.MaxBytes ?? 0);
  }, [config.Logs?.Path, config.Logs?.MaxBytes]);

  return (
    <SectionCard title="Logs">
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="space-y-2">
          <label className="block text-sm text-bc-text">Log Path</label>
          <input
            type="text"
            value={path}
            onChange={(e) => setPath(e.target.value)}
            placeholder="/path/to/logs"
            className="w-full max-w-xs px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>
        <div className="space-y-2">
          <label className="block text-sm text-bc-text">Max Bytes</label>
          <input
            type="number"
            value={maxBytes}
            onChange={(e) => setMaxBytes(Number(e.target.value))}
            min={0}
            className="w-full max-w-xs px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>
      </div>
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={() => save({ logs: { Path: path, MaxBytes: maxBytes } })}
          disabled={status.type === "saving"}
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          Save
        </button>
        <StatusMessage status={status} />
      </div>
    </SectionCard>
  );
}

const POLL_INTERVAL_KEYS = [
  "AgentPollMs",
  "ChannelPollMs",
  "CostPollMs",
  "EventPollMs",
  "StatsPollMs",
  "DoctorPollMs",
  "CronPollMs",
  "DaemonPollMs",
  "SecretsPollMs",
  "ToolsPollMs",
  "MCPPollMs",
  "RolesPollMs",
  "WorkspacePollMs",
  "SettingsPollMs",
];

const CACHE_TTL_KEYS = [
  "AgentCacheTTLMs",
  "ChannelCacheTTLMs",
  "CostCacheTTLMs",
  "StatsCacheTTLMs",
];

function formatLabel(key: string): string {
  return key
    .replace(/Ms$/, "")
    .replace(/TTL/, " TTL")
    .replace(/([A-Z])/g, " $1")
    .trim();
}

function PerformanceSection({
  config,
  refresh,
}: {
  config: SettingsConfig;
  refresh: () => void;
}) {
  const [values, setValues] = useState<Record<string, number>>(
    config.Performance ?? {},
  );
  const { status, save } = useSectionSave(refresh);

  useEffect(() => {
    setValues(config.Performance ?? {});
  }, [config.Performance]);

  const updateValue = (key: string, val: number) => {
    setValues((prev) => ({ ...prev, [key]: val }));
  };

  const pollKeys = POLL_INTERVAL_KEYS.filter((k) => k in values);
  const cacheKeys = CACHE_TTL_KEYS.filter((k) => k in values);
  const otherKeys = Object.keys(values).filter(
    (k) => !POLL_INTERVAL_KEYS.includes(k) && !CACHE_TTL_KEYS.includes(k),
  );

  return (
    <SectionCard title="Performance">
      {pollKeys.length > 0 && (
        <div className="space-y-3">
          <h3 className="text-xs font-medium text-bc-muted uppercase tracking-wide">
            Poll Intervals (ms)
          </h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {pollKeys.map((key) => (
              <div key={key} className="space-y-1">
                <label className="block text-sm text-bc-text">
                  {formatLabel(key)}
                </label>
                <input
                  type="number"
                  value={values[key] ?? 0}
                  onChange={(e) => updateValue(key, Number(e.target.value))}
                  min={0}
                  className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
                />
              </div>
            ))}
          </div>
        </div>
      )}
      {cacheKeys.length > 0 && (
        <div className="space-y-3">
          <h3 className="text-xs font-medium text-bc-muted uppercase tracking-wide">
            Cache TTLs (ms)
          </h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {cacheKeys.map((key) => (
              <div key={key} className="space-y-1">
                <label className="block text-sm text-bc-text">
                  {formatLabel(key)}
                </label>
                <input
                  type="number"
                  value={values[key] ?? 0}
                  onChange={(e) => updateValue(key, Number(e.target.value))}
                  min={0}
                  className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
                />
              </div>
            ))}
          </div>
        </div>
      )}
      {otherKeys.length > 0 && (
        <div className="space-y-3">
          <h3 className="text-xs font-medium text-bc-muted uppercase tracking-wide">
            Other
          </h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {otherKeys.map((key) => (
              <div key={key} className="space-y-1">
                <label className="block text-sm text-bc-text">
                  {formatLabel(key)}
                </label>
                <input
                  type="number"
                  value={values[key] ?? 0}
                  onChange={(e) => updateValue(key, Number(e.target.value))}
                  min={0}
                  className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
                />
              </div>
            ))}
          </div>
        </div>
      )}
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={() => save({ performance: values })}
          disabled={status.type === "saving"}
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          Save
        </button>
        <StatusMessage status={status} />
      </div>
    </SectionCard>
  );
}

function DockerSection({
  config,
  refresh,
}: {
  config: SettingsConfig;
  refresh: () => void;
}) {
  const docker = config.Runtime.Docker;
  const [image, setImage] = useState(docker?.Image ?? "");
  const [network, setNetwork] = useState(docker?.Network ?? "");
  const [cpus, setCpus] = useState(docker?.CPUs ?? 0);
  const [memoryMB, setMemoryMB] = useState(docker?.MemoryMB ?? 0);
  const { status, save } = useSectionSave(refresh);

  useEffect(() => {
    setImage(docker?.Image ?? "");
    setNetwork(docker?.Network ?? "");
    setCpus(docker?.CPUs ?? 0);
    setMemoryMB(docker?.MemoryMB ?? 0);
  }, [docker?.Image, docker?.Network, docker?.CPUs, docker?.MemoryMB]);

  if (config.Runtime.Backend !== "docker") return null;

  return (
    <SectionCard title="Docker Runtime">
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="space-y-2">
          <label className="block text-sm text-bc-text">Image</label>
          <input
            type="text"
            value={image}
            onChange={(e) => setImage(e.target.value)}
            placeholder="docker-image:tag"
            className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>
        <div className="space-y-2">
          <label className="block text-sm text-bc-text">Network</label>
          <input
            type="text"
            value={network}
            onChange={(e) => setNetwork(e.target.value)}
            placeholder="bridge"
            className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>
        <div className="space-y-2">
          <label className="block text-sm text-bc-text">CPUs</label>
          <input
            type="number"
            value={cpus}
            onChange={(e) => setCpus(Number(e.target.value))}
            min={0}
            step={0.5}
            className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>
        <div className="space-y-2">
          <label className="block text-sm text-bc-text">Memory (MB)</label>
          <input
            type="number"
            value={memoryMB}
            onChange={(e) => setMemoryMB(Number(e.target.value))}
            min={0}
            className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>
      </div>
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={() =>
            save({
              runtime: {
                Backend: config.Runtime.Backend,
                Docker: {
                  Image: image,
                  Network: network,
                  CPUs: cpus,
                  MemoryMB: memoryMB,
                },
              },
            })
          }
          disabled={status.type === "saving"}
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          Save
        </button>
        <StatusMessage status={status} />
      </div>
    </SectionCard>
  );
}

const SERVICE_NAMES = ["GitHub", "GitLab", "Jira"] as const;
type ServiceName = (typeof SERVICE_NAMES)[number];

function ServicesSection({
  config,
  refresh,
}: {
  config: SettingsConfig;
  refresh: () => void;
}) {
  const [services, setServices] = useState<
    Record<ServiceName, { Command: string; Enabled: boolean }>
  >(() => {
    const result = {} as Record<
      ServiceName,
      { Command: string; Enabled: boolean }
    >;
    for (const name of SERVICE_NAMES) {
      result[name] = {
        Command: config.Services?.[name]?.Command ?? "",
        Enabled: config.Services?.[name]?.Enabled ?? false,
      };
    }
    return result;
  });
  const { status, save } = useSectionSave(refresh);

  useEffect(() => {
    const result = {} as Record<
      ServiceName,
      { Command: string; Enabled: boolean }
    >;
    for (const name of SERVICE_NAMES) {
      result[name] = {
        Command: config.Services?.[name]?.Command ?? "",
        Enabled: config.Services?.[name]?.Enabled ?? false,
      };
    }
    setServices(result);
  }, [config.Services]);

  const updateCommand = (name: ServiceName, command: string) => {
    setServices((prev) => ({
      ...prev,
      [name]: { ...prev[name], Command: command },
    }));
  };

  const toggleEnabled = (name: ServiceName) => {
    setServices((prev) => ({
      ...prev,
      [name]: { ...prev[name], Enabled: !prev[name].Enabled },
    }));
  };

  return (
    <SectionCard title="Services">
      <div className="space-y-4">
        {SERVICE_NAMES.map((name) => (
          <div
            key={name}
            className="flex items-center gap-4 px-4 py-3 rounded border border-bc-border bg-bc-bg"
          >
            <div className="flex-shrink-0 w-16">
              <span className="text-sm font-medium text-bc-text">{name}</span>
            </div>
            <input
              type="text"
              value={services[name].Command}
              onChange={(e) => updateCommand(name, e.target.value)}
              placeholder="command"
              className="flex-1 px-3 py-2 rounded border border-bc-border bg-bc-surface text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
              aria-label={`${name} command`}
            />
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={services[name].Enabled}
                onChange={() => toggleEnabled(name)}
                className="w-4 h-4 rounded border-bc-border text-bc-accent focus:ring-bc-accent"
                aria-label={`Enable ${name}`}
              />
              <span className="text-sm text-bc-muted">Enabled</span>
            </label>
          </div>
        ))}
      </div>
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={() => save({ services: services })}
          disabled={status.type === "saving"}
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          Save
        </button>
        <StatusMessage status={status} />
      </div>
    </SectionCard>
  );
}

function ServerSection({
  config,
  refresh,
}: {
  config: SettingsConfig;
  refresh: () => void;
}) {
  const [addr, setAddr] = useState(config.Server?.Addr ?? "");
  const [corsOrigin, setCorsOrigin] = useState(config.Server?.CORSOrigin ?? "");
  const { status, save } = useSectionSave(refresh);

  useEffect(() => {
    setAddr(config.Server?.Addr ?? "");
    setCorsOrigin(config.Server?.CORSOrigin ?? "");
  }, [config.Server?.Addr, config.Server?.CORSOrigin]);

  return (
    <SectionCard title="Server">
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="space-y-2">
          <label className="block text-sm text-bc-text">Address</label>
          <input
            type="text"
            value={addr}
            onChange={(e) => setAddr(e.target.value)}
            placeholder=":9374"
            className="w-full max-w-xs px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>
        <div className="space-y-2">
          <label className="block text-sm text-bc-text">CORS Origin</label>
          <input
            type="text"
            value={corsOrigin}
            onChange={(e) => setCorsOrigin(e.target.value)}
            placeholder="*"
            className="w-full max-w-xs px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>
      </div>
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={() =>
            save({ server: { Addr: addr, CORSOrigin: corsOrigin } })
          }
          disabled={status.type === "saving"}
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          Save
        </button>
        <StatusMessage status={status} />
      </div>
    </SectionCard>
  );
}

// --- Main component ---

export function Settings() {
  const fetcher = useCallback(() => api.getSettings(), []);
  const {
    data: config,
    loading,
    error,
    refresh,
    timedOut,
  } = usePolling(fetcher, 30000);

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
      <DockerSection config={config} refresh={refresh} />
      <ProvidersSection config={config} refresh={refresh} />
      <MCPSection refresh={refresh} />
      <EnvSection config={config} refresh={refresh} />
      <LogsSection config={config} refresh={refresh} />
      <PerformanceSection config={config} refresh={refresh} />
      <ServicesSection config={config} refresh={refresh} />
      <ServerSection config={config} refresh={refresh} />
    </div>
  );
}
