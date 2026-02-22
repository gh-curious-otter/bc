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

import React, { useReducer, useEffect, useCallback, useMemo } from 'react';
import { Box, Text, useInput } from 'ink';
import { getWorktrees } from '../services/bc';
import type { Worktree } from '../types';
import { useTheme } from '../theme';
import { useFileTree, useGitStatus, useResponsiveLayout, type FileTreeEntry, type GitFileStatus } from '../hooks';
import * as fs from 'fs';

// Focus areas within the view
type FocusArea = 'worktree' | 'tree' | 'preview';

// State shape for useReducer (#1601)
interface FilesViewState {
  // Worktree state
  worktrees: Worktree[];
  selectedWorktree: Worktree | null;
  worktreeIndex: number;
  worktreeSelectorOpen: boolean;
  worktreesLoading: boolean;
  worktreesError: string | null;
  // Navigation state
  selectedPath: string | null;
  treeIndex: number;
  focusArea: FocusArea;
}

// Action types for reducer
type FilesViewAction =
  | { type: 'SET_WORKTREES'; worktrees: Worktree[] }
  | { type: 'SET_WORKTREES_LOADING'; loading: boolean }
  | { type: 'SET_WORKTREES_ERROR'; error: string | null }
  | { type: 'SELECT_WORKTREE'; worktree: Worktree }
  | { type: 'SET_WORKTREE_INDEX'; index: number }
  | { type: 'TOGGLE_WORKTREE_SELECTOR' }
  | { type: 'CLOSE_WORKTREE_SELECTOR' }
  | { type: 'SET_TREE_INDEX'; index: number }
  | { type: 'SET_SELECTED_PATH'; path: string | null }
  | { type: 'CYCLE_FOCUS_FORWARD' }
  | { type: 'CYCLE_FOCUS_BACKWARD' }
  | { type: 'SET_FOCUS_AREA'; area: FocusArea }
  | { type: 'RESET_NAVIGATION' };

// Initial state
const initialState: FilesViewState = {
  worktrees: [],
  selectedWorktree: null,
  worktreeIndex: 0,
  worktreeSelectorOpen: false,
  worktreesLoading: true,
  worktreesError: null,
  selectedPath: null,
  treeIndex: 0,
  focusArea: 'tree',
};

// Reducer function (#1601: Consolidate state management)
function filesViewReducer(state: FilesViewState, action: FilesViewAction): FilesViewState {
  switch (action.type) {
    case 'SET_WORKTREES':
      return {
        ...state,
        worktrees: action.worktrees,
        selectedWorktree: action.worktrees[0] ?? null,
        worktreesLoading: false,
      };
    case 'SET_WORKTREES_LOADING':
      return { ...state, worktreesLoading: action.loading };
    case 'SET_WORKTREES_ERROR':
      return { ...state, worktreesError: action.error, worktreesLoading: false };
    case 'SELECT_WORKTREE':
      return {
        ...state,
        selectedWorktree: action.worktree,
        worktreeSelectorOpen: false,
        treeIndex: 0,
        selectedPath: null,
      };
    case 'SET_WORKTREE_INDEX':
      return { ...state, worktreeIndex: action.index };
    case 'TOGGLE_WORKTREE_SELECTOR':
      return { ...state, worktreeSelectorOpen: !state.worktreeSelectorOpen };
    case 'CLOSE_WORKTREE_SELECTOR':
      return { ...state, worktreeSelectorOpen: false };
    case 'SET_TREE_INDEX':
      return { ...state, treeIndex: action.index };
    case 'SET_SELECTED_PATH':
      return { ...state, selectedPath: action.path };
    case 'CYCLE_FOCUS_FORWARD':
      return {
        ...state,
        focusArea: state.focusArea === 'worktree' ? 'tree'
          : state.focusArea === 'tree' ? 'preview'
          : 'worktree',
      };
    case 'CYCLE_FOCUS_BACKWARD':
      return {
        ...state,
        focusArea: state.focusArea === 'worktree' ? 'preview'
          : state.focusArea === 'tree' ? 'worktree'
          : 'tree',
      };
    case 'SET_FOCUS_AREA':
      return { ...state, focusArea: action.area };
    case 'RESET_NAVIGATION':
      return { ...state, treeIndex: 0, selectedPath: null };
    default:
      return state;
  }
}

