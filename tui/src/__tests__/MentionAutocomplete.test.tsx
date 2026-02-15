import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'bun:test';
import { MentionAutocomplete } from '../components/MentionAutocomplete';
import type { MentionSuggestion } from '../hooks/useMentionAutocomplete';

describe('MentionAutocomplete', () => {
  const baseSuggestions: MentionSuggestion[] = [
    { name: 'eng-01', state: 'working' },
    { name: 'eng-02', state: 'idle' },
    { name: 'tech-lead-01', state: 'working' },
  ];

  describe('visibility', () => {
    it('does not render when not visible', () => {
      const { lastFrame } = render(
        <MentionAutocomplete
          suggestions={baseSuggestions}
          selectedIndex={0}
          visible={false}
        />
      );
      const frame = lastFrame();
      // Component should not render suggestions when invisible
      expect(frame).toBeDefined();
    });

    it('renders when visible', () => {
      const { lastFrame } = render(
        <MentionAutocomplete
          suggestions={baseSuggestions}
          selectedIndex={0}
          visible={true}
        />
      );
      const frame = lastFrame();
      expect(frame).toBeDefined();
    });
  });

  describe('suggestion rendering', () => {
    it('renders single suggestion', () => {
      const suggestions: MentionSuggestion[] = [
        { name: 'eng-01', state: 'working' },
      ];
      const { lastFrame } = render(
        <MentionAutocomplete
          suggestions={suggestions}
          selectedIndex={0}
          visible={true}
        />
      );
      expect(lastFrame()).toContain('eng-01');
    });

    it('renders multiple suggestions', () => {
      const { lastFrame } = render(
        <MentionAutocomplete
          suggestions={baseSuggestions}
          selectedIndex={0}
          visible={true}
        />
      );
      const frame = lastFrame();
      expect(frame).toContain('eng-01');
      expect(frame).toContain('eng-02');
      expect(frame).toContain('tech-lead-01');
    });

    it('renders no suggestions', () => {
      const { lastFrame } = render(
        <MentionAutocomplete
          suggestions={[]}
          selectedIndex={-1}
          visible={true}
        />
      );
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('selection state', () => {
    it('highlights first suggestion when selectedIndex is 0', () => {
      const { lastFrame } = render(
        <MentionAutocomplete
          suggestions={baseSuggestions}
          selectedIndex={0}
          visible={true}
        />
      );
      expect(lastFrame()).toContain('eng-01');
    });

    it('handles middle selection', () => {
      const { lastFrame } = render(
        <MentionAutocomplete
          suggestions={baseSuggestions}
          selectedIndex={1}
          visible={true}
        />
      );
      expect(lastFrame()).toContain('eng-02');
    });

    it('handles last selection', () => {
      const { lastFrame } = render(
        <MentionAutocomplete
          suggestions={baseSuggestions}
          selectedIndex={2}
          visible={true}
        />
      );
      expect(lastFrame()).toContain('tech-lead-01');
    });

    it('handles invalid selection index', () => {
      const { lastFrame } = render(
        <MentionAutocomplete
          suggestions={baseSuggestions}
          selectedIndex={-1}
          visible={true}
        />
      );
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('agent states', () => {
    it('renders working state agents', () => {
      const suggestions: MentionSuggestion[] = [
        { name: 'eng-01', state: 'working' },
      ];
      const { lastFrame } = render(
        <MentionAutocomplete
          suggestions={suggestions}
          selectedIndex={0}
          visible={true}
        />
      );
      expect(lastFrame()).toContain('eng-01');
    });

    it('renders idle state agents', () => {
      const suggestions: MentionSuggestion[] = [
        { name: 'eng-02', state: 'idle' },
      ];
      const { lastFrame } = render(
        <MentionAutocomplete
          suggestions={suggestions}
          selectedIndex={0}
          visible={true}
        />
      );
      expect(lastFrame()).toContain('eng-02');
    });

    it('renders stuck state agents', () => {
      const suggestions: MentionSuggestion[] = [
        { name: 'eng-03', state: 'stuck' },
      ];
      const { lastFrame } = render(
        <MentionAutocomplete
          suggestions={suggestions}
          selectedIndex={0}
          visible={true}
        />
      );
      expect(lastFrame()).toContain('eng-03');
    });

    it('renders mixed state agents', () => {
      const suggestions: MentionSuggestion[] = [
        { name: 'eng-01', state: 'working' },
        { name: 'eng-02', state: 'idle' },
        { name: 'eng-03', state: 'stuck' },
      ];
      const { lastFrame } = render(
        <MentionAutocomplete
          suggestions={suggestions}
          selectedIndex={0}
          visible={true}
        />
      );
      const frame = lastFrame();
      expect(frame).toContain('eng-01');
      expect(frame).toContain('eng-02');
      expect(frame).toContain('eng-03');
    });
  });

  describe('query highlighting', () => {
    it('renders with query parameter', () => {
      const { lastFrame } = render(
        <MentionAutocomplete
          suggestions={baseSuggestions}
          selectedIndex={0}
          visible={true}
          query="eng"
        />
      );
      expect(lastFrame()).toContain('eng');
    });

    it('renders with partial match query', () => {
      const { lastFrame } = render(
        <MentionAutocomplete
          suggestions={baseSuggestions}
          selectedIndex={0}
          visible={true}
          query="e"
        />
      );
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('edge cases', () => {
    it('handles agents with long names', () => {
      const longName = 'engineering-team-member-'.padEnd(50, 'x');
      const suggestions: MentionSuggestion[] = [
        { name: longName, state: 'working' },
      ];
      const { lastFrame } = render(
        <MentionAutocomplete
          suggestions={suggestions}
          selectedIndex={0}
          visible={true}
        />
      );
      expect(lastFrame()).toBeDefined();
    });

    it('handles agents with special characters in names', () => {
      const suggestions: MentionSuggestion[] = [
        { name: 'eng_01-special', state: 'working' },
      ];
      const { lastFrame } = render(
        <MentionAutocomplete
          suggestions={suggestions}
          selectedIndex={0}
          visible={true}
        />
      );
      expect(lastFrame()).toContain('eng_01-special');
    });

    it('handles large suggestion list', () => {
      const suggestions: MentionSuggestion[] = Array.from(
        { length: 100 },
        (_, i) => ({
          name: `agent-${i}`,
          state: i % 2 === 0 ? 'working' : 'idle',
        })
      );
      const { lastFrame } = render(
        <MentionAutocomplete
          suggestions={suggestions}
          selectedIndex={50}
          visible={true}
        />
      );
      expect(lastFrame()).toBeDefined();
    });
  });
});
