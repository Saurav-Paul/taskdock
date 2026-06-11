import { useEffect, useRef, useState } from "react";
import type { DispatcherStatus } from "../api";
import { getDispatcherLog } from "../api";

interface Props {
  status: DispatcherStatus;
  onClose: () => void;
  onOpenIssue: (key: string) => void;
}

/** "2026-06-11T16:17:59+03:00" → "16:17" (local); "" when unparseable. */
function fmtTime(iso: string): string {
  const d = new Date(iso);
  return isNaN(d.getTime())
    ? ""
    : d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function statusText(status: DispatcherStatus): string {
  if (!status.running) return "Not running — start it with ./bin/dispatcher";
  const parts: string[] = [status.paused ? "Paused" : "Running"];
  if (status.dry_run) parts.push("dry run");
  const n = status.projects?.length ?? 0;
  parts.push(`${n} project${n === 1 ? "" : "s"} mapped`);
  const since = status.started_at ? fmtTime(status.started_at) : "";
  if (since) parts.push(`since ${since}`);
  return parts.join(" · ");
}

export function DispatcherModal({ status, onClose, onOpenIssue }: Props) {
  const [lines, setLines] = useState<string[] | null>(null);
  const [logError, setLogError] = useState<string | null>(null);
  const logRef = useRef<HTMLPreElement>(null);
  // Auto-scroll only while "pinned" to the bottom — scrolling up to read
  // history must not be yanked back down by the next poll.
  const pinnedRef = useRef(true);

  // Poll the log every 2s, but only while the modal is mounted (= open).
  useEffect(() => {
    let cancelled = false;
    const load = () =>
      getDispatcherLog()
        .then((r) => {
          if (cancelled) return;
          setLines(r.lines);
          setLogError(null);
        })
        .catch((e) => {
          if (!cancelled) setLogError((e as Error).message);
        });
    void load();
    const timer = window.setInterval(load, 2000);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, []);

  useEffect(() => {
    const el = logRef.current;
    if (el && pinnedRef.current) el.scrollTop = el.scrollHeight;
  }, [lines]);

  function onLogScroll() {
    const el = logRef.current;
    if (!el) return;
    pinnedRef.current = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
  }

  const dot = status.running ? (status.paused ? "paused" : "on") : "off";
  const sessions = status.sessions ?? [];

  return (
    <div className="modal-overlay" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal modal-dispatcher">
        <div className="modal-header">
          <span className={`dispatcher-dot ${dot}`} />
          <span className="dispatcher-status-text">{statusText(status)}</span>
          <span className="detail-header-spacer" />
          <button className="btn btn-small" onClick={onClose} title="Close (Esc)">
            ✕
          </button>
        </div>

        {sessions.length > 0 && (
          <div className="dispatcher-sessions">
            <span className="prop-label dispatcher-sessions-label">Sessions</span>
            {sessions.map((s) => (
              <button
                key={s.pane_id}
                className="dispatcher-session-chip"
                onClick={() => onOpenIssue(s.key)}
                title={`Open ${s.key} (pane ${s.pane_id})`}
              >
                <span className="issue-key">{s.key}</span>
                <span className="muted">{s.project}</span>
              </button>
            ))}
          </div>
        )}

        {lines === null && logError ? (
          <div className="dispatcher-log dispatcher-log-empty muted">
            Log unavailable — {logError}
          </div>
        ) : (
          <pre ref={logRef} className="dispatcher-log" onScroll={onLogScroll}>
            {lines === null ? "Loading…" : lines.length ? lines.join("") : "Log is empty."}
          </pre>
        )}
      </div>
    </div>
  );
}
