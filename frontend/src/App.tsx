import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { Issue, Label, Project, Status, User } from "./api";
import { getIssue, getIssues, getLabels, getProjects, getUsers, STATUS_LABELS } from "./api";
import { flattenVisible, groupIssues } from "./grouping";
import { CreateIssueModal } from "./components/CreateIssueModal";
import { CreateProjectModal } from "./components/CreateProjectModal";
import { IssueDetail } from "./components/IssueDetail";
import { IssueList } from "./components/IssueList";
import { McpIcon, McpModal } from "./components/McpModal";
import { ProjectHeader } from "./components/ProjectHeader";
import { Sidebar } from "./components/Sidebar";

export default function App() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [labels, setLabels] = useState<Label[]>([]);
  const [issues, setIssues] = useState<Issue[]>([]);
  const [loadError, setLoadError] = useState<string | null>(null);

  // Filters survive reloads (same pattern as the detail fullscreen pref).
  const [projectFilter, setProjectFilter] = useState<string | null>(
    () => localStorage.getItem("taskdock.filter-project")
  );
  const [statusFilter, setStatusFilter] = useState<Status | null>(
    () => localStorage.getItem("taskdock.filter-status") as Status | null
  );
  const [query, setQuery] = useState("");

  useEffect(() => {
    if (projectFilter) localStorage.setItem("taskdock.filter-project", projectFilter);
    else localStorage.removeItem("taskdock.filter-project");
  }, [projectFilter]);

  useEffect(() => {
    if (statusFilter) localStorage.setItem("taskdock.filter-status", statusFilter);
    else localStorage.removeItem("taskdock.filter-status");
  }, [statusFilter]);

  // Collapsed status groups survive reloads (same pattern as the filters above).
  const [collapsedStatuses, setCollapsedStatuses] = useState<Set<Status>>(() => {
    try {
      const raw = localStorage.getItem("taskdock.collapsed-statuses");
      return new Set(raw ? (JSON.parse(raw) as Status[]) : []);
    } catch {
      return new Set();
    }
  });

  useEffect(() => {
    localStorage.setItem("taskdock.collapsed-statuses", JSON.stringify([...collapsedStatuses]));
  }, [collapsedStatuses]);

  const toggleGroup = useCallback((status: Status) => {
    setCollapsedStatuses((prev) => {
      const next = new Set(prev);
      if (next.has(status)) next.delete(status);
      else next.add(status);
      return next;
    });
  }, []);

  const groups = useMemo(() => groupIssues(issues), [issues]);
  // What j/k actually walks: display order, collapsed groups skipped.
  const visibleIssues = useMemo(
    () => flattenVisible(groups, collapsedStatuses),
    [groups, collapsedStatuses]
  );

  const [selectedIndex, setSelectedIndex] = useState(0);

  // Reloads and collapses can shrink the visible list — keep the index in range.
  useEffect(() => {
    setSelectedIndex((i) => Math.min(i, Math.max(visibleIssues.length - 1, 0)));
  }, [visibleIssues.length]);
  const [openIssue, setOpenIssue] = useState<Issue | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [showCreateProject, setShowCreateProject] = useState(false);
  const [showMcp, setShowMcp] = useState(false);

  useEffect(() => {
    Promise.all([getProjects(), getUsers(), getLabels()])
      .then(([p, u, l]) => {
        setProjects(p);
        setUsers(u);
        setLabels(l);
        // The remembered project may have been deleted since last visit.
        setProjectFilter((key) => (key && !p.some((pr) => pr.key === key) ? null : key));
      })
      .catch((e) => setLoadError(`Failed to load workspace: ${(e as Error).message}`));
  }, []);

  const refreshProjects = useCallback(() => {
    getProjects()
      .then(setProjects)
      .catch((e) => setLoadError(`Failed to load projects: ${(e as Error).message}`));
  }, []);

  const loadIssues = useCallback(() => {
    getIssues({
      project: projectFilter ?? undefined,
      status: statusFilter ?? undefined,
      q: query.trim() || undefined,
    })
      .then((list) => {
        setIssues(list);
        setLoadError(null);
      })
      .catch((e) => setLoadError(`Failed to load issues: ${(e as Error).message}`));
  }, [projectFilter, statusFilter, query]);

  useEffect(() => {
    loadIssues();
  }, [loadIssues]);

  // Keyboard shortcuts — read latest state via ref to keep a single stable listener.
  const stateRef = useRef({ visibleIssues, selectedIndex, openIssue, showCreate, showCreateProject, showMcp });
  stateRef.current = { visibleIssues, selectedIndex, openIssue, showCreate, showCreateProject, showMcp };

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
        if (s.showMcp) setShowMcp(false);
        else if (s.showCreateProject) setShowCreateProject(false);
        else if (s.showCreate) setShowCreate(false);
        else if (s.openIssue) setOpenIssue(null);
        return;
      }
      if (typing || e.metaKey || e.ctrlKey || e.altKey) return;

      if (e.key === "c" && !s.showCreate && !s.showCreateProject && !s.showMcp) {
        e.preventDefault();
        setShowCreate(true);
        return;
      }
      if (s.showCreate || s.showCreateProject || s.showMcp || s.openIssue) return;

      if (e.key === "j") {
        e.preventDefault();
        setSelectedIndex((i) => Math.min(i + 1, Math.max(s.visibleIssues.length - 1, 0)));
      } else if (e.key === "k") {
        e.preventDefault();
        setSelectedIndex((i) => Math.max(i - 1, 0));
      } else if (e.key === "Enter") {
        const issue = s.visibleIssues[s.selectedIndex];
        if (issue) setOpenIssue(issue);
      }
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  const heading =
    (projectFilter ? projectFilter : "All issues") +
    (statusFilter ? ` · ${STATUS_LABELS[statusFilter]}` : "");

  const selectedProject = projectFilter
    ? projects.find((p) => p.key === projectFilter) ?? null
    : null;

  return (
    <div className="app">
      <Sidebar
        projects={projects}
        selectedProject={projectFilter}
        selectedStatus={statusFilter}
        onSelectProject={setProjectFilter}
        onSelectStatus={setStatusFilter}
        onNewIssue={() => setShowCreate(true)}
        onNewProject={() => setShowCreateProject(true)}
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
          <button
            className="btn btn-small mcp-btn"
            onClick={() => setShowMcp(true)}
            title="Connect via MCP"
          >
            <McpIcon /> MCP
          </button>
        </div>
        {loadError && <div className="error-bar">{loadError}</div>}
        {selectedProject && (
          <ProjectHeader
            project={selectedProject}
            issueCount={issues.length}
            onChanged={refreshProjects}
            onDeleted={() => {
              refreshProjects();
              setProjectFilter(null);
            }}
          />
        )}
        <IssueList
          groups={groups}
          collapsed={collapsedStatuses}
          onToggleGroup={toggleGroup}
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
          onOpenIssue={(key) =>
            getIssue(key)
              .then(setOpenIssue)
              .catch((e) => setLoadError(`Failed to open ${key}: ${(e as Error).message}`))
          }
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

      {showMcp && (
        <McpModal projectKey={projectFilter} projects={projects} onClose={() => setShowMcp(false)} />
      )}

      {showCreateProject && (
        <CreateProjectModal
          onClose={() => setShowCreateProject(false)}
          onCreated={(project) => {
            setShowCreateProject(false);
            refreshProjects();
            setProjectFilter(project.key);
          }}
        />
      )}
    </div>
  );
}
