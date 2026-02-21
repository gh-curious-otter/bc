/**
 * useFileTree hook - Directory traversal for file explorer
 *
 * RFC 002: File Explorer for TUI
 * Provides lazy-loaded directory tree with expand/collapse support.
 *
 * Features:
 * - Lazy loading: only load children when directory is expanded
 * - Gitignore-aware: respects .gitignore patterns
 * - Performance: caches directory contents
 */

import { useState, useCallback, useRef, useEffect, useMemo } from 'react';
import * as fs from 'fs';
import * as path from 'path';

/** Directory entry in the file tree */
export interface FileTreeEntry {
  name: string;
  path: string;
  isDirectory: boolean;
  expanded: boolean;
  children: FileTreeEntry[];
  loaded: boolean;
  gitStatus?: 'modified' | 'added' | 'deleted' | 'untracked';
}

/** Patterns to always ignore */
const DEFAULT_IGNORE_PATTERNS = [
  '.git',
  'node_modules',
  '.bc',
  '__pycache__',
  '.pytest_cache',
  '.mypy_cache',
  '.tox',
  '.eggs',
  '*.egg-info',
  '.venv',
  'venv',
  '.env',
  'dist',
  'build',
  '.DS_Store',
  'Thumbs.db',
];

export interface UseFileTreeOptions {
  /** Root directory path */
  rootPath: string;
  /** Additional patterns to ignore */
  ignorePatterns?: string[];
  /** Maximum depth to load initially */
  initialDepth?: number;
}

export interface UseFileTreeResult {
  /** The file tree structure */
  tree: FileTreeEntry[];
  /** Loading state */
  loading: boolean;
  /** Error message if any */
  error: string | null;
  /** Expand a directory (loads children if needed) */
  expandDirectory: (entryPath: string) => void;
  /** Collapse a directory */
  collapseDirectory: (entryPath: string) => void;
  /** Toggle directory expansion */
  toggleDirectory: (entryPath: string) => void;
  /** Refresh the tree */
  refresh: () => void;
}

/**
 * Check if a name matches any ignore pattern
 */
function shouldIgnore(name: string, patterns: string[]): boolean {
  for (const pattern of patterns) {
    // Simple glob matching
    if (pattern.startsWith('*')) {
      const suffix = pattern.slice(1);
      if (name.endsWith(suffix)) return true;
    } else if (pattern.endsWith('*')) {
      const prefix = pattern.slice(0, -1);
      if (name.startsWith(prefix)) return true;
    } else if (name === pattern) {
      return true;
    }
  }
  return false;
}

/**
 * Read directory contents and return sorted entries (directories first)
 */
function readDirectoryContents(
  dirPath: string,
  ignorePatterns: string[]
): FileTreeEntry[] {
  try {
    const entries = fs.readdirSync(dirPath, { withFileTypes: true });
    const result: FileTreeEntry[] = [];

    for (const entry of entries) {
      // Skip hidden files and ignored patterns
      if (entry.name.startsWith('.') && entry.name !== '.gitignore') {
        continue;
      }
      if (shouldIgnore(entry.name, ignorePatterns)) {
        continue;
      }

      result.push({
        name: entry.name,
        path: path.join(dirPath, entry.name),
        isDirectory: entry.isDirectory(),
        expanded: false,
        children: [],
        loaded: false,
      });
    }

    // Sort: directories first, then alphabetically
    result.sort((a, b) => {
      if (a.isDirectory && !b.isDirectory) return -1;
      if (!a.isDirectory && b.isDirectory) return 1;
      return a.name.localeCompare(b.name);
    });

    return result;
  } catch {
    return [];
  }
}

/**
 * Find an entry in the tree by path
 */
function findEntry(
  tree: FileTreeEntry[],
  targetPath: string
): FileTreeEntry | null {
  for (const entry of tree) {
    if (entry.path === targetPath) {
      return entry;
    }
    if (entry.isDirectory && entry.children.length > 0) {
      const found = findEntry(entry.children, targetPath);
      if (found) return found;
    }
  }
  return null;
}

