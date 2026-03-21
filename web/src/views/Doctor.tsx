import { useCallback } from 'react';
import { api } from '../api/client';
import type { DoctorCategory } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import { EmptyState } from '../components/EmptyState';

const severityIcon = (s: number) => {
  switch (s) {
    case 0: return <span className="text-green-400">&#10003;</span>;
    case 1: return <span className="text-yellow-400">&#9888;</span>;
    case 2: return <span className="text-red-400">&#10007;</span>;
    default: return <span className="text-bc-muted">?</span>;
  }
};

export function Doctor() {
  const fetcher = useCallback(() => api.getDoctor(), []);
  const { data: report, loading, error, refresh, timedOut } = usePolling(fetcher, 30000);

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

  const totalPassed = report.Categories.reduce((n: number, c: DoctorCategory) => n + c.Items.filter(i => i.Severity === 0).length, 0);
  const totalFailed = report.Categories.reduce((n: number, c: DoctorCategory) => n + c.Items.filter(i => i.Severity === 2).length, 0);
  const totalWarnings = report.Categories.reduce((n: number, c: DoctorCategory) => n + c.Items.filter(i => i.Severity === 1).length, 0);

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Doctor</h1>
        <div className="flex gap-4 text-sm">
          <span className="text-green-400">{totalPassed} passed</span>
          {totalFailed > 0 && <span className="text-red-400">{totalFailed} failed</span>}
          {totalWarnings > 0 && <span className="text-yellow-400">{totalWarnings} warnings</span>}
        </div>
      </div>

      {report.Categories.map((cat: DoctorCategory) => (
        <CategorySection key={cat.Name} category={cat} />
      ))}
    </div>
  );
}

function CategorySection({ category }: { category: DoctorCategory }) {
  return (
    <div className="space-y-2">
      <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">{category.Name}</h2>
      <div className="rounded border border-bc-border overflow-hidden">
        {category.Items.map((item, i) => (
          <div
            key={i}
            className="flex items-start gap-3 px-4 py-2 border-b border-bc-border/50 last:border-b-0"
          >
            <span className="mt-0.5">{severityIcon(item.Severity)}</span>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span className="font-medium text-sm">{item.Name}</span>
                <span className="text-xs text-bc-muted truncate">{item.Message}</span>
              </div>
              {item.Fix && (
                <code className="text-xs text-bc-accent mt-0.5 block">{item.Fix}</code>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
