import { useRef, useState } from "react";

interface Props {
  /** Selected project key (e.g. "TD"), or null for "All issues". */
  projectKey: string | null;
  onClose: () => void;
}

type CopyTarget = "command" | "json" | "url";

export function McpModal({ projectKey, onClose }: Props) {
  const [copied, setCopied] = useState<CopyTarget | null>(null);
  const [error, setError] = useState<string | null>(null);
  const copiedTimer = useRef<number | null>(null);

  const host = window.location.host;
  const serverName = projectKey ? `taskdock-${projectKey.toLowerCase()}` : "taskdock";
  const url = `http://${host}/mcp${projectKey ? `/${projectKey}` : ""}`;
  const command = `claude mcp add --transport http --scope project ${serverName} ${url}`;
  const json = JSON.stringify(
    { mcpServers: { [serverName]: { type: "http", url } } },
    null,
    2
  );

  function copyText(text: string, which: CopyTarget) {
    navigator.clipboard
      .writeText(text)
      .then(() => {
        setCopied(which);
        if (copiedTimer.current !== null) window.clearTimeout(copiedTimer.current);
        copiedTimer.current = window.setTimeout(() => setCopied(null), 1500);
      })
      .catch((e) => setError(`Failed to copy: ${(e as Error).message}`));
  }

  return (
    <div className="modal-overlay" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal">
        <div className="modal-header">
          <span className="muted">Connect via MCP</span>
          <span className="detail-header-spacer" />
          <button className="btn btn-small" onClick={onClose} title="Close (Esc)">
            ✕
          </button>
        </div>
        {error && <div className="error-bar">{error}</div>}

        <p className="mcp-scope-note">
          {projectKey
            ? `Project-scoped: tools are locked to ${projectKey} — new issues land in this project.`
            : "Workspace-wide: tools can touch every project."}
        </p>

        <div className="mcp-block">
          <div className="prop-label">Claude Code command</div>
          <CodeBox text={command} copied={copied === "command"} onCopy={() => copyText(command, "command")} />
        </div>

        <div className="mcp-block">
          <div className="prop-label">.mcp.json snippet</div>
          <CodeBox text={json} copied={copied === "json"} onCopy={() => copyText(json, "json")} />
        </div>

        <div className="mcp-block">
          <div className="prop-label">URL</div>
          <CodeBox text={url} copied={copied === "url"} onCopy={() => copyText(url, "url")} />
        </div>

        <p className="muted mcp-hint">
          Paste any of these to your coding agent (Claude Code, Codex, …) and ask it to add the MCP
          server.
        </p>
      </div>
    </div>
  );
}

function CodeBox({ text, copied, onCopy }: { text: string; copied: boolean; onCopy: () => void }) {
  return (
    <div className="code-box">
      <pre className="code-box-text">{text}</pre>
      <button className="icon-btn" onClick={onCopy} title="Copy">
        {copied ? <CheckIcon /> : <CopyIcon />}
      </button>
    </div>
  );
}

export function McpIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" aria-label="MCP">
      <path
        d="M5 1.5 V4 M9 1.5 V4 M4 4 H10 V7 A3 3 0 0 1 7 10 A3 3 0 0 1 4 7 Z M7 10 V12.5"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.3"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function CopyIcon() {
  return (
    <svg width="13" height="13" viewBox="0 0 14 14" aria-label="Copy">
      <rect x="4.5" y="4.5" width="7" height="7" rx="1.5" fill="none" stroke="currentColor" strokeWidth="1.3" />
      <path
        d="M9.5 2.5 H4 A1.5 1.5 0 0 0 2.5 4 V9.5"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.3"
        strokeLinecap="round"
      />
    </svg>
  );
}

function CheckIcon() {
  return (
    <svg width="13" height="13" viewBox="0 0 14 14" aria-label="Copied">
      <path
        d="M2.5 7.5 L5.5 10.5 L11.5 3.5"
        fill="none"
        stroke="#4cb782"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
