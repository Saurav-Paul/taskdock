import { useEffect, useState } from "react";
import type { Comment, Issue, IssuePatch, IssueRef, Label, User } from "../api";
import {
  addComment,
  createIssue,
  deleteIssue,
  getComments,
  getIssue,
  getIssues,
  PRIORITIES,
  PRIORITY_LABELS,
  STATUSES,
  STATUS_LABELS,
  updateIssue,
} from "../api";
import { Avatar, relativeTime, StatusIcon, STATUS_COLORS } from "./bits";
import { Markdown } from "./Markdown";
import { MarkdownEditor } from "./MarkdownEditor";

interface Props {
  issue: Issue;
  users: User[];
  labels: Label[];
  onClose: () => void;
  onChanged: () => void;
  onDeleted: () => void;
  onOpenIssue: (key: string) => void;
}

export function IssueDetail({ issue: initial, users, labels, onClose, onChanged, onDeleted, onOpenIssue }: Props) {
  const [issue, setIssue] = useState(initial);
  const [editingTitle, setEditingTitle] = useState(false);
  const [titleDraft, setTitleDraft] = useState(initial.title);
  const [comments, setComments] = useState<Comment[]>([]);
  const [commentDraft, setCommentDraft] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [addingSubtask, setAddingSubtask] = useState(false);
  const [subtaskTitle, setSubtaskTitle] = useState("");
  const [addingDep, setAddingDep] = useState(false);
  const [depQuery, setDepQuery] = useState("");
  const [depResults, setDepResults] = useState<Issue[]>([]);

  useEffect(() => {
    setIssue(initial);
    setTitleDraft(initial.title);
    setEditingTitle(false);
    setAddingSubtask(false);
    setSubtaskTitle("");
    setAddingDep(false);
    setDepQuery("");
    getComments(initial.key).then(setComments).catch(() => setComments([]));
  }, [initial.key]);

  // Search candidates for the "Blocked by" add control.
  useEffect(() => {
    if (!addingDep) return;
    let stale = false;
    getIssues({ q: depQuery.trim() || undefined })
      .then((list) => {
        if (stale) return;
        setDepResults(
          list
            .filter((i) => i.key !== issue.key && !issue.depends_on.some((d) => d.key === i.key))
            .slice(0, 6)
        );
      })
      .catch(() => setDepResults([]));
    return () => {
      stale = true;
    };
  }, [addingDep, depQuery, issue.key, issue.depends_on]);

  async function patch(p: IssuePatch) {
    try {
      const updated = await updateIssue(issue.key, p);
      setIssue(updated);
      setError(null);
      onChanged();
      return true;
    } catch (e) {
      setError(`Failed to save: ${(e as Error).message}`);
      return false;
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

  async function submitSubtask() {
    const title = subtaskTitle.trim();
    if (!title) return;
    try {
      await createIssue({ project: issue.project, title, parent: issue.key });
      setSubtaskTitle("");
      setIssue(await getIssue(issue.key));
      setError(null);
      onChanged();
    } catch (e) {
      setError(`Failed to add subtask: ${(e as Error).message}`);
    }
  }

  async function addDependency(key: string) {
    const ok = await patch({ depends_on: [...issue.depends_on.map((d) => d.key), key] });
    if (ok) {
      setAddingDep(false);
      setDepQuery("");
    }
  }

  function removeDependency(key: string) {
    void patch({ depends_on: issue.depends_on.filter((d) => d.key !== key).map((d) => d.key) });
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

  const subtasksDone = issue.subtasks.filter((s) => s.status === "done").length;

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
            {issue.parent && (
              <button
                className="parent-chip"
                onClick={() => onOpenIssue(issue.parent!.key)}
                title={`${issue.parent.key} ${issue.parent.title}`}
              >
                ↑ <span className="issue-key">{issue.parent.key}</span>
                <span className="parent-chip-title">{issue.parent.title}</span>
              </button>
            )}

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

            <div className="subtasks">
              <div className="subtasks-header">
                <h3 className="subtasks-title">Subtasks</h3>
                {issue.subtasks.length > 0 && (
                  <span className="muted">
                    {issue.subtasks.length} · {subtasksDone}/{issue.subtasks.length} done
                  </span>
                )}
                <span className="detail-header-spacer" />
                {!addingSubtask && (
                  <button className="btn btn-small" onClick={() => setAddingSubtask(true)}>
                    + Add subtask
                  </button>
                )}
              </div>
              {issue.subtasks.map((s) => (
                <div key={s.key} className="subtask-row" onClick={() => onOpenIssue(s.key)}>
                  <StatusIcon status={s.status} />
                  <span className="issue-key">{s.key}</span>
                  <span className="issue-title">{s.title}</span>
                </div>
              ))}
              {addingSubtask && (
                <input
                  className="subtask-input"
                  placeholder="Subtask title… (Enter to create, Esc to cancel)"
                  value={subtaskTitle}
                  autoFocus
                  onChange={(e) => setSubtaskTitle(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") void submitSubtask();
                    if (e.key === "Escape") {
                      setSubtaskTitle("");
                      setAddingSubtask(false);
                    }
                  }}
                />
              )}
            </div>

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
            <PropRow label="Blocked by">
              <div className="relation-list">
                {issue.depends_on.map((r) => (
                  <RelationChip key={r.key} issue={r} onOpen={onOpenIssue} onRemove={() => removeDependency(r.key)} />
                ))}
                {addingDep ? (
                  <div className="relation-add">
                    <input
                      placeholder="Search issues…"
                      value={depQuery}
                      autoFocus
                      onChange={(e) => setDepQuery(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === "Escape") {
                          setAddingDep(false);
                          setDepQuery("");
                        }
                      }}
                    />
                    {depResults.length > 0 && (
                      <div className="relation-results">
                        {depResults.map((i) => (
                          <button key={i.key} className="relation-result" onClick={() => void addDependency(i.key)}>
                            <StatusIcon status={i.status} />
                            <span className="issue-key">{i.key}</span>
                            <span className="issue-title">{i.title}</span>
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                ) : (
                  <button className="label-toggle" onClick={() => setAddingDep(true)}>
                    + Add
                  </button>
                )}
              </div>
            </PropRow>
            <PropRow label="Blocks">
              <div className="relation-list">
                {issue.blocks.length === 0 && <span className="muted">None</span>}
                {issue.blocks.map((r) => (
                  <RelationChip key={r.key} issue={r} onOpen={onOpenIssue} />
                ))}
              </div>
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

function RelationChip({
  issue,
  onOpen,
  onRemove,
}: {
  issue: IssueRef;
  onOpen: (key: string) => void;
  onRemove?: () => void;
}) {
  return (
    <span className="relation-chip">
      <button
        className="relation-key"
        style={{ color: STATUS_COLORS[issue.status] }}
        onClick={() => onOpen(issue.key)}
        title={issue.title}
      >
        {issue.key}
      </button>
      {onRemove && (
        <button className="relation-remove" onClick={onRemove} title="Remove">
          ×
        </button>
      )}
    </span>
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
