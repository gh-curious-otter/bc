/**
 * FilesView - File explorer for agent worktrees
 *
 * RFC 002: File Explorer for TUI
 * Phase 1 MVP: Worktree selector, directory tree, file preview
 *
 * Features:
 * - WorktreeSelector: Switch between agent worktrees
 * - FileTree: Expandable/collapsible directory navigation
 * - FilePreview: Read-only file content preview
 * - Keyboard navigation: j/k/Enter/Esc
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Box, Text, useInput, useStdout } from 'ink';
import { getWorktrees } from '../services/bc';
import type { Worktree } from '../types';
import { useTheme } from '../theme';

export interface FilesViewProps {
  /** Callback when user presses Esc to go back */
  onBack?: () => void;
}

// Directory entry type for file tree
interface DirectoryEntry {
  name: string;
  path: string;
  isDirectory: boolean;
  children?: DirectoryEntry[];
  expanded?: boolean;
}

// Focus areas within the view
type FocusArea = 'worktree' | 'tree' | 'preview';

export function FilesView({ onBack }: FilesViewProps): React.ReactElement {
  const { stdout } = useStdout();
  const { theme } = useTheme();
  const terminalWidth = stdout.columns;
  const terminalHeight = stdout.rows;

  // Worktree state
  const [worktrees, setWorktrees] = useState<Worktree[]>([]);
  const [selectedWorktree, setSelectedWorktree] = useState<Worktree | null>(null);
  const [worktreeIndex, setWorktreeIndex] = useState(0);
  const [worktreeSelectorOpen, setWorktreeSelectorOpen] = useState(false);

  // File tree state
  const [fileTree, setFileTree] = useState<DirectoryEntry[]>([]);
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [treeIndex, setTreeIndex] = useState(0);

  // View state
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [focusArea, setFocusArea] = useState<FocusArea>('tree');

  // Load worktrees on mount
  useEffect(() => {
    const loadWorktrees = async (): Promise<void> => {
      try {
        setLoading(true);
        setError(null);
        const wts = await getWorktrees();
        // Filter to only OK worktrees
        const activeWorktrees = wts.filter(w => w.status === 'OK');
        setWorktrees(activeWorktrees);
        if (activeWorktrees.length > 0) {
          setSelectedWorktree(activeWorktrees[0]);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load worktrees');
      } finally {
        setLoading(false);
      }
    };

    void loadWorktrees();
  }, []);

  // Flatten visible tree entries for navigation
  const flattenTree = useCallback((entries: DirectoryEntry[], depth = 0): { entry: DirectoryEntry; depth: number }[] => {
    const result: { entry: DirectoryEntry; depth: number }[] = [];
    for (const entry of entries) {
      result.push({ entry, depth });
      if (entry.isDirectory && entry.expanded && entry.children) {
        result.push(...flattenTree(entry.children, depth + 1));
      }
    }
    return result;
  }, []);

  const flatTree = useMemo(() => flattenTree(fileTree), [flattenTree, fileTree]);

  // Handle keyboard input
  useInput((input, key) => {
    // Escape: close selector or go back
    if (key.escape) {
      if (worktreeSelectorOpen) {
        setWorktreeSelectorOpen(false);
      } else if (onBack) {
        onBack();
      }
      return;
    }

    // Tab: cycle focus areas
    if (key.tab) {
      if (!key.shift) {
        setFocusArea(prev => {
          if (prev === 'worktree') return 'tree';
          if (prev === 'tree') return 'preview';
          return 'worktree';
        });
      } else {
        setFocusArea(prev => {
          if (prev === 'worktree') return 'preview';
          if (prev === 'tree') return 'worktree';
          return 'tree';
        });
      }
      return;
    }

    // w: toggle worktree selector
    if (input === 'w') {
      setWorktreeSelectorOpen(prev => !prev);
      return;
    }

    // Handle worktree selector navigation
    if (worktreeSelectorOpen) {
      if (input === 'j' || key.downArrow) {
        setWorktreeIndex(prev => Math.min(prev + 1, worktrees.length - 1));
      } else if (input === 'k' || key.upArrow) {
        setWorktreeIndex(prev => Math.max(prev - 1, 0));
      } else if (key.return) {
        if (worktrees[worktreeIndex]) {
          setSelectedWorktree(worktrees[worktreeIndex]);
          setWorktreeSelectorOpen(false);
        }
      }
      return;
    }

    // Handle tree navigation when focused on tree
    if (focusArea === 'tree' && flatTree.length > 0) {
      if (input === 'j' || key.downArrow) {
        setTreeIndex(prev => Math.min(prev + 1, flatTree.length - 1));
      } else if (input === 'k' || key.upArrow) {
        setTreeIndex(prev => Math.max(prev - 1, 0));
      } else if (input === 'g') {
        setTreeIndex(0);
      } else if (input === 'G') {
        setTreeIndex(flatTree.length - 1);
      } else if (key.return && flatTree[treeIndex]) {
        const item = flatTree[treeIndex];
        if (item.entry.isDirectory) {
          // Toggle directory expansion
          item.entry.expanded = !item.entry.expanded;
          setFileTree([...fileTree]); // Trigger re-render
        } else {
          // Select file for preview
          setSelectedPath(item.entry.path);
          setFocusArea('preview');
        }
      }
    }
  });

  // Calculate layout
  const treeWidth = Math.min(40, Math.floor(terminalWidth * 0.4));
  const previewWidth = terminalWidth - treeWidth - 4;

  if (loading) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text color="cyan">Files</Text>
        <Text dimColor>Loading worktrees...</Text>
      </Box>
    );
  }

  if (error) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text color="cyan">Files</Text>
        <Text color="red">Error: {error}</Text>
      </Box>
    );
  }

  if (worktrees.length === 0) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text color="cyan">Files</Text>
        <Box marginTop={1}>
          <Text dimColor>No active worktrees found.</Text>
        </Box>
        <Box marginTop={1}>
          <Text dimColor>Create agents to explore their worktrees.</Text>
        </Box>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" height={terminalHeight - 4}>
      {/* Header with worktree selector */}
      <Box marginBottom={1}>
        <Text bold color="cyan">Files</Text>
        <Text> </Text>
        <WorktreeSelector
          worktrees={worktrees}
          selected={selectedWorktree}
          selectedIndex={worktreeIndex}
          isOpen={worktreeSelectorOpen}
          onToggle={() => { setWorktreeSelectorOpen(prev => !prev); }}
        />
      </Box>

      {/* Main content: tree + preview */}
      <Box flexDirection="row" flexGrow={1}>
        {/* File tree panel */}
        <Box
          flexDirection="column"
          width={treeWidth}
          borderStyle="single"
          borderColor={focusArea === 'tree' ? theme.colors.accent : undefined}
          paddingX={1}
        >
          <Text bold dimColor>Tree</Text>
          {fileTree.length === 0 ? (
            <Box marginTop={1} flexDirection="column">
              <Text dimColor>Select a worktree to browse files.</Text>
              <Text dimColor>Press w to open worktree selector.</Text>
            </Box>
          ) : (
            <FileTreeDisplay
              flatTree={flatTree}
              selectedIndex={treeIndex}
              maxHeight={terminalHeight - 10}
            />
          )}
        </Box>

        {/* File preview panel */}
        <Box
          flexDirection="column"
          width={previewWidth}
          borderStyle="single"
          borderColor={focusArea === 'preview' ? theme.colors.accent : undefined}
          paddingX={1}
          marginLeft={1}
        >
          <Text bold dimColor>Preview</Text>
          {selectedPath ? (
            <Box marginTop={1}>
              <Text dimColor>{selectedPath}</Text>
            </Box>
          ) : (
            <Box marginTop={1}>
              <Text dimColor>Select a file to preview.</Text>
            </Box>
          )}
        </Box>
      </Box>

      {/* Footer with hints */}
      <Box marginTop={1}>
        <Text dimColor>
          j/k: nav | Enter: expand/select | w: worktree | Tab: focus | Esc: back
        </Text>
      </Box>
    </Box>
  );
}

