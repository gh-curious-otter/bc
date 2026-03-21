/**
 * InlineEditor - TUI-native text editing component
 * Issue #858 - Replace nano dependency with inline editing
 */

import React, { useState, useCallback, useEffect, memo } from 'react';
import { Box, Text, useInput } from 'ink';
import { useTheme } from '../theme';

export interface InlineEditorProps {
  /** Initial value */
  initialValue?: string;
  /** Placeholder text when empty */
  placeholder?: string;
  /** Label shown above editor */
  label?: string;
  /** Whether this is multi-line */
  multiline?: boolean;
  /** Max height for multi-line (default: 10) */
  maxHeight?: number;
  /** Whether the editor is focused */
  focused?: boolean;
  /** Called when value changes */
  onChange?: (value: string) => void;
  /** Called when save is triggered (Ctrl+S or Enter for single-line) */
  onSave?: (value: string) => void;
  /** Called when cancel is triggered (Escape) */
  onCancel?: () => void;
  /** Disable input handling */
  disableInput?: boolean;
}

/**
 * InlineEditor - Single or multi-line text editor
 *
 * Single-line: Enter saves, Escape cancels
 * Multi-line: Ctrl+S saves, Escape cancels, Enter adds newline
 */
export const InlineEditor = memo(function InlineEditor({
  initialValue = '',
  placeholder = 'Enter text...',
  label,
  multiline = false,
  maxHeight = 10,
  focused = true,
  onChange,
  onSave,
  onCancel,
  disableInput = false,
}: InlineEditorProps): React.ReactElement {
  const { theme } = useTheme();
  const [value, setValue] = useState(initialValue);
  const [cursorPos, setCursorPos] = useState(initialValue.length);
  const [cursorLine, setCursorLine] = useState(0);

  // Sync with initialValue changes
  useEffect(() => {
    setValue(initialValue);
    setCursorPos(initialValue.length);
  }, [initialValue]);

  // Get current line info for multi-line
  const lines = value.split('\n');
  const currentLineStart = lines
    .slice(0, cursorLine)
    .reduce((acc, line) => acc + line.length + 1, 0);
  const currentLineLength = lines[cursorLine]?.length ?? 0;
  const cursorPosInLine = cursorPos - currentLineStart;

  const handleInput = useCallback(
    (input: string, key: { ctrl: boolean; return: boolean; escape: boolean; backspace: boolean; delete: boolean; upArrow: boolean; downArrow: boolean; leftArrow: boolean; rightArrow: boolean; meta: boolean; tab: boolean }) => {
      // Save: Ctrl+S (multi-line) or Enter (single-line)
      if ((key.ctrl && input === 's') || (!multiline && key.return)) {
        onSave?.(value);
        return;
      }

      // Cancel: Escape
      if (key.escape) {
        onCancel?.();
        return;
      }

      // Newline (multi-line only)
      if (multiline && key.return) {
        const newValue =
          value.slice(0, cursorPos) + '\n' + value.slice(cursorPos);
        setValue(newValue);
        setCursorPos(cursorPos + 1);
        setCursorLine(cursorLine + 1);
        onChange?.(newValue);
        return;
      }

      // Backspace
      if (key.backspace || key.delete) {
        if (cursorPos > 0) {
          const newValue =
            value.slice(0, cursorPos - 1) + value.slice(cursorPos);
          setValue(newValue);
          setCursorPos(cursorPos - 1);
          // Handle line change for multi-line
          if (multiline && value[cursorPos - 1] === '\n') {
            setCursorLine(Math.max(0, cursorLine - 1));
          }
          onChange?.(newValue);
        }
        return;
      }

      // Arrow keys - navigation
      if (key.leftArrow) {
        if (cursorPos > 0) {
          setCursorPos(cursorPos - 1);
          if (multiline && value[cursorPos - 1] === '\n') {
            setCursorLine(Math.max(0, cursorLine - 1));
          }
        }
        return;
      }

      if (key.rightArrow) {
        if (cursorPos < value.length) {
          if (multiline && value[cursorPos] === '\n') {
            setCursorLine(cursorLine + 1);
          }
          setCursorPos(cursorPos + 1);
        }
        return;
      }

      if (multiline && key.upArrow) {
        if (cursorLine > 0) {
          const prevLineStart = lines
            .slice(0, cursorLine - 1)
            .reduce((acc, line) => acc + line.length + 1, 0);
          const prevLineLength = lines[cursorLine - 1]?.length ?? 0;
          const newPosInLine = Math.min(cursorPosInLine, prevLineLength);
          setCursorPos(prevLineStart + newPosInLine);
          setCursorLine(cursorLine - 1);
        }
        return;
      }

      if (multiline && key.downArrow) {
        if (cursorLine < lines.length - 1) {
          const nextLineStart =
            currentLineStart + currentLineLength + 1;
          const nextLineLength = lines[cursorLine + 1]?.length ?? 0;
          const newPosInLine = Math.min(cursorPosInLine, nextLineLength);
          setCursorPos(nextLineStart + newPosInLine);
          setCursorLine(cursorLine + 1);
        }
        return;
      }

      // Tab - ignore for now
      if (key.tab) {
        return;
      }

      // Regular character input
      if (input && !key.ctrl && !key.meta) {
        const newValue =
          value.slice(0, cursorPos) + input + value.slice(cursorPos);
        setValue(newValue);
        setCursorPos(cursorPos + input.length);
        onChange?.(newValue);
      }
    },
    [value, cursorPos, cursorLine, lines, currentLineStart, currentLineLength, cursorPosInLine, multiline, onChange, onSave, onCancel]
  );

  useInput(handleInput, { isActive: focused && !disableInput });

  // Render single-line
  if (!multiline) {
    const beforeCursor = value.slice(0, cursorPos);
    const atCursor = value[cursorPos] ?? ' ';
    const afterCursor = value.slice(cursorPos + 1);

    return (
      <Box flexDirection="column">
        {label && (
          <Box marginBottom={1}>
            <Text bold color={theme.colors.primary}>{label}</Text>
          </Box>
        )}
        <Box
          borderStyle="single"
          borderColor={focused ? theme.colors.primary : theme.colors.textMuted}
          paddingX={1}
        >
          {value.length === 0 ? (
            <Text dimColor>{placeholder}</Text>
          ) : (
            <Text>
              {beforeCursor}
              <Text inverse>{atCursor}</Text>
              {afterCursor}
            </Text>
          )}
        </Box>
        <Box marginTop={1}>
          <Text dimColor>
            [Enter] save | [Esc] cancel
          </Text>
        </Box>
      </Box>
    );
  }

  // Render multi-line
  const displayLines = lines.slice(0, maxHeight);
  const hasMore = lines.length > maxHeight;

  return (
    <Box flexDirection="column">
      {label && (
        <Box marginBottom={1}>
          <Text bold color="cyan">{label}</Text>
        </Box>
      )}
      <Box
        flexDirection="column"
        borderStyle="single"
        borderColor={focused ? theme.colors.primary : theme.colors.textMuted}
        paddingX={1}
        minHeight={3}
      >
        {value.length === 0 ? (
          <Text dimColor>{placeholder}</Text>
        ) : (
          displayLines.map((line, lineIdx) => {
            // Render cursor if this is the cursor line
            if (lineIdx === cursorLine && lineIdx < maxHeight) {
              const beforeCursor = line.slice(0, cursorPosInLine);
              const atCursor = line[cursorPosInLine] ?? ' ';
              const afterCursor = line.slice(cursorPosInLine + 1);
              return (
                <Text key={lineIdx}>
                  {beforeCursor}
                  <Text inverse>{atCursor}</Text>
                  {afterCursor}
                </Text>
              );
            }
            return <Text key={lineIdx}>{line || ' '}</Text>;
          })
        )}
        {hasMore && (
          <Text dimColor>... {lines.length - maxHeight} more lines</Text>
        )}
      </Box>
      <Box marginTop={1}>
        <Text dimColor>
          [Ctrl+S] save | [Esc] cancel | [Enter] newline
        </Text>
      </Box>
    </Box>
  );
});

/**
 * Modal wrapper for InlineEditor
 * Centers editor on screen with backdrop
 */
export interface EditorModalProps extends InlineEditorProps {
  /** Whether modal is visible */
  visible: boolean;
  /** Modal title */
  title?: string;
}

export const EditorModal = memo(function EditorModal({
  visible,
  title = 'Edit',
  ...editorProps
}: EditorModalProps): React.ReactElement | null {
  const { theme } = useTheme();

  if (!visible) return null;

  return (
    <Box
      flexDirection="column"
      alignItems="center"
      justifyContent="center"
      width="100%"
    >
      <Box
        flexDirection="column"
        borderStyle="double"
        borderColor={theme.colors.primary}
        padding={1}
        minWidth={50}
      >
        <Box marginBottom={1}>
          <Text bold color={theme.colors.primary}>{title}</Text>
        </Box>
        <InlineEditor {...editorProps} />
      </Box>
    </Box>
  );
});

export default InlineEditor;
