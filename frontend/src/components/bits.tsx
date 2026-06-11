import type { Label, Priority, Status } from "../api";

export const STATUS_COLORS: Record<Status, string> = {
  backlog: "#62666d",
  todo: "#8a8f98",
  in_progress: "#f2c94c",
  in_review: "#26b5ce",
  done: "#5e6ad2",
  canceled: "#62666d",
};

export function StatusIcon({ status }: { status: Status }) {
  const c = STATUS_COLORS[status];
  switch (status) {
    case "backlog":
      return (
        <svg className="status-icon" width="14" height="14" viewBox="0 0 14 14" aria-label="Backlog">
          <circle cx="7" cy="7" r="5.5" fill="none" stroke={c} strokeWidth="1.5" strokeDasharray="2.4 2" />
        </svg>
      );
    case "todo":
      return (
        <svg className="status-icon" width="14" height="14" viewBox="0 0 14 14" aria-label="Todo">
          <circle cx="7" cy="7" r="5.5" fill="none" stroke={c} strokeWidth="1.5" />
        </svg>
      );
    case "in_progress":
      return (
        <svg className="status-icon" width="14" height="14" viewBox="0 0 14 14" aria-label="In Progress">
          <circle cx="7" cy="7" r="5.5" fill="none" stroke={c} strokeWidth="1.5" />
          <path d="M7 3.5 A3.5 3.5 0 0 1 7 10.5 Z" fill={c} />
        </svg>
      );
    case "in_review":
      return (
        <svg className="status-icon" width="14" height="14" viewBox="0 0 14 14" aria-label="In Review">
          <circle cx="7" cy="7" r="5.5" fill="none" stroke={c} strokeWidth="1.5" />
          <path d="M7 3.5 A3.5 3.5 0 1 1 3.5 7 L7 7 Z" fill={c} />
        </svg>
      );
    case "done":
      return (
        <svg className="status-icon" width="14" height="14" viewBox="0 0 14 14" aria-label="Done">
          <circle cx="7" cy="7" r="6" fill={c} />
          <path d="M4.2 7.2 L6.2 9.2 L9.8 5" fill="none" stroke="#0e0f11" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      );
    case "canceled":
      return (
        <svg className="status-icon" width="14" height="14" viewBox="0 0 14 14" aria-label="Canceled">
          <circle cx="7" cy="7" r="6" fill={c} />
          <path d="M4.8 4.8 L9.2 9.2 M9.2 4.8 L4.8 9.2" stroke="#0e0f11" strokeWidth="1.5" strokeLinecap="round" />
        </svg>
      );
  }
}

export function PriorityIcon({ priority }: { priority: Priority }) {
  if (priority === "urgent") {
    return (
      <svg className="priority-icon" width="14" height="14" viewBox="0 0 14 14" aria-label="Urgent">
        <rect x="1" y="1" width="12" height="12" rx="3" fill="#fc7840" />
        <path d="M7 3.6 V8" stroke="#0e0f11" strokeWidth="1.6" strokeLinecap="round" />
        <circle cx="7" cy="10.4" r="1" fill="#0e0f11" />
      </svg>
    );
  }
  const filled = priority === "low" ? 1 : priority === "medium" ? 2 : priority === "high" ? 3 : 0;
  if (filled === 0) {
    return (
      <svg className="priority-icon" width="14" height="14" viewBox="0 0 14 14" aria-label="No priority">
        <rect x="2" y="6.3" width="2" height="1.4" rx="0.7" fill="#62666d" />
        <rect x="6" y="6.3" width="2" height="1.4" rx="0.7" fill="#62666d" />
        <rect x="10" y="6.3" width="2" height="1.4" rx="0.7" fill="#62666d" />
      </svg>
    );
  }
  return (
    <svg className="priority-icon" width="14" height="14" viewBox="0 0 14 14" aria-label={priority}>
      <rect x="1.5" y="8" width="3" height="4.5" rx="1" fill={filled >= 1 ? "#8a8f98" : "#383b42"} />
      <rect x="5.5" y="5.5" width="3" height="7" rx="1" fill={filled >= 2 ? "#8a8f98" : "#383b42"} />
      <rect x="9.5" y="3" width="3" height="9.5" rx="1" fill={filled >= 3 ? "#8a8f98" : "#383b42"} />
    </svg>
  );
}

const AVATAR_COLORS = ["#5e6ad2", "#26b5ce", "#f2994a", "#bb87fc", "#4cb782", "#eb5757"];

export function Avatar({ name, title }: { name: string; title?: string }) {
  let hash = 0;
  for (let i = 0; i < name.length; i++) hash = (hash * 31 + name.charCodeAt(i)) | 0;
  const color = AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length];
  return (
    <span className="avatar" style={{ background: color }} title={title ?? name}>
      {name.charAt(0).toUpperCase()}
    </span>
  );
}

export function LabelChip({ name, labels }: { name: string; labels: Label[] }) {
  const color = labels.find((l) => l.name === name)?.color ?? "#8a8f98";
  return (
    <span className="label-chip">
      <span className="label-dot" style={{ background: color }} />
      {name}
    </span>
  );
}

const DAY_MS = 86_400_000;

/** Whole days from today (local) to a "YYYY-MM-DD" due date; negative = overdue. */
export function daysUntilDue(dueDate: string): number {
  const [y, m, d] = dueDate.split("-").map(Number);
  const now = new Date();
  const due = new Date(y, m - 1, d).getTime();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  return Math.round((due - today) / DAY_MS);
}

/** Short label for a "YYYY-MM-DD" due date, e.g. "Jun 24" (with year if not this year). */
export function formatDueDate(dueDate: string): string {
  const [y, m, d] = dueDate.split("-").map(Number);
  const opts: Intl.DateTimeFormatOptions = { month: "short", day: "numeric" };
  if (y !== new Date().getFullYear()) opts.year = "numeric";
  return new Date(y, m - 1, d).toLocaleDateString("en-US", opts);
}

export const isClosed = (status: Status) => status === "done" || status === "canceled";

export function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;
  return new Date(iso).toLocaleDateString();
}
