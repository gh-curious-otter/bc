/**
 * MembersPanel - Display channel members with collapsible view
 * Issue #847 - Channel member list + description in TUI
 */

import React, { memo, useState } from 'react';
import { Box, Text, useInput } from 'ink';
import { useTheme } from '../theme';
import { Panel } from './Panel';

export interface MemberInfo {
  name: string;
  role?: string;
  state?: string;
}

export interface MembersPanelProps {
  /** List of members to display */
  members: MemberInfo[] | string[];
  /** Panel title */
  title?: string;
  /** Whether panel is collapsible */
  collapsible?: boolean;
  /** Initial collapsed state */
  defaultCollapsed?: boolean;
  /** Maximum members to show before "and X more" */
  maxVisible?: number;
  /** Whether this panel has focus */
  focused?: boolean;
}

/**
 * Format member for display
 */
function formatMember(member: MemberInfo | string): { name: string; detail?: string } {
  if (typeof member === 'string') {
    return { name: member };
  }
  const details: string[] = [];
  if (member.role) details.push(member.role);
  if (member.state) details.push(member.state);
  return {
    name: member.name,
    detail: details.length > 0 ? details.join(' · ') : undefined,
  };
}

/**
 * MembersPanel component - Displays list of channel members
 */
export const MembersPanel = memo(function MembersPanel({
  members,
  title = 'Members',
  collapsible = true,
  defaultCollapsed = false,
  maxVisible = 10,
  focused = false,
}: MembersPanelProps): React.ReactElement {
  const { theme } = useTheme();
  const [collapsed, setCollapsed] = useState(defaultCollapsed);

  // Handle keyboard input for collapse toggle
  useInput(
    (input) => {
      if (collapsible && (input === ' ' || input === 'c')) {
        setCollapsed((prev) => !prev);
      }
    },
    { isActive: focused }
  );

  const memberCount = members.length;
  const displayTitle = `${title} (${String(memberCount)})`;

  // Collapsed view
  if (collapsed) {
    return (
      <Panel title={displayTitle} focused={focused}>
        <Box>
          <Text dimColor>
            {collapsible ? 'Press space to expand' : `${String(memberCount)} members`}
          </Text>
        </Box>
      </Panel>
    );
  }

  // Calculate visible members
  const visibleMembers = members.slice(0, maxVisible);
  const hiddenCount = memberCount - maxVisible;

  return (
    <Panel title={displayTitle} focused={focused}>
      <Box flexDirection="column">
        {visibleMembers.map((member, idx) => {
          const { name, detail } = formatMember(member);
          return (
            <Box key={`${name}-${String(idx)}`}>
              <Text color={theme.colors.primary}>{name}</Text>
              {detail && <Text dimColor> ({detail})</Text>}
            </Box>
          );
        })}
        {hiddenCount > 0 && <Text dimColor>... and {String(hiddenCount)} more</Text>}
        {collapsible && (
          <Text dimColor italic>
            {'\n'}Press space to collapse
          </Text>
        )}
      </Box>
    </Panel>
  );
});

/**
 * Compact member count badge for channel list
 */
export interface MemberCountBadgeProps {
  count: number;
  color?: string;
}

export const MemberCountBadge = memo(function MemberCountBadge({
  count,
  color = 'gray',
}: MemberCountBadgeProps): React.ReactElement {
  return <Text color={color}>[{String(count)}]</Text>;
});

export default MembersPanel;
