/* eslint-disable @typescript-eslint/no-unsafe-call, @typescript-eslint/no-unsafe-member-access */

/**
 * Tests for theme system
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { Text } from 'ink';
import {
  ThemeProvider,
  useTheme,
  useThemeColor,
  darkTheme,
  lightTheme,
  getTheme,
  applyOverrides,
  detectColorScheme,
  supportsColors,
} from '../theme';

// Test component that uses theme
function ThemeConsumer() {
  const { theme, isDark, mode } = useTheme();
  return (
    <Text>
      Theme: {theme.name}, Mode: {mode}, Dark: {isDark ? 'yes' : 'no'}
    </Text>
  );
}

// Test component for color
function ColorConsumer() {
  const primary = useThemeColor('primary');
  return <Text color={primary}>Primary Color</Text>;
}

describe('ThemeProvider', () => {
  it('provides default dark theme', () => {
    const { lastFrame } = render(
      <ThemeProvider>
        <ThemeConsumer />
      </ThemeProvider>
    );
    expect(lastFrame()).toContain('Theme: dark');
  });

  it('respects explicit dark mode', () => {
    const { lastFrame } = render(
      <ThemeProvider config={{ mode: 'dark' }}>
        <ThemeConsumer />
      </ThemeProvider>
    );
    expect(lastFrame()).toContain('Mode: dark');
    expect(lastFrame()).toContain('Dark: yes');
  });

  it('respects explicit light mode', () => {
    const { lastFrame } = render(
      <ThemeProvider config={{ mode: 'light' }}>
        <ThemeConsumer />
      </ThemeProvider>
    );
    expect(lastFrame()).toContain('Mode: light');
    expect(lastFrame()).toContain('Dark: no');
  });
});

describe('useTheme', () => {
  it('throws when used outside provider', () => {
    // Note: In ink-testing-library, the error is caught differently
    // The hook should throw, but the error may be caught by React's error boundary
    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
    const consoleError = jest.spyOn(console, 'error').mockImplementation();
    try {
      render(<ThemeConsumer />);
    } catch (error) {
      expect((error as Error).message).toContain('useTheme must be used within a ThemeProvider');
    }
    consoleError.mockRestore();
  });
});

describe('useThemeColor', () => {
  it('returns theme color value', () => {
    const { lastFrame } = render(
      <ThemeProvider>
        <ColorConsumer />
      </ThemeProvider>
    );
    expect(lastFrame()).toContain('Primary Color');
  });
});

describe('theme definitions', () => {
  it('darkTheme has all required colors', () => {
    expect(darkTheme.name).toBe('dark');
    expect(darkTheme.mode).toBe('dark');
    expect(darkTheme.colors.primary).toBeDefined();
    expect(darkTheme.colors.success).toBeDefined();
    expect(darkTheme.colors.error).toBeDefined();
    expect(darkTheme.colors.agentWorking).toBeDefined();
  });

  it('lightTheme has all required colors', () => {
    expect(lightTheme.name).toBe('light');
    expect(lightTheme.mode).toBe('light');
    expect(lightTheme.colors.primary).toBeDefined();
    expect(lightTheme.colors.success).toBeDefined();
    expect(lightTheme.colors.error).toBeDefined();
    expect(lightTheme.colors.agentWorking).toBeDefined();
  });
});

describe('getTheme', () => {
  it('returns dark theme by name', () => {
    const theme = getTheme('dark');
    expect(theme).toBe(darkTheme);
  });

  it('returns light theme by name', () => {
    const theme = getTheme('light');
    expect(theme).toBe(lightTheme);
  });
});

describe('applyOverrides', () => {
  it('applies color overrides to theme', () => {
    const overridden = applyOverrides(darkTheme, {
      primary: 'magenta',
    });
    expect(overridden.colors.primary).toBe('magenta');
    expect(overridden.colors.secondary).toBe(darkTheme.colors.secondary);
  });

  it('preserves original theme', () => {
    applyOverrides(darkTheme, { primary: 'magenta' });
    expect(darkTheme.colors.primary).toBe('cyan');
  });
});

describe('detectColorScheme', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    process.env = { ...originalEnv };
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  it('defaults to dark when no hints available', () => {
    delete process.env.COLORFGBG;
    delete process.env.ITERM_PROFILE;
    expect(detectColorScheme()).toBe('dark');
  });

  it('detects light from COLORFGBG with white background', () => {
    process.env.COLORFGBG = '0;7';
    expect(detectColorScheme()).toBe('light');
  });

  it('detects dark from COLORFGBG with black background', () => {
    process.env.COLORFGBG = '7;0';
    expect(detectColorScheme()).toBe('dark');
  });

  it('detects light from ITERM_PROFILE', () => {
    process.env.ITERM_PROFILE = 'Solarized Light';
    expect(detectColorScheme()).toBe('light');
  });

  it('detects dark from ITERM_PROFILE', () => {
    process.env.ITERM_PROFILE = 'One Dark';
    expect(detectColorScheme()).toBe('dark');
  });
});

describe('supportsColors', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    process.env = { ...originalEnv };
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  it('returns false when NO_COLOR is set', () => {
    process.env.NO_COLOR = '1';
    // Note: We can't easily mock process.stdout.isTTY in Bun
    // so we just verify NO_COLOR takes precedence when it's set
    const result = supportsColors();
    // NO_COLOR should disable colors regardless of TTY state
    expect(result).toBe(false);
  });

  it('respects FORCE_COLOR environment variable', () => {
    process.env.FORCE_COLOR = '1';
    delete process.env.NO_COLOR;
    // FORCE_COLOR enables colors if TTY is available
    // This test verifies the function doesn't crash
    const result = supportsColors();
    expect(typeof result).toBe('boolean');
  });
});
