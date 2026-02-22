/**
 * RolesView - View and manage agent roles
 * Issue #859 - Add Roles tab with CRUD operations
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Box, Text, useInput } from 'ink';
import { Panel } from '../components/Panel';
import { LoadingIndicator } from '../components/LoadingIndicator';
import { HeaderBar } from '../components/HeaderBar';
import { useFocus } from '../navigation/FocusContext';
import { useAgents } from '../hooks';
import type { Role } from '../types';
import { getRoles, getRole, deleteRole } from '../services/bc';

interface RolesViewProps {
  disableInput?: boolean;
}

/**
 * RolesView - Display and manage workspace roles
 */
export function RolesView({
  disableInput = false,
}: RolesViewProps): React.ReactElement {
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [showDetails, setShowDetails] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchMode, setSearchMode] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const { setFocus } = useFocus();

  // #968 fix: Fetch agents to compute accurate role counts
  // Backend's agent_count is incorrect when running from worktree
  const agents = useAgents();

  // Compute agent counts by role (consistent with Dashboard approach)
  const agentCountByRole = useMemo(() => {
    const counts: Record<string, number> = {};
    const agentList = agents.data ?? [];
    for (const agent of agentList) {
      counts[agent.role] = (counts[agent.role] || 0) + 1;
    }
    return counts;
  }, [agents.data]);

  // #971 fix: Calculate dynamic name column width based on longest role name
  // Add 3 for selection indicator "▸ " and padding, cap at 25 for readability
  const nameColumnWidth = useMemo(() => {
    if (roles.length === 0) return 15; // Default
    const maxNameLen = Math.max(...roles.map((r) => r.name.length));
    return Math.min(25, Math.max(15, maxNameLen + 3));
  }, [roles]);

  // Manage focus state for nested view navigation
  // When showing details, set focus='view' to prevent global ESC from firing
  useEffect(() => {
    if (showDetails) {
      setFocus('view');
    } else {
      setFocus('main');
    }
  }, [showDetails, setFocus]);

  // Fetch roles
  const fetchRoles = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await getRoles();
      setRoles(response.roles);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch roles');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchRoles();
  }, [fetchRoles]);

  // Filter roles by search
  const filteredRoles = useMemo(() => {
    if (searchQuery.length === 0) return roles;
    const lower = searchQuery.toLowerCase();
    return roles.filter(
      (r) =>
        r.name.toLowerCase().includes(lower) ||
        (r.description?.toLowerCase().includes(lower) ?? false) ||
        r.capabilities.some((c) => c.toLowerCase().includes(lower))
    );
  }, [roles, searchQuery]);

  // Reset index when filtered results change
  useEffect(() => {
    setSelectedIndex(0);
  }, [searchQuery]);

  // Get valid index
  const validIndex = Math.min(selectedIndex, Math.max(0, filteredRoles.length - 1));
  const currentRole = filteredRoles[validIndex] as Role | undefined;

  // Fetch role details
  const fetchRoleDetails = useCallback(async (name: string) => {
    try {
      const role = await getRole(name);
      setSelectedRole(role);
      setShowDetails(true);
    } catch {
      setError('Failed to fetch role details');
    }
  }, []);

  // Handle delete confirmation
  const handleDelete = useCallback(async () => {
    if (!currentRole) return;
    try {
      await deleteRole(currentRole.name);
      setConfirmDelete(false);
      await fetchRoles();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete role');
      setConfirmDelete(false);
    }
  }, [currentRole, fetchRoles]);

  // Keyboard handling
  useInput(
    (input, key) => {
      // Confirm delete mode
      if (confirmDelete) {
        if (input === 'y' || input === 'Y') {
          void handleDelete();
        } else {
          setConfirmDelete(false);
        }
        return;
      }

      // Details view mode
      if (showDetails) {
        if (key.escape || input === 'q') {
          setShowDetails(false);
          setSelectedRole(null);
        }
        return;
      }

      // Search mode
      if (searchMode) {
        if (key.return) {
          setSearchMode(false);
        } else if (key.escape) {
          setSearchQuery('');
          setSearchMode(false);
        } else if (key.backspace || key.delete) {
          setSearchQuery((q) => q.slice(0, -1));
        } else if (input && !key.ctrl && !key.meta && !key.tab) {
          setSearchQuery((q) => q + input);
        }
        return;
      }

      // Navigation mode
      if (input === '/') {
        setSearchMode(true);
      } else if (key.upArrow || input === 'k') {
        if (filteredRoles.length > 0) {
          setSelectedIndex(Math.max(0, validIndex - 1));
        }
      } else if (key.downArrow || input === 'j') {
        if (filteredRoles.length > 0) {
          setSelectedIndex(Math.min(filteredRoles.length - 1, validIndex + 1));
        }
      } else if (input === 'g') {
        setSelectedIndex(0);
      } else if (input === 'G') {
        if (filteredRoles.length > 0) {
          setSelectedIndex(filteredRoles.length - 1);
        }
      } else if (key.return && currentRole) {
        void fetchRoleDetails(currentRole.name);
      } else if (input === 'd' && currentRole) {
        // Only allow delete for non-builtin roles
        if (!isBuiltinRole(currentRole.name)) {
          setConfirmDelete(true);
        }
      } else if (input === 'r') {
        void fetchRoles();
      }
    },
    { isActive: !disableInput }
  );

  // Loading state
  if (loading && roles.length === 0) {
    return <LoadingIndicator message="Loading roles..." />;
  }

  // Error state
  if (error && roles.length === 0) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text color="red">Error: {error}</Text>
        <Text dimColor>Press r to retry, q to go back</Text>
      </Box>
    );
  }

  // Delete confirmation modal
  if (confirmDelete && currentRole) {
    return (
      <Box flexDirection="column" padding={1}>
        <Panel title="Confirm Delete" borderColor="red">
          <Box flexDirection="column">
            <Text color="red">Delete role &quot;{currentRole.name}&quot;?</Text>
            <Text dimColor>This action cannot be undone.</Text>
            <Box marginTop={1}>
              <Text>Press </Text>
              <Text color="red" bold>y</Text>
              <Text> to confirm, any other key to cancel</Text>
            </Box>
          </Box>
        </Panel>
      </Box>
    );
  }

  // Details view
  if (showDetails && selectedRole) {
    return (
      <Box flexDirection="column" padding={1}>
        <RoleDetails
          role={selectedRole}
          agentCount={agentCountByRole[selectedRole.name] ?? 0}
        />
        <Box marginTop={1}>
          <Text dimColor>[Esc/q] back to list</Text>
        </Box>
      </Box>
    );
  }

  // Main list view
  return (
    <Box flexDirection="column" width="100%">
      {/* Header - using shared HeaderBar component (#1419) */}
      <HeaderBar
        title="Roles"
        count={filteredRoles.length}
        loading={loading}
        color="cyan"
      />

      {/* Search bar */}
      <Box
        marginBottom={1}
        paddingX={1}
        borderStyle="single"
        borderColor={searchMode ? 'cyan' : 'gray'}
      >
        {searchMode ? (
          <Box>
            <Text color="cyan">{'/ '}</Text>
            <Text>{searchQuery}</Text>
            <Text color="cyan">▌</Text>
          </Box>
        ) : (
          <Text dimColor>Press / to search, j/k to navigate, Enter for details</Text>
        )}
      </Box>

      {/* Roles table */}
      <Box flexDirection="column" marginBottom={1}>
        {/* Header row - #971 fix: dynamic name column width */}
        <Box paddingX={1}>
          <Box width={nameColumnWidth}>
            <Text bold dimColor>NAME</Text>
          </Box>
          <Box width={30}>
            <Text bold dimColor>CAPABILITIES</Text>
          </Box>
          <Box width={8}>
            <Text bold dimColor>AGENTS</Text>
          </Box>
          <Box flexGrow={1}>
            <Text bold dimColor>DESCRIPTION</Text>
          </Box>
        </Box>

        {/* Role rows */}
        {filteredRoles.length === 0 ? (
          <Box paddingX={1} marginTop={1}>
            <Text dimColor>
              {searchQuery.length > 0
                ? `No roles match "${searchQuery}"`
                : 'No roles defined'}
            </Text>
          </Box>
        ) : (
          filteredRoles.map((role, idx) => (
            <RoleRow
              key={role.name}
              role={role}
              selected={idx === validIndex}
              agentCount={agentCountByRole[role.name] ?? 0}
              nameWidth={nameColumnWidth}
            />
          ))
        )}
      </Box>

      {/* Error display */}
      {error && (
        <Box marginBottom={1} paddingX={1}>
          <Text color="red">Error: {error}</Text>
        </Box>
      )}

      {/* Footer */}
      <Box>
        <Text dimColor>
          {searchMode
            ? 'Type to search, Enter/Esc to exit'
            : 'j/k: navigate | g/G: top/bottom | Enter: details | d: delete | r: refresh | q/ESC: back'}
        </Text>
      </Box>
    </Box>
  );
}

