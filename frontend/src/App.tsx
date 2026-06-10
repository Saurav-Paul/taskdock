import { useCallback, useEffect, useRef, useState } from "react";
import type { Issue, Label, Project, Status, User } from "./api";
import { getIssues, getLabels, getProjects, getUsers, STATUS_LABELS } from "./api";
import { CreateIssueModal } from "./components/CreateIssueModal";
import { IssueDetail } from "./components/IssueDetail";
import { IssueList } from "./components/IssueList";
import { Sidebar } from "./components/Sidebar";

export default function App() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [labels, setLabels] = useState<Label[]>([]);
  const [issues, setIssues] = useState<Issue[]>([]);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [projectFilter, setProjectFilter] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<Status | null>(null);
  const [query, setQuery] = useState("");

  const [selectedIndex, setSelectedIndex] = useState(0);
  const [openIssue, setOpenIssue] = useState<Issue | null>(null);
  const [showCreate, setShowCreate] = useState(false);

  useEffect(() => {
    Promise.all([getProjects(), getUsers(), getLabels()])
      .then(([p, u, l]) => {
        setProjects(p);
        setUsers(u);
        setLabels(l);
      })
      .catch((e) => setLoadError(`Failed to load workspace: ${(e as Error).message}`));
  }, []);

  const loadIssues = useCallback(() => {
    getIssues({
      project: projectFilter ?? undefined,
      status: statusFilter ?? undefined,
      q: query.trim() || undefined,
    })
      .then((list) => {
        setIssues(list);
        setSelectedIndex((i) => Math.min(i, Math.max(list.length - 1, 0)));
        setLoadError(null);
      })
      .catch((e) => setLoadError(`Failed to load issues: ${(e as Error).message}`));
  }, [projectFilter, statusFilter, query]);

  useEffect(() => {
    loadIssues();
  }, [loadIssues]);

  // Keyboard shortcuts — read latest state via ref to keep a single stable listener.
  const stateRef = useRef({ issues, selectedIndex, openIssue, showCreate });
  stateRef.current = { issues, selectedIndex, openIssue, showCreate };

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      const s = stateRef.current;
      const target = e.target as HTMLElement;
      const typing =
        target.tagName === "INPUT" ||
        target.tagName === "TEXTAREA" ||
        target.tagName === "SELECT" ||
        target.isContentEditable;

      if (e.key === "Escape") {
        if (s.showCreate) setShowCreate(false);
        else if (s.openIssue) setOpenIssue(null);
        return;
      }
      if (typing || e.metaKey || e.ctrlKey || e.altKey) return;

      if (e.key === "c" && !s.showCreate) {
        e.preventDefault();
        setShowCreate(true);
        return;
      }
      if (s.showCreate || s.openIssue) return;

      if (e.key === "j") {
        e.preventDefault();
        setSelectedIndex((i) => Math.min(i + 1, Math.max(s.issues.length - 1, 0)));
      } else if (e.key === "k") {
        e.preventDefault();
        setSelectedIndex((i) => Math.max(i - 1, 0));
      } else if (e.key === "Enter") {
        const issue = s.issues[s.selectedIndex];
        if (issue) setOpenIssue(issue);
      }
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  const heading =
    (projectFilter ? projectFilter : "All issues") +
    (statusFilter ? ` · ${STATUS_LABELS[statusFilter]}` : "");

  return (
    <div className="app">
      <Sidebar
        projects={projects}
        selectedProject={projectFilter}
        selectedStatus={statusFilter}
        onSelectProject={setProjectFilter}
        onSelectStatus={setStatusFilter}
        onNewIssue={() => setShowCreate(true)}
      />
      <main className="main">
        <div className="main-header">
          <h2 className="main-heading">{heading}</h2>
          <input
            className="search-input"
            placeholder="Search issues…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <span className="issue-count muted">
            {issues.length} issue{issues.length === 1 ? "" : "s"}
          </span>
        </div>
        {loadError && <div className="error-bar">{loadError}</div>}
        <IssueList
          issues={issues}
          labels={labels}
          selectedIndex={selectedIndex}
          onSelect={setSelectedIndex}
          onOpen={setOpenIssue}
        />
      </main>

      {openIssue && (
        <IssueDetail
          issue={openIssue}
          users={users}
          labels={labels}
          onClose={() => setOpenIssue(null)}
          onChanged={loadIssues}
          onDeleted={() => {
            setOpenIssue(null);
            loadIssues();
          }}
        />
      )}

      {showCreate && (
        <CreateIssueModal
          projects={projects}
          users={users}
          labels={labels}
          defaultProject={projectFilter}
          onClose={() => setShowCreate(false)}
          onCreated={() => {
            setShowCreate(false);
            loadIssues();
          }}
        />
      )}
    </div>
  );
}