// WorktreeSelector component
interface WorktreeSelectorProps {
  worktrees: Worktree[];
  selected: Worktree | null;
  selectedIndex: number;
  isOpen: boolean;
  onToggle: () => void;
}

function WorktreeSelector({
  worktrees,
  selected,
  selectedIndex,
  isOpen,
  onToggle: _onToggle,
}: WorktreeSelectorProps): React.ReactElement {
  const { theme } = useTheme();

  if (!isOpen) {
    return (
      <Box>
        <Text>[</Text>
        <Text color={theme.colors.accent}>{selected?.agent ?? 'none'}</Text>
        <Text>]</Text>
        <Text dimColor> (w to switch)</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" borderStyle="single" paddingX={1}>
      <Text bold>Select Worktree:</Text>
      {worktrees.map((wt, index) => (
        <Box key={wt.agent}>
          <Text color={index === selectedIndex ? theme.colors.accent : undefined}>
            {index === selectedIndex ? '>' : ' '} {wt.agent}
          </Text>
          {wt.branch && <Text dimColor> ({wt.branch})</Text>}
        </Box>
      ))}
    </Box>
  );
}

// FileTreeDisplay component
interface FileTreeDisplayProps {
  flatTree: { entry: DirectoryEntry; depth: number }[];
  selectedIndex: number;
  maxHeight: number;
}

function FileTreeDisplay({
  flatTree,
  selectedIndex,
  maxHeight,
}: FileTreeDisplayProps): React.ReactElement {
  const { theme } = useTheme();

  // Calculate visible window
  const visibleCount = Math.max(1, maxHeight - 2);
  const start = Math.max(0, Math.min(selectedIndex - Math.floor(visibleCount / 2), flatTree.length - visibleCount));
  const visibleItems = flatTree.slice(start, start + visibleCount);

  return (
    <Box flexDirection="column" marginTop={1}>
      {visibleItems.map((item, idx) => {
        const globalIdx = start + idx;
        const isSelected = globalIdx === selectedIndex;
        const indent = '  '.repeat(item.depth);
        const icon = item.entry.isDirectory
          ? (item.entry.expanded ? '[-]' : '[+]')
          : '   ';

        return (
          <Text key={item.entry.path}>
            <Text color={isSelected ? theme.colors.accent : undefined} bold={isSelected}>
              {indent}{icon} {item.entry.name}
            </Text>
          </Text>
        );
      })}
    </Box>
  );
}

export default FilesView;