export function FilesView(): React.ReactNode {
  const { theme } = useTheme();
  const { width: terminalWidth, height: terminalHeight, responsive } = useResponsiveLayout();

  // Consolidated state management with useReducer (#1601)
  const [state, dispatch] = useReducer(filesViewReducer, initialState);
  const {
    worktrees,
    selectedWorktree,
    worktreeIndex,
    worktreeSelectorOpen,
    worktreesLoading,
    worktreesError,
    selectedPath,
    treeIndex,
    focusArea,
  } = state;

  // File tree state - use the hook
  const {
    tree: fileTree,
    loading: treeLoading,
    error: treeError,
    expandDirectory,
    collapseDirectory,
  } = useFileTree({
    rootPath: selectedWorktree?.path ?? '',
  });

  // Git status for the selected worktree
  const {
    getStatus: getGitStatus,
    summary: gitSummary,
    loading: gitLoading,
  } = useGitStatus({
    workingDir: selectedWorktree?.path ?? '',
  });

  // Load worktrees on mount
  useEffect(() => {
    const loadWorktrees = async (): Promise<void> => {
      try {
        dispatch({ type: 'SET_WORKTREES_LOADING', loading: true });
        dispatch({ type: 'SET_WORKTREES_ERROR', error: null });
        const wts = await getWorktrees();
        // Filter to only OK worktrees
        const activeWorktrees = wts.filter(w => w.status === 'OK');
        dispatch({ type: 'SET_WORKTREES', worktrees: activeWorktrees });
      } catch (err) {
        dispatch({
          type: 'SET_WORKTREES_ERROR',
          error: err instanceof Error ? err.message : 'Failed to load worktrees',
        });
      }
    };

    void loadWorktrees();
  }, []);

  // Reset tree index when worktree changes
  useEffect(() => {
    dispatch({ type: 'RESET_NAVIGATION' });
  }, [selectedWorktree]);

  // Flatten visible tree entries for navigation
  const flattenTree = useCallback((entries: FileTreeEntry[], depth = 0): { entry: FileTreeEntry; depth: number }[] => {
    const result: { entry: FileTreeEntry; depth: number }[] = [];
    for (const entry of entries) {
      result.push({ entry, depth });
      if (entry.isDirectory && entry.expanded && entry.children.length > 0) {
        result.push(...flattenTree(entry.children, depth + 1));
      }
    }
    return result;
  }, []);

  const flatTree = useMemo(() => flattenTree(fileTree), [flattenTree, fileTree]);

  // Handle keyboard input
  useInput((input, key) => {
    // Escape: close selector
    if (key.escape) {
      if (worktreeSelectorOpen) {
        dispatch({ type: 'CLOSE_WORKTREE_SELECTOR' });
      }
      return;
    }

    // f/F: cycle focus areas (Tab reserved for global view navigation #1520)
    if (input === 'f') {
      dispatch({ type: 'CYCLE_FOCUS_FORWARD' });
      return;
    }
    if (input === 'F') {
      dispatch({ type: 'CYCLE_FOCUS_BACKWARD' });
      return;
    }

    // w: toggle worktree selector
    if (input === 'w') {
      dispatch({ type: 'TOGGLE_WORKTREE_SELECTOR' });
      return;
    }

    // Handle worktree selector navigation
    if (worktreeSelectorOpen) {
      if (input === 'j' || key.downArrow) {
        dispatch({ type: 'SET_WORKTREE_INDEX', index: Math.min(worktreeIndex + 1, worktrees.length - 1) });
      } else if (input === 'k' || key.upArrow) {
        dispatch({ type: 'SET_WORKTREE_INDEX', index: Math.max(worktreeIndex - 1, 0) });
      } else if (key.return) {
        if (worktrees[worktreeIndex]) {
          dispatch({ type: 'SELECT_WORKTREE', worktree: worktrees[worktreeIndex] });
        }
      }
      return;
    }

    // Handle tree navigation when focused on tree
    if (focusArea === 'tree' && flatTree.length > 0) {
      if (input === 'j' || key.downArrow) {
        dispatch({ type: 'SET_TREE_INDEX', index: Math.min(treeIndex + 1, flatTree.length - 1) });
      } else if (input === 'k' || key.upArrow) {
        dispatch({ type: 'SET_TREE_INDEX', index: Math.max(treeIndex - 1, 0) });
      } else if (input === 'g') {
        dispatch({ type: 'SET_TREE_INDEX', index: 0 });
      } else if (input === 'G') {
        dispatch({ type: 'SET_TREE_INDEX', index: flatTree.length - 1 });
      } else if (key.return && flatTree[treeIndex]) {
        const item = flatTree[treeIndex];
        if (item.entry.isDirectory) {
          // Toggle directory expansion
          if (item.entry.expanded) {
            collapseDirectory(item.entry.path);
          } else {
            expandDirectory(item.entry.path);
          }
        } else {
          // Select file for preview
          dispatch({ type: 'SET_SELECTED_PATH', path: item.entry.path });
          dispatch({ type: 'SET_FOCUS_AREA', area: 'preview' });
        }
      }
    }
  });

  // Calculate layout (#1448: Responsive tree width per UX spec)
  // XS: 16 cols, SM: 20 cols (per spec), MD: 25, LG: 30, XL: 35
  const treeWidth = responsive({
    xs: 16,
    sm: 20,
    md: 25,
    lg: 30,
    xl: 35,
    default: 20,
  });
  const previewWidth = terminalWidth - treeWidth - 4;

  // Get current path for breadcrumb (selected tree item path relative to worktree)
  const currentTreePath = useMemo(() => {
    if (!selectedWorktree || flatTree.length === 0 || treeIndex >= flatTree.length) {
      return null;
    }
    const entry = flatTree[treeIndex].entry;
    // Get path relative to worktree root
    const rootPath = selectedWorktree.path;
    if (entry.path.startsWith(rootPath)) {
      return entry.path.slice(rootPath.length + 1); // +1 for trailing slash
    }
    return entry.name;
  }, [flatTree, treeIndex, selectedWorktree]);

  // Loading states
  if (worktreesLoading) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text color="cyan">Files</Text>
        <Text dimColor>Loading worktrees...</Text>
      </Box>
    );
  }

  if (worktreesError) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text color="cyan">Files</Text>
        <Text color="red">Error: {worktreesError}</Text>
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
          onToggle={() => { dispatch({ type: 'TOGGLE_WORKTREE_SELECTOR' }); }}
        />
        {/* Git status summary */}
        {!gitLoading && gitSummary.total > 0 && (
          <Box marginLeft={2}>
            <Text dimColor>[</Text>
            {gitSummary.modified > 0 && <Text color="yellow">~{gitSummary.modified}</Text>}
            {gitSummary.added > 0 && <Text color="green"> +{gitSummary.added}</Text>}
            {gitSummary.deleted > 0 && <Text color="red"> -{gitSummary.deleted}</Text>}
            {gitSummary.untracked > 0 && <Text dimColor> ?{gitSummary.untracked}</Text>}
            <Text dimColor>]</Text>
          </Box>
        )}
      </Box>

      {/* Path breadcrumb */}
      {currentTreePath && (
        <PathBreadcrumb path={currentTreePath} maxWidth={terminalWidth - 4} />
      )}

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
          {treeLoading ? (
            <Box marginTop={1}>
              <Text dimColor>Loading files...</Text>
            </Box>
          ) : treeError ? (
            <Box marginTop={1}>
              <Text color="red">{treeError}</Text>
            </Box>
          ) : fileTree.length === 0 ? (
            <Box marginTop={1} flexDirection="column">
              <Text dimColor>No files in worktree.</Text>
            </Box>
          ) : (
            <FileTreeDisplay
              flatTree={flatTree}
              selectedIndex={treeIndex}
              maxHeight={terminalHeight - 10}
              getGitStatus={getGitStatus}
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
            <FilePreview path={selectedPath} maxHeight={terminalHeight - 10} />
          ) : (
            <Box marginTop={1}>
              <Text dimColor>Select a file to preview.</Text>
            </Box>
          )}
        </Box>
      </Box>

      {/* Footer with hints (#1448: Responsive hints per breakpoint) */}
      <Box marginTop={1}>
        <Text dimColor>
          {responsive({
            xs: 'j/k nav · Enter sel · w tree · Esc',
            sm: 'j/k nav | Enter expand | w worktree | Esc back',
            default: 'j/k: nav | Enter: expand/select | w: worktree | Tab: focus | Esc: back',
          })}
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
            {index === selectedIndex ? '▸' : ' '} {wt.agent}
          </Text>
          {wt.branch && <Text dimColor> ({wt.branch})</Text>}
        </Box>
      ))}
    </Box>
  );
}