interface RoleRowProps {
  role: Role;
  selected: boolean;
  /** Agent count computed from agents list (fixes #968) */
  agentCount: number;
  /** Dynamic name column width (fixes #971) */
  nameWidth: number;
}

function RoleRow({ role, selected, agentCount, nameWidth }: RoleRowProps): React.ReactElement {
  const capabilitiesStr =
    role.capabilities.length > 0
      ? role.capabilities.slice(0, 3).join(', ') +
        (role.capabilities.length > 3 ? '...' : '')
      : '-';

  // #971 fix: Use dynamic width, truncate with 3 chars reserved for "▸ " indicator
  const truncateLen = nameWidth - 3;

  return (
    <Box paddingX={1}>
      <Box width={nameWidth}>
        <Text color={selected ? 'cyan' : undefined} bold={selected}>
          {selected ? '▸ ' : '  '}
          {truncate(role.name, truncateLen)}
        </Text>
      </Box>
      <Box width={30}>
        <Text dimColor>{truncate(capabilitiesStr, 28)}</Text>
      </Box>
      <Box width={8}>
        <Text>{String(agentCount)}</Text>
      </Box>
      <Box flexGrow={1}>
        <Text dimColor>{truncate(role.description ?? '-', 30)}</Text>
      </Box>
    </Box>
  );
}

