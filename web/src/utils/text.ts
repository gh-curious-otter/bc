/** Strip ANSI escape codes (CSI, OSC, charset, DEC private mode) from terminal output. */
// eslint-disable-next-line no-control-regex
const ANSI_RE = /\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?(?:\x07|\x1b\\)|\x1b[()][0-9A-Z]|\x1b\[\??[0-9;]*[hlm]/g;

export function stripAnsi(str: string): string {
  return str.replace(ANSI_RE, "");
}

/** Truncate string to maxLen characters with ellipsis. */
export function truncate(str: string, maxLen: number): string {
  return str.length > maxLen ? str.slice(0, maxLen) + "…" : str;
}
