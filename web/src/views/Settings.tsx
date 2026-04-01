import { useCallback, useState, useEffect } from "react";
import { api } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

type SaveStatus = "idle" | "saving" | "saved" | "error";

const SENSITIVE_KEYS = /token|password|secret|key/i;

function formatLabel(key: string): string {
  return key.replace(/[_-]/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

function deepClone<T>(v: T): T {
  return JSON.parse(JSON.stringify(v));
}

function deepEqual(a: unknown, b: unknown): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}

/* ------------------------------------------------------------------ */
/*  Primitive editors                                                   */
/* ------------------------------------------------------------------ */

function TextInput({
  value,
  onChange,
  type = "text",
  disabled = false,
}: {
  value: string | number;
  onChange: (v: string) => void;
  type?: string;
  disabled?: boolean;
}) {
  return (
    <input
      type={type}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      disabled={disabled}
      className="w-full px-3 py-1.5 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent disabled:opacity-50 font-mono"
    />
  );
}

function PasswordField({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const [visible, setVisible] = useState(false);
  return (
    <div className="relative">
      <input
        type={visible ? "text" : "password"}
        value={value}
        onChange={(e) => onChange(e.target.value)}
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

/* ------------------------------------------------------------------ */
/*  Recursive JSON editor                                               */
/* ------------------------------------------------------------------ */

function JsonEditor({
  value,
  onChange,
  path = [],
  readOnly = false,
}: {
  value: unknown;
  onChange: (path: string[], newValue: unknown) => void;
  path?: string[];
  readOnly?: boolean;
}) {
  const key = path[path.length - 1] ?? "";

  if (value === null || value === undefined) {
    return (
      <TextInput
        value=""
        onChange={(v) => onChange(path, v || null)}
        disabled={readOnly}
      />
    );
  }

  if (typeof value === "boolean") {
    return <Toggle checked={value} onChange={(v) => !readOnly && onChange(path, v)} />;
  }

  if (typeof value === "number") {
    return (
      <TextInput
        value={value}
        onChange={(v) => onChange(path, v === "" ? 0 : Number(v))}
        type="number"
        disabled={readOnly}
      />
    );
  }

  if (typeof value === "string") {
    if (SENSITIVE_KEYS.test(key)) {
      return readOnly ? (
        <TextInput value="********" onChange={() => {}} disabled />
      ) : (
        <PasswordField value={value} onChange={(v) => onChange(path, v)} />
      );
    }
    return (
      <TextInput value={value} onChange={(v) => onChange(path, v)} disabled={readOnly} />
    );
  }

  if (Array.isArray(value)) {
    const jsonStr = JSON.stringify(value, null, 2);
    return (
      <textarea
        value={jsonStr}
        disabled={readOnly}
        onChange={(e) => {
          try {
            onChange(path, JSON.parse(e.target.value));
          } catch {
            /* keep current value on invalid JSON */
          }
        }}
        rows={Math.min(Math.max(jsonStr.split("\n").length, 2), 8)}
        className="w-full px-3 py-1.5 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent font-mono disabled:opacity-50 resize-y"
      />
    );
  }

  if (typeof value === "object") {
    const entries = Object.entries(value as Record<string, unknown>);
    return (
      <div className="space-y-3 pl-4 border-l-2 border-bc-border/50">
        {entries.map(([k, v]) => (
          <div key={k} className="flex items-start gap-4">
            <label className="text-sm text-bc-muted w-40 shrink-0 pt-1.5 truncate" title={k}>
              {formatLabel(k)}
            </label>
            <div className="flex-1 min-w-0">
              <JsonEditor
                value={v}
                onChange={onChange}
                path={[...path, k]}
                readOnly={readOnly || k === "version"}
              />
            </div>
          </div>
        ))}
      </div>
    );
  }

  return <TextInput value={String(value)} onChange={(v) => onChange(path, v)} disabled={readOnly} />;
}

/* ------------------------------------------------------------------ */
/*  Collapsible section with save button                                */
/* ------------------------------------------------------------------ */

function ConfigSection({
  sectionKey,
  original,
  edited,
  onChange,
  onSave,
}: {
  sectionKey: string;
  original: unknown;
  edited: unknown;
  onChange: (path: string[], newValue: unknown) => void;
  onSave: (key: string) => Promise<void>;
}) {
  const [open, setOpen] = useState(true);
  const [status, setStatus] = useState<SaveStatus>("idle");
  const dirty = !deepEqual(original, edited);
  const isVersion = sectionKey === "version";

  const save = async () => {
    setStatus("saving");
    try {
      await onSave(sectionKey);
      setStatus("saved");
      setTimeout(() => setStatus("idle"), 2000);
    } catch {
      setStatus("error");
    }
  };

  const label =
    status === "saving" ? "Saving..." : status === "saved" ? "Saved!" : status === "error" ? "Error - Retry" : "Save";

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
          <h2 className="text-sm font-semibold text-bc-text uppercase tracking-wide">
            {formatLabel(sectionKey)}
          </h2>
          {dirty && (
            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-bc-accent/20 text-bc-accent">
              unsaved
            </span>
          )}
        </div>
      </button>
      {open && (
        <div className="px-5 pb-5 pt-2 space-y-4 border-t border-bc-border">
          {typeof edited === "object" && edited !== null && !Array.isArray(edited) ? (
            Object.entries(edited as Record<string, unknown>).map(([k, v]) => (
              <div key={k} className="flex items-start gap-4">
                <label className="text-sm text-bc-muted w-44 shrink-0 pt-1.5 truncate" title={k}>
                  {formatLabel(k)}
                </label>
                <div className="flex-1 min-w-0">
                  <JsonEditor
                    value={v}
                    onChange={onChange}
                    path={[sectionKey, k]}
                    readOnly={isVersion}
                  />
                </div>
              </div>
            ))
          ) : (
            <JsonEditor value={edited} onChange={onChange} path={[sectionKey]} readOnly={isVersion} />
          )}
          {!isVersion && (
            <div className="flex items-center gap-3 pt-2">
              <button
                onClick={save}
                disabled={!dirty || status === "saving"}
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
          )}
        </div>
      )}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Main Settings page                                                  */
/* ------------------------------------------------------------------ */

export function Settings() {
  const fetcher = useCallback(() => api.getSettings(), []);
  const { data: config, loading, error, refresh, timedOut } = usePolling(fetcher, 30000);

  const [edited, setEdited] = useState<Record<string, unknown> | null>(null);
  const [original, setOriginal] = useState<Record<string, unknown> | null>(null);

  useEffect(() => {
    if (config) {
      const raw = config as unknown as Record<string, unknown>;
      setOriginal(deepClone(raw));
      setEdited((prev) => (prev === null ? deepClone(raw) : prev));
    }
  }, [config]);

  const handleChange = (path: string[], newValue: unknown) => {
    if (!edited || path.length === 0) return;
    const next = deepClone(edited);
    let cursor: Record<string, unknown> = next;
    for (let i = 0; i < path.length - 1; i++) {
      const k = path[i]!;
      if (typeof cursor[k] !== "object" || cursor[k] === null) {
        cursor[k] = {};
      }
      cursor = cursor[k] as Record<string, unknown>;
    }
    cursor[path[path.length - 1]!] = newValue;
    setEdited(next);
  };

  const handleSave = async (sectionKey: string) => {
    if (!edited) return;
    const patch = { [sectionKey]: edited[sectionKey] };
    await api.updateSettings(patch);
    refresh();
    setOriginal((prev) => {
      if (!prev) return prev;
      const next = deepClone(prev);
      next[sectionKey] = deepClone(edited[sectionKey]);
      return next;
    });
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
  if (!config || !edited || !original) return null;

  const sectionKeys = Object.keys(edited);

  return (
    <div className="p-6 space-y-4 max-w-3xl">
      <div className="flex items-center justify-between mb-2">
        <div>
          <h1 className="text-xl font-bold text-bc-text">System Configuration</h1>
          <p className="text-xs text-bc-muted mt-0.5">
            settings.json{typeof edited.version !== "undefined" ? ` v${edited.version}` : ""}
          </p>
        </div>
      </div>
      {sectionKeys.map((key) => (
        <ConfigSection
          key={key}
          sectionKey={key}
          original={original[key]}
          edited={edited[key]}
          onChange={handleChange}
          onSave={handleSave}
        />
      ))}
    </div>
  );
}
