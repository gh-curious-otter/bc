/**
 * Terminal color scheme detection
 *
 * Attempts to detect whether the terminal is using a dark or light color scheme.
 */

/**
 * Detect terminal color scheme from environment variables
 *
 * Checks common environment variables that indicate color scheme:
 * - COLORFGBG: Set by some terminals (format: "fg;bg")
 * - TERM_PROGRAM: Terminal application name
 * - ITERM_PROFILE: iTerm2 profile (may contain "light" or "dark")
 * - COLORTERM: Color terminal capabilities
 *
 * @returns 'dark' | 'light' based on detection, defaults to 'dark'
 */
export function detectColorScheme(): 'dark' | 'light' {
  const env = process.env;

  // Check COLORFGBG (format: "foreground;background" or "foreground;ignored;background")
  // Background values: 0-6 or 8 = dark, 7 or 15 = light
  const colorFgBg = env.COLORFGBG;
  if (colorFgBg) {
    const parts = colorFgBg.split(';');
    const bg = parseInt(parts[parts.length - 1], 10);
    if (!isNaN(bg)) {
      // 7 = white, 15 = bright white (light backgrounds)
      if (bg === 7 || bg === 15) {
        return 'light';
      }
      // 0 = black, 8 = bright black (dark backgrounds)
      if (bg === 0 || bg === 8 || (bg >= 1 && bg <= 6)) {
        return 'dark';
      }
    }
  }

  // Check macOS appearance via terminal profile hints
  const itermProfile = env.ITERM_PROFILE?.toLowerCase() || '';
  if (itermProfile.includes('light')) {
    return 'light';
  }
  if (itermProfile.includes('dark')) {
    return 'dark';
  }

  // Check for Terminal.app on macOS
  const termProgram = env.TERM_PROGRAM?.toLowerCase() || '';
  if (termProgram === 'apple_terminal') {
    // Apple Terminal defaults to light in recent macOS
    // But many users customize it, so we can't be certain
    // Check if there's additional context
  }

  // Check for explicit dark mode preference (some systems)
  const darkMode = env.DARK_MODE || env.THEME || '';
  if (darkMode.toLowerCase() === 'light') {
    return 'light';
  }
  if (darkMode.toLowerCase() === 'dark') {
    return 'dark';
  }

  // Check for VS Code integrated terminal
  if (env.TERM_PROGRAM === 'vscode') {
    // VS Code theme can be detected via VSCODE_INJECTION
    // But default assumption is it follows system
  }

  // Default to dark theme (most common for developers)
  return 'dark';
}

/**
 * Check if terminal supports 256 colors or true color
 */
export function supportsExtendedColors(): boolean {
  const colorTerm = process.env.COLORTERM?.toLowerCase() || '';
  const term = process.env.TERM?.toLowerCase() || '';

  // True color support
  if (colorTerm === 'truecolor' || colorTerm === '24bit') {
    return true;
  }

  // 256 color support
  if (term.includes('256color') || term.includes('256')) {
    return true;
  }

  return false;
}

/**
 * Check if terminal supports basic colors
 */
export function supportsColors(): boolean {
  // Check if stdout is a TTY
  if (!process.stdout.isTTY) {
    return false;
  }

  // Check TERM environment variable
  const term = process.env.TERM?.toLowerCase() || '';
  if (term === 'dumb') {
    return false;
  }

  // Check for NO_COLOR standard
  if ('NO_COLOR' in process.env) {
    return false;
  }

  // Check for FORCE_COLOR
  if ('FORCE_COLOR' in process.env) {
    return true;
  }

  return true;
}
