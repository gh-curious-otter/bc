import { useCallback, useState, useEffect } from "react";
import { api } from "../api/client";
import type { SettingsConfig } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

type SaveStatus = "idle" | "saving" | "saved" | "error";

/* ------------------------------------------------------------------ */
/*  Shared primitives                                                  */
/* ------------------------------------------------------------------ */

function Section({
  title,
  description,
  defaultOpen = true,
  dirty = false,
  children,
}: {
  title: string;
  description?: string;
  defaultOpen?: boolean;
  dirty?: boolean;
  children: React.ReactNode;
}) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <div className="rounded-lg border border-bc-border bg-bc-surface overflow-hidden">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="w-full flex items-center justify-between px-5 py-3 hover:bg-bc-bg/50 transition-colors"
      >
        <div className="flex items-center gap-3">
          <svg
            className={`w-4 h-4 text-bc-muted transition-transform ${open ? "rotate-90" : ""}`}
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2}
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
          </svg>
          <h2 className="text-sm font-semibold text-bc-text uppercase tracking-wide">{title}</h2>
          {dirty && (
            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-bc-accent/20 text-bc-accent">
              unsaved
            </span>
          )}
        </div>
        {description && <span className="text-xs text-bc-muted hidden sm:block">{description}</span>}
      </button>
      {open && <div className="px-5 pb-5 pt-2 space-y-4 border-t border-bc-border">{children}</div>}
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-start gap-4">
      <label className="text-sm text-bc-muted w-44 shrink-0 pt-1.5">{label}</label>
      <div className="flex-1 min-w-0">{children}</div>
    </div>
  );
}

function Input({
  value,
  onChange,
  placeholder,
  type = "text",
  disabled = false,
}: {
  value: string | number;
  onChange: (v: string) => void;
  placeholder?: string;
  type?: string;
  disabled?: boolean;
}) {
  return (
    <input
      type={type}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      disabled={disabled}
      className="w-full px-3 py-1.5 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent disabled:opacity-50"
    />
  );
}

function PasswordInput({
  value,
  onChange,
  placeholder,
}: {
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
}) {
  const [visible, setVisible] = useState(false);
  return (
    <div className="relative">
      <input
        type={visible ? "text" : "password"}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="w-full px-3 py-1.5 pr-10 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent font-mono"
      />
      <button
        type="button"
        onClick={() => setVisible(!visible)}
        className="absolute inset-y-0 right-0 flex items-center px-3 text-bc-muted hover:text-bc-text"
        tabIndex={-1}
      >
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          {visible ? (
            <path strokeLinecap="round" strokeLinejoin="round" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.878 9.878L3 3m6.878 6.878L21 21" />
          ) : (
            <>
              <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              <path strokeLinecap="round" strokeLinejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
            </>
          )}
        </svg>
      </button>
    </div>
  );
}

