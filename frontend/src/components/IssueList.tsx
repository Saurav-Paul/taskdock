import { useEffect, useRef } from "react";
import type { Issue, Label } from "../api";
import { Avatar, LabelChip, PriorityIcon, StatusIcon } from "./bits";

interface Props {
  issues: Issue[];
  labels: Label[];
  selectedIndex: number;
  onSelect: (index: number) => void;
  onOpen: (issue: Issue) => void;
}

export function IssueList({ issues, labels, selectedIndex, onSelect, onOpen }: Props) {
  const listRef = useRef<HTMLDivElement>(null);

  // Keep the keyboard-selected row in view.
  useEffect(() => {
    const el = listRef.current?.querySelector<HTMLElement>(`[data-index="${selectedIndex}"]`);
    el?.scrollIntoView({ block: "nearest" });
  }, [selectedIndex]);

  if (issues.length === 0) {
    return <div className="empty-state">No issues found. Press <kbd>c</kbd> to create one.</div>;
  }

  return (
    <div className="issue-list" ref={listRef}>
      {issues.map((issue, i) => (
        <div
          key={issue.key}
          data-index={i}
          className={`issue-row ${i === selectedIndex ? "selected" : ""}`}
          onMouseEnter={() => onSelect(i)}
          onClick={() => onOpen(issue)}
        >
          <PriorityIcon priority={issue.priority} />
          <span className="issue-key">{issue.key}</span>
          <StatusIcon status={issue.status} />
          <span className="issue-title">{issue.title}</span>
          {issue.depends_on.some((d) => d.status !== "done" && d.status !== "canceled") && (
            <span className="blocked-chip" title={`Blocked by ${issue.depends_on.map((d) => d.key).join(", ")}`}>
              ⊘ blocked
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
          {issue.assignee ? (
            <Avatar name={issue.assignee} />
          ) : (
            <span className="avatar avatar-empty" title="Unassigned" />
          )}
        </div>
      ))}
    </div>
  );
}
