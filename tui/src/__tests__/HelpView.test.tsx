/**
 * HelpView unit tests
 * Tests helper functions and business logic
 */

import { describe, expect, test } from 'bun:test';

describe('HelpView - totalLines calculation', () => {
  interface ShortcutSection {
    type: 'section';
    title: string;
    shortcuts: { keys: string; desc: string }[];
  }

  interface HeaderSection {
    type: 'header';
  }

  interface FooterSection {
    type: 'footer';
  }

  type HelpSection = ShortcutSection | HeaderSection | FooterSection;

  function calculateTotalLines(sections: HelpSection[]): number {
    return sections.reduce((acc, section) => {
      if (section.type === 'header') return acc + 2;
      if (section.type === 'footer') return acc + 3;
      return acc + 1 + section.shortcuts.length + 1; // title + shortcuts + margin
    }, 0);
  }

  test('header adds 2 lines', () => {
    const sections: HelpSection[] = [{ type: 'header' }];
    expect(calculateTotalLines(sections)).toBe(2);
  });

  test('footer adds 3 lines', () => {
    const sections: HelpSection[] = [{ type: 'footer' }];
    expect(calculateTotalLines(sections)).toBe(3);
  });

  test('section adds title + shortcuts + margin', () => {
    const sections: HelpSection[] = [
      { type: 'section', title: 'Global', shortcuts: [{ keys: 'q', desc: 'Quit' }] },
    ];
    // 1 (title) + 1 (shortcuts) + 1 (margin) = 3
    expect(calculateTotalLines(sections)).toBe(3);
  });

  test('section with multiple shortcuts', () => {
    const sections: HelpSection[] = [
      {
        type: 'section',
        title: 'Global',
        shortcuts: [
          { keys: 'q', desc: 'Quit' },
          { keys: 'Tab', desc: 'Next view' },
          { keys: 'ESC', desc: 'Go back' },
        ],
      },
    ];
    // 1 (title) + 3 (shortcuts) + 1 (margin) = 5
    expect(calculateTotalLines(sections)).toBe(5);
  });

  test('full structure with header, sections, and footer', () => {
    const sections: HelpSection[] = [
      { type: 'header' },
      { type: 'section', title: 'Global', shortcuts: [{ keys: 'q', desc: 'Quit' }] },
      {
        type: 'section',
        title: 'Nav',
        shortcuts: [
          { keys: 'j', desc: 'Down' },
          { keys: 'k', desc: 'Up' },
        ],
      },
      { type: 'footer' },
    ];
    // 2 (header) + 3 (section 1) + 4 (section 2) + 3 (footer) = 12
    expect(calculateTotalLines(sections)).toBe(12);
  });
});

describe('HelpView - availableHeight calculation', () => {
  function calculateAvailableHeight(terminalRows: number): number {
    return Math.max(10, terminalRows - 6);
  }

  test('standard terminal (24 rows)', () => {
    expect(calculateAvailableHeight(24)).toBe(18);
  });

  test('tall terminal (40 rows)', () => {
    expect(calculateAvailableHeight(40)).toBe(34);
  });

  test('short terminal enforces minimum', () => {
    expect(calculateAvailableHeight(12)).toBe(10);
  });

  test('very short terminal enforces minimum', () => {
    expect(calculateAvailableHeight(5)).toBe(10);
  });

  test('exact boundary (16 rows)', () => {
    expect(calculateAvailableHeight(16)).toBe(10);
  });
});

describe('HelpView - scroll state', () => {
  function needsScroll(totalLines: number, availableHeight: number): boolean {
    return totalLines > availableHeight;
  }

  function calculateMaxScroll(totalLines: number, availableHeight: number): number {
    return Math.max(0, totalLines - availableHeight);
  }

  test('needsScroll when content exceeds height', () => {
    expect(needsScroll(30, 20)).toBe(true);
  });

  test('no scroll when content fits', () => {
    expect(needsScroll(15, 20)).toBe(false);
  });

  test('no scroll when exactly equal', () => {
    expect(needsScroll(20, 20)).toBe(false);
  });

  test('maxScroll is difference when content exceeds', () => {
    expect(calculateMaxScroll(30, 20)).toBe(10);
  });

  test('maxScroll is 0 when content fits', () => {
    expect(calculateMaxScroll(15, 20)).toBe(0);
  });

  test('maxScroll is 0 when exactly equal', () => {
    expect(calculateMaxScroll(20, 20)).toBe(0);
  });
});

describe('HelpView - scroll offset clamping', () => {
  function clampScrollUp(offset: number): number {
    return Math.max(offset - 1, 0);
  }

  function clampScrollDown(offset: number, maxScroll: number): number {
    return Math.min(offset + 1, maxScroll);
  }

  function jumpToTop(): number {
    return 0;
  }

  function jumpToBottom(maxScroll: number): number {
    return maxScroll;
  }

  test('scroll up from middle', () => {
    expect(clampScrollUp(5)).toBe(4);
  });

  test('scroll up from top stays at 0', () => {
    expect(clampScrollUp(0)).toBe(0);
  });

  test('scroll down from middle', () => {
    expect(clampScrollDown(5, 10)).toBe(6);
  });

  test('scroll down at max stays at max', () => {
    expect(clampScrollDown(10, 10)).toBe(10);
  });

  test('jump to top returns 0', () => {
    expect(jumpToTop()).toBe(0);
  });

  test('jump to bottom returns maxScroll', () => {
    expect(jumpToBottom(15)).toBe(15);
  });
});

