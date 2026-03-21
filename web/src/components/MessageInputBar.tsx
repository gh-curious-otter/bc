import { useState, useCallback } from 'react';

interface MessageInputBarProps {
  onSend: (message: string) => Promise<void>;
  placeholder?: string;
  disabled?: boolean;
}

/**
 * Text input with send button for messaging agents or channels.
 *
 * Supports Enter to send, Esc to clear, and shows a loading state while sending.
 */
export function MessageInputBar({
  onSend,
  placeholder = 'Send a message...',
  disabled = false,
}: MessageInputBarProps) {
  const [message, setMessage] = useState('');
  const [sending, setSending] = useState(false);

  const handleSend = useCallback(async () => {
    const trimmed = message.trim();
    if (!trimmed || sending || disabled) return;
    setSending(true);
    try {
      await onSend(trimmed);
      setMessage('');
    } finally {
      setSending(false);
    }
  }, [message, sending, disabled, onSend]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter') {
        void handleSend();
      } else if (e.key === 'Escape') {
        setMessage('');
      }
    },
    [handleSend],
  );

  const isDisabled = disabled || sending;

  return (
    <div className="flex gap-2">
      <input
        type="text"
        value={message}
        onChange={(e) => setMessage(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        disabled={isDisabled}
        className="flex-1 bg-bc-bg border border-bc-border rounded px-3 py-1.5 text-sm focus:outline-none focus:border-bc-accent disabled:opacity-50"
      />
      <button
        onClick={() => void handleSend()}
        disabled={isDisabled || !message.trim()}
        className="px-3 py-1.5 bg-bc-accent text-bc-bg rounded text-sm font-medium disabled:opacity-50"
      >
        {sending ? 'Sending...' : 'Send'}
      </button>
    </div>
  );
}
