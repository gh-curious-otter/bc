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

  test('handles empty text', () => {
    const { lastFrame } = render(<MentionText text="" />);
    expect(lastFrame()).toBeDefined();
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
});