interface RoleDetailsProps {
  role: Role;
  /** Agent count computed from agents list (fixes #968) */
  agentCount: number;
}

function RoleDetails({ role, agentCount }: RoleDetailsProps): React.ReactElement {
  return (
    <Panel title={`Role: ${role.name}`} borderColor="cyan">
      <Box flexDirection="column">
        {/* Basic info */}
        <Box marginBottom={1}>
          <Box width={15}>
            <Text dimColor>Description:</Text>
          </Box>
          <Text>{role.description ?? 'No description'}</Text>
        </Box>

        {role.parent && (
          <Box marginBottom={1}>
            <Box width={15}>
              <Text dimColor>Parent:</Text>
            </Box>
            <Text color="cyan">{role.parent}</Text>
          </Box>
        )}

        <Box marginBottom={1}>
          <Box width={15}>
            <Text dimColor>Agents:</Text>
          </Box>
          <Text>{String(agentCount)}</Text>
        </Box>

        {/* Capabilities */}
        <Box flexDirection="column" marginBottom={1}>
          <Text dimColor>Capabilities:</Text>
          <Box flexDirection="column" marginLeft={2}>
            {role.capabilities.length === 0 ? (
              <Text dimColor>None</Text>
            ) : (
              role.capabilities.map((cap) => (
                <Text key={cap} color="green">
                  • {cap}
                </Text>
              ))
            )}
          </Box>
        </Box>

        {/* Prompt preview */}
        {role.prompt && (
          <Box flexDirection="column">
            <Text dimColor>Prompt preview:</Text>
            <Box
              marginLeft={2}
              borderStyle="single"
              borderColor="gray"
              paddingX={1}
            >
              <Text dimColor wrap="wrap">
                {truncate(role.prompt, 200)}
              </Text>
            </Box>
          </Box>
        )}
      </Box>
    </Panel>
  );
}

/**
 * Check if a role is a builtin role that cannot be deleted
 */
function isBuiltinRole(name: string): boolean {
  const builtinRoles = ['root', 'manager', 'engineer', 'tech-lead', 'product-manager'];
  return builtinRoles.includes(name);
}

/**
 * Truncate string to max length
 */
function truncate(str: string, maxLen: number): string {
  if (str.length <= maxLen) return str;
  return str.slice(0, maxLen - 1) + '…';
}

export default RolesView;
