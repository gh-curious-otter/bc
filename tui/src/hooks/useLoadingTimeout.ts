/**
 * useLoadingTimeout - Track elapsed time during loading states
 * Issue #1898: CostsView and ToolsView stuck loading
 *
 * Returns elapsed seconds while loading is true. Resets to 0 when loading becomes false.
 * Use to show progressive timeout messages (e.g. 5s slow warning, 10s timeout).
 */

import { useState, useEffect } from 'react';

export function useLoadingTimeout(loading: boolean): number {
  const [elapsed, setElapsed] = useState(0);

  useEffect(() => {
    if (!loading) {
      setElapsed(0);
      return;
    }
    const start = Date.now();
    const timer = setInterval(() => {
      setElapsed(Math.floor((Date.now() - start) / 1000));
    }, 1000);
    return () => {
      clearInterval(timer);
    };
  }, [loading]);

  return elapsed;
}
