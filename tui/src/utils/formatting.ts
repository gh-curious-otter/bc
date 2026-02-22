/**
 * Shared formatting utilities
 *
 * Issue #1598: Remove duplicated code
 */

/**
 * Format a timestamp as relative time (e.g., "5m ago", "2h ago")
 *
 * @param timestamp - ISO timestamp string or Date
 * @returns Human-readable relative time string
 */
export function formatRelativeTime(timestamp: string | Date): string {
  try {
    const date = typeof timestamp === 'string' ? new Date(timestamp) : timestamp;
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    if (diffMins < 1) return 'now';
    if (diffMins < 60) return `${String(diffMins)}m ago`;
    if (diffHours < 24) return `${String(diffHours)}h ago`;
    if (diffDays < 7) return `${String(diffDays)}d ago`;

    // For older items, show date
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
    });
  } catch {
    return typeof timestamp === 'string' ? timestamp : 'unknown';
  }
}

/**
 * Format a duration in milliseconds as human-readable string
 *
 * @param ms - Duration in milliseconds
 * @returns Human-readable duration (e.g., "5m 30s", "2h 15m")
 */
export function formatDuration(ms: number): string {
  if (ms < 1000) return `${String(Math.round(ms))}ms`;

  const seconds = Math.floor(ms / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);

  if (hours > 0) {
    const remainingMins = minutes % 60;
    return remainingMins > 0 ? `${String(hours)}h ${String(remainingMins)}m` : `${String(hours)}h`;
  }
  if (minutes > 0) {
    const remainingSecs = seconds % 60;
    return remainingSecs > 0 ? `${String(minutes)}m ${String(remainingSecs)}s` : `${String(minutes)}m`;
  }
  return `${String(seconds)}s`;
}

/**
 * Truncate a string to a maximum length with ellipsis
 *
 * @param str - String to truncate
 * @param maxLength - Maximum length including ellipsis
 * @returns Truncated string
 */
export function truncate(str: string, maxLength: number): string {
  if (str.length <= maxLength) return str;
  return str.slice(0, maxLength - 3) + '...';
}

/**
 * Format a number with thousands separators
 *
 * @param num - Number to format
 * @returns Formatted string (e.g., "1,234,567")
 */
export function formatNumber(num: number): string {
  return num.toLocaleString('en-US');
}

/**
 * Format bytes as human-readable size
 *
 * @param bytes - Number of bytes
 * @returns Human-readable size (e.g., "1.5 MB")
 */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';

  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  const size = bytes / Math.pow(1024, i);

  return `${size.toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

/**
 * Format a cost value as currency
 *
 * @param cost - Cost in dollars
 * @returns Formatted cost string (e.g., "$1.23")
 */
export function formatCost(cost: number): string {
  if (cost < 0.01) return '<$0.01';
  return `$${cost.toFixed(2)}`;
}

/**
 * Capitalize the first letter of a string
 *
 * @param str - String to capitalize
 * @returns Capitalized string
 */
export function capitalize(str: string): string {
  if (!str) return str;
  return str.charAt(0).toUpperCase() + str.slice(1);
}

/**
 * Convert a string to title case
 *
 * @param str - String to convert
 * @returns Title-cased string
 */
export function toTitleCase(str: string): string {
  return str
    .split(/[\s_-]+/)
    .map(capitalize)
    .join(' ');
}
