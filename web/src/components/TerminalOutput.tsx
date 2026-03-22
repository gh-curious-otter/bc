import { useEffect, useRef, useCallback } from "react";

interface TerminalOutputProps {
  lines: string[];
  maxHeight?: string;
  autoScroll?: boolean;
  onScrollChange?: (nearBottom: boolean) => void;
}

/**
 * Renders terminal output lines in a monospace dark panel with auto-scroll.
 *
 * Reusable across AgentDetail, peek panel, and inline terminal views.
 */
export function TerminalOutput({
  lines,
  maxHeight = "32rem",
  autoScroll = true,
  onScrollChange,
}: TerminalOutputProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const isNearBottomRef = useRef(true);

  const checkNearBottom = useCallback(() => {
    const el = containerRef.current;
    if (!el) return true;
    return el.scrollHeight - el.scrollTop - el.clientHeight < 120;
  }, []);

  const handleScroll = useCallback(() => {
    const nearBottom = checkNearBottom();
    isNearBottomRef.current = nearBottom;
    onScrollChange?.(nearBottom);
  }, [checkNearBottom, onScrollChange]);

  // Auto-scroll when lines change, only if near bottom
  useEffect(() => {
    const el = containerRef.current;
    if (!el || !autoScroll) return;
    if (isNearBottomRef.current) {
      el.scrollTop = el.scrollHeight;
    }
  }, [lines, autoScroll]);

  return (
    <div
      ref={containerRef}
      onScroll={handleScroll}
      className="rounded-lg border border-bc-border/50 bg-[#0a0a0f] overflow-y-auto shadow-inner"
      style={{ maxHeight }}
    >
      <pre
        className="p-4 text-xs leading-relaxed whitespace-pre-wrap break-words text-bc-text/90"
        style={{
          fontFamily:
            "'Space Mono', ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
        }}
      >
        {lines.length > 0 ? (
          lines.join("\n")
        ) : (
          <span className="text-bc-muted italic">No output yet.</span>
        )}
      </pre>
    </div>
  );
}
