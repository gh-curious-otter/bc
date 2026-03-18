/**
 * SecretsView - Display secret metadata (never values)
 * Issue #1927 - k9s-style resource view for secrets
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Box, Text } from 'ink';
import { LoadingIndicator } from '../components/LoadingIndicator';
import { HeaderBar } from '../components/HeaderBar';
import { Footer } from '../components/Footer';
import { useDisableInput, useListNavigation, useLoadingTimeout } from '../hooks';
import { truncate } from '../utils';
import { getSecretList, type SecretMeta } from '../services/bc';

export function SecretsView(): React.ReactElement {
  const { isDisabled: disableInput } = useDisableInput();
  const [secrets, setSecrets] = useState<SecretMeta[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchSecrets = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getSecretList();
      setSecrets(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch secrets');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchSecrets();
  }, [fetchSecrets]);

  const customKeys = useMemo(
    () => ({
      'r': () => { void fetchSecrets(); },
    }),
    [fetchSecrets]
  );

  const { selectedIndex } = useListNavigation({
    items: secrets,
    disabled: disableInput,
    customKeys,
  });

  const showTimeout = useLoadingTimeout(loading);

  const viewHints = [
    { key: 'r', label: 'refresh', priority: 10 },
    { key: 'j/k', label: 'navigate', priority: 11 },
  ];

  if (loading && showTimeout) {
    return (
      <Box flexDirection="column">
        <HeaderBar title="Secrets" />
        <LoadingIndicator message="Loading secrets..." />
        <Footer hints={viewHints} />
      </Box>
    );
  }

  if (error) {
    return (
      <Box flexDirection="column">
        <HeaderBar title="Secrets" />
        <Box paddingLeft={1}><Text color="red">{error}</Text></Box>
        <Footer hints={viewHints} />
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      <HeaderBar title="Secrets" count={secrets.length} />

      {secrets.length === 0 ? (
        <Box paddingLeft={1} paddingTop={1}>
          <Text dimColor>No secrets configured. Use &apos;bc secret set&apos; to add one.</Text>
        </Box>
      ) : (
        <Box flexDirection="column" paddingTop={1}>
          <Box paddingLeft={1}>
            <Box width={25}><Text bold>NAME</Text></Box>
            <Box width={35}><Text bold>DESCRIPTION</Text></Box>
            <Box width={20}><Text bold>UPDATED</Text></Box>
          </Box>

          {secrets.map((secret, index) => {
            const isSelected = index === selectedIndex;
            const updated = secret.updated_at ? new Date(secret.updated_at).toLocaleDateString() : '';
            return (
              <Box key={secret.name} paddingLeft={1}>
                <Box width={25}>
                  <Text inverse={isSelected} color={isSelected ? 'blue' : undefined}>
                    {truncate(secret.name, 23)}
                  </Text>
                </Box>
                <Box width={35}>
                  <Text inverse={isSelected}>{truncate(secret.description || '', 33)}</Text>
                </Box>
                <Box width={20}>
                  <Text inverse={isSelected} dimColor>{updated}</Text>
                </Box>
              </Box>
            );
          })}
        </Box>
      )}

      <Footer hints={viewHints} />
    </Box>
  );
}
