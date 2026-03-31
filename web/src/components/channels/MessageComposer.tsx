import { useState, useCallback, useRef } from "react";

interface PendingFile {
  file: File;
  preview: string | null;
}

export function MessageComposer({
  channelName,
  onSend,
  onFileUpload,
  disabled = false,
}: {
  channelName: string;
  onSend: (content: string) => Promise<void>;
  onFileUpload?: (file: File) => Promise<void>;
  disabled?: boolean;
}) {
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const [pendingFile, setPendingFile] = useState<PendingFile | null>(null);
  const [uploading, setUploading] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

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

  const handleFileSelect = (file: File) => {
    const isImage = file.type.startsWith("image/");
    const preview = isImage ? URL.createObjectURL(file) : null;
    setPendingFile({ file, preview });
  };

  const handleFileUpload = async () => {
    if (!pendingFile || !onFileUpload || uploading) return;
    setUploading(true);
    try {
      await onFileUpload(pendingFile.file);
      if (pendingFile.preview) URL.revokeObjectURL(pendingFile.preview);
      setPendingFile(null);
    } catch {
      // keep pending
    } finally {
      setUploading(false);
    }
  };

  const cancelFile = () => {
    if (pendingFile?.preview) URL.revokeObjectURL(pendingFile.preview);
    setPendingFile(null);
  };

  const handlePaste = (e: React.ClipboardEvent) => {
    if (!onFileUpload) return;
    const items = e.clipboardData.items;
    for (let i = 0; i < items.length; i++) {
      const item = items[i];
      if (item && item.type.startsWith("image/")) {
        e.preventDefault();
        const file = item.getAsFile();
        if (file) handleFileSelect(file);
        return;
      }
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    if (!onFileUpload) return;
    e.preventDefault();
    const file = e.dataTransfer.files[0];
    if (file) handleFileSelect(file);
  };

  return (
    <div
      className="border-t border-bc-border"
      onDrop={handleDrop}
      onDragOver={(e) => e.preventDefault()}
    >
      {/* File preview */}
      {pendingFile && (
        <div className="px-3 pt-2 flex items-center gap-2">
          {pendingFile.preview ? (
            <img
              src={pendingFile.preview}
              alt="preview"
              className="w-16 h-16 rounded border border-bc-border object-cover"
            />
          ) : (
            <div className="w-16 h-16 rounded border border-bc-border bg-bc-surface flex items-center justify-center text-xs text-bc-muted">
              📎
            </div>
          )}
          <div className="flex-1 min-w-0">
            <p className="text-xs text-bc-text truncate">{pendingFile.file.name}</p>
            <p className="text-[10px] text-bc-muted">
              {(pendingFile.file.size / 1024).toFixed(1)} KB
            </p>
          </div>
          <button
            type="button"
            onClick={() => void handleFileUpload()}
            disabled={uploading}
            className="text-xs px-2 py-1 rounded bg-bc-accent text-bc-bg font-medium disabled:opacity-50"
          >
            {uploading ? "..." : "Upload"}
          </button>
          <button
            type="button"
            onClick={cancelFile}
            className="text-xs px-2 py-1 rounded border border-bc-border text-bc-muted hover:text-bc-text"
          >
            Cancel
          </button>
        </div>
      )}

      {/* Input area */}
      <div className="p-3 flex gap-2 items-end">
        {onFileUpload && (
          <>
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              disabled={disabled}
              className="px-2 py-1.5 rounded border border-bc-border text-bc-muted hover:text-bc-accent hover:border-bc-accent transition-colors disabled:opacity-50 focus-visible:ring-1 focus-visible:ring-bc-accent"
              aria-label="Attach file"
              title="Attach file"
            >
              📎
            </button>
            <input
              ref={fileInputRef}
              type="file"
              className="hidden"
              onChange={(e) => {
                const file = e.target.files?.[0];
                if (file) handleFileSelect(file);
                e.target.value = "";
              }}
            />
          </>
        )}
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
          onPaste={handlePaste}
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
    </div>
  );
}
