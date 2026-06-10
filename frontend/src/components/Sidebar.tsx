import type { Project, Status } from "../api";
import { STATUSES, STATUS_LABELS } from "../api";
import { StatusIcon } from "./bits";

interface Props {
  projects: Project[];
  selectedProject: string | null;
  selectedStatus: Status | null;
  onSelectProject: (key: string | null) => void;
  onSelectStatus: (status: Status | null) => void;
  onNewIssue: () => void;
  onNewProject: () => void;
}

export function Sidebar(props: Props) {
  const { projects, selectedProject, selectedStatus, onSelectProject, onSelectStatus, onNewIssue, onNewProject } = props;
  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <span className="logo">taskdock</span>
        <button className="btn btn-small" onClick={onNewIssue} title="New issue (c)">
          + New
        </button>
      </div>

      <nav className="sidebar-section">
        <button
          className={`sidebar-item ${selectedProject === null ? "active" : ""}`}
          onClick={() => onSelectProject(null)}
        >
          All issues
        </button>
      </nav>

      <div className="sidebar-section">
        <div className="sidebar-title">Projects</div>
        {projects.map((p) => (
          <button
            key={p.key}
            className={`sidebar-item ${selectedProject === p.key ? "active" : ""}`}
            onClick={() => onSelectProject(p.key)}
            title={p.description}
          >
            <span className="project-key">{p.key}</span>
            {p.name}
          </button>
        ))}
        <button className="sidebar-item sidebar-item-muted" onClick={onNewProject}>
          + New project
        </button>
      </div>

      <div className="sidebar-section">
        <div className="sidebar-title">Status</div>
        <button
          className={`sidebar-item ${selectedStatus === null ? "active" : ""}`}
          onClick={() => onSelectStatus(null)}
        >
          All statuses
        </button>
        {STATUSES.map((s) => (
          <button
            key={s}
            className={`sidebar-item ${selectedStatus === s ? "active" : ""}`}
            onClick={() => onSelectStatus(s)}
          >
            <StatusIcon status={s} />
            {STATUS_LABELS[s]}
          </button>
        ))}
      </div>
    </aside>
  );
}
