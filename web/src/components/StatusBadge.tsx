const COLORS: Record<string, string> = {
  idle: "bg-bc-muted/20 text-bc-muted",
  working: "bg-bc-accent/20 text-bc-accent",
  done: "bg-bc-success/20 text-bc-success",
  stuck: "bg-bc-warning/20 text-bc-warning",
  error: "bg-bc-error/20 text-bc-error",
  stopped: "bg-bc-muted/10 text-bc-muted",
};

export function StatusBadge({ status }: { status: string }) {
  const cls = COLORS[status] ?? COLORS["idle"]!;
  return (
    <span
      className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${cls}`}
    >
      {status}
    </span>
  );
}
