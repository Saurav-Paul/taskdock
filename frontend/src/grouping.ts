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

/** A row in the list: an issue plus the subtasks nested under it.
 *  A subtask nests only when its parent shares the status group —
 *  otherwise it stays a top-level row in its own group. */
export interface IssueNode {
  issue: Issue;
  children: IssueNode[];
}

export interface IssueGroup {
  status: Status;
  nodes: IssueNode[];
  /** Total issues in the group (incl. nested subtasks) — the header count. */
  count: number;
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

/** What get_next_task would actually pick first: actionable work on top —
 *  blocked issues sink (you can't act on them), overdue/due-today float,
 *  then priority desc, then created_at desc. */
function byActionability(a: Issue, b: Issue): number {
  return (
    Number(isBlocked(a)) - Number(isBlocked(b)) ||
    Number(isDueNow(b)) - Number(isDueNow(a)) ||
    PRIORITY_RANK[b.priority] - PRIORITY_RANK[a.priority] ||
    new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  );
}

/** Nests subtasks under their parent within one status bucket. Issues whose
 *  parent is absent (different status, filtered out, or no parent) stay top
 *  level. Handles arbitrary depth; sibling order follows byActionability. */
function buildTree(bucket: Issue[]): IssueNode[] {
  const present = new Set(bucket.map((issue) => issue.key));
  const childrenOf = new Map<string, Issue[]>();
  const tops: Issue[] = [];

  for (const issue of bucket) {
    const parentKey = issue.parent?.key;
    if (parentKey && present.has(parentKey) && parentKey !== issue.key) {
      const siblings = childrenOf.get(parentKey) ?? [];
      siblings.push(issue);
      childrenOf.set(parentKey, siblings);
    } else {
      tops.push(issue);
    }
  }

  const toNode = (issue: Issue): IssueNode => ({
    issue,
    children: (childrenOf.get(issue.key) ?? []).map(toNode),
  });
  return tops.map(toNode);
}

/** Buckets issues by status in GROUP_ORDER (empty groups omitted), with
 *  subtasks nested under their parent when both land in the same group. */
export function groupIssues(issues: Issue[]): IssueGroup[] {
  return GROUP_ORDER.map((status) => {
    const bucket = issues.filter((issue) => issue.status === status).sort(byActionability);
    return { status, nodes: buildTree(bucket), count: bucket.length };
  }).filter((group) => group.count > 0);
}

/** Issues in display order, skipping collapsed groups and the subtrees of
 *  collapsed parents — what j/k navigates. */
export function flattenVisible(
  groups: IssueGroup[],
  collapsed: ReadonlySet<Status>,
  collapsedParents: ReadonlySet<string>
): Issue[] {
  const out: Issue[] = [];
  const walk = (node: IssueNode) => {
    out.push(node.issue);
    if (!collapsedParents.has(node.issue.key)) node.children.forEach(walk);
  };
  for (const group of groups) {
    if (!collapsed.has(group.status)) group.nodes.forEach(walk);
  }
  return out;
}
