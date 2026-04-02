import { useParams, Link } from "react-router-dom";

export function ProviderDetail() {
  const { provider } = useParams<{ provider: string }>();

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center gap-3">
        <Link to="/tools" className="text-sm text-bc-accent hover:underline">&larr; Tools</Link>
        <h1 className="text-xl font-bold">{provider}</h1>
      </div>
      <div className="rounded border border-bc-border bg-bc-surface p-6 text-center">
        <p className="text-bc-muted">Provider detail page coming soon.</p>
        <p className="text-xs text-bc-muted mt-2">This page will show configuration, MCP servers, agent list, and cost breakdown for <span className="font-medium text-bc-text">{provider}</span>.</p>
      </div>
    </div>
  );
}
