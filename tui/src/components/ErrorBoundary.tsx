import React, { Component, type ErrorInfo, type ReactNode } from 'react';
import { Box, Text } from 'ink';

export interface ErrorBoundaryProps {
  /** The view name for context in error messages */
  viewName?: string;
  /** Children to render */
  children: ReactNode;
  /** Custom fallback component */
  fallback?: ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
}

/**
 * ErrorBoundary - Catches errors in child components
 *
 * Prevents a single component error from crashing the entire TUI.
 * Displays an error message with recovery options.
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
    };
  }

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    return { hasError: true, error };
  }

  override componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    this.setState({ errorInfo });
    // Log error for debugging
    // eslint-disable-next-line no-console
    console.error(`[ErrorBoundary] Error in ${this.props.viewName ?? 'component'}:`, error);
    // eslint-disable-next-line no-console
    console.error('[ErrorBoundary] Component stack:', errorInfo.componentStack);
  }

  handleReset = (): void => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
    });
  };

  override render(): ReactNode {
    const { hasError, error } = this.state;
    const { children, viewName, fallback } = this.props;

    if (hasError) {
      if (fallback) {
        return fallback;
      }

      return (
        <Box
          flexDirection="column"
          borderStyle="single"
          borderColor="red"
          padding={1}
        >
          <Text color="red" bold>
            Error{viewName ? ` in ${viewName}` : ''}
          </Text>
          <Box marginTop={1}>
            <Text color="red">
              {error?.message ?? 'An unexpected error occurred'}
            </Text>
          </Box>
          <Box marginTop={1}>
            <Text dimColor>
              The rest of the application should continue to work.
            </Text>
          </Box>
          <Box marginTop={1}>
            <Text dimColor>
              Use navigation keys to switch to another view.
            </Text>
          </Box>
        </Box>
      );
    }

    return children;
  }
}

/**
 * ViewErrorBoundary - Error boundary specifically for views
 *
 * Wraps a view component and catches any rendering errors.
 * Shows a friendly error message without crashing the app.
 */
export function ViewErrorBoundary({
  viewName,
  children,
}: {
  viewName: string;
  children: ReactNode;
}): React.ReactElement {
  return (
    <ErrorBoundary viewName={viewName}>
      {children}
    </ErrorBoundary>
  );
}

export default ErrorBoundary;
