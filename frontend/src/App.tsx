import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { Issue, IssuePatch, Label, Project, Status, User } from "./api";
import {
  getIssue,
  getIssues,
  getLabels,
  getProjects,
  getUsers,
  PRIORITIES,
  PRIORITY_LABELS,
  STATUSES,
  STATUS_LABELS,
  updateIssue,
} from "./api";
import { flattenVisible, groupIssues } from "./grouping";
import { CommandPalette, type PaletteCommand } from "./components/CommandPalette";
import { CreateIssueModal } from "./components/CreateIssueModal";
import { CreateProjectModal } from "./components/CreateProjectModal";
import { IssueDetail } from "./components/IssueDetail";
import { IssueList } from "./components/IssueList";
import { McpIcon, McpModal } from "./components/McpModal";
import { ProjectHeader } from "./components/ProjectHeader";
import { Sidebar } from "./components/Sidebar";

/** "/PRO-967" → "PRO-967"; anything else → null. */
function parseIssuePath(pathname: string): string | null {
  const m = /^\/([A-Za-z]+-\d+)\/?$/.exec(pathname);
  return m ? m[1].toUpperCase() : null;
}

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

  // ── Issue deep-linking ──────────────────────────────
  // The key (if any) the page was loaded on; cleared once that fetch settles.
  // While set, the URL-sync effect below must not clobber the deep link.
  const initialKeyRef = useRef<string | null>(parseIssuePath(location.pathname));

  useEffect(() => {
    const key = initialKeyRef.current;
    if (!key) return;
    getIssue(key)
      .then((issue) => {
        // Canonicalize in place (e.g. /pro-967 → /PRO-967) without a new entry.
        history.replaceState(null, "", `/${issue.key}`);
        setOpenIssue(issue);
      })
      .catch((e) => {
        setLoadError(`Failed to open ${key}: ${(e as Error).message}`);
        history.replaceState(null, "", "/");
      })
      .finally(() => {
        initialKeyRef.current = null;
      });
  }, []);

  // Single source of truth for the URL: whatever issue is open. Every open
  // path (list click, Enter, palette, parent/subtask chips) funnels through
  // setOpenIssue, so this one effect covers them all. Pushing only when the
  // path differs means popstate-driven updates (where the URL already
  // changed) never push again — no loops, no double entries.
  const openKey = openIssue?.key ?? null;
  useEffect(() => {
    if (initialKeyRef.current) return; // deep-link fetch still pending
    const path = openKey ? `/${openKey}` : "/";
    if (location.pathname !== path) history.pushState(null, "", path);
  }, [openKey]);

  // Back/forward: the URL changed under us — make the UI follow.
  useEffect(() => {
    function onPopState() {
      const key = parseIssuePath(location.pathname);
      if (!key) {
        setOpenIssue(null);
        return;
      }
      getIssue(key)
        .then(setOpenIssue)
        .catch((e) => setLoadError(`Failed to open ${key}: ${(e as Error).message}`));
    }
    window.addEventListener("popstate", onPopState);
    return () => window.removeEventListener("popstate", onPopState);
  }, []);

  const [showCreate, setShowCreate] = useState(false);
  const [showCreateProject, setShowCreateProject] = useState(false);
  const [showMcp, setShowMcp] = useState(false);
  const [showPalette, setShowPalette] = useState(false);

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

  const loadIssues = useCallback(
    () =>
      getIssues({
        project: projectFilter ?? undefined,
        status: statusFilter ?? undefined,
        q: query.trim() || undefined,
      })
        .then((list) => {
          setIssues(list);
          setLoadError(null);
        })
        .catch((e) => setLoadError(`Failed to load issues: ${(e as Error).message}`)),
    [projectFilter, statusFilter, query]
  );

  useEffect(() => {
    void loadIssues();
  }, [loadIssues]);

  // ── Refresh ─────────────────────────────────────────
  const [refreshing, setRefreshing] = useState(false);
  const openIssueRef = useRef(openIssue);
  openIssueRef.current = openIssue;

  const refreshAll = useCallback(() => {
    setRefreshing(true);
    const meta = Promise.all([getProjects(), getUsers(), getLabels()])
      .then(([p, u, l]) => {
        setProjects(p);
        setUsers(u);
        setLabels(l);
      })
      .catch((e) => setLoadError(`Failed to refresh workspace: ${(e as Error).message}`));
    const open = openIssueRef.current
      ? getIssue(openIssueRef.current.key)
          .then((issue) =>
            // Only apply if that issue is still the one open.
            setOpenIssue((cur) => (cur && cur.key === issue.key ? issue : cur))
          )
          .catch((e) => setLoadError(`Failed to refresh issue: ${(e as Error).message}`))
      : Promise.resolve();
    void Promise.allSettled([loadIssues(), meta, open]).then(() => setRefreshing(false));
  }, [loadIssues]);

  const refreshRef = useRef(refreshAll);
  refreshRef.current = refreshAll;

  // Issue the palette acts on: the open detail wins, else the list selection.
  const contextIssue = openIssue ?? visibleIssues[selectedIndex] ?? null;

  // Patch + refresh, shared by the palette's status/priority/assign commands.
  const patchIssue = useCallback(
    (key: string, patch: IssuePatch) => {
      updateIssue(key, patch)
        .then((updated) => {
          loadIssues();
          setOpenIssue((cur) => (cur && cur.key === key ? updated : cur));
        })
        .catch((e) => setLoadError(`Failed to update ${key}: ${(e as Error).message}`));
    },
    [loadIssues]
  );

  const copyText = useCallback((text: string) => {
    navigator.clipboard
      .writeText(text)
      .catch((e) => setLoadError(`Failed to copy: ${(e as Error).message}`));
  }, []);

  const paletteCommands = useMemo<PaletteCommand[]>(() => {
    const cmds: PaletteCommand[] = [];
    const issue = contextIssue;
    if (issue) {
      // Issue-scoped commands come first; "mark as …" aliases make
      // Linear-style queries like "mip" hit "Set status: In Progress".
      for (const st of STATUSES)
        cmds.push({
          id: `status:${st}`,
          label: `Set status: ${STATUS_LABELS[st]}`,
          keywords: `mark as ${STATUS_LABELS[st]} move`,
          hint: issue.key,
          run: () => patchIssue(issue.key, { status: st }),
        });
      for (const pr of PRIORITIES)
        cmds.push({
          id: `priority:${pr}`,
          label: `Set priority: ${PRIORITY_LABELS[pr]}`,
          keywords: `make priority ${PRIORITY_LABELS[pr]}`,
          hint: issue.key,
          run: () => patchIssue(issue.key, { priority: pr }),
        });
      for (const u of users)
        cmds.push({
          id: `assign:${u.name}`,
          label: `Assign to: ${u.display_name}`,
          keywords: `assignee ${u.name}`,
          hint: issue.key,
          run: () => patchIssue(issue.key, { assignee: u.name }),
        });
      cmds.push({
        id: "copy-key",
        label: "Copy issue key",
        hint: issue.key,
        run: () => copyText(issue.key),
      });
      cmds.push({
        id: "copy-branch",
        label: "Copy branch name",
        keywords: "git checkout",
        hint: issue.key,
        run: () => copyText(issue.branch),
      });
    }
    cmds.push({ id: "new-issue", label: "Create new issue…", keywords: "add", run: () => setShowCreate(true) });
    for (const p of projects)
      cmds.push({
        id: `project:${p.key}`,
        label: `Go to project: ${p.name}`,
        keywords: `navigate ${p.key}`,
        hint: p.key,
        run: () => setProjectFilter(p.key),
      });
    cmds.push({
      id: "project:all",
      label: "Go to All issues",
      keywords: "navigate everything",
      run: () => setProjectFilter(null),
    });
    // Issues join the same list (core requirement: one fuzzy list).
    for (const i of issues)
      cmds.push({ id: `open:${i.key}`, label: i.title, keywords: i.key, hint: i.key, run: () => setOpenIssue(i) });
    return cmds;
  }, [contextIssue, users, projects, issues, patchIssue, copyText]);

  // Keyboard shortcuts — read latest state via ref to keep a single stable listener.
  const stateRef = useRef({ visibleIssues, selectedIndex, openIssue, showCreate, showCreateProject, showMcp, showPalette, refreshing });
  stateRef.current = { visibleIssues, selectedIndex, openIssue, showCreate, showCreateProject, showMcp, showPalette, refreshing };

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      const s = stateRef.current;
      const target = e.target as HTMLElement;
      const typing =
        target.tagName === "INPUT" ||
        target.tagName === "TEXTAREA" ||
        target.tagName === "SELECT" ||
        target.isContentEditable;

      // cmd+k / ctrl+k toggles the palette. Never hijack the browser (or
      // Tiptap) default while typing — except inside the palette itself.
      if ((e.metaKey || e.ctrlKey) && !e.altKey && !e.shiftKey && e.key.toLowerCase() === "k") {
        if (s.showPalette) {
          e.preventDefault();
          setShowPalette(false);
        } else if (!typing) {
          e.preventDefault();
          setShowPalette(true);
        }
        return;
      }

      if (e.key === "Escape") {
        if (s.showPalette) setShowPalette(false);
        else if (s.showMcp) setShowMcp(false);
        else if (s.showCreateProject) setShowCreateProject(false);
        else if (s.showCreate) setShowCreate(false);
        else if (s.openIssue) setOpenIssue(null);
        return;
      }
      if (typing || e.metaKey || e.ctrlKey || e.altKey) return;

      if (e.key === "c" && !s.showPalette && !s.showCreate && !s.showCreateProject && !s.showMcp) {
        e.preventDefault();
        setShowCreate(true);
        return;
      }
      // Refresh works with the detail open too (it re-fetches the open issue).
      if (e.key === "r" && !s.showPalette && !s.showCreate && !s.showCreateProject && !s.showMcp) {
        e.preventDefault();
        if (!s.refreshing) refreshRef.current();
        return;
      }
      if (s.showPalette || s.showCreate || s.showCreateProject || s.showMcp || s.openIssue) return;

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
            className="btn btn-small btn-icon refresh-btn"
            onClick={refreshAll}
            disabled={refreshing}
            title="Refresh (r)"
          >
            <RefreshIcon spinning={refreshing} />
          </button>
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

      {showPalette && (
        <CommandPalette commands={paletteCommands} onClose={() => setShowPalette(false)} />
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

function RefreshIcon({ spinning }: { spinning: boolean }) {
  return (
    <svg
      width="14"
      height="14"
      viewBox="0 0 14 14"
      aria-label="Refresh"
      className={spinning ? "spinning" : undefined}
    >
      <path
        d="M11.8 7 A4.8 4.8 0 1 1 9.8 3.1 M9.8 1 V3.4 H7.4"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.3"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
