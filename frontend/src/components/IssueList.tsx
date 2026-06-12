import { useEffect, useMemo, useRef, type ReactNode } from "react";
import type { Issue, Label, Status, User } from "../api";
import { STATUS_LABELS } from "../api";
import type { IssueGroup, IssueNode } from "../grouping";
import { Avatar, daysUntilDue, formatDueDate, isClosed, LabelChip, PriorityIcon, StatusIcon } from "./bits";

interface Props {
  groups: IssueGroup[];
  collapsed: ReadonlySet<Status>;
  onToggleGroup: (status: Status) => void;
  /** Issue keys whose nested subtasks are hidden. */
  collapsedParents: ReadonlySet<string>;
  onToggleParent: (key: string) => void;
  labels: Label[];
  users: User[];
  /** Index into the visible (non-collapsed) issues, in display order. */
  selectedIndex: number;
  onSelect: (index: number) => void;
  onOpen: (issue: Issue) => void;
}

function DueChip({ dueDate }: { dueDate: string }) {
  const days = daysUntilDue(dueDate);
  const cls = days < 0 ? "overdue" : days <= 3 ? "soon" : "";
  const text =
    days < 0 ? "overdue" : days === 0 ? "today" : days <= 3 ? `${days}d` : formatDueDate(dueDate);
  return (
    <span className={`due-chip ${cls}`} title={`Due ${dueDate}`}>
      {text}
    </span>
  );
}

function Chevron({ collapsed }: { collapsed: boolean }) {
  return (
    <svg
      className={`group-chevron ${collapsed ? "collapsed" : ""}`}
      width="12"
      height="12"
      viewBox="0 0 12 12"
      aria-hidden="true"
    >
      <path
        d="M3.5 1.5 L8 6 L3.5 10.5"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

export function IssueList({
  groups,
  collapsed,
  onToggleGroup,
  collapsedParents,
  onToggleParent,
  labels,
  users,
  selectedIndex,
  onSelect,
  onOpen,
}: Props) {
  const listRef = useRef<HTMLDivElement>(null);
  const userByName = useMemo(() => new Map(users.map((u) => [u.name, u])), [users]);

  // Keep the keyboard-selected row in view.
  useEffect(() => {
    const el = listRef.current?.querySelector<HTMLElement>(`[data-index="${selectedIndex}"]`);
    el?.scrollIntoView({ block: "nearest" });
  }, [selectedIndex]);

  if (groups.length === 0) {
    return <div className="empty-state">No issues found. Press <kbd>c</kbd> to create one.</div>;
  }

  // Running index over visible rows only — must match App's visibleIssues order.
  let visibleIndex = 0;

  function renderNode(node: IssueNode, depth: number): ReactNode {
    const issue = node.issue;
    const i = visibleIndex++;
    const parentCollapsed = collapsedParents.has(issue.key);
    // Flag actionable work assigned to a runner that isn't heartbeating.
    const assignedUser = issue.assignee ? userByName.get(issue.assignee) : undefined;
    const runnerOffline =
      assignedUser?.kind === "runner" &&
      !assignedUser.online &&
      (issue.status === "todo" || issue.status === "in_progress");
    return (
      <div key={issue.key}>
        <div
          data-index={i}
          className={`issue-row ${i === selectedIndex ? "selected" : ""}`}
          style={depth > 0 ? { paddingLeft: 20 + depth * 24 } : undefined}
          onMouseEnter={() => onSelect(i)}
          onClick={() => onOpen(issue)}
        >
          <span className="row-gutter">
            {node.children.length > 0 ? (
              <button
                className="sub-toggle"
                onClick={(e) => {
                  e.stopPropagation();
                  onToggleParent(issue.key);
                }}
                title={parentCollapsed ? "Show subtasks" : "Hide subtasks"}
                aria-expanded={!parentCollapsed}
              >
                <Chevron collapsed={parentCollapsed} />
              </button>
            ) : depth > 0 ? (
              <span className="sub-tree-mark" aria-hidden="true" />
            ) : null}
          </span>
          <PriorityIcon priority={issue.priority} />
          <span className="issue-key">{issue.key}</span>
          <StatusIcon status={issue.status} />
          <span className="issue-title">{issue.title}</span>
          {issue.depends_on.some((d) => d.status !== "done" && d.status !== "canceled") && (
            <span
              className="blocked-chip"
              title={`Blocked by ${issue.depends_on.map((d) => d.key).join(", ")}`}
            >
              ⊘ blocked
            </span>
          )}
          {runnerOffline && (
            <span
              className="runner-offline-chip"
              title={`Runner ${issue.assignee} is offline`}
            >
              ○ runner offline
            </span>
          )}
          {issue.subtasks.length > 0 && (
            <span className="sub-count-chip" title={`${issue.subtasks.length} subtasks`}>
              {issue.subtasks.length} sub
            </span>
          )}
          <span className="issue-labels">
            {issue.labels.map((name) => (
              <LabelChip key={name} name={name} labels={labels} />
            ))}
          </span>
          {issue.due_date && !isClosed(issue.status) && <DueChip dueDate={issue.due_date} />}
          {issue.assignee ? (
            <Avatar name={issue.assignee} />
          ) : (
            <span className="avatar avatar-empty" title="Unassigned" />
          )}
        </div>
        {!parentCollapsed && node.children.map((child) => renderNode(child, depth + 1))}
      </div>
    );
  }

  return (
    <div className="issue-list" ref={listRef}>
      {groups.map((group) => {
        const isCollapsed = collapsed.has(group.status);
        return (
          <div key={group.status} className="issue-group">
            <div
              className="issue-group-header"
              onClick={() => onToggleGroup(group.status)}
              role="button"
              aria-expanded={!isCollapsed}
            >
              <Chevron collapsed={isCollapsed} />
              <StatusIcon status={group.status} />
              <span className="issue-group-label">{STATUS_LABELS[group.status]}</span>
              <span className="issue-group-count muted">{group.count}</span>
            </div>
            {!isCollapsed && group.nodes.map((node) => renderNode(node, 0))}
          </div>
        );
      })}
    </div>
  );
}
