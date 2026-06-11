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

/** Buckets issues by status in GROUP_ORDER (empty groups omitted),
 *  sorting each group by priority desc, then created_at desc. */
export function groupIssues(issues: Issue[]): IssueGroup[] {
  return GROUP_ORDER.map((status) => ({
    status,
    issues: issues
      .filter((issue) => issue.status === status)
      .sort(
        (a, b) =>
          PRIORITY_RANK[b.priority] - PRIORITY_RANK[a.priority] ||
          new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
      ),
  })).filter((group) => group.issues.length > 0);
}

/** Issues in display order, skipping collapsed groups — what j/k navigates. */
export function flattenVisible(groups: IssueGroup[], collapsed: ReadonlySet<Status>): Issue[] {
  return groups.flatMap((group) => (collapsed.has(group.status) ? [] : group.issues));
}
