"use client";

import { useEffect } from "react";
import { reportWebVitals } from "../../lib/vitals";

/**
 * Client component that initialises Core Web Vitals monitoring on mount.
 * Uses the web-vitals library for accurate, production-grade metrics.
 */
export function WebVitals() {
  useEffect(() => {
    reportWebVitals();
  }, []);

  return null;
}
