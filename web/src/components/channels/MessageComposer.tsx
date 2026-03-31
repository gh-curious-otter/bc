import { useState, useCallback, useRef } from "react";

export function MessageComposer({
  channelName,
  onSend,
  disabled = false,
}: {
  channelName: string;
  onSend: (content: string) => Promise<void>;
  disabled?: boolean;
}) {
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const autoGrow = useCallback(() => {
    const ta = textareaRef.current;
    if (!ta) return;
    ta.style.height = "auto";
    ta.style.height = Math.min(ta.scrollHeight, 120) + "px";
  }, []);

  const handleSend = async () => {
    const content = input.trim();
    if (!content || sending || disabled) return;
    setSending(true);
    setInput("");
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
    }
    try {
      await onSend(content);
    } catch {
      setInput(content);
    } finally {
      setSending(false);
    }
  };

  return (
    <div className="p-3 border-t border-bc-border flex gap-2 items-end">
      <textarea
        ref={textareaRef}
        rows={1}
        value={input}
        onChange={(e) => {
          setInput(e.target.value);
          autoGrow();
        }}
        onKeyDown={(e) => {
          if (e.key === "Enter" && !e.shiftKey) {
            e.preventDefault();
            void handleSend();
          }
          if (e.key === "Escape") {
            setInput("");
            if (textareaRef.current) {
              textareaRef.current.style.height = "auto";
              textareaRef.current.blur();
            }
          }
        }}
        placeholder={`Message #${channelName}...`}
        disabled={disabled}
        className="flex-1 bg-bc-bg border border-bc-border rounded px-3 py-1.5 text-sm focus:outline-none focus:border-bc-accent focus-visible:ring-1 focus-visible:ring-bc-accent resize-none disabled:opacity-50"
      />
      <button
        type="button"
        onClick={() => void handleSend()}
        disabled={sending || !input.trim() || disabled}
        className="px-4 py-1.5 bg-bc-accent text-bc-bg rounded text-sm font-medium disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bc-bg"
      >
        {sending ? "..." : "Send"}
      </button>
    </div>
  );
}
