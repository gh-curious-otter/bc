import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'bun:test';
import { Reaction, ReactionBar } from '../components/Reaction';

describe('Reaction', () => {
  describe('basic rendering', () => {
    it('renders ack reaction', () => {
      const { lastFrame } = render(<Reaction type="ack" />);
      expect(lastFrame()).toContain('✓');
    });

    it('renders plus reaction', () => {
      const { lastFrame } = render(<Reaction type="plus" />);
      expect(lastFrame()).toContain('➕');
    });

    it('renders check reaction', () => {
      const { lastFrame } = render(<Reaction type="check" />);
      expect(lastFrame()).toContain('✅');
    });

    it('renders thumbsup reaction', () => {
      const { lastFrame } = render(<Reaction type="thumbsup" />);
      expect(lastFrame()).toContain('👍');
    });

    it('renders heart reaction', () => {
      const { lastFrame } = render(<Reaction type="heart" />);
      expect(lastFrame()).toContain('❤️');
    });
  });

  describe('reaction count', () => {
    it('renders without count when count is 1', () => {
      const { lastFrame } = render(<Reaction type="ack" count={1} />);
      const frame = lastFrame();
      expect(frame).toContain('✓');
      // Should not show count for 1
      expect(frame).not.toContain('1');
    });

    it('renders count when greater than 1', () => {
      const { lastFrame } = render(<Reaction type="thumbsup" count={3} />);
      expect(lastFrame()).toContain('3');
    });

    it('renders default count of 1', () => {
      const { lastFrame } = render(<Reaction type="ack" />);
      expect(lastFrame()).toContain('✓');
    });

    it('renders large count', () => {
      const { lastFrame } = render(<Reaction type="heart" count={99} />);
      expect(lastFrame()).toContain('99');
    });

    it('renders zero count', () => {
      const { lastFrame } = render(<Reaction type="ack" count={0} />);
      const frame = lastFrame();
      expect(frame).toContain('✓');
    });
  });

  describe('ownership', () => {
    it('renders as not own by default', () => {
      const { lastFrame } = render(<Reaction type="heart" />);
      expect(lastFrame()).toContain('❤️');
    });

    it('renders as own reaction', () => {
      const { lastFrame } = render(<Reaction type="thumbsup" isOwn={true} />);
      expect(lastFrame()).toContain('👍');
    });

    it('renders as not own reaction', () => {
      const { lastFrame } = render(<Reaction type="ack" isOwn={false} />);
      expect(lastFrame()).toContain('✓');
    });
  });

  describe('color mapping', () => {
    it('ack reaction has green color', () => {
      const { lastFrame } = render(<Reaction type="ack" />);
      expect(lastFrame()).toContain('✓');
    });

    it('heart reaction has red color', () => {
      const { lastFrame } = render(<Reaction type="heart" />);
      expect(lastFrame()).toContain('❤️');
    });

    it('own reaction has cyan color', () => {
      const { lastFrame } = render(<Reaction type="check" isOwn={true} />);
      expect(lastFrame()).toContain('✅');
    });
  });
});

describe('ReactionBar', () => {
  describe('basic rendering', () => {
    it('renders single reaction', () => {
      const reactions = [{ type: 'thumbsup' as const, count: 1 }];
      const { lastFrame } = render(<ReactionBar reactions={reactions} />);
      expect(lastFrame()).toContain('👍');
    });

    it('renders multiple reactions', () => {
      const reactions = [
        { type: 'thumbsup' as const, count: 2 },
        { type: 'heart' as const, count: 1 },
      ];
      const { lastFrame } = render(<ReactionBar reactions={reactions} />);
      const frame = lastFrame();
      expect(frame).toContain('👍');
      expect(frame).toContain('❤️');
    });

    it('renders empty reactions', () => {
      const { lastFrame } = render(<ReactionBar reactions={[]} />);
      // Should render without error
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('reaction counts', () => {
    it('renders reaction with multiple counts', () => {
      const reactions = [
        { type: 'ack' as const, count: 5 },
        { type: 'check' as const, count: 3 },
      ];
      const { lastFrame } = render(<ReactionBar reactions={reactions} />);
      const frame = lastFrame();
      expect(frame).toContain('✓');
      expect(frame).toContain('✅');
      expect(frame).toContain('5');
      expect(frame).toContain('3');
    });
  });

  describe('ownership indicators', () => {
    it('renders own and other reactions', () => {
      const reactions = [
        { type: 'thumbsup' as const, count: 2, isOwn: true },
        { type: 'heart' as const, count: 1, isOwn: false },
      ];
      const { lastFrame } = render(<ReactionBar reactions={reactions} />);
      const frame = lastFrame();
      expect(frame).toContain('👍');
      expect(frame).toContain('❤️');
    });

    it('renders all own reactions', () => {
      const reactions = [
        { type: 'ack' as const, count: 1, isOwn: true },
        { type: 'check' as const, count: 1, isOwn: true },
      ];
      const { lastFrame } = render(<ReactionBar reactions={reactions} />);
      const frame = lastFrame();
      expect(frame).toContain('✓');
      expect(frame).toContain('✅');
    });
  });

  describe('all reaction types', () => {
    it('renders all reaction types together', () => {
      const reactions = [
        { type: 'ack' as const, count: 1 },
        { type: 'plus' as const, count: 1 },
        { type: 'check' as const, count: 1 },
        { type: 'thumbsup' as const, count: 1 },
        { type: 'heart' as const, count: 1 },
      ];
      const { lastFrame } = render(<ReactionBar reactions={reactions} />);
      const frame = lastFrame();
      expect(frame).toContain('✓');
      expect(frame).toContain('➕');
      expect(frame).toContain('✅');
      expect(frame).toContain('👍');
      expect(frame).toContain('❤️');
    });
  });
});
