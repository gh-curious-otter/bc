"use client";

import { useEffect } from "react";

/**
 * Lightweight Web Vitals reporter using native PerformanceObserver API.
 * Tracks Core Web Vitals (LCP, CLS, FCP) without external dependencies.
 */
export function WebVitals() {
  useEffect(() => {
    if (typeof window === "undefined" || typeof PerformanceObserver === "undefined") return;

    const isDev = process.env.NODE_ENV === "development";
    const observers: PerformanceObserver[] = [];

    function reportMetric(name: string, value: number, unit: string = "ms") {
      if (isDev) {
        console.log(`[Web Vitals] ${name}: ${value.toFixed(2)}${unit}`);
      }
    }

    // Largest Contentful Paint (LCP)
    try {
      const lcpObserver = new PerformanceObserver((list) => {
        const entries = list.getEntries();
        const lastEntry = entries[entries.length - 1];
        if (lastEntry) reportMetric("LCP", lastEntry.startTime);
      });
      lcpObserver.observe({ type: "largest-contentful-paint", buffered: true });
      observers.push(lcpObserver);
    } catch {
      // LCP not supported
    }

    // Cumulative Layout Shift (CLS)
    try {
      let clsValue = 0;
      const clsObserver = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          const layoutShift = entry as PerformanceEntry & { hadRecentInput: boolean; value: number };
          if (!layoutShift.hadRecentInput) {
            clsValue += layoutShift.value;
          }
        }
        reportMetric("CLS", clsValue, "");
      });
      clsObserver.observe({ type: "layout-shift", buffered: true });
      observers.push(clsObserver);
    } catch {
      // CLS not supported
    }

    // First Contentful Paint (FCP)
    try {
      const fcpObserver = new PerformanceObserver((list) => {
        const entries = list.getEntries();
        const fcpEntry = entries.find((e) => e.name === "first-contentful-paint");
        if (fcpEntry) reportMetric("FCP", fcpEntry.startTime);
      });
      fcpObserver.observe({ type: "paint", buffered: true });
      observers.push(fcpObserver);
    } catch {
      // FCP not supported
    }

    return () => {
      observers.forEach((o) => o.disconnect());
    };
  }, []);

  return null;
}
