import type { Issue, Priority, Status } from "./api";

/** Linear-style display order — active work first, terminal states last. */
export const GROUP_ORDER: Status[] = [
  "in_review",
  "in_progress",
  "todo",
  "backlog",
  "done",
  "canceled",
];

const PRIORITY_RANK: Record<Priority, number> = {
  urgent: 4,
  high: 3,
  medium: 2,
  low: 1,
  none: 0,
};

export interface IssueGroup {
  status: Status;
  issues: Issue[];
}

/** An issue is blocked while any of its dependencies is still open. */
export function isBlocked(issue: Issue): boolean {
  return issue.depends_on.some((dep) => dep.status !== "done" && dep.status !== "canceled");
}

/** Due today or overdue — mirrors the server-side boost in get_next_task. */
function isDueNow(issue: Issue): boolean {
  if (!issue.due_date) return false;
  const today = new Date();
  const iso = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, "0")}-${String(today.getDate()).padStart(2, "0")}`;
  return issue.due_date <= iso;
}

/** Buckets issues by status in GROUP_ORDER (empty groups omitted).
 *
 *  Within a group, the order mirrors what get_next_task would actually
 *  pick: actionable work first — blocked issues sink to the bottom (you
 *  can't act on them), overdue/due-today float above the rest, then
 *  priority desc, then created_at desc. */
export function groupIssues(issues: Issue[]): IssueGroup[] {
  return GROUP_ORDER.map((status) => ({
    status,
    issues: issues
      .filter((issue) => issue.status === status)
      .sort(
        (a, b) =>
          Number(isBlocked(a)) - Number(isBlocked(b)) ||
          Number(isDueNow(b)) - Number(isDueNow(a)) ||
          PRIORITY_RANK[b.priority] - PRIORITY_RANK[a.priority] ||
          new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
      ),
  })).filter((group) => group.issues.length > 0);
}

/** Issues in display order, skipping collapsed groups — what j/k navigates. */
export function flattenVisible(groups: IssueGroup[], collapsed: ReadonlySet<Status>): Issue[] {
  return groups.flatMap((group) => (collapsed.has(group.status) ? [] : group.issues));
}