/**
 * Deep clone the tree to trigger React re-renders
 */
function cloneTree(tree: FileTreeEntry[]): FileTreeEntry[] {
  return tree.map((entry) => ({
    ...entry,
    children: cloneTree(entry.children),
  }));
}

/**
 * Hook to manage file tree state and operations
 */
export function useFileTree(options: UseFileTreeOptions): UseFileTreeResult {
  const { rootPath, ignorePatterns = [], initialDepth = 1 } = options;

  // Memoize ignore patterns to prevent dependency changes on every render
  const allIgnorePatterns = useMemo(
    () => [...DEFAULT_IGNORE_PATTERNS, ...ignorePatterns],
    [ignorePatterns]
  );

  const [tree, setTree] = useState<FileTreeEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Cache for loaded directories
  const cacheRef = useRef<Map<string, FileTreeEntry[]>>(new Map());

  /**
   * Load the root directory
   */
  const loadRoot = useCallback(() => {
    setLoading(true);
    setError(null);

    try {
      if (!fs.existsSync(rootPath)) {
        setError(`Path does not exist: ${rootPath}`);
        setTree([]);
        return;
      }

      const stat = fs.statSync(rootPath);
      if (!stat.isDirectory()) {
        setError(`Path is not a directory: ${rootPath}`);
        setTree([]);
        return;
      }

      const rootEntries = readDirectoryContents(rootPath, allIgnorePatterns);

      // Load initial depth
      if (initialDepth > 0) {
        for (const entry of rootEntries) {
          if (entry.isDirectory) {
            entry.children = readDirectoryContents(entry.path, allIgnorePatterns);
            entry.loaded = true;
            cacheRef.current.set(entry.path, entry.children);
          }
        }
      }

      setTree(rootEntries);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to read directory');
      setTree([]);
    } finally {
      setLoading(false);
    }
  }, [rootPath, allIgnorePatterns, initialDepth]);

  // Load on mount and when rootPath changes
  useEffect(() => {
    if (rootPath) {
      loadRoot();
    } else {
      setTree([]);
      setLoading(false);
    }
  }, [rootPath, loadRoot]);

  /**
   * Expand a directory (loads children if needed)
   */
  const expandDirectory = useCallback(
    (entryPath: string) => {
      setTree((prevTree) => {
        const newTree = cloneTree(prevTree);
        const entry = findEntry(newTree, entryPath);

        if (entry?.isDirectory) {
          // Load children if not already loaded
          if (!entry.loaded) {
            const cached = cacheRef.current.get(entryPath);
            if (cached) {
              entry.children = cached;
            } else {
              entry.children = readDirectoryContents(entryPath, allIgnorePatterns);
              cacheRef.current.set(entryPath, entry.children);
            }
            entry.loaded = true;
          }
          entry.expanded = true;
        }

        return newTree;
      });
    },
    [allIgnorePatterns]
  );

  /**
   * Collapse a directory
   */
  const collapseDirectory = useCallback((entryPath: string) => {
    setTree((prevTree) => {
      const newTree = cloneTree(prevTree);
      const entry = findEntry(newTree, entryPath);

      if (entry?.isDirectory) {
        entry.expanded = false;
      }

      return newTree;
    });
  }, []);

  /**
   * Toggle directory expansion
   */
  const toggleDirectory = useCallback(
    (entryPath: string) => {
      setTree((prevTree) => {
        const entry = findEntry(prevTree, entryPath);

        if (entry?.isDirectory) {
          if (entry.expanded) {
            collapseDirectory(entryPath);
          } else {
            expandDirectory(entryPath);
          }
        }

        return prevTree;
      });
    },
    [expandDirectory, collapseDirectory]
  );

  /**
   * Refresh the tree (clear cache and reload)
   */
  const refresh = useCallback(() => {
    cacheRef.current.clear();
    loadRoot();
  }, [loadRoot]);

  return {
    tree,
    loading,
    error,
    expandDirectory,
    collapseDirectory,
    toggleDirectory,
    refresh,
  };
}

export default useFileTree;
