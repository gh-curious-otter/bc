import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'bun:test';
import { ChatMessage } from '../components/ChatMessage';

describe('ChatMessage', () => {
  const baseProps = {
    sender: 'eng-01',
    message: 'Hello world',
    timestamp: new Date().toISOString(),
  };

  describe('basic rendering', () => {
    it('renders sender name', () => {
      const { lastFrame } = render(<ChatMessage {...baseProps} />);
      expect(lastFrame()).toContain('eng-01');
    });

    it('renders message text', () => {
      const { lastFrame } = render(<ChatMessage {...baseProps} />);
      expect(lastFrame()).toContain('Hello world');
    });

    it('renders timestamp in relative format', () => {
      const { lastFrame } = render(<ChatMessage {...baseProps} />);
      const frame = lastFrame();
      // Should contain time indicator like "now", "1m ago", etc or gray colored text
      expect(frame).toContain('now');
    });
  });

  describe('role-based colors', () => {
    it('renders root sender with special color', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} sender="root" />
      );
      expect(lastFrame()).toContain('root');
    });

    it('renders tech-lead sender', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} sender="tech-lead-01" />
      );
      expect(lastFrame()).toContain('tech-lead-01');
    });

    it('renders engineer sender', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} sender="eng-02" />
      );
      expect(lastFrame()).toContain('eng-02');
    });
  });

  describe('read status', () => {
    it('does not show read indicator when isRead is true', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} isRead={true} />
      );
      // Should not have unread indicator
      expect(lastFrame()).not.toContain('●');
    });

    it('shows read indicator when isRead is false', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} isRead={false} />
      );
      expect(lastFrame()).toContain('●');
    });
  });

  describe('selection state', () => {
    it('renders without border when not selected', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} isSelected={false} />
      );
      const frame = lastFrame();
      // Should render normally without selection border
      expect(frame).toContain('eng-01');
    });

    it('renders with border when selected', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} isSelected={true} />
      );
      const frame = lastFrame();
      expect(frame).toContain('eng-01');
    });
  });

  describe('reactions', () => {
    it('does not render reaction bar when no reactions', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} reactions={[]} />
      );
      // Should only have message content, no reaction area
      expect(lastFrame()).toContain('Hello world');
    });

    it('renders reactions when provided', () => {
      const reactions = [
        { type: 'thumbsup' as const, count: 2, isOwn: false },
      ];
      const { lastFrame } = render(
        <ChatMessage {...baseProps} reactions={reactions} />
      );
      const frame = lastFrame();
      expect(frame).toContain('Hello world');
      // Reactions should be rendered
    });

    it('handles multiple reactions', () => {
      const reactions = [
        { type: 'thumbsup' as const, count: 2 },
        { type: 'heart' as const, count: 1 },
      ];
      const { lastFrame } = render(
        <ChatMessage {...baseProps} reactions={reactions} />
      );
      expect(lastFrame()).toContain('Hello world');
    });
  });

  describe('timestamp formatting', () => {
    it('shows "now" for very recent messages', () => {
      const recent = new Date().toISOString();
      const { lastFrame } = render(
        <ChatMessage {...baseProps} timestamp={recent} />
      );
      expect(lastFrame()).toContain('now');
    });

    it('handles old timestamps gracefully', () => {
      const oldDate = new Date('2025-01-01').toISOString();
      const { lastFrame } = render(
        <ChatMessage {...baseProps} timestamp={oldDate} />
      );
      expect(lastFrame()).toContain('Jan');
    });

    it('handles invalid timestamp gracefully', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} timestamp="invalid" />
      );
      // When given an invalid timestamp, component catches error and displays fallback
      const frame = lastFrame();
      expect(frame).toContain('eng-01');
      expect(frame).toContain('Hello world');
    });
  });

  describe('special characters', () => {
    it('renders messages with @mentions', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} message="@eng-02 please review this" />
      );
      expect(lastFrame()).toContain('eng-02');
    });

    it('renders messages with special characters', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} message="Message with !@#$% chars" />
      );
      expect(lastFrame()).toContain('!@#$%');
    });

    it('renders multiline messages', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} message="Line 1\nLine 2\nLine 3" />
      );
      expect(lastFrame()).toContain('Line 1');
    });
  });

  describe('bubble styling and alignment', () => {
    it('shows "(you)" indicator for own messages', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} sender="eng-01" currentUser="eng-01" />
      );
      expect(lastFrame()).toContain('(you)');
    });

    it('does not show "(you)" for other users messages', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} sender="eng-02" currentUser="eng-01" />
      );
      expect(lastFrame()).not.toContain('(you)');
    });

    it('renders message without "(you)" when no currentUser provided', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} sender="eng-01" />
      );
      expect(lastFrame()).not.toContain('(you)');
    });

    it('renders own message with bubble border', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} sender="eng-01" currentUser="eng-01" />
      );
      // Should render with round border (indicated by border characters)
      expect(lastFrame()).toContain('eng-01');
      expect(lastFrame()).toContain('(you)');
    });

    it('renders other user message with bubble border', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} sender="eng-02" currentUser="eng-01" />
      );
      // Should render with round border (gray for others)
      expect(lastFrame()).toContain('eng-02');
      expect(lastFrame()).not.toContain('(you)');
    });

    it('applies custom maxBubbleWidth', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} maxBubbleWidth={40} />
      );
      // Should render without errors with custom width
      expect(lastFrame()).toContain('eng-01');
    });
  });

  describe('role colors for additional roles', () => {
    it('renders tl- prefix sender (tech lead)', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} sender="tl-01" />
      );
      expect(lastFrame()).toContain('tl-01');
    });

    it('renders mgr- prefix sender (manager)', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} sender="mgr-01" />
      );
      expect(lastFrame()).toContain('mgr-01');
    });

    it('renders pm- prefix sender (product manager)', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} sender="pm-01" />
      );
      expect(lastFrame()).toContain('pm-01');
    });

    it('renders ux- prefix sender', () => {
      const { lastFrame } = render(
        <ChatMessage {...baseProps} sender="ux-01" />
      );
      expect(lastFrame()).toContain('ux-01');
    });
  });
});
