import { useEffect, useState } from "react";
import type { Project } from "../api";
import { deleteProject, updateProject } from "../api";

interface Props {
  project: Project;
  issueCount: number;
  onChanged: () => void;
  onDeleted: () => void;
}

export function ProjectHeader({ project, issueCount, onChanged, onDeleted }: Props) {
  const [editing, setEditing] = useState(false);
  const [nameDraft, setNameDraft] = useState(project.name);
  const [descriptionDraft, setDescriptionDraft] = useState(project.description);
  const [webhookDraft, setWebhookDraft] = useState(project.webhook_url ?? "");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setEditing(false);
    setNameDraft(project.name);
    setDescriptionDraft(project.description);
    setWebhookDraft(project.webhook_url ?? "");
    setError(null);
  }, [project.key, project.name, project.description, project.webhook_url]);

  function cancelEdit() {
    setEditing(false);
    setNameDraft(project.name);
    setDescriptionDraft(project.description);
    setWebhookDraft(project.webhook_url ?? "");
    setError(null);
  }

  async function save() {
    const name = nameDraft.trim();
    if (!name || saving) return;
    setSaving(true);
    setError(null);
    try {
      await updateProject(project.key, {
        name,
        description: descriptionDraft.trim(),
        webhook_url: webhookDraft.trim(),
      });
      setEditing(false);
      onChanged();
    } catch (e) {
      setError(`Failed to save: ${(e as Error).message}`);
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (
      !window.confirm(
        `Delete project ${project.key} (${project.name})? ALL issues in this project will be deleted.`
      )
    )
      return;
    try {
      await deleteProject(project.key);
      onDeleted();
    } catch (e) {
      setError(`Failed to delete: ${(e as Error).message}`);
    }
  }

  return (
    <>
      <div className={`project-header ${editing ? "editing" : ""}`}>
        {editing ? (
          <>
            <div className="project-header-row">
              <input
                className="project-name-input"
                value={nameDraft}
                autoFocus
                placeholder="Project name"
                onChange={(e) => setNameDraft(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") void save();
                  if (e.key === "Escape") cancelEdit();
                }}
              />
              <span className="project-key-chip">{project.key}</span>
              <input
                className="project-description-input"
                value={descriptionDraft}
                placeholder="Description"
                onChange={(e) => setDescriptionDraft(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") void save();
                  if (e.key === "Escape") cancelEdit();
                }}
              />
              <button className="btn btn-small" onClick={cancelEdit}>
                Cancel
              </button>
              <button
                className="btn btn-primary btn-small"
                onClick={save}
                disabled={!nameDraft.trim() || saving}
              >
                Save
              </button>
            </div>
            <div className="project-header-row">
              <span className="prop-label webhook-label">Webhook URL</span>
              <input
                className="project-webhook-input"
                value={webhookDraft}
                placeholder="https://… (fires issue/comment events)"
                onChange={(e) => setWebhookDraft(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") void save();
                  if (e.key === "Escape") cancelEdit();
                }}
              />
            </div>
          </>
        ) : (
          <>
            <span className="project-name">{project.name}</span>
            <span className="project-key-chip">{project.key}</span>
            {project.description && (
              <span className="project-description muted" title={project.description}>
                {project.description}
              </span>
            )}
            {project.webhook_url && (
              <span className="webhook-indicator" title="Webhook configured">
                <ZapIcon />
              </span>
            )}
            <span className="project-header-spacer" />
            <span className="muted issue-count">
              {issueCount} issue{issueCount === 1 ? "" : "s"}
            </span>
            <button className="icon-btn" onClick={() => setEditing(true)} title="Edit project">
              <PencilIcon />
            </button>
            <button className="icon-btn" onClick={handleDelete} title="Delete project">
              <TrashIcon />
            </button>
          </>
        )}
      </div>
      {error && <div className="error-bar">{error}</div>}
    </>
  );
}

function ZapIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 14 14" aria-label="Webhook configured">
      <path
        d="M8 1.5 L3.5 8 H6.7 L6 12.5 L10.5 6 H7.3 Z"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.2"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function PencilIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" aria-label="Edit">
      <path
        d="M2.5 11.5 L2.9 9.3 L9.6 2.6 A1 1 0 0 1 11 2.6 L11.4 3 A1 1 0 0 1 11.4 4.4 L4.7 11.1 Z"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.3"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function TrashIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" aria-label="Delete">
      <path
        d="M2.5 4 H11.5 M5.5 4 V2.8 A0.8 0.8 0 0 1 6.3 2 H7.7 A0.8 0.8 0 0 1 8.5 2.8 V4 M4 4 L4.5 11.2 A1 1 0 0 0 5.5 12 H8.5 A1 1 0 0 0 9.5 11.2 L10 4"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.3"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path d="M6 6.2 V9.8 M8 6.2 V9.8" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
    </svg>
  );
}
