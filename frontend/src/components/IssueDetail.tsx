import { useEffect, useState } from "react";
import type { Comment, Issue, IssuePatch, Label, User } from "../api";
import {
  addComment,
  deleteIssue,
  getComments,
  PRIORITIES,
  PRIORITY_LABELS,
  STATUSES,
  STATUS_LABELS,
  updateIssue,
} from "../api";
import { Avatar, relativeTime } from "./bits";
import { Markdown } from "./Markdown";
import { MarkdownEditor } from "./MarkdownEditor";

interface Props {
  issue: Issue;
  users: User[];
  labels: Label[];
  onClose: () => void;
  onChanged: () => void;
  onDeleted: () => void;
}

export function IssueDetail({ issue: initial, users, labels, onClose, onChanged, onDeleted }: Props) {
  const [issue, setIssue] = useState(initial);
  const [editingTitle, setEditingTitle] = useState(false);
  const [titleDraft, setTitleDraft] = useState(initial.title);
  const [comments, setComments] = useState<Comment[]>([]);
  const [commentDraft, setCommentDraft] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setIssue(initial);
    setTitleDraft(initial.title);
    setEditingTitle(false);
    getComments(initial.key).then(setComments).catch(() => setComments([]));
  }, [initial.key]);

  async function patch(p: IssuePatch) {
    try {
      const updated = await updateIssue(issue.key, p);
      setIssue(updated);
      setError(null);
      onChanged();
    } catch (e) {
      setError(`Failed to save: ${(e as Error).message}`);
    }
  }

  function saveTitle() {
    setEditingTitle(false);
    const t = titleDraft.trim();
    if (t && t !== issue.title) void patch({ title: t });
    else setTitleDraft(issue.title);
  }

  function saveDescription(md: string) {
    if (md !== issue.description) void patch({ description: md });
  }

  function toggleLabel(name: string) {
    const next = issue.labels.includes(name)
      ? issue.labels.filter((l) => l !== name)
      : [...issue.labels, name];
    void patch({ labels: next });
  }

  async function submitComment() {
    const body = commentDraft.trim();
    if (!body) return;
    try {
      await addComment(issue.key, "saurav", body);
      setCommentDraft("");
      setComments(await getComments(issue.key));
    } catch (e) {
      setError(`Failed to comment: ${(e as Error).message}`);
    }
  }

  async function handleDelete() {
    if (!confirm(`Delete ${issue.key}?`)) return;
    try {
      await deleteIssue(issue.key);
      onDeleted();
    } catch (e) {
      setError(`Failed to delete: ${(e as Error).message}`);
    }
  }

  return (
    <div className="detail-overlay" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className="detail-panel">
        <div className="detail-header">
          <span className="issue-key">{issue.key}</span>
          <span className="detail-header-spacer" />
          <button className="btn btn-small btn-danger" onClick={handleDelete}>
            Delete
          </button>
          <button className="btn btn-small" onClick={onClose} title="Close (Esc)">
            ✕
          </button>
        </div>
        {error && <div className="error-bar">{error}</div>}
        <div className="detail-body">
          <div className="detail-main">
            {editingTitle ? (
              <input
                className="title-input"
                value={titleDraft}
                autoFocus
                onChange={(e) => setTitleDraft(e.target.value)}
                onBlur={saveTitle}
                onKeyDown={(e) => {
                  if (e.key === "Enter") saveTitle();
                  if (e.key === "Escape") {
                    setTitleDraft(issue.title);
                    setEditingTitle(false);
                  }
                }}
              />
            ) : (
              <h1 className="detail-title" onClick={() => setEditingTitle(true)} title="Click to edit">
                {issue.title}
              </h1>
            )}

            <MarkdownEditor
              key={issue.key}
              value={issue.description}
              placeholder="Add a description…"
              onSave={saveDescription}
            />

            <div className="comments">
              <h3 className="comments-title">Comments</h3>
              {comments.length === 0 && <div className="muted">No comments yet.</div>}
              {comments.map((c) => (
                <div key={c.id} className="comment">
                  <div className="comment-meta">
                    <Avatar name={c.author} />
                    <span className="comment-author">{c.author}</span>
                    <span className="muted">{relativeTime(c.created_at)}</span>
                  </div>
                  <Markdown source={c.body} />
                </div>
              ))}
              <div className="comment-box">
                <textarea
                  placeholder="Leave a comment… (markdown supported, ⌘↵ to post)"
                  value={commentDraft}
                  onChange={(e) => setCommentDraft(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) void submitComment();
                  }}
                />
                <button className="btn btn-primary btn-small" onClick={submitComment} disabled={!commentDraft.trim()}>
                  Comment
                </button>
              </div>
            </div>
          </div>

          <div className="detail-props">
            <PropRow label="Status">
              <select value={issue.status} onChange={(e) => void patch({ status: e.target.value as Issue["status"] })}>
                {STATUSES.map((s) => (
                  <option key={s} value={s}>
                    {STATUS_LABELS[s]}
                  </option>
                ))}
              </select>
            </PropRow>
            <PropRow label="Priority">
              <select
                value={issue.priority}
                onChange={(e) => void patch({ priority: e.target.value as Issue["priority"] })}
              >
                {PRIORITIES.map((p) => (
                  <option key={p} value={p}>
                    {PRIORITY_LABELS[p]}
                  </option>
                ))}
              </select>
            </PropRow>
            <PropRow label="Assignee">
              <select value={issue.assignee ?? ""} onChange={(e) => void patch({ assignee: e.target.value })}>
                <option value="">Unassigned</option>
                {users.map((u) => (
                  <option key={u.name} value={u.name}>
                    {u.display_name}
                  </option>
                ))}
              </select>
            </PropRow>
            <PropRow label="Labels">
              <div className="label-toggle-list">
                {labels.length === 0 && <span className="muted">No labels</span>}
                {labels.map((l) => {
                  const active = issue.labels.includes(l.name);
                  return (
                    <button
                      key={l.id}
                      className={`label-toggle ${active ? "active" : ""}`}
                      onClick={() => toggleLabel(l.name)}
                    >
                      <span className="label-dot" style={{ background: l.color }} />
                      {l.name}
                    </button>
                  );
                })}
              </div>
            </PropRow>
            <PropRow label="Created">
              <span className="muted">{relativeTime(issue.created_at)}</span>
            </PropRow>
            <PropRow label="Updated">
              <span className="muted">{relativeTime(issue.updated_at)}</span>
            </PropRow>
          </div>
        </div>
      </div>
    </div>
  );
}

function PropRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="prop-row">
      <div className="prop-label">{label}</div>
      <div className="prop-value">{children}</div>
    </div>
  );
}
