/**
 * ErrorBoundary - Catches React errors and displays fallback UI
 *
 * Issue #1585: Prevent TUI crashes from component errors
 *
 * React error boundaries must be class components to use
 * componentDidCatch and getDerivedStateFromError lifecycle methods.
 */

import React, { Component, type ReactNode } from 'react';
import { Box, Text } from 'ink';

export interface ErrorBoundaryProps {
  /** Content to render */
  children: ReactNode;
  /** Name of the view/component being wrapped (for error messages) */
  viewName?: string;
  /** Custom fallback UI (optional) */
  fallback?: ReactNode;
  /** Callback when an error is caught */
  onError?: (error: Error, errorInfo: React.ErrorInfo) => void;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

/**
 * ViewErrorBoundary wraps views to catch and display errors gracefully.
 * Instead of crashing the entire TUI, shows an error message for the
 * affected view while keeping the rest of the app functional.
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  override componentDidCatch(error: Error, errorInfo: React.ErrorInfo): void {
    // Log error for debugging
    // eslint-disable-next-line no-console
    console.error(`[ErrorBoundary] Error in ${this.props.viewName ?? 'component'}:`, error);
    // eslint-disable-next-line no-console
    console.error('[ErrorBoundary] Component stack:', errorInfo.componentStack);

    // Call optional error handler
    this.props.onError?.(error, errorInfo);
  }

  override render(): ReactNode {
    if (this.state.hasError) {
      // Custom fallback if provided
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // Default error UI
      return (
        <Box flexDirection="column" padding={1} borderStyle="single" borderColor="red">
          <Text color="red" bold>
            Error in {this.props.viewName ?? 'view'}
          </Text>
          <Box marginTop={1}>
            <Text color="yellow">
              {this.state.error?.message ?? 'An unexpected error occurred'}
            </Text>
          </Box>
          <Box marginTop={1}>
            <Text dimColor>Press Tab to switch to another view, or q to quit.</Text>
          </Box>
        </Box>
      );
    }

    return this.props.children;
  }
}

/**
 * ViewErrorBoundary - Convenience wrapper for views with view name
 */
export interface ViewErrorBoundaryProps {
  children: ReactNode;
  viewName: string;
  onError?: (error: Error, errorInfo: React.ErrorInfo) => void;
}

export function ViewErrorBoundary({
  children,
  viewName,
  onError,
}: ViewErrorBoundaryProps): React.ReactElement {
  return (
    <ErrorBoundary viewName={viewName} onError={onError}>
      {children}
    </ErrorBoundary>
  );
}
