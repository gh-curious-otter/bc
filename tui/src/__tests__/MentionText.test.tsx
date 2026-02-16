import { describe, expect, test } from 'bun:test';
import { render } from 'ink-testing-library';
import React from 'react';
import { MentionText } from '../components/MentionText';

describe('MentionText', () => {
  test('renders plain text without mentions', () => {
    const { lastFrame } = render(<MentionText text="Hello world" />);
    expect(lastFrame()).toContain('Hello world');
  });

  test('highlights @mentions', () => {
    const { lastFrame } = render(<MentionText text="Hello @eng-01" />);
    const output = lastFrame() ?? '';
    expect(output).toContain('Hello');
    expect(output).toContain('@eng-01');
  });

  test('highlights multiple mentions', () => {
    const { lastFrame } = render(
      <MentionText text="@eng-01 and @eng-02 are working" />
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('@eng-01');
    expect(output).toContain('@eng-02');
    expect(output).toContain('are working');
  });

  test('highlights self-mentions differently', () => {
    const { lastFrame } = render(
      <MentionText text="Hello @eng-04" currentUser="eng-04" />
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('@eng-04');
  });

  test('highlights broadcast mentions (@all)', () => {
    const { lastFrame } = render(<MentionText text="@all please review" />);
    const output = lastFrame() ?? '';
    expect(output).toContain('@all');
    expect(output).toContain('please review');
  });

  test('highlights broadcast mentions (@everyone)', () => {
    const { lastFrame } = render(<MentionText text="@everyone meeting now" />);
    const output = lastFrame() ?? '';
    expect(output).toContain('@everyone');
  });

  test('handles text with no mentions', () => {
    const { lastFrame } = render(<MentionText text="No mentions here" />);
    expect(lastFrame()).toContain('No mentions here');
  });

  test('handles empty text with placeholder', () => {
    const { lastFrame } = render(<MentionText text="" />);
    expect(lastFrame()).toContain('(empty)');
  });

  test('handles whitespace-only text with placeholder', () => {
    const { lastFrame } = render(<MentionText text="   " />);
    expect(lastFrame()).toContain('(empty)');
  });

  test('handles newline-only text with placeholder', () => {
    const newlineText = "\n\n";
    const { lastFrame } = render(<MentionText text={newlineText} />);
    expect(lastFrame()).toContain('(empty)');
  });

  test('handles tab-only text with placeholder', () => {
    const tabText = "\t\t";
    const { lastFrame } = render(<MentionText text={tabText} />);
    expect(lastFrame()).toContain('(empty)');
  });

  test('handles mention at start of text', () => {
    const { lastFrame } = render(<MentionText text="@wise-owl says hello" />);
    const output = lastFrame() ?? '';
    expect(output).toContain('@wise-owl');
    expect(output).toContain('says hello');
  });

  test('handles mention at end of text', () => {
    const { lastFrame } = render(<MentionText text="Message for @clever-fox" />);
    const output = lastFrame() ?? '';
    expect(output).toContain('Message for');
    expect(output).toContain('@clever-fox');
  });

  // Edge cases for #915 - ensure all empty/invalid cases show placeholder
  test('handles undefined text gracefully', () => {
    // TypeScript wouldn't allow this, but runtime JSON parsing might produce it
    const { lastFrame } = render(<MentionText text={undefined as unknown as string} />);
    expect(lastFrame()).toContain('(empty)');
  });

  test('handles null text gracefully', () => {
    // TypeScript wouldn't allow this, but runtime JSON parsing might produce it
    const { lastFrame } = render(<MentionText text={null as unknown as string} />);
    expect(lastFrame()).toContain('(empty)');
  });

  test('handles mixed whitespace text with placeholder', () => {
    const mixedWhitespace = "  \n\t  \n  ";
    const { lastFrame } = render(<MentionText text={mixedWhitespace} />);
    expect(lastFrame()).toContain('(empty)');
  });

  test('handles very long text without truncation', () => {
    const longText = 'A'.repeat(500);
    const { lastFrame } = render(<MentionText text={longText} />);
    // Should not show (empty), should render the text
    expect(lastFrame()).not.toContain('(empty)');
    expect(lastFrame()).toContain('AAA');
  });

  test('handles text with special characters', () => {
    const { lastFrame } = render(<MentionText text="Hello <script>alert('xss')</script> world" />);
    const output = lastFrame() ?? '';
    expect(output).toContain('Hello');
    expect(output).toContain('world');
  });

  test('handles text with unicode/emoji', () => {
    const { lastFrame } = render(<MentionText text="Hello 👋 world 🌍" />);
    const output = lastFrame() ?? '';
    expect(output).toContain('Hello');
    expect(output).toContain('world');
  });

  test('handles multiline text', () => {
    const { lastFrame } = render(<MentionText text="Line 1\nLine 2\nLine 3" />);
    const output = lastFrame() ?? '';
    expect(output).toContain('Line 1');
    expect(output).toContain('Line 2');
  });
});