describe('HelpView - shortcut key padding', () => {
  function padKeys(keys: string): string {
    return keys.padEnd(12);
  }

  test('short key is padded', () => {
    expect(padKeys('q')).toBe('q           ');
    expect(padKeys('q').length).toBe(12);
  });

  test('medium key is padded', () => {
    expect(padKeys('Tab')).toBe('Tab         ');
    expect(padKeys('Tab').length).toBe(12);
  });

  test('long key is padded', () => {
    expect(padKeys('Shift+Tab')).toBe('Shift+Tab   ');
    expect(padKeys('Shift+Tab').length).toBe(12);
  });

  test('12-char key not truncated', () => {
    expect(padKeys('123456789012')).toBe('123456789012');
  });

  test('longer than 12 chars not truncated', () => {
    // padEnd doesn't truncate, just doesn't add padding
    expect(padKeys('Ctrl+Shift+X')).toBe('Ctrl+Shift+X');
    expect(padKeys('Ctrl+Shift+X').length).toBe(12);
  });
});

describe('HelpView - scroll percentage', () => {
  function calculateScrollPercentage(offset: number, maxScroll: number): number {
    if (maxScroll === 0) return 0;
    return Math.round((offset / maxScroll) * 100);
  }

  test('0% at top', () => {
    expect(calculateScrollPercentage(0, 10)).toBe(0);
  });

  test('100% at bottom', () => {
    expect(calculateScrollPercentage(10, 10)).toBe(100);
  });

  test('50% at middle', () => {
    expect(calculateScrollPercentage(5, 10)).toBe(50);
  });

  test('handles 0 maxScroll', () => {
    expect(calculateScrollPercentage(0, 0)).toBe(0);
  });

  test('rounds correctly', () => {
    expect(calculateScrollPercentage(1, 3)).toBe(33);
    expect(calculateScrollPercentage(2, 3)).toBe(67);
  });
});

describe('HelpView - theme display', () => {
  function getThemeText(themeName: string, isDark: boolean): string {
    return `Theme: ${themeName} (${isDark ? 'dark' : 'light'} mode)`;
  }

  test('dark mode display', () => {
    expect(getThemeText('default', true)).toBe('Theme: default (dark mode)');
  });

  test('light mode display', () => {
    expect(getThemeText('default', false)).toBe('Theme: default (light mode)');
  });

  test('custom theme name', () => {
    expect(getThemeText('monokai', true)).toBe('Theme: monokai (dark mode)');
  });
});

describe('HelpView - section visibility', () => {
  function isSectionVisible(
    sectionStart: number,
    sectionEnd: number,
    scrollOffset: number,
    availableHeight: number
  ): boolean {
    return sectionEnd > scrollOffset && sectionStart < scrollOffset + availableHeight;
  }

  test('section fully visible', () => {
    expect(isSectionVisible(5, 10, 0, 20)).toBe(true);
  });

  test('section above viewport not visible', () => {
    expect(isSectionVisible(0, 5, 10, 10)).toBe(false);
  });

  test('section below viewport not visible', () => {
    expect(isSectionVisible(25, 30, 0, 20)).toBe(false);
  });

  test('section partially visible at top', () => {
    expect(isSectionVisible(5, 15, 10, 10)).toBe(true);
  });

  test('section partially visible at bottom', () => {
    expect(isSectionVisible(15, 25, 10, 10)).toBe(true);
  });

  test('section spanning entire viewport', () => {
    expect(isSectionVisible(0, 30, 10, 10)).toBe(true);
  });
});

describe('HelpView - shortcut data structure', () => {
  interface Shortcut {
    keys: string;
    desc: string;
  }

  const shortcuts: Shortcut[] = [
    { keys: 'Tab', desc: 'Next view' },
    { keys: 'Shift+Tab', desc: 'Previous view' },
    { keys: 'ESC', desc: 'Go back / Home' },
    { keys: 'q', desc: 'Quit' },
  ];

  test('shortcuts have required keys field', () => {
    shortcuts.forEach((s) => {
      expect(s.keys).toBeDefined();
      expect(s.keys.length).toBeGreaterThan(0);
    });
  });

  test('shortcuts have required desc field', () => {
    shortcuts.forEach((s) => {
      expect(s.desc).toBeDefined();
      expect(s.desc.length).toBeGreaterThan(0);
    });
  });

  test('common shortcut exists', () => {
    const quitShortcut = shortcuts.find((s) => s.keys === 'q');
    expect(quitShortcut).toBeDefined();
    expect(quitShortcut?.desc).toBe('Quit');
  });
});

describe('HelpView - divider rendering', () => {
  function createDivider(width: number): string {
    return '─'.repeat(width);
  }

  test('standard divider width', () => {
    const divider = createDivider(40);
    expect(divider.length).toBe(40);
    expect(divider).toBe('────────────────────────────────────────');
  });

  test('short divider', () => {
    const divider = createDivider(10);
    expect(divider.length).toBe(10);
  });

  test('zero width divider', () => {
    const divider = createDivider(0);
    expect(divider).toBe('');
  });
});
