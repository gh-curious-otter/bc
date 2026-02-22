/**
 * Utility exports
 */

export {
  handleApiError,
  withErrorHandling,
  withFallback,
  logError,
  isRecoverable,
  getErrorMessage,
  ok,
  err,
  type ApiErrorResult,
  type Result,
} from './errorHandling';

export {
  formatRelativeTime,
  formatDuration,
  truncate,
  formatNumber,
  formatBytes,
  formatCost,
  capitalize,
  toTitleCase,
} from './formatting';