function Select({
  value,
  onChange,
  options,
}: {
  value: string;
  onChange: (v: string) => void;
  options: { value: string; label: string }[];
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="w-full px-3 py-1.5 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
    >
      {options.map((o) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
    </select>
  );
}

function Toggle({ checked, onChange }: { checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <button
      type="button"
      onClick={() => onChange(!checked)}
      className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${checked ? "bg-bc-accent" : "bg-bc-border"}`}
    >
      <span
        className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${checked ? "translate-x-6" : "translate-x-1"}`}
      />
    </button>
  );
}

function SaveButton({ status, onClick, disabled = false }: { status: SaveStatus; onClick: () => void; disabled?: boolean }) {
  const label = status === "saving" ? "Saving..." : status === "saved" ? "Saved!" : status === "error" ? "Error - Retry" : "Save";
  return (
    <div className="flex items-center gap-3 pt-2">
      <button
        onClick={onClick}
        disabled={disabled || status === "saving"}
        className={`px-4 py-1.5 rounded text-sm font-medium transition-all disabled:opacity-50 ${
          status === "error"
            ? "bg-red-600 text-white hover:bg-red-700"
            : status === "saved"
              ? "bg-green-600 text-white"
              : "bg-bc-accent text-white hover:opacity-90"
        }`}
      >
        {label}
      </button>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Helper: deep equality for dirty detection                          */
/* ------------------------------------------------------------------ */
function deepEqual(a: unknown, b: unknown): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}

/* ------------------------------------------------------------------ */
/*  Section components                                                 */
/* ------------------------------------------------------------------ */

function ServerSection({
  config,
  onSave,
}: {
  config: SettingsConfig;
  onSave: (patch: Record<string, unknown>) => Promise<void>;
}) {
  const [host, setHost] = useState(config.server.host);
  const [port, setPort] = useState(String(config.server.port));
  const [cors, setCors] = useState(config.server.cors_origin);
  const [status, setStatus] = useState<SaveStatus>("idle");

  useEffect(() => {
    setHost(config.server.host);
    setPort(String(config.server.port));
    setCors(config.server.cors_origin);
  }, [config.server.host, config.server.port, config.server.cors_origin]);

  const dirty =
    host !== config.server.host ||
    port !== String(config.server.port) ||
    cors !== config.server.cors_origin;

  const save = async () => {
    setStatus("saving");
    try {
      await onSave({ server: { host, port: parseInt(port, 10), cors_origin: cors } });
      setStatus("saved");
      setTimeout(() => setStatus("idle"), 2000);
    } catch {
      setStatus("error");
    }
  };

  return (
    <Section title="Server" description="HTTP server configuration" dirty={dirty}>
      <Field label="Host">
        <Input value={host} onChange={setHost} placeholder="0.0.0.0" />
      </Field>
      <Field label="Port">
        <Input value={port} onChange={setPort} type="number" placeholder="9374" />
      </Field>
      <Field label="CORS Origin">
        <Input value={cors} onChange={setCors} placeholder="*" />
      </Field>
      <SaveButton status={status} onClick={save} disabled={!dirty} />
    </Section>
  );
}

function StorageSection({
  config,
  onSave,
}: {
  config: SettingsConfig;
  onSave: (patch: Record<string, unknown>) => Promise<void>;
}) {
  const s = config.storage;
  const [backend, setBackend] = useState(s.default);
  const [sqlitePath, setSqlitePath] = useState(s.sqlite?.path ?? "");
  const [sqlHost, setSqlHost] = useState(s.timescale?.host ?? "");
  const [sqlPort, setSqlPort] = useState(String(s.timescale?.port ?? 5432));
  const [sqlUser, setSqlUser] = useState(s.timescale?.user ?? "");
  const [sqlPassword, setSqlPassword] = useState(s.timescale?.password ?? "");
  const [sqlDatabase, setSqlDatabase] = useState(s.timescale?.database ?? "");
  const [status, setStatus] = useState<SaveStatus>("idle");

  useEffect(() => {
    setBackend(s.default);
    setSqlitePath(s.sqlite?.path ?? "");
    setSqlHost(s.timescale?.host ?? "");
    setSqlPort(String(s.timescale?.port ?? 5432));
    setSqlUser(s.timescale?.user ?? "");
    setSqlPassword(s.timescale?.password ?? "");
    setSqlDatabase(s.timescale?.database ?? "");
  }, [s.default, s.sqlite?.path, s.timescale?.host, s.timescale?.port, s.timescale?.user, s.timescale?.password, s.timescale?.database]);

  const current = {
    default: backend,
    sqlite: { path: sqlitePath },
    timescale: { host: sqlHost, port: parseInt(sqlPort, 10), user: sqlUser, password: sqlPassword, database: sqlDatabase },
  };
  const dirty = !deepEqual(current, { default: s.default, sqlite: s.sqlite, timescale: s.timescale });

  const save = async () => {
    setStatus("saving");
    try {
      await onSave({ storage: current });
      setStatus("saved");
      setTimeout(() => setStatus("idle"), 2000);
    } catch {
      setStatus("error");
    }
  };

  return (
    <Section title="Storage" description="Database backend settings" dirty={dirty}>
      <Field label="Backend">
        <Select
          value={backend}
          onChange={setBackend}
          options={[
            { value: "sqlite", label: "SQLite" },
            { value: "timescale", label: "TimescaleDB" },
          ]}
        />
      </Field>
      {backend === "sqlite" && (
        <Field label="SQLite Path">
          <Input value={sqlitePath} onChange={setSqlitePath} placeholder=".bc/bc.db" />
        </Field>
      )}
      {(backend === "timescale" || backend === "sql") && (
        <>
          <Field label="Host">
            <Input value={sqlHost} onChange={setSqlHost} placeholder="localhost" />
          </Field>
          <Field label="Port">
            <Input value={sqlPort} onChange={setSqlPort} type="number" placeholder="5432" />
          </Field>
          <Field label="User">
            <Input value={sqlUser} onChange={setSqlUser} placeholder="postgres" />
          </Field>
          <Field label="Password">
            <PasswordInput value={sqlPassword} onChange={setSqlPassword} placeholder="password" />
          </Field>
          <Field label="Database">
            <Input value={sqlDatabase} onChange={setSqlDatabase} placeholder="bc" />
          </Field>
        </>
      )}
      <SaveButton status={status} onClick={save} disabled={!dirty} />
    </Section>
  );
}

function RuntimeSection({
  config,
  onSave,
}: {
  config: SettingsConfig;
  onSave: (patch: Record<string, unknown>) => Promise<void>;
}) {
  const rt = config.runtime;
  const [backend, setBackend] = useState(rt.default);
  const [dockerImage, setDockerImage] = useState(rt.docker?.image ?? "");
  const [dockerNetwork, setDockerNetwork] = useState(rt.docker?.network ?? "");
  const [dockerCpus, setDockerCpus] = useState(String(rt.docker?.cpus ?? 2));
  const [dockerMemory, setDockerMemory] = useState(String(rt.docker?.memory_mb ?? 2048));
  const [tmuxPrefix, setTmuxPrefix] = useState(rt.tmux?.session_prefix ?? "");
  const [tmuxHistory, setTmuxHistory] = useState(String(rt.tmux?.history_limit ?? 10000));
  const [status, setStatus] = useState<SaveStatus>("idle");

  useEffect(() => {
    setBackend(rt.default);
    setDockerImage(rt.docker?.image ?? "");
    setDockerNetwork(rt.docker?.network ?? "");
    setDockerCpus(String(rt.docker?.cpus ?? 2));
    setDockerMemory(String(rt.docker?.memory_mb ?? 2048));
    setTmuxPrefix(rt.tmux?.session_prefix ?? "");
    setTmuxHistory(String(rt.tmux?.history_limit ?? 10000));
  }, [rt.default, rt.docker?.image, rt.docker?.network, rt.docker?.cpus, rt.docker?.memory_mb, rt.tmux?.session_prefix, rt.tmux?.history_limit]);

  const buildPatch = () => ({
    ...config.runtime,
    default: backend,
    docker: {
      ...config.runtime.docker,
      image: dockerImage,
      network: dockerNetwork,
      cpus: parseFloat(dockerCpus) || 0,
      memory_mb: parseInt(dockerMemory, 10) || 0,
    },
    tmux: {
      ...config.runtime.tmux,
      session_prefix: tmuxPrefix,
      history_limit: parseInt(tmuxHistory, 10) || 10000,
    },
  });

  const dirty = !deepEqual(buildPatch(), config.runtime);

  const save = async () => {
    setStatus("saving");
    try {
      await onSave({ runtime: buildPatch() });
      setStatus("saved");
      setTimeout(() => setStatus("idle"), 2000);
    } catch {
      setStatus("error");
    }
  };

  return (
    <Section title="Runtime" description="Agent execution backend" dirty={dirty}>
      <Field label="Default Backend">
        <Select
          value={backend}
          onChange={setBackend}
          options={[
            { value: "tmux", label: "tmux" },
            { value: "docker", label: "Docker" },
          ]}
        />
      </Field>
      <div className="border-t border-bc-border pt-3 mt-1">
        <p className="text-xs text-bc-muted font-medium uppercase tracking-wide mb-3">Docker</p>
        <div className="space-y-3">
          <Field label="Image">
            <Input value={dockerImage} onChange={setDockerImage} placeholder="bc-agent:latest" />
          </Field>
          <Field label="Network">
            <Input value={dockerNetwork} onChange={setDockerNetwork} placeholder="bc-net" />
          </Field>
          <Field label="CPUs">
            <Input value={dockerCpus} onChange={setDockerCpus} type="number" placeholder="2" />
          </Field>
          <Field label="Memory (MB)">
            <Input value={dockerMemory} onChange={setDockerMemory} type="number" placeholder="2048" />
          </Field>
        </div>
      </div>
      <div className="border-t border-bc-border pt-3 mt-1">
        <p className="text-xs text-bc-muted font-medium uppercase tracking-wide mb-3">tmux</p>
        <div className="space-y-3">
          <Field label="Session Prefix">
            <Input value={tmuxPrefix} onChange={setTmuxPrefix} placeholder="bc-" />
          </Field>
          <Field label="History Limit">
            <Input value={tmuxHistory} onChange={setTmuxHistory} type="number" placeholder="10000" />
          </Field>
        </div>
      </div>
      <SaveButton status={status} onClick={save} disabled={!dirty} />
    </Section>
  );
}

function GatewaysSection({
  config,
  onSave,
}: {
  config: SettingsConfig;
  onSave: (patch: Record<string, unknown>) => Promise<void>;
}) {
  const gw = config.gateways;
  const [telegramEnabled, setTelegramEnabled] = useState(gw.telegram?.enabled ?? false);
  const [telegramToken, setTelegramToken] = useState(gw.telegram?.bot_token ?? "");
  const [slackEnabled, setSlackEnabled] = useState(gw.slack?.enabled ?? false);
  const [slackBotToken, setSlackBotToken] = useState(gw.slack?.bot_token ?? "");
  const [slackAppToken, setSlackAppToken] = useState(gw.slack?.app_token ?? "");
  const [status, setStatus] = useState<SaveStatus>("idle");

  useEffect(() => {
    setTelegramEnabled(gw.telegram?.enabled ?? false);
    setTelegramToken(gw.telegram?.bot_token ?? "");
    setSlackEnabled(gw.slack?.enabled ?? false);
    setSlackBotToken(gw.slack?.bot_token ?? "");
    setSlackAppToken(gw.slack?.app_token ?? "");
  }, [gw.telegram?.enabled, gw.telegram?.bot_token, gw.slack?.enabled, gw.slack?.bot_token, gw.slack?.app_token]);

  const buildPatch = () => ({
    ...config.gateways,
    telegram: {
      ...config.gateways.telegram,
      enabled: telegramEnabled,
      bot_token: telegramToken,
    },
    slack: {
      ...config.gateways.slack,
      enabled: slackEnabled,
      bot_token: slackBotToken,
      app_token: slackAppToken,
    },
  });

  const dirty = !deepEqual(buildPatch(), config.gateways);

  const save = async () => {
    setStatus("saving");
    try {
      await onSave({ gateways: buildPatch() });
      setStatus("saved");
      setTimeout(() => setStatus("idle"), 2000);
    } catch {
      setStatus("error");
    }
  };

  return (
    <Section title="Gateways" description="External messaging integrations" dirty={dirty}>
      <div className="border-b border-bc-border pb-4">
        <div className="flex items-center justify-between mb-3">
          <p className="text-xs text-bc-muted font-medium uppercase tracking-wide">Telegram</p>
          <Toggle checked={telegramEnabled} onChange={setTelegramEnabled} />
        </div>
        <Field label="Bot Token">
          <PasswordInput value={telegramToken} onChange={setTelegramToken} placeholder="123456:ABC-DEF..." />
        </Field>
      </div>
      <div className="pt-1">
        <div className="flex items-center justify-between mb-3">
          <p className="text-xs text-bc-muted font-medium uppercase tracking-wide">Slack</p>
          <Toggle checked={slackEnabled} onChange={setSlackEnabled} />
        </div>
        <div className="space-y-3">
          <Field label="Bot Token">
            <PasswordInput value={slackBotToken} onChange={setSlackBotToken} placeholder="xoxb-..." />
          </Field>
          <Field label="App Token">
            <PasswordInput value={slackAppToken} onChange={setSlackAppToken} placeholder="xapp-..." />
          </Field>
        </div>
      </div>
      <SaveButton status={status} onClick={save} disabled={!dirty} />
    </Section>
  );
}

function CronSection({
  config,
  onSave,
}: {
  config: SettingsConfig;
  onSave: (patch: Record<string, unknown>) => Promise<void>;
}) {
  const c = config.cron;
  const [pollInterval, setPollInterval] = useState(String(c.poll_interval_seconds));
  const [jobTimeout, setJobTimeout] = useState(String(c.job_timeout_seconds));
  const [status, setStatus] = useState<SaveStatus>("idle");

  useEffect(() => {
    setPollInterval(String(c.poll_interval_seconds));
    setJobTimeout(String(c.job_timeout_seconds));
  }, [c.poll_interval_seconds, c.job_timeout_seconds]);

  const dirty =
    pollInterval !== String(c.poll_interval_seconds) || jobTimeout !== String(c.job_timeout_seconds);

  const save = async () => {
    setStatus("saving");
    try {
      await onSave({
        cron: {
          poll_interval_seconds: parseInt(pollInterval, 10),
          job_timeout_seconds: parseInt(jobTimeout, 10),
        },
      });
      setStatus("saved");
      setTimeout(() => setStatus("idle"), 2000);
    } catch {
      setStatus("error");
    }
  };

  return (
    <Section title="Cron" description="Scheduled job settings" dirty={dirty}>
      <Field label="Poll Interval (s)">
        <Input value={pollInterval} onChange={setPollInterval} type="number" placeholder="10" />
      </Field>
      <Field label="Job Timeout (s)">
        <Input value={jobTimeout} onChange={setJobTimeout} type="number" placeholder="300" />
      </Field>
      <SaveButton status={status} onClick={save} disabled={!dirty} />
    </Section>
  );
}

function ProvidersSection({
  config,
  onSave,
}: {
  config: SettingsConfig;
  onSave: (patch: Record<string, unknown>) => Promise<void>;
}) {
  const p = config.providers;
  const [defaultProvider, setDefaultProvider] = useState(p.default);
  const [providers, setProviders] = useState<Record<string, { command: string }>>(p.providers ?? {});
  const [status, setStatus] = useState<SaveStatus>("idle");

  useEffect(() => {
    setDefaultProvider(p.default);
    setProviders(p.providers ?? {});
  }, [p.default, p.providers]);

  const dirty = !deepEqual({ default: defaultProvider, providers }, { default: p.default, providers: p.providers ?? {} });

  const updateProviderCommand = (name: string, command: string) => {
    setProviders((prev) => ({ ...prev, [name]: { command } }));
  };

  const providerNames = Object.keys(providers);

  const save = async () => {
    setStatus("saving");
    try {
      await onSave({ providers: { default: defaultProvider, providers } });
      setStatus("saved");
      setTimeout(() => setStatus("idle"), 2000);
    } catch {
      setStatus("error");
    }
  };

  return (
    <Section title="Providers" description="AI provider configuration" dirty={dirty}>
      <Field label="Default Provider">
        {providerNames.length > 0 ? (
          <Select
            value={defaultProvider}
            onChange={setDefaultProvider}
            options={providerNames.map((n) => ({ value: n, label: n }))}
          />
        ) : (
          <Input value={defaultProvider} onChange={setDefaultProvider} placeholder="claude" />
        )}
      </Field>
      {providerNames.map((name) => (
        <Field key={name} label={name}>
          <Input
            value={providers[name]?.command ?? ""}
            onChange={(v) => updateProviderCommand(name, v)}
            placeholder="command"
          />
        </Field>
      ))}
      <SaveButton status={status} onClick={save} disabled={!dirty} />
    </Section>
  );
}

function LogsSection({
  config,
  onSave,
}: {
  config: SettingsConfig;
  onSave: (patch: Record<string, unknown>) => Promise<void>;
}) {
  const l = config.logs;
  const [path, setPath] = useState(l.path);
  const [maxBytes, setMaxBytes] = useState(String(l.max_bytes));
  const [status, setStatus] = useState<SaveStatus>("idle");

  useEffect(() => {
    setPath(l.path);
    setMaxBytes(String(l.max_bytes));
  }, [l.path, l.max_bytes]);

  const dirty = path !== l.path || maxBytes !== String(l.max_bytes);

  const save = async () => {
    setStatus("saving");
    try {
      await onSave({ logs: { path, max_bytes: parseInt(maxBytes, 10) } });
      setStatus("saved");
      setTimeout(() => setStatus("idle"), 2000);
    } catch {
      setStatus("error");
    }
  };

  return (
    <Section title="Logs" description="Log file settings" dirty={dirty}>
      <Field label="Path">
        <Input value={path} onChange={setPath} placeholder=".bc/logs" />
      </Field>
      <Field label="Max Bytes">
        <Input value={maxBytes} onChange={setMaxBytes} type="number" placeholder="10485760" />
      </Field>
      <SaveButton status={status} onClick={save} disabled={!dirty} />
    </Section>
  );
}

function UISection({
  config,
  onSave,
}: {
  config: SettingsConfig;
  onSave: (patch: Record<string, unknown>) => Promise<void>;
}) {
  const u = config.ui;
  const [theme, setTheme] = useState(u.theme);
  const [mode, setMode] = useState(u.mode);
  const [defaultView, setDefaultView] = useState(u.default_view);
  const [status, setStatus] = useState<SaveStatus>("idle");

  useEffect(() => {
    setTheme(u.theme);
    setMode(u.mode);
    setDefaultView(u.default_view);
  }, [u.theme, u.mode, u.default_view]);

  const dirty = theme !== u.theme || mode !== u.mode || defaultView !== u.default_view;

  const save = async () => {
    setStatus("saving");
    try {
      await onSave({ ui: { theme, mode, default_view: defaultView } });
      setStatus("saved");
      setTimeout(() => setStatus("idle"), 2000);
    } catch {
      setStatus("error");
    }
  };

  return (
    <Section title="UI" description="Interface appearance" dirty={dirty}>
      <Field label="Theme">
        <Select
          value={theme}
          onChange={setTheme}
          options={[
            { value: "solar-flare", label: "Solar Flare" },
            { value: "dark", label: "Dark" },
            { value: "light", label: "Light" },
            { value: "matrix", label: "Matrix" },
            { value: "synthwave", label: "Synthwave" },
            { value: "high-contrast", label: "High Contrast" },
          ]}
        />
      </Field>
      <Field label="Color Mode">
        <Select
          value={mode}
          onChange={setMode}
          options={[
            { value: "auto", label: "Auto" },
            { value: "dark", label: "Dark" },
            { value: "light", label: "Light" },
          ]}
        />
      </Field>
      <Field label="Default View">
        <Input value={defaultView} onChange={setDefaultView} placeholder="dashboard" />
      </Field>
      <SaveButton status={status} onClick={save} disabled={!dirty} />
    </Section>
  );
}

/* ------------------------------------------------------------------ */
/*  Main Settings page                                                 */
/* ------------------------------------------------------------------ */

export function Settings() {
  const fetcher = useCallback(() => api.getSettings(), []);
  const { data: config, loading, error, refresh, timedOut } = usePolling(fetcher, 30000);

  const handleSave = async (patch: Record<string, unknown>) => {
    await api.updateSettings(patch);
    refresh();
  };

  if (loading && !config)
    return (
      <div className="p-6">
        <LoadingSkeleton variant="cards" rows={4} />
      </div>
    );
  if (timedOut && !config)
    return (
      <div className="p-6">
        <EmptyState icon="!" title="Settings timed out" actionLabel="Retry" onAction={refresh} />
      </div>
    );
  if (error && !config)
    return (
      <div className="p-6">
        <EmptyState icon="!" title="Failed to load settings" description={error} actionLabel="Retry" onAction={refresh} />
      </div>
    );
  if (!config) return null;

  return (
    <div className="p-6 space-y-4 max-w-3xl">
      <div className="flex items-center justify-between mb-2">
        <div>
          <h1 className="text-xl font-bold text-bc-text">System Configuration</h1>
          <p className="text-xs text-bc-muted mt-0.5">settings.json v{config.version}</p>
        </div>
        <span className="text-[10px] font-mono text-bc-muted px-2 py-1 rounded bg-bc-surface border border-bc-border">
          {config.storage.default === "timescale" ? "TimescaleDB" : "SQLite"} · {config.runtime.default}
        </span>
      </div>
      <ServerSection config={config} onSave={handleSave} />
      <StorageSection config={config} onSave={handleSave} />
      <RuntimeSection config={config} onSave={handleSave} />
      <GatewaysSection config={config} onSave={handleSave} />
      <CronSection config={config} onSave={handleSave} />
      <ProvidersSection config={config} onSave={handleSave} />
      <LogsSection config={config} onSave={handleSave} />
      <UISection config={config} onSave={handleSave} />
    </div>
  );
}
