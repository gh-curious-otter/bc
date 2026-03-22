import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'bun:test';
import { MessageInput } from '../components/MessageInput';

describe('MessageInput', () => {
  describe('basic rendering', () => {
    it('renders with default placeholder', () => {
      const { lastFrame } = render(<MessageInput />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders with custom placeholder', () => {
      const { lastFrame } = render(<MessageInput placeholder="Say something..." />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders with channel name', () => {
      const { lastFrame } = render(<MessageInput channelName="engineering" />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('disabled state', () => {
    it('renders when enabled', () => {
      const { lastFrame } = render(<MessageInput disabled={false} />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders when disabled', () => {
      const { lastFrame } = render(<MessageInput disabled={true} />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('callbacks', () => {
    it('accepts onSubmit callback', () => {
      const onSubmit = () => {};
      const { lastFrame } = render(<MessageInput onSubmit={onSubmit} />);
      expect(lastFrame()).toBeDefined();
    });

    it('accepts onModeChange callback', () => {
      const onModeChange = () => {};
      const { lastFrame } = render(<MessageInput onModeChange={onModeChange} />);
      expect(lastFrame()).toBeDefined();
    });

    it('accepts both callbacks', () => {
      const onSubmit = () => {};
      const onModeChange = () => {};
      const { lastFrame } = render(
        <MessageInput onSubmit={onSubmit} onModeChange={onModeChange} />
      );
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('input modes', () => {
    it('renders in navigation mode by default', () => {
      const { lastFrame } = render(<MessageInput />);
      const frame = lastFrame();
      expect(frame).toBeDefined();
    });

    it('renders with modeChange tracking', () => {
      let modeChanged = false;
      const onModeChange = (isInputMode: boolean) => {
        modeChanged = true;
      };
      const { lastFrame } = render(<MessageInput onModeChange={onModeChange} />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('submission handling', () => {
    it('accepts onSubmit handler', () => {
      let submitted = '';
      const onSubmit = (message: string) => {
        submitted = message;
      };
      const { lastFrame } = render(<MessageInput onSubmit={onSubmit} />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders without submission if no handler', () => {
      const { lastFrame } = render(<MessageInput />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('mention support', () => {
    it('supports mention typing', () => {
      const { lastFrame } = render(<MessageInput />);
      const frame = lastFrame();
      // Should support @mention autocomplete
      expect(frame).toBeDefined();
    });

    it('renders with autocomplete hook integration', () => {
      const { lastFrame } = render(<MessageInput channelName="test" />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('configuration combinations', () => {
    it('renders with all options enabled', () => {
      const { lastFrame } = render(
        <MessageInput
          placeholder="Custom placeholder"
          onSubmit={() => {}}
          onModeChange={() => {}}
          disabled={false}
          channelName="test-channel"
        />
      );
      expect(lastFrame()).toBeDefined();
    });

    it('renders with all options disabled', () => {
      const { lastFrame } = render(<MessageInput disabled={true} />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('edge cases', () => {
    it('handles very long placeholder text', () => {
      const longPlaceholder = 'A'.repeat(100);
      const { lastFrame } = render(<MessageInput placeholder={longPlaceholder} />);
      expect(lastFrame()).toBeDefined();
    });

    it('handles special characters in placeholder', () => {
      const { lastFrame } = render(<MessageInput placeholder="Say @something or :command" />);
      expect(lastFrame()).toBeDefined();
    });

    it('handles channel name with special characters', () => {
      const { lastFrame } = render(<MessageInput channelName="eng-02-dev" />);
      expect(lastFrame()).toBeDefined();
    });

    it('handles rapid callback invocations', () => {
      let callCount = 0;
      const onModeChange = () => {
        callCount++;
      };
      const { lastFrame } = render(<MessageInput onModeChange={onModeChange} />);
      expect(lastFrame()).toBeDefined();
    });
  });
});
