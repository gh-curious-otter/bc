import React, { useState, useMemo, useCallback } from 'react';
import { Box, Text } from 'ink';
import { Panel } from '../components/Panel.js';
import { DataTable } from '../components/DataTable.js';
import { Footer } from '../components/Footer.js';
import { LoadingIndicator } from '../components/LoadingIndicator.js';
import { ErrorDisplay } from '../components/ErrorDisplay.js';
import { useTeams, useListNavigation } from '../hooks';
import { truncate } from '../utils';
import type { Team } from '../types';

// Extended team type for DataTable compatibility
interface TeamRow extends Record<string, unknown> {
  name: string;
  members: string[];
  lead: string;
  description: string;
}

/**
 * TeamsView - Display and manage teams
 * Issue #556 - Teams view
 */
export function TeamsView(): React.ReactElement {
  const { data: teams, loading, error, refresh } = useTeams();
  const [expandedTeam, setExpandedTeam] = useState<string | null>(null);

  const teamList = teams ?? [];
  const teamCount = teamList.length;

  // Toggle expanded view for selected team
  const handleSelect = useCallback((team: Team) => {
    setExpandedTeam((current) => (current === team.name ? null : team.name));
  }, []);

  // Custom key handlers (#1741)
  const customKeys = useMemo(
    () => ({
      r: () => { void refresh(); },
    }),
    [refresh]
  );

  // #1741: useListNavigation for consolidated keyboard patterns
  // Space key triggers onSelect same as Enter (built into hook)
  const { selectedIndex } = useListNavigation({
    items: teamList,
    onSelect: handleSelect,
    customKeys,
  });

  if (error) {
    return <ErrorDisplay error={error} onRetry={() => { void refresh(); }} />;
  }

  if (loading && !teams) {
    return <LoadingIndicator message="Loading teams..." />;
  }

  // Convert to TeamRow format for DataTable
  const teamRows: TeamRow[] = teamList.map((t) => ({
    name: t.name,
    members: t.members,
    lead: t.lead ?? '',
    description: t.description ?? '',
  }));

  return (
    <Box flexDirection="column" padding={1}>
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color="blue">
          Teams
        </Text>
        <Text dimColor> ({teamCount})</Text>
        {loading && <Text color="yellow"> (refreshing...)</Text>}
      </Box>

      {/* Teams List */}
      <Panel title="Team List">
        {teamCount === 0 ? (
          <Box flexDirection="column">
            <Text dimColor>No teams configured</Text>
            <Text dimColor>Create a team with: bc team create {'<name>'}</Text>
          </Box>
        ) : (
          <DataTable<TeamRow>
            columns={[
              {
                key: 'name',
                header: 'TEAM',
                width: 20,
              },
              {
                key: 'members',
                header: 'MEMBERS',
                width: 10,
                render: (value) => (
                  <Text>{(value as string[]).length}</Text>
                ),
              },
              {
                key: 'lead',
                header: 'LEAD',
                width: 15,
                render: (value) => (
                  <Text color="green">{value as string}</Text>
                ),
              },
              {
                key: 'description',
                header: 'DESCRIPTION',
                render: (value) => (
                  <Text dimColor>{truncate(value as string, 30)}</Text>
                ),
              },
            ]}
            data={teamRows}
            selectedIndex={selectedIndex}
          />
        )}
      </Panel>

      {/* Expanded Team Details */}
      {expandedTeam && (
        <TeamDetails team={teamList.find((t) => t.name === expandedTeam)} />
      )}

      {/* Footer with keyboard hints */}
      <Footer
        hints={[
          { key: 'j/k', label: 'navigate' },
          { key: 'g/G', label: 'top/bottom' },
          { key: 'Enter', label: 'expand' },
          { key: 'r', label: 'refresh' },
          { key: 'q/ESC', label: 'back' },
        ]}
      />
    </Box>
  );
}

interface TeamDetailsProps {
  team?: Team;
}

function TeamDetails({ team }: TeamDetailsProps) {
  if (!team) return null;

  return (
    <Panel title={`Team: ${team.name}`}>
      <Box flexDirection="column">
        {team.description && (
          <Box marginBottom={1}>
            <Text dimColor>Description: </Text>
            <Text>{team.description}</Text>
          </Box>
        )}

        {team.lead && (
          <Box>
            <Text dimColor>Lead: </Text>
            <Text color="green" bold>
              {team.lead}
            </Text>
          </Box>
        )}

        <Box marginTop={1}>
          <Text dimColor>Members ({String(team.members.length)}):</Text>
        </Box>

        {team.members.length > 0 ? (
          <Box flexDirection="column" marginLeft={2}>
            {team.members.map((member) => (
              <Box key={member}>
                <Text color={member === team.lead ? 'green' : 'cyan'}>
                  {member === team.lead ? '★ ' : '• '}
                  {member}
                </Text>
              </Box>
            ))}
          </Box>
        ) : (
          <Box marginLeft={2}>
            <Text dimColor>No members</Text>
          </Box>
        )}

        <Box marginTop={1}>
          <Text dimColor>
            Created: {formatDate(team.created_at)} · Updated:{' '}
            {formatDate(team.updated_at)}
          </Text>
        </Box>
      </Box>
    </Panel>
  );
}

/**
 * Format ISO date string
 */
function formatDate(isoString: string | undefined): string {
  if (!isoString) return '-';
  try {
    const date = new Date(isoString);
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    });
  } catch {
    return '-';
  }
}

export default TeamsView;
