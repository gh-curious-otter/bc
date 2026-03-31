import { useCallback, useState } from "react";
import { api } from "../api/client";
import type { DoctorCategory, DoctorItem } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

const severityIcon = (s: string) => {
  switch (s) {
    case "ok":
      return <span className="text-bc-success">&#10003;</span>;
    case "warn":
      return <span className="text-bc-warning">&#9888;</span>;
    case "error":
      return <span className="text-bc-error">&#10007;</span>;
    default:
      return <span className="text-bc-muted">?</span>;
  }
};

export function Doctor() {
  const fetcher = useCallback(() => api.getDoctor(), []);
  const {
    data: report,
    loading,
    error,
    refresh,
    timedOut,
  } = usePolling(fetcher, 30000);

  if (loading && !report) {
    return (
      <div className="p-6 space-y-4">
        <div className="h-6 w-24 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="text" rows={4} />
        <LoadingSkeleton variant="text" rows={3} />
      </div>
    );
  }
  if (timedOut && !report) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Diagnostics took too long to run"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !report) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to run diagnostics"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (!report) return null;

  const categories = report.categories ?? [];

  const allItems = categories.flatMap((c) => c.items ?? []);
  const totalPassed = allItems.filter((i) => i.severity === "ok").length;
  const totalFailed = allItems.filter((i) => i.severity === "error").length;
  const totalWarnings = allItems.filter((i) => i.severity === "warn").length;

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Doctor</h1>
        <div className="flex gap-4 text-sm">
          <span className="text-bc-success">{totalPassed} passed</span>
          {totalFailed > 0 && (
            <span className="text-bc-error">{totalFailed} failed</span>
          )}
          {totalWarnings > 0 && (
            <span className="text-bc-warning">{totalWarnings} warnings</span>
          )}
        </div>
      </div>

      {categories.map((cat: DoctorCategory) => (
        <CategorySection key={cat.name} category={cat} />
      ))}
    </div>
  );
}

function FixButton({ fix }: { fix: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(fix).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  };

  return (
    <div className="flex items-center gap-2 mt-1">
      <code className="text-xs text-bc-accent">{fix}</code>
      <button
        onClick={handleCopy}
        className="text-xs px-2 py-0.5 rounded bg-bc-error/20 text-bc-error hover:bg-bc-error/30 transition-colors shrink-0 focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-offset-1 focus-visible:ring-offset-bc-bg"
        title="Copy fix command to clipboard"
        aria-label="Copy fix command to clipboard"
      >
        {copied ? "Copied!" : "Fix"}
      </button>
    </div>
  );
}

function CategorySection({ category }: { category: DoctorCategory }) {
  const items = category.items ?? [];
  return (
    <div className="space-y-2">
      <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">
        {category.name}
      </h2>
      <div className="rounded border border-bc-border overflow-hidden">
        {items.map((item: DoctorItem, i: number) => (
          <div
            key={i}
            className="flex items-start gap-3 px-4 py-2 border-b border-bc-border/50 last:border-b-0"
          >
            <span className="mt-0.5">{severityIcon(item.severity)}</span>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span className="font-medium text-sm">{item.name}</span>
                <span className="text-xs text-bc-muted truncate">
                  {item.message}
                </span>
              </div>
              {item.severity === "error" && item.fix ? (
                <FixButton fix={item.fix} />
              ) : item.fix ? (
                <code className="text-xs text-bc-accent mt-0.5 block">
                  {item.fix}
                </code>
              ) : null}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
