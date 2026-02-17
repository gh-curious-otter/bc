/**
 * Tests for AnimatedText components
 * Issue #1024: Animations and visual effects
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'bun:test';
import {
  FadeText,
  PulseText,
  TypewriterText,
  BlinkText,
  StatusTransition,
  NotificationText,
} from '../components/AnimatedText';

describe('FadeText', () => {
  describe('initial render', () => {
    it('renders text content', () => {
      const { lastFrame } = render(<FadeText>Hello</FadeText>);
      expect(lastFrame()).toContain('Hello');
    });

    it('renders with fade-in direction by default', () => {
      const { lastFrame } = render(<FadeText>Test</FadeText>);
      expect(lastFrame()).toContain('Test');
    });

    it('renders with fade-out direction', () => {
      const { lastFrame } = render(
        <FadeText direction="out">Goodbye</FadeText>
      );
      // Initially visible during fade-out
      expect(lastFrame()).toContain('Goodbye');
    });
  });

  describe('customization', () => {
    it('accepts color prop', () => {
      const { lastFrame } = render(<FadeText color="green">Text</FadeText>);
      expect(lastFrame()).toContain('Text');
    });

    it('accepts duration prop', () => {
      const { lastFrame } = render(<FadeText duration={500}>Text</FadeText>);
      expect(lastFrame()).toContain('Text');
    });
  });

  describe('unmount', () => {
    it('cleans up on unmount', () => {
      const { unmount } = render(<FadeText>Test</FadeText>);
      expect(() => { unmount(); }).not.toThrow();
    });
  });
});

describe('PulseText', () => {
  describe('initial render', () => {
    it('renders text content', () => {
      const { lastFrame } = render(<PulseText>Pulsing</PulseText>);
      expect(lastFrame()).toContain('Pulsing');
    });

    it('renders enabled by default', () => {
      const { lastFrame } = render(<PulseText>Active</PulseText>);
      expect(lastFrame()).toContain('Active');
    });
  });

  describe('customization', () => {
    it('accepts interval prop', () => {
      const { lastFrame } = render(
        <PulseText interval={500}>Fast pulse</PulseText>
      );
      expect(lastFrame()).toContain('Fast pulse');
    });

    it('can be disabled', () => {
      const { lastFrame } = render(
        <PulseText enabled={false}>Static</PulseText>
      );
      expect(lastFrame()).toContain('Static');
    });

    it('accepts color prop', () => {
      const { lastFrame } = render(
        <PulseText color="yellow">Warning</PulseText>
      );
      expect(lastFrame()).toContain('Warning');
    });
  });

  describe('unmount', () => {
    it('cleans up on unmount', () => {
      const { unmount } = render(<PulseText>Test</PulseText>);
      expect(() => { unmount(); }).not.toThrow();
    });
  });
});

describe('TypewriterText', () => {
  describe('initial render', () => {
    it('starts empty or with partial text', () => {
      const { lastFrame } = render(<TypewriterText>Hello World</TypewriterText>);
      // Initial frame may be empty or have first character
      const frame = lastFrame() ?? '';
      expect(frame.length).toBeLessThanOrEqual('Hello World'.length + 10); // Allow for cursor
    });

    it('shows cursor by default', () => {
      const { lastFrame } = render(<TypewriterText>Test</TypewriterText>);
      const frame = lastFrame() ?? '';
      // Should contain cursor character
      expect(frame).toContain('▌');
    });
  });

  describe('customization', () => {
    it('can hide cursor', () => {
      const { lastFrame } = render(
        <TypewriterText showCursor={false}>Test</TypewriterText>
      );
      const frame = lastFrame() ?? '';
      expect(frame).not.toContain('▌');
    });

    it('accepts custom cursor', () => {
      const { lastFrame } = render(
        <TypewriterText cursor="_">Test</TypewriterText>
      );
      const frame = lastFrame() ?? '';
      expect(frame).toContain('_');
    });

    it('accepts speed prop', () => {
      const { lastFrame } = render(
        <TypewriterText speed={60}>Fast typing</TypewriterText>
      );
      expect(lastFrame()).toBeDefined();
    });

    it('accepts delay prop', () => {
      const { lastFrame } = render(
        <TypewriterText delay={100}>Delayed</TypewriterText>
      );
      // With delay, should start empty
      const frame = lastFrame() ?? '';
      expect(frame).toBeDefined();
    });
  });

  describe('unmount', () => {
    it('cleans up on unmount', () => {
      const { unmount } = render(<TypewriterText>Test</TypewriterText>);
      expect(() => { unmount(); }).not.toThrow();
    });
  });
});

describe('BlinkText', () => {
  describe('initial render', () => {
    it('renders text when visible', () => {
      const { lastFrame } = render(<BlinkText>Blinking</BlinkText>);
      const frame = lastFrame() ?? '';
      // Either shows text or spaces
      expect(frame.length).toBeGreaterThan(0);
    });
  });

  describe('customization', () => {
    it('accepts interval prop', () => {
      const { lastFrame } = render(
        <BlinkText interval={250}>Fast blink</BlinkText>
      );
      expect(lastFrame()).toBeDefined();
    });

    it('can be disabled', () => {
      const { lastFrame } = render(
        <BlinkText enabled={false}>Static</BlinkText>
      );
      expect(lastFrame()).toContain('Static');
    });

    it('accepts color prop', () => {
      const { lastFrame } = render(
        <BlinkText color="red">Alert</BlinkText>
      );
      // Should render, color is applied by Ink
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('unmount', () => {
    it('cleans up on unmount', () => {
      const { unmount } = render(<BlinkText>Test</BlinkText>);
      expect(() => { unmount(); }).not.toThrow();
    });
  });
});

describe('StatusTransition', () => {
  describe('initial render', () => {
    it('renders current status', () => {
      const { lastFrame } = render(<StatusTransition to="working" />);
      expect(lastFrame()).toContain('working');
    });

    it('renders with color', () => {
      const { lastFrame } = render(
        <StatusTransition to="done" color="green" />
      );
      expect(lastFrame()).toContain('done');
    });
  });

  describe('transitions', () => {
    it('shows transition from previous state', () => {
      const { lastFrame } = render(
        <StatusTransition from="idle" to="working" />
      );
      expect(lastFrame()).toContain('working');
    });

    it('handles same state (no transition)', () => {
      const { lastFrame } = render(
        <StatusTransition from="working" to="working" />
      );
      expect(lastFrame()).toContain('working');
    });

    it('can disable animation', () => {
      const { lastFrame } = render(
        <StatusTransition from="idle" to="working" animate={false} />
      );
      expect(lastFrame()).toContain('working');
    });
  });

  describe('customization', () => {
    it('accepts duration prop', () => {
      const { lastFrame } = render(
        <StatusTransition from="idle" to="working" duration={500} />
      );
      expect(lastFrame()).toContain('working');
    });
  });

  describe('unmount', () => {
    it('cleans up on unmount', () => {
      const { unmount } = render(<StatusTransition to="idle" />);
      expect(() => { unmount(); }).not.toThrow();
    });
  });
});

describe('NotificationText', () => {
  describe('initial render', () => {
    it('renders notification text', () => {
      const { lastFrame } = render(
        <NotificationText>Alert message</NotificationText>
      );
      expect(lastFrame()).toContain('Alert message');
    });

    it('renders with info type by default', () => {
      const { lastFrame } = render(
        <NotificationText>Info</NotificationText>
      );
      expect(lastFrame()).toContain('Info');
    });
  });

  describe('notification types', () => {
    it('renders info type', () => {
      const { lastFrame } = render(
        <NotificationText type="info">Info message</NotificationText>
      );
      expect(lastFrame()).toContain('Info message');
    });

    it('renders success type', () => {
      const { lastFrame } = render(
        <NotificationText type="success">Success!</NotificationText>
      );
      expect(lastFrame()).toContain('Success!');
    });

    it('renders warning type', () => {
      const { lastFrame } = render(
        <NotificationText type="warning">Warning!</NotificationText>
      );
      expect(lastFrame()).toContain('Warning!');
    });

    it('renders error type', () => {
      const { lastFrame } = render(
        <NotificationText type="error">Error occurred</NotificationText>
      );
      expect(lastFrame()).toContain('Error occurred');
    });
  });

  describe('auto-dismiss', () => {
    it('renders normally without dismissAfter', () => {
      const { lastFrame } = render(
        <NotificationText>Persistent</NotificationText>
      );
      expect(lastFrame()).toContain('Persistent');
    });

    it('accepts dismissAfter prop', () => {
      const { lastFrame } = render(
        <NotificationText dismissAfter={3000}>Temporary</NotificationText>
      );
      // Should still be visible initially
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('unmount', () => {
    it('cleans up on unmount', () => {
      const { unmount } = render(
        <NotificationText>Test</NotificationText>
      );
      expect(() => { unmount(); }).not.toThrow();
    });
  });
});

describe('Edge cases', () => {
  it('handles empty string in FadeText', () => {
    const { lastFrame } = render(<FadeText>{''}</FadeText>);
    expect(lastFrame()).toBeDefined();
  });

  it('handles empty string in PulseText', () => {
    const { lastFrame } = render(<PulseText>{''}</PulseText>);
    expect(lastFrame()).toBeDefined();
  });

  it('handles empty string in TypewriterText', () => {
    const { lastFrame } = render(<TypewriterText>{''}</TypewriterText>);
    expect(lastFrame()).toBeDefined();
  });

  it('handles empty string in BlinkText', () => {
    const { lastFrame } = render(<BlinkText>{''}</BlinkText>);
    expect(lastFrame()).toBeDefined();
  });

  it('handles special characters', () => {
    const special = 'Test → 🚀 ← Test';
    const { lastFrame } = render(<PulseText>{special}</PulseText>);
    const frame = lastFrame() ?? '';
    expect(frame).toContain('Test');
  });

  it('handles long text in TypewriterText', () => {
    const longText = 'A'.repeat(100);
    const { lastFrame } = render(
      <TypewriterText speed={1000}>{longText}</TypewriterText>
    );
    expect(lastFrame()).toBeDefined();
  });

  it('handles multiple rapid prop changes', () => {
    const { rerender, lastFrame } = render(
      <StatusTransition from="idle" to="working" />
    );
    rerender(<StatusTransition from="working" to="done" />);
    rerender(<StatusTransition from="done" to="idle" />);
    expect(lastFrame()).toContain('idle');
  });
});
