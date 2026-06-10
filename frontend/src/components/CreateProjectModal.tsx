import { useState } from "react";
import type { NewProject, Project } from "../api";
import { createProject } from "../api";

function deriveKey(name: string): string {
  return name.replace(/[^a-z]/gi, "").slice(0, 3).toUpperCase();
}

interface Props {
  onClose: () => void;
  onCreated: (project: Project) => void;
}

export function CreateProjectModal({ onClose, onCreated }: Props) {
  const [name, setName] = useState("");
  const [key, setKey] = useState("");
  const [keyTouched, setKeyTouched] = useState(false);
  const [description, setDescription] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const canSubmit = name.trim() !== "" && !submitting;

  async function submit() {
    if (!canSubmit) return;
    setSubmitting(true);
    setError(null);
    const body: NewProject = { name: name.trim() };
    if (key.trim()) body.key = key.trim();
    if (description.trim()) body.description = description.trim();
    try {
      onCreated(await createProject(body));
    } catch (e) {
      setError(`Failed to create: ${(e as Error).message}`);
      setSubmitting(false);
    }
  }

  return (
    <div className="modal-overlay" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div
        className="modal modal-small"
        onKeyDown={(e) => {
          if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
            e.preventDefault();
            void submit();
          }
        }}
      >
        <div className="modal-header">
          <span className="muted">New project</span>
        </div>
        {error && <div className="error-bar">{error}</div>}

        <input
          className="title-input"
          placeholder="Project name"
          value={name}
          autoFocus
          onChange={(e) => {
            setName(e.target.value);
            if (!keyTouched) setKey(deriveKey(e.target.value));
          }}
        />

        <input
          className="key-input"
          placeholder="Key"
          value={key}
          onChange={(e) => {
            const v = e.target.value.toUpperCase();
            setKey(v);
            // An emptied key resumes auto-fill from the name.
            setKeyTouched(v !== "");
          }}
        />

        <textarea
          className="description-input"
          placeholder="Description (optional)"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />

        <div className="modal-footer">
          <span className="muted">⌘↵ to create</span>
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button className="btn btn-primary" onClick={submit} disabled={!canSubmit}>
            Create project
          </button>
        </div>
      </div>
    </div>
  );
}
