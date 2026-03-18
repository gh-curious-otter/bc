import { useCallback } from 'react';
import { api } from '../api/client';
import type { Role } from '../api/client';
import { usePolling } from '../hooks/usePolling';

export function Roles() {
  const fetcher = useCallback(async () => {
    const res = await api.listRoles();
    return Object.entries(res).map(([name, role]) => ({ name, ...role }));
  }, []);
  const { data: roles, loading, error } = usePolling(fetcher, 30000);

  if (loading && !roles) {
    return <div className="p-6 text-bc-muted">Loading roles...</div>;
  }
  if (error && !roles) {
    return <div className="p-6 text-bc-error">Error: {error}</div>;
  }

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Roles</h1>
        <span className="text-sm text-bc-muted">{roles?.length ?? 0} roles</span>
      </div>

      <div className="grid gap-4">
        {(roles ?? []).map((role) => (
          <RoleCard key={role.name} name={role.name} role={role} />
        ))}
      </div>
    </div>
  );
}

function RoleCard({ name, role }: { name: string; role: Role & { name: string } }) {
  return (
    <div className="rounded border border-bc-border bg-bc-surface p-4 space-y-2">
      <div className="flex items-center justify-between">
        <h3 className="font-medium text-lg">{name}</h3>
        <div className="flex gap-2">
          {role.Metadata.IsSingleton && (
            <span className="text-xs px-2 py-0.5 rounded bg-bc-accent/20 text-bc-accent">singleton</span>
          )}
          <span className="text-xs px-2 py-0.5 rounded bg-bc-border text-bc-muted">
            level {role.Metadata.Level}
          </span>
        </div>
      </div>
      {role.Metadata.Description && (
        <p className="text-sm text-bc-muted">{role.Metadata.Description}</p>
      )}
      <div className="flex flex-wrap gap-2">
        {(role.Metadata.Capabilities ?? []).map((cap) => (
          <span key={cap} className="text-xs px-2 py-0.5 rounded border border-bc-border text-bc-muted">
            {cap}
          </span>
        ))}
      </div>
      {(role.Metadata.Permissions ?? []).length > 0 && (
        <div className="flex flex-wrap gap-2">
          {role.Metadata.Permissions.map((perm) => (
            <span key={perm} className="text-xs px-2 py-0.5 rounded border border-bc-accent/30 text-bc-accent">
              {perm}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
