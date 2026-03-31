import type { ReactNode } from "react";

/**
 * Renders message content with basic markdown-like formatting:
 * - URLs become clickable links (images rendered inline)
 * - **bold** text
 * - `code` backticks
 * - #channel references link to /channels/<name>
 * - @mentions link to agent detail page
 * - [file:ID] attachment references rendered inline
 */
export function MessageContent({ content }: { content: string }) {
  return <>{parseContent(content)}</>;
}

const IMAGE_EXT = /\.(png|jpg|jpeg|gif|webp|svg)(\?|$)/i;

/** Tokenize and render inline formatting. */
function parseContent(text: string): ReactNode[] {
  // Split on patterns we want to handle, preserving delimiters
  // Order: file refs, URLs (greedy), bold, code, #channel, @mention
  const pattern =
    /(\[file:[a-zA-Z0-9_-]+\])|(https?:\/\/[^\s<>)"']+)|(\*\*(?:[^*]|\*(?!\*))+\*\*)|(`[^`]+`)|(\B#(?=[a-zA-Z0-9_-]*[a-zA-Z])[a-zA-Z0-9_-]+\b)|(@[a-zA-Z0-9_-]+)/g;

  const nodes: ReactNode[] = [];
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = pattern.exec(text)) !== null) {
    // Push preceding plain text
    if (match.index > lastIndex) {
      nodes.push(text.slice(lastIndex, match.index));
    }

    const [full] = match;
    const key = `${match.index}`;

    if (match[1]) {
      // [file:ID] attachment reference
      const fileId = full.slice(6, -1);
      const fileUrl = `/api/files/${encodeURIComponent(fileId)}`;
      nodes.push(
        <a
          key={key}
          href={fileUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-block my-1"
        >
          <img
            src={fileUrl}
            alt="attachment"
            className="max-w-sm max-h-64 rounded border border-bc-border"
            loading="lazy"
            onError={(e) => {
              const el = e.currentTarget;
              const parent = el.parentElement;
              if (parent) {
                parent.className = "text-bc-accent underline-offset-2 hover:underline text-xs";
                parent.textContent = `📎 ${fileId}`;
              }
            }}
          />
        </a>,
      );
    } else if (match[2]) {
      // URL — render images inline
      if (IMAGE_EXT.test(full)) {
        nodes.push(
          <a key={key} href={full} target="_blank" rel="noopener noreferrer" className="inline-block my-1">
            <img src={full} alt="" className="max-w-sm max-h-64 rounded border border-bc-border" loading="lazy" />
          </a>,
        );
      } else {
        nodes.push(
          <a
            key={key}
            href={full}
            target="_blank"
            rel="noopener noreferrer"
            className="text-bc-accent underline-offset-2 hover:underline"
          >
            {full}
          </a>,
        );
      }
    } else if (match[3]) {
      // Bold **text**
      const inner = full.slice(2, -2);
      nodes.push(<strong key={key}>{inner}</strong>);
    } else if (match[4]) {
      // Inline code `text`
      const inner = full.slice(1, -1);
      nodes.push(
        <code
          key={key}
          className="rounded bg-bc-surface px-1 py-0.5 font-mono text-[0.85em]"
        >
          {inner}
        </code>,
      );
    } else if (match[5]) {
      // #channel reference → link to /channels/<name>
      const channelName = full.slice(1);
      nodes.push(
        <a
          key={key}
          href={`/channels/${channelName}`}
          className="text-bc-accent font-medium hover:underline"
        >
          {full}
        </a>,
      );
    } else if (match[6]) {
      // @mention → link to agent detail page
      const name = full.slice(1);
      nodes.push(
        <a
          key={key}
          href={`/agents/${name}`}
          className="text-bc-accent font-medium hover:underline"
        >
          {full}
        </a>,
      );
    }

    lastIndex = match.index + full.length;
  }

  // Push trailing plain text
  if (lastIndex < text.length) {
    nodes.push(text.slice(lastIndex));
  }

  return nodes;
}
