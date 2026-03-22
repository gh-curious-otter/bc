import { useEffect, useRef } from "react";

interface KeyboardHelpProps {
  open: boolean;
  onClose: () => void;
}

const SHORTCUTS = [
  { key: "?", description: "Toggle this help overlay" },
  { key: "j", description: "Next item in list" },
  { key: "k", description: "Previous item in list" },
  { key: "/", description: "Focus search input" },
  { key: "1-9", description: "Switch sidebar navigation tabs" },
  { key: "Esc", description: "Close overlay / unfocus" },
  { key: "⌘K", description: "Open command palette" },
] as const;

export function KeyboardHelp({ open, onClose }: KeyboardHelpProps) {
  const overlayRef = useRef<HTMLDivElement>(null);

  // Close on click outside the modal content
  useEffect(() => {
    if (!open) return;

    function handleClick(e: MouseEvent) {
      if (
        overlayRef.current &&
        e.target instanceof Node &&
        !overlayRef.current.contains(e.target)
      ) {
        onClose();
      }
    }

    // Defer to avoid catching the same click that opened it
    const id = requestAnimationFrame(() => {
      document.addEventListener("mousedown", handleClick);
    });

    return () => {
      cancelAnimationFrame(id);
      document.removeEventListener("mousedown", handleClick);
    };
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-[100] flex items-center justify-center bg-black/60"
      role="dialog"
      aria-modal="true"
      aria-label="Keyboard shortcuts"
    >
      <div
        ref={overlayRef}
        className="w-full max-w-md mx-4 rounded-lg border border-bc-border bg-bc-surface shadow-xl"
      >
        <div className="flex items-center justify-between px-5 py-4 border-b border-bc-border">
          <h2 className="text-base font-semibold text-bc-text">
            Keyboard Shortcuts
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="p-1 rounded text-bc-muted hover:text-bc-text transition-colors"
            aria-label="Close shortcuts help"
          >
            <svg
              width="16"
              height="16"
              viewBox="0 0 16 16"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
            >
              <path d="M4 4l8 8M12 4l-8 8" />
            </svg>
          </button>
        </div>
        <div className="px-5 py-4 space-y-3">
          {SHORTCUTS.map(({ key, description }) => (
            <div key={key} className="flex items-center justify-between">
              <span className="text-sm text-bc-muted">{description}</span>
              <kbd className="inline-flex items-center px-2 py-0.5 rounded border border-bc-border bg-bc-bg text-xs font-mono text-bc-text">
                {key}
              </kbd>
            </div>
          ))}
        </div>
        <div className="px-5 py-3 border-t border-bc-border">
          <p className="text-xs text-bc-muted">
            Shortcuts are disabled when typing in input fields.
          </p>
        </div>
      </div>
    </div>
  );
}
