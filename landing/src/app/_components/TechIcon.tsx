/**
 * TechIcon — renders inline SVG brand icons for tech tags on the pricing page.
 * Monochrome, 16x16, uses currentColor so it inherits the muted text color.
 * Falls back to a monospace text label for techs without a clean SVG.
 */

interface TechIconProps {
  name: string;
  className?: string;
}

function DockerIcon() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {/* Whale body */}
      <path d="M3 14c0-3 2-6 9-6h1c4 0 7 1.5 8 4 .5 1.2.3 3-1 4.5C18.5 18 16 19 12 19c-5 0-8-2-9-5z" />
      {/* Container boxes */}
      <rect x="5" y="10" width="2" height="2" rx="0.3" />
      <rect x="8" y="10" width="2" height="2" rx="0.3" />
      <rect x="11" y="10" width="2" height="2" rx="0.3" />
      <rect x="8" y="7" width="2" height="2" rx="0.3" />
      <rect x="11" y="7" width="2" height="2" rx="0.3" />
      <rect x="14" y="10" width="2" height="2" rx="0.3" />
    </svg>
  );
}

function PostgreSQLIcon() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {/* Elephant head shape */}
      <path d="M12 3C7 3 4 6 4 10c0 3 1.5 5 4 6l-1 4c0 .5.5 1 1 .5l3-2 3 2c.5.5 1 0 1-.5l-1-4c2.5-1 4-3 4-6 0-4-3-7-7-7z" />
      {/* Eye */}
      <circle cx="10" cy="9" r="1" fill="currentColor" stroke="none" />
      {/* Trunk */}
      <path d="M14 11c1 1 2 3 1.5 5" />
    </svg>
  );
}

function KubernetesIcon() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {/* Helm wheel / steering wheel shape */}
      <circle cx="12" cy="12" r="8" />
      <circle cx="12" cy="12" r="2" />
      {/* Spokes */}
      <line x1="12" y1="4" x2="12" y2="10" />
      <line x1="12" y1="14" x2="12" y2="20" />
      <line x1="4.9" y1="7.9" x2="10" y2="11" />
      <line x1="14" y1="13" x2="19.1" y2="16.1" />
      <line x1="4.9" y1="16.1" x2="10" y2="13" />
      <line x1="14" y1="11" x2="19.1" y2="7.9" />
    </svg>
  );
}

function AWSIcon() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {/* Simplified AWS "smile" logo */}
      <path d="M3 14c3 2 6 3 9 3s6-1 9-3" />
      <path d="M19 14l2 1" />
      {/* "A" shape */}
      <path d="M8 16V7l4-3 4 3v9" />
      <path d="M8 11h8" />
    </svg>
  );
}

function GCPIcon() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {/* Simplified hexagonal cloud shape */}
      <polygon points="12,3 19,7 19,17 12,21 5,17 5,7" />
      {/* Inner triangle */}
      <polygon points="12,8 16,14 8,14" />
    </svg>
  );
}

function SQLiteIcon() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {/* Database cylinder */}
      <ellipse cx="12" cy="5" rx="8" ry="3" />
      <path d="M4 5v14c0 1.66 3.58 3 8 3s8-1.34 8-3V5" />
      <path d="M4 12c0 1.66 3.58 3 8 3s8-1.34 8-3" />
    </svg>
  );
}

const iconMap: Record<string, React.ComponentType> = {
  docker: DockerIcon,
  postgresql: PostgreSQLIcon,
  kubernetes: KubernetesIcon,
  aws: AWSIcon,
  gcp: GCPIcon,
  sqlite: SQLiteIcon,
};

// Tags that render as monospace text (no suitable minimal SVG)
const textOnlyTags = new Set([
  "tmux",
  "ssh",
  "mcp",
  "sso",
  "saml",
  "audit",
  "sla",
]);

export function TechIcon({ name, className }: TechIconProps) {
  const key = name.toLowerCase();
  const Icon = iconMap[key];

  return (
    <span
      className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-mono bg-muted/40 text-muted-foreground ${className ?? ""}`}
    >
      {Icon && !textOnlyTags.has(key) ? (
        <>
          <Icon />
          <span>{key}</span>
        </>
      ) : (
        <span>{key}</span>
      )}
    </span>
  );
}

export function TechTags({ tags }: { tags: string[] }) {
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {tags.map((t) => (
        <TechIcon key={t} name={t} />
      ))}
    </div>
  );
}
