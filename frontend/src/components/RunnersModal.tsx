import { useEffect, useState } from "react";
import type { Runner } from "../api";
import { getRunners } from "../api";

interface Props {
  /** Seed from App's 10s poll so the panel renders instantly. */
  runners: Runner[];
  onClose: () => void;
  onOpenIssue: (key: string) => void;
}

/** Heartbeats are seconds-fresh, so go finer than relativeTime's minutes. */
function lastSeen(iso: string): string {
  const secs = Math.max(0, Math.floor((Date.now() - new Date(iso).getTime()) / 1000));
  if (secs < 60) return `${secs}s ago`;
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

export function RunnersModal({ runners: initial, onClose, onOpenIssue }: Props) {
  const [runners, setRunners] = useState(initial);

  // Poll every 5s, but only while the modal is mounted (= open). Failures
  // keep the last snapshot — App's slower poll will catch up either way.
  useEffect(() => {
    let cancelled = false;
    const poll = () =>
      getRunners()
        .then((r) => {
          if (!cancelled) setRunners(r);
        })
        .catch(() => {});
    void poll();
    const timer = window.setInterval(poll, 5000);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, []);

  const online = runners.filter((r) => r.online).length;

  return (
    <div className="modal-overlay" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal modal-runners">
        <div className="modal-header">
          <span className={`runner-dot ${online > 0 ? "on" : "off"}`} />
          <span className="runners-status-text">
            {runners.length === 0
              ? "Runners"
              : `${online}/${runners.length} runner${runners.length === 1 ? "" : "s"} online`}
          </span>
          <span className="detail-header-spacer" />
          <button className="btn btn-small" onClick={onClose} title="Close (Esc)">
            ✕
          </button>
        </div>

        {runners.length === 0 ? (
          <div className="runners-empty">No runners. Start one with: dispatcher .</div>
        ) : (
          <div className="runners-list">
            {runners.map((r) => (
              <div key={r.id} className="runner-row">
                <span className={`runner-dot ${r.online ? "on" : "off"}`} />
                <span className="runner-name">{r.display_name}</span>
                {r.path && (
                  <span className="runner-path" title={r.path}>
                    {r.path}
                  </span>
                )}
                {r.hostname && <span className="runner-host muted">{r.hostname}</span>}
                {r.last_seen && (
                  <span className="runner-last-seen muted">last seen {lastSeen(r.last_seen)}</span>
                )}
                {r.current_issue && (
                  <button
                    className="runner-issue-chip"
                    onClick={() => onOpenIssue(r.current_issue!.key)}
                    title={`Open ${r.current_issue.key} — ${r.current_issue.title}`}
                  >
                    <span className="issue-key">{r.current_issue.key}</span>
                    <span className="runner-issue-title">{r.current_issue.title}</span>
                  </button>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