// PathBreadcrumb component - shows current path as clickable segments
interface PathBreadcrumbProps {
  path: string;
  maxWidth: number;
}

function PathBreadcrumb({ path, maxWidth }: PathBreadcrumbProps): React.ReactElement {
  const { theme } = useTheme();
  const segments = path.split('/').filter(Boolean);

  // Truncate if path is too long
  let displaySegments = segments;
  let truncated = false;
  const separator = ' › ';
  const fullDisplay = segments.join(separator);

  if (fullDisplay.length > maxWidth - 4) {
    // Show first and last segments with ellipsis
    truncated = true;
    if (segments.length > 2) {
      displaySegments = [segments[0], '...', segments[segments.length - 1]];
    }
  }

  return (
    <Box marginBottom={1}>
      <Text dimColor>📁 </Text>
      {displaySegments.map((segment, idx) => (
        <React.Fragment key={idx}>
          {idx > 0 && <Text dimColor>{separator}</Text>}
          <Text color={idx === displaySegments.length - 1 ? theme.colors.accent : undefined}>
            {segment}
          </Text>
        </React.Fragment>
      ))}
      {truncated && segments.length <= 2 && <Text dimColor>...</Text>}
    </Box>
  );
}

// Git status indicator helper
function getGitStatusIndicator(status: GitFileStatus | undefined): { icon: string; color: string } {
  switch (status) {
    case 'modified':
      return { icon: '✱', color: 'yellow' };
    case 'added':
      return { icon: '+', color: 'green' };
    case 'deleted':
      return { icon: '−', color: 'red' };
    case 'renamed':
      return { icon: '→', color: 'blue' };
    case 'untracked':
      return { icon: '?', color: 'gray' };
    case 'ignored':
      return { icon: '!', color: 'gray' };
    default:
      return { icon: ' ', color: '' };
  }
}

