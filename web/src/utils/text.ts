/** Strip ANSI escape codes (CSI, OSC, charset) from terminal output. */
export function stripAnsi(str: string): string {
  return str
    .replace(/\x1b\[[0-9;]*[a-zA-Z]/g, '')
    .replace(/\x1b\][^\x07]*\x07/g, '')
    .replace(/\x1b\(B/g, '');
}

/** Truncate string to maxLen characters with ellipsis. */
export function truncate(str: string, maxLen: number): string {
  return str.length > maxLen ? str.slice(0, maxLen) + '…' : str;
}
