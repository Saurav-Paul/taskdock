import { useRef, useState } from "react";
import type { Label, NewIssue, Priority, Project, Status, User } from "../api";
import { createIssue, PRIORITIES, PRIORITY_LABELS, STATUSES, STATUS_LABELS } from "../api";
import { AssigneeOptions } from "./bits";
import { MarkdownEditor } from "./MarkdownEditor";

interface Props {
  projects: Project[];
  users: User[];
  labels: Label[];
  defaultProject: string | null;
  onClose: () => void;
  onCreated: () => void;
}

export function CreateIssueModal({ projects, users, labels, defaultProject, onClose, onCreated }: Props) {
  const [project, setProject] = useState(defaultProject ?? projects[0]?.key ?? "");
  const [title, setTitle] = useState("");
  const [status, setStatus] = useState<Status>("todo");
  const [priority, setPriority] = useState<Priority>("none");
  const [assignee, setAssignee] = useState("");
  const [dueDate, setDueDate] = useState("");
  const [selectedLabels, setSelectedLabels] = useState<string[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const descriptionRef = useRef("");

  const canSubmit = title.trim() !== "" && project !== "" && !submitting;

  async function submit() {
    if (!canSubmit) return;
    setSubmitting(true);
    setError(null);
    const body: NewIssue = {
      project,
      title: title.trim(),
      status,
      priority,
      labels: selectedLabels,
    };
    if (descriptionRef.current.trim()) body.description = descriptionRef.current;
    if (assignee) body.assignee = assignee;
    if (dueDate) body.due_date = dueDate;
    try {
      await createIssue(body);
      onCreated();
    } catch (e) {
      setError(`Failed to create: ${(e as Error).message}`);
      setSubmitting(false);
    }
  }

  function toggleLabel(name: string) {
    setSelectedLabels((prev) =>
      prev.includes(name) ? prev.filter((l) => l !== name) : [...prev, name]
    );
  }

  return (
    <div className="modal-overlay" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div
        className="modal"
        onKeyDown={(e) => {
          if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
            e.preventDefault();
            void submit();
          }
        }}
      >
        <div className="modal-header">
          <select className="project-select" value={project} onChange={(e) => setProject(e.target.value)}>
            {projects.map((p) => (
              <option key={p.key} value={p.key}>
                {p.key} · {p.name}
              </option>
            ))}
          </select>
          <span className="muted">New issue</span>
        </div>
        {error && <div className="error-bar">{error}</div>}

        <input
          className="title-input"
          placeholder="Issue title"
          value={title}
          autoFocus
          onChange={(e) => setTitle(e.target.value)}
        />

        <MarkdownEditor
          value=""
          placeholder="Add a description…"
          onChange={(md) => (descriptionRef.current = md)}
        />

        <div className="modal-props">
          <select value={status} onChange={(e) => setStatus(e.target.value as Status)}>
            {STATUSES.map((s) => (
              <option key={s} value={s}>
                {STATUS_LABELS[s]}
              </option>
            ))}
          </select>
          <select value={priority} onChange={(e) => setPriority(e.target.value as Priority)}>
            {PRIORITIES.map((p) => (
              <option key={p} value={p}>
                {PRIORITY_LABELS[p]}
              </option>
            ))}
          </select>
          <select value={assignee} onChange={(e) => setAssignee(e.target.value)}>
            <option value="">Unassigned</option>
            <AssigneeOptions users={users} />
          </select>
          <input
            type="date"
            className="due-date-input"
            title="Due date (optional)"
            value={dueDate}
            onChange={(e) => setDueDate(e.target.value)}
          />
        </div>

        {labels.length > 0 && (
          <div className="label-toggle-list">
            {labels.map((l) => {
              const active = selectedLabels.includes(l.name);
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
        )}

        <div className="modal-footer">
          <span className="muted">⌘↵ to create</span>
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button className="btn btn-primary" onClick={submit} disabled={!canSubmit}>
            Create issue
          </button>
        </div>
      </div>
    </div>
  );
}
