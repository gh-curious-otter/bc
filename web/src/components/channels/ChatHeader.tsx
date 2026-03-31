import { useState } from "react";
import type { Channel } from "../../api/client";

export function ChatHeader({
  channelName,
  channel,
  messageCount,
  showMembers,
  onToggleMembers,
  onDescriptionSave,
}: {
  channelName: string;
  channel?: Channel;
  messageCount: number;
  showMembers: boolean;
  onToggleMembers: () => void;
  onDescriptionSave: (description: string) => Promise<void>;
}) {
  const [editingDesc, setEditingDesc] = useState(false);
  const [descDraft, setDescDraft] = useState("");
  const [savingDesc, setSavingDesc] = useState(false);

  const handleSaveDescription = async () => {
    setSavingDesc(true);
    try {
      await onDescriptionSave(descDraft);
      setEditingDesc(false);
    } catch {
      // keep editing open
    } finally {
      setSavingDesc(false);
    }
  };

  return (
    <div className="px-4 py-2 border-b border-bc-border bg-bc-surface space-y-1">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="font-medium">#{channelName}</span>
          <span className="text-xs text-bc-muted">
            {messageCount} message{messageCount !== 1 ? "s" : ""}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={onToggleMembers}
            className={`px-2 py-1 rounded border text-xs transition-colors focus-visible:ring-1 focus-visible:ring-bc-accent ${
              showMembers
                ? "border-bc-accent text-bc-accent bg-bc-accent/10"
                : "border-bc-border text-bc-muted hover:text-bc-text"
            }`}
            aria-label="Toggle members panel"
            aria-pressed={showMembers}
          >
            Members ({channel?.member_count ?? 0})
          </button>
          <button
            type="button"
            onClick={() => {
              setDescDraft(channel?.description ?? "");
              setEditingDesc(true);
            }}
            className="px-2 py-1 rounded border border-bc-border text-xs text-bc-muted hover:text-bc-text transition-colors focus-visible:ring-1 focus-visible:ring-bc-accent"
            aria-label="Edit channel description"
          >
            Edit
          </button>
        </div>
      </div>
      {editingDesc && (
        <div className="flex items-center gap-2">
          <input
            type="text"
            value={descDraft}
            onChange={(e) => setDescDraft(e.target.value)}
            placeholder="Channel description"
            className="flex-1 px-2 py-1 rounded border border-bc-border bg-bc-bg text-sm text-bc-text focus:outline-none focus:border-bc-accent"
            onKeyDown={(e) => {
              if (e.key === "Enter") void handleSaveDescription();
              if (e.key === "Escape") setEditingDesc(false);
            }}
            aria-label="Channel description"
          />
          <button
            type="button"
            onClick={() => void handleSaveDescription()}
            disabled={savingDesc}
            className="px-2 py-1 rounded bg-bc-accent text-bc-bg text-xs font-medium disabled:opacity-50"
          >
            {savingDesc ? "Saving..." : "Save"}
          </button>
          <button
            type="button"
            onClick={() => setEditingDesc(false)}
            className="px-2 py-1 rounded border border-bc-border text-xs text-bc-muted hover:text-bc-text"
          >
            Cancel
          </button>
        </div>
      )}
      {!editingDesc && channel?.description && (
        <p className="text-xs text-bc-muted">{channel.description}</p>
      )}
    </div>
  );
}
