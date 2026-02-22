/**
 * Tests for useDebounce hook
 * Issue #1602: Add debounce to expensive input operations
 */

import { describe, it, expect, afterEach } from 'bun:test';
import React from 'react';
import { render, cleanup } from 'ink-testing-library';
import { Text, Box } from 'ink';
import {
  useDebounce,
  useDebouncedCallback,
  useDebouncedSearch,
  DEFAULT_DEBOUNCE_MS,
} from '../useDebounce';

describe('useDebounce', () => {
  afterEach(() => {
    cleanup();
  });

  it('exports DEFAULT_DEBOUNCE_MS constant', () => {
    expect(DEFAULT_DEBOUNCE_MS).toBe(300);
  });

  it('returns initial value immediately', () => {
    const TestComponent = (): React.ReactElement => {
      const debouncedValue = useDebounce('initial', 100);
      return <Text>value:{debouncedValue}</Text>;
    };

    const { lastFrame } = render(<TestComponent />);
    expect(lastFrame()).toContain('value:initial');
  });

  it('accepts different types', () => {
    const TestComponentNumber = (): React.ReactElement => {
      const debouncedValue = useDebounce(42, 100);
      return <Text>value:{debouncedValue}</Text>;
    };

    const { lastFrame: frame1 } = render(<TestComponentNumber />);
    expect(frame1()).toContain('value:42');

    cleanup();

    const TestComponentObject = (): React.ReactElement => {
      const debouncedValue = useDebounce({ key: 'value' }, 100);
      return <Text>value:{debouncedValue.key}</Text>;
    };

    const { lastFrame: frame2 } = render(<TestComponentObject />);
    expect(frame2()).toContain('value:value');
  });

  it('uses default delay when not specified', () => {
    const TestComponent = (): React.ReactElement => {
      const debouncedValue = useDebounce('test');
      return <Text>value:{debouncedValue}</Text>;
    };

    const { lastFrame } = render(<TestComponent />);
    expect(lastFrame()).toContain('value:test');
  });
});

describe('useDebouncedCallback', () => {
  afterEach(() => {
    cleanup();
  });

  it('returns callback, cancel, flush, and isPending', () => {
    const TestComponent = (): React.ReactElement => {
      const result = useDebouncedCallback(() => {}, { delay: 100 });
      return (
        <Box flexDirection="column">
          <Text>hasCallback:{typeof result.callback === 'function' ? 'yes' : 'no'}</Text>
          <Text>hasCancel:{typeof result.cancel === 'function' ? 'yes' : 'no'}</Text>
          <Text>hasFlush:{typeof result.flush === 'function' ? 'yes' : 'no'}</Text>
          <Text>hasPending:{typeof result.isPending === 'boolean' ? 'yes' : 'no'}</Text>
        </Box>
      );
    };

    const { lastFrame } = render(<TestComponent />);
    expect(lastFrame()).toContain('hasCallback:yes');
    expect(lastFrame()).toContain('hasCancel:yes');
    expect(lastFrame()).toContain('hasFlush:yes');
    expect(lastFrame()).toContain('hasPending:yes');
  });

  it('accepts options with maxWait', () => {
    const TestComponent = (): React.ReactElement => {
      const { callback } = useDebouncedCallback(
        () => {},
        { delay: 100, maxWait: 500, leading: false, trailing: true }
      );
      return <Text>callback:{typeof callback}</Text>;
    };

    const { lastFrame } = render(<TestComponent />);
    expect(lastFrame()).toContain('callback:function');
  });

  it('isPending is false initially', () => {
    const TestComponent = (): React.ReactElement => {
      const { isPending } = useDebouncedCallback(() => {}, { delay: 100 });
      return <Text>isPending:{isPending ? 'yes' : 'no'}</Text>;
    };

    const { lastFrame } = render(<TestComponent />);
    expect(lastFrame()).toContain('isPending:no');
  });

  it('uses default options when not specified', () => {
    const TestComponent = (): React.ReactElement => {
      const result = useDebouncedCallback(() => {});
      return <Text>hasCallback:{typeof result.callback === 'function' ? 'yes' : 'no'}</Text>;
    };

    const { lastFrame } = render(<TestComponent />);
    expect(lastFrame()).toContain('hasCallback:yes');
  });
});