// FileTreeDisplay component
interface FileTreeDisplayProps {
  flatTree: { entry: FileTreeEntry; depth: number }[];
  selectedIndex: number;
  maxHeight: number;
  getGitStatus?: (filePath: string) => { status: GitFileStatus; staged: boolean } | null;
}

function FileTreeDisplay({
  flatTree,
  selectedIndex,
  maxHeight,
  getGitStatus,
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

        // Get git status for the file
        const gitStatus = getGitStatus?.(item.entry.path);
        const statusIndicator = getGitStatusIndicator(gitStatus?.status);

        return (
          <Text key={item.entry.path}>
            <Text color={isSelected ? theme.colors.accent : undefined} bold={isSelected}>
              {indent}{icon} {item.entry.name}
            </Text>
            {statusIndicator.icon !== ' ' && (
              <Text color={statusIndicator.color}> {statusIndicator.icon}</Text>
            )}
            {gitStatus?.staged && <Text color="green">*</Text>}
          </Text>
        );
      })}
    </Box>
  );
}

// FilePreview component
interface FilePreviewProps {
  path: string;
  maxHeight: number;
}

// FilePreview state and reducer (#1601)
interface FilePreviewState {
  content: string | null;
  error: string | null;
  loading: boolean;
}

type FilePreviewAction =
  | { type: 'LOADING' }
  | { type: 'SUCCESS'; content: string }
  | { type: 'ERROR'; error: string };

function filePreviewReducer(state: FilePreviewState, action: FilePreviewAction): FilePreviewState {
  switch (action.type) {
    case 'LOADING':
      return { content: null, error: null, loading: true };
    case 'SUCCESS':
      return { content: action.content, error: null, loading: false };
    case 'ERROR':
      return { content: null, error: action.error, loading: false };
    default:
      return state;
  }
}

function FilePreview({ path, maxHeight }: FilePreviewProps): React.ReactElement {
  const [state, dispatch] = useReducer(filePreviewReducer, {
    content: null,
    error: null,
    loading: true,
  });
  const { content, error, loading } = state;

  useEffect(() => {
    dispatch({ type: 'LOADING' });

    try {
      const stats = fs.statSync(path);

      // Don't preview files larger than 100KB
      if (stats.size > 100 * 1024) {
        dispatch({ type: 'ERROR', error: 'File too large to preview (>100KB)' });
        return;
      }

      // Read file content
      const fileContent = fs.readFileSync(path, 'utf-8');
      dispatch({ type: 'SUCCESS', content: fileContent });
    } catch (err) {
      dispatch({ type: 'ERROR', error: err instanceof Error ? err.message : 'Failed to read file' });
    }
  }, [path]);

  if (loading) {
    return (
      <Box marginTop={1}>
        <Text dimColor>Loading...</Text>
      </Box>
    );
  }

  if (error) {
    return (
      <Box marginTop={1}>
        <Text color="red">{error}</Text>
      </Box>
    );
  }

  if (!content) {
    return (
      <Box marginTop={1}>
        <Text dimColor>Empty file</Text>
      </Box>
    );
  }

  // Split into lines and limit display
  const lines = content.split('\n');
  const visibleLines = lines.slice(0, maxHeight - 2);

  return (
    <Box flexDirection="column" marginTop={1}>
      <Text dimColor>{path}</Text>
      <Text dimColor>{'─'.repeat(30)}</Text>
      {visibleLines.map((line, idx) => (
        <Text key={idx} wrap="truncate">
          {line}
        </Text>
      ))}
      {lines.length > visibleLines.length && (
        <Text dimColor>... ({lines.length - visibleLines.length} more lines)</Text>
      )}
    </Box>
  );
}

export default FilesView;
