import { useCallback, useState } from "react";
import { api } from "../api/client";
import type { SettingsConfig } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

type SaveStatus = "idle" | "saving" | "saved" | "error";

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="space-y-3">
      <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">{title}</h2>
      <div className="rounded border border-bc-border bg-bc-surface p-5 space-y-4">
        {children}
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center gap-4">
      <label className="text-sm text-bc-muted w-40 shrink-0">{label}</label>
      <div className="flex-1">{children}</div>
    </div>
  );
}

function Input({ value, onChange, placeholder, type = "text" }: {
  value: string | number;
  onChange: (v: string) => void;
  placeholder?: string;
  type?: string;
}) {
  return (
    <input
      type={type}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      className="w-full px-3 py-1.5 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
    />
  );
}

function Select({ value, onChange, options }: {
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
        <option key={o.value} value={o.value}>{o.label}</option>
      ))}
    </select>
  );
}

function SaveButton({ status, onClick }: { status: SaveStatus; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      disabled={status === "saving"}
      className="px-4 py-1.5 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
    >
      {status === "saving" ? "Saving..." : status === "saved" ? "Saved!" : "Save"}
    </button>
  );
}

function UserSection({ config, onSave }: { config: SettingsConfig; onSave: (patch: Record<string, unknown>) => Promise<void> }) {
  const [name, setName] = useState(config.user.name);
  const [status, setStatus] = useState<SaveStatus>("idle");
  const save = async () => {
    setStatus("saving");
    try { await onSave({ user: { name } }); setStatus("saved"); setTimeout(() => setStatus("idle"), 2000); }
    catch { setStatus("error"); }
  };
  return (
    <Section title="User">
      <Field label="Name"><Input value={name} onChange={setName} placeholder="Your name" /></Field>
      <SaveButton status={status} onClick={save} />
    </Section>
  );
}

function ServerSection({ config, onSave }: { config: SettingsConfig; onSave: (patch: Record<string, unknown>) => Promise<void> }) {
  const [host, setHost] = useState(config.server.host);
  const [port, setPort] = useState(String(config.server.port));
  const [cors, setCors] = useState(config.server.cors_origin);
  const [status, setStatus] = useState<SaveStatus>("idle");
  const save = async () => {
    setStatus("saving");
    try { await onSave({ server: { host, port: parseInt(port, 10), cors_origin: cors } }); setStatus("saved"); setTimeout(() => setStatus("idle"), 2000); }
    catch { setStatus("error"); }
  };
  return (
    <Section title="Server">
      <Field label="Host"><Input value={host} onChange={setHost} /></Field>
      <Field label="Port"><Input value={port} onChange={setPort} type="number" /></Field>
      <Field label="CORS Origin"><Input value={cors} onChange={setCors} /></Field>
      <SaveButton status={status} onClick={save} />
    </Section>
  );
}

function RuntimeSection({ config, onSave }: { config: SettingsConfig; onSave: (patch: Record<string, unknown>) => Promise<void> }) {
  const [backend, setBackend] = useState(config.runtime.default);
  const [status, setStatus] = useState<SaveStatus>("idle");
  const save = async () => {
    setStatus("saving");
    try { await onSave({ runtime: { ...config.runtime, default: backend } }); setStatus("saved"); setTimeout(() => setStatus("idle"), 2000); }
    catch { setStatus("error"); }
  };
  return (
    <Section title="Runtime">
      <Field label="Backend">
        <Select value={backend} onChange={setBackend} options={[
          { value: "tmux", label: "tmux" },
          { value: "docker", label: "Docker" },
        ]} />
      </Field>
      <SaveButton status={status} onClick={save} />
    </Section>
  );
}

function ProvidersSection({ config }: { config: SettingsConfig }) {
  const providers = config.providers.providers ?? {};
  return (
    <Section title="Providers">
      <Field label="Default">
        <span className="text-sm font-medium text-bc-accent">{config.providers.default}</span>
      </Field>
      {Object.entries(providers).map(([name, p]) => (
        <Field key={name} label={name}>
          <code className="text-xs text-bc-muted">{p.command}</code>
        </Field>
      ))}
    </Section>
  );
}

