/**
 * FilterBar - k9s-style /filter input
 *
 * Activated by pressing '/'. Shows a text input at the bottom.
 * Filter text is passed to the current view via useFilter() context.
 */

import React, { useState } from 'react';
import { Box, Text, useInput } from 'ink';
import { useTheme } from '../theme';
import { useFilter } from '../hooks/useFilter';

interface FilterBarProps {
  onClose: () => void;
}

export function FilterBar({ onClose }: FilterBarProps): React.ReactElement {
  const { theme } = useTheme();
  const { query, setFilter, clearFilter } = useFilter();
  const [input, setInput] = useState(query);

  useInput((char, key) => {
    if (key.escape) {
      clearFilter();
      onClose();
      return;
    }

    if (key.return) {
      setFilter(input);
      onClose();
      return;
    }

    if (key.backspace || key.delete) {
      setInput((prev) => {
        const next = prev.slice(0, -1);
        setFilter(next);
        return next;
      });
      return;
    }

    if (char && !key.ctrl && !key.meta) {
      setInput((prev) => {
        const next = prev + char;
        setFilter(next);
        return next;
      });
    }
  });

  return (
    <Box>
      <Text color={theme.colors.warning} bold>
        /{' '}
      </Text>
      <Text>{input}</Text>
      <Text color={theme.colors.textMuted}>|</Text>
      <Text dimColor>{'  [Enter] apply  [Esc] clear'}</Text>
    </Box>
  );
}
