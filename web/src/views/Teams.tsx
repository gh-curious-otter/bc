import { useCallback } from "react";
import { api } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { Table } from "../components/Table";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";
import type { Team } from "../api/client";

export function Teams() {
  const fetcher = useCallback(() => api.listTeams(), []);
  const {
    data: teams,
    loading,
    error,
    refresh,
    timedOut,
  } = usePolling(fetcher, 15000);

  if (loading && !teams) {
    return (
      <div className="p-6 space-y-4">
        <div className="h-6 w-20 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="text" rows={5} />
      </div>
    );
  }
  if (timedOut && !teams) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Teams took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !teams) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load teams"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Teams</h1>
        <span className="text-sm text-bc-muted">
          {teams?.length ?? 0} teams
        </span>
      </div>

      <Table<Team>
        columns={[
          {
            key: "name",
            label: "Name",
            render: (t) => <span className="font-medium">{t.name}</span>,
          },
          {
            key: "description",
            label: "Description",
            render: (t) => (
              <span className="text-bc-muted">{t.description || "-"}</span>
            ),
          },
          {
            key: "lead",
            label: "Lead",
            render: (t) => (
              <span className="text-bc-accent">{t.lead || "-"}</span>
            ),
          },
          {
            key: "members",
            label: "Members",
            render: (t) => (
              <div className="flex flex-wrap gap-1">
                {(t.members ?? []).length === 0 ? (
                  <span className="text-bc-muted">-</span>
                ) : (
                  (t.members ?? []).map((m) => (
                    <span
                      key={m}
                      className="text-xs px-2 py-0.5 rounded bg-bc-accent/10 text-bc-accent"
                    >
                      {m}
                    </span>
                  ))
                )}
              </div>
            ),
          },
        ]}
        data={teams ?? []}
        keyFn={(t) => t.name}
        emptyMessage="No teams"
        emptyIcon="G"
        emptyDescription="Teams are created via bc team create or the config file."
      />
    </div>
  );
}