function UISection({ config, onSave }: { config: SettingsConfig; onSave: (patch: Record<string, unknown>) => Promise<void> }) {
  const [theme, setTheme] = useState(config.ui.theme);
  const [mode, setMode] = useState(config.ui.mode);
  const [status, setStatus] = useState<SaveStatus>("idle");
  const save = async () => {
    setStatus("saving");
    try { await onSave({ ui: { theme, mode, default_view: config.ui.default_view } }); setStatus("saved"); setTimeout(() => setStatus("idle"), 2000); }
    catch { setStatus("error"); }
  };
  return (
    <Section title="UI">
      <Field label="Theme">
        <Select value={theme} onChange={setTheme} options={[
          { value: "dark", label: "Dark" }, { value: "light", label: "Light" },
          { value: "matrix", label: "Matrix" }, { value: "synthwave", label: "Synthwave" },
          { value: "high-contrast", label: "High Contrast" },
        ]} />
      </Field>
      <Field label="Color Mode">
        <Select value={mode} onChange={setMode} options={[
          { value: "auto", label: "Auto" }, { value: "dark", label: "Dark" }, { value: "light", label: "Light" },
        ]} />
      </Field>
      <SaveButton status={status} onClick={save} />
    </Section>
  );
}

function GatewaysSection({ config }: { config: SettingsConfig }) {
  const gw = config.gateways;
  return (
    <Section title="Gateways">
      <Field label="Telegram">
        <span className={`text-sm ${gw.telegram?.enabled ? "text-green-400" : "text-bc-muted"}`}>
          {gw.telegram?.enabled ? "enabled" : "disabled"}
        </span>
      </Field>
      <Field label="Discord">
        <span className={`text-sm ${gw.discord?.enabled ? "text-green-400" : "text-bc-muted"}`}>
          {gw.discord?.enabled ? "enabled" : "disabled"}
        </span>
      </Field>
      <Field label="Slack">
        <span className={`text-sm ${gw.slack?.enabled ? "text-green-400" : "text-bc-muted"}`}>
          {gw.slack?.enabled ? "enabled" : "disabled"}
        </span>
      </Field>
    </Section>
  );
}

function LogsSection({ config, onSave }: { config: SettingsConfig; onSave: (patch: Record<string, unknown>) => Promise<void> }) {
  const [path, setPath] = useState(config.logs.path);
  const [maxBytes, setMaxBytes] = useState(String(config.logs.max_bytes));
  const [status, setStatus] = useState<SaveStatus>("idle");
  const save = async () => {
    setStatus("saving");
    try { await onSave({ logs: { path, max_bytes: parseInt(maxBytes, 10) } }); setStatus("saved"); setTimeout(() => setStatus("idle"), 2000); }
    catch { setStatus("error"); }
  };
  return (
    <Section title="Logs">
      <Field label="Path"><Input value={path} onChange={setPath} /></Field>
      <Field label="Max Bytes"><Input value={maxBytes} onChange={setMaxBytes} type="number" /></Field>
      <SaveButton status={status} onClick={save} />
    </Section>
  );
}

function CronSection({ config }: { config: SettingsConfig }) {
  return (
    <Section title="Cron Scheduler">
      <Field label="Poll Interval"><span className="text-sm">{config.cron.poll_interval_seconds}s</span></Field>
      <Field label="Job Timeout"><span className="text-sm">{config.cron.job_timeout_seconds}s</span></Field>
    </Section>
  );
}

function StorageSection({ config }: { config: SettingsConfig }) {
  return (
    <Section title="Storage">
      <Field label="Backend"><span className="text-sm font-medium">{config.storage.default}</span></Field>
      {config.storage.default === "sqlite" && (
        <Field label="Path"><span className="text-sm text-bc-muted">{config.storage.sqlite.path}</span></Field>
      )}
      {config.storage.default === "sql" && (
        <Field label="Host"><span className="text-sm text-bc-muted">{config.storage.sql.host}:{config.storage.sql.port}</span></Field>
      )}
    </Section>
  );
}

export function Settings() {
  const fetcher = useCallback(() => api.getSettings(), []);
  const { data: config, loading, error, refresh, timedOut } = usePolling(fetcher, 30000);

  const handleSave = async (patch: Record<string, unknown>) => {
    await api.updateSettings(patch);
    refresh();
  };

  if (loading && !config) return <div className="p-6"><LoadingSkeleton variant="cards" rows={4} /></div>;
  if (timedOut && !config) return <div className="p-6"><EmptyState icon="!" title="Settings timed out" actionLabel="Retry" onAction={refresh} /></div>;
  if (error && !config) return <div className="p-6"><EmptyState icon="!" title="Failed to load settings" description={error} actionLabel="Retry" onAction={refresh} /></div>;
  if (!config) return null;

  return (
    <div className="p-6 space-y-6 max-w-3xl">
      <h1 className="text-xl font-bold">Settings</h1>
      <UserSection config={config} onSave={handleSave} />
      <ServerSection config={config} onSave={handleSave} />
      <RuntimeSection config={config} onSave={handleSave} />
      <ProvidersSection config={config} />
      <UISection config={config} onSave={handleSave} />
      <GatewaysSection config={config} />
      <LogsSection config={config} onSave={handleSave} />
      <CronSection config={config} />
      <StorageSection config={config} />
    </div>
  );
}
