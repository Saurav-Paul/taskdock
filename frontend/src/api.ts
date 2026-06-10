export type Status = "backlog" | "todo" | "in_progress" | "done" | "canceled";
export type Priority = "none" | "low" | "medium" | "high" | "urgent";

export const STATUSES: Status[] = ["backlog", "todo", "in_progress", "done", "canceled"];
export const PRIORITIES: Priority[] = ["none", "low", "medium", "high", "urgent"];

export const STATUS_LABELS: Record<Status, string> = {
  backlog: "Backlog",
  todo: "Todo",
  in_progress: "In Progress",
  done: "Done",
  canceled: "Canceled",
};

export const PRIORITY_LABELS: Record<Priority, string> = {
  none: "No priority",
  low: "Low",
  medium: "Medium",
  high: "High",
  urgent: "Urgent",
};

export interface Project {
  id: number;
  name: string;
  key: string;
  description: string;
  created_at: string;
}

export interface User {
  id: number;
  name: string;
  display_name: string;
}

export interface Label {
  id: number;
  name: string;
  color: string;
}

export interface Issue {
  id: number;
  key: string;
  project: string;
  title: string;
  description: string;
  status: Status;
  priority: Priority;
  assignee?: string;
  labels: string[];
  created_at: string;
  updated_at: string;
}

export interface Comment {
  id: number;
  author: string;
  body: string;
  created_at: string;
}

export interface IssueFilter {
  project?: string;
  status?: Status;
  assignee?: string;
  q?: string;
}

export interface IssuePatch {
  title?: string;
  description?: string;
  status?: Status;
  priority?: Priority;
  assignee?: string;
  labels?: string[];
}

export interface NewIssue extends IssuePatch {
  project: string;
  title: string;
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(`${res.status} ${res.statusText}${text ? `: ${text}` : ""}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

export const getProjects = () => request<Project[]>("/api/projects");
export const getUsers = () => request<User[]>("/api/users");
export const getLabels = () => request<Label[]>("/api/labels");

export function getIssues(filter: IssueFilter = {}): Promise<Issue[]> {
  const params = new URLSearchParams();
  if (filter.project) params.set("project", filter.project);
  if (filter.status) params.set("status", filter.status);
  if (filter.assignee) params.set("assignee", filter.assignee);
  if (filter.q) params.set("q", filter.q);
  const qs = params.toString();
  return request<Issue[]>(`/api/issues${qs ? `?${qs}` : ""}`);
}

export const createIssue = (body: NewIssue) =>
  request<Issue>("/api/issues", { method: "POST", body: JSON.stringify(body) });

export const updateIssue = (key: string, patch: IssuePatch) =>
  request<Issue>(`/api/issues/${key}`, { method: "PATCH", body: JSON.stringify(patch) });

export const deleteIssue = (key: string) =>
  request<void>(`/api/issues/${key}`, { method: "DELETE" });

export const getComments = (key: string) =>
  request<Comment[]>(`/api/issues/${key}/comments`);

export const addComment = (key: string, author: string, body: string) =>
  request<Comment>(`/api/issues/${key}/comments`, {
    method: "POST",
    body: JSON.stringify({ author, body }),
  });