describe('useDebouncedSearch', () => {
  afterEach(() => {
    cleanup();
  });

  it('returns search state and controls', () => {
    const TestComponent = (): React.ReactElement => {
      const result = useDebouncedSearch();
      return (
        <Box flexDirection="column">
          <Text>hasQuery:{typeof result.query === 'string' ? 'yes' : 'no'}</Text>
          <Text>hasDebouncedQuery:{typeof result.debouncedQuery === 'string' ? 'yes' : 'no'}</Text>
          <Text>hasSetQuery:{typeof result.setQuery === 'function' ? 'yes' : 'no'}</Text>
          <Text>hasClear:{typeof result.clear === 'function' ? 'yes' : 'no'}</Text>
          <Text>hasIsDebouncing:{typeof result.isDebouncing === 'boolean' ? 'yes' : 'no'}</Text>
        </Box>
      );
    };

    const { lastFrame } = render(<TestComponent />);
    expect(lastFrame()).toContain('hasQuery:yes');
    expect(lastFrame()).toContain('hasDebouncedQuery:yes');
    expect(lastFrame()).toContain('hasSetQuery:yes');
    expect(lastFrame()).toContain('hasClear:yes');
    expect(lastFrame()).toContain('hasIsDebouncing:yes');
  });

  it('uses initial query', () => {
    const TestComponent = (): React.ReactElement => {
      const { query, debouncedQuery } = useDebouncedSearch({ initialQuery: 'test' });
      return (
        <Box flexDirection="column">
          <Text>query:{query}</Text>
          <Text>debounced:{debouncedQuery}</Text>
        </Box>
      );
    };

    const { lastFrame } = render(<TestComponent />);
    expect(lastFrame()).toContain('query:test');
    expect(lastFrame()).toContain('debounced:test');
  });

  it('isDebouncing is false initially when query matches debouncedQuery', () => {
    const TestComponent = (): React.ReactElement => {
      const { isDebouncing } = useDebouncedSearch({ initialQuery: 'test' });
      return <Text>isDebouncing:{isDebouncing ? 'yes' : 'no'}</Text>;
    };

    const { lastFrame } = render(<TestComponent />);
    expect(lastFrame()).toContain('isDebouncing:no');
  });

  it('accepts minLength option', () => {
    const TestComponent = (): React.ReactElement => {
      const { query } = useDebouncedSearch({ minLength: 3 });
      return <Text>query:{query}</Text>;
    };

    const { lastFrame } = render(<TestComponent />);
    expect(lastFrame()).toContain('query:');
  });

  it('accepts onSearch callback option', () => {
    let onSearchProvided = false;
    const TestComponent = (): React.ReactElement => {
      useDebouncedSearch({
        onSearch: () => {
          onSearchProvided = true;
        },
      });
      return <Text>rendered</Text>;
    };

    render(<TestComponent />);
    expect(onSearchProvided).toBe(false); // Not called yet since no query change
  });

  it('accepts delay option', () => {
    const TestComponent = (): React.ReactElement => {
      const { query } = useDebouncedSearch({ delay: 500 });
      return <Text>query:{query}</Text>;
    };

    const { lastFrame } = render(<TestComponent />);
    expect(lastFrame()).toContain('query:');
  });

  it('empty initial query by default', () => {
    const TestComponent = (): React.ReactElement => {
      const { query, debouncedQuery } = useDebouncedSearch();
      return (
        <Box flexDirection="column">
          <Text>query:{query || 'empty'}</Text>
          <Text>debounced:{debouncedQuery || 'empty'}</Text>
        </Box>
      );
    };

    const { lastFrame } = render(<TestComponent />);
    expect(lastFrame()).toContain('query:empty');
    expect(lastFrame()).toContain('debounced:empty');
  });
});
