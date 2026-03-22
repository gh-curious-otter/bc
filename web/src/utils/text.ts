/** Strip ANSI escape codes (CSI, OSC, charset) from terminal output. */
// eslint-disable-next-line no-control-regex
const CSI = /\x1b\[[0-9;]*[a-zA-Z]/g;
// eslint-disable-next-line no-control-regex
const OSC = /\x1b\][^\x07]*\x07/g;
// eslint-disable-next-line no-control-regex
const CHARSET = /\x1b\(B/g;

export function stripAnsi(str: string): string {
  return str.replace(CSI, "").replace(OSC, "").replace(CHARSET, "");
}

/** Truncate string to maxLen characters with ellipsis. */
export function truncate(str: string, maxLen: number): string {
  return str.length > maxLen ? str.slice(0, maxLen) + "…" : str;
}
