import { useEffect, useMemo, useRef, useState } from "react";
import { fuzzyMatch } from "../lib/fuzzy";

export interface PaletteCommand {
  id: string;
  label: string;
  /** Hidden alias text also searched (e.g. "mark as in progress"). */
  keywords?: string;
  /** Right-aligned mono hint, e.g. the issue key the command targets. */
  hint?: string;
  run: () => void;
}

const RECENTS_KEY = "taskdock.recent-commands";
const MAX_RECENTS = 8;

function loadRecents(): string[] {
  try {
    const raw = JSON.parse(localStorage.getItem(RECENTS_KEY) ?? "[]") as unknown;
    return Array.isArray(raw) ? raw.filter((x): x is string => typeof x === "string") : [];
  } catch {
    return [];
  }
}

interface Props {
  commands: PaletteCommand[];
  /** Escape is handled by App's global chain; this covers click-away + run. */
  onClose: () => void;
}

export function CommandPalette({ commands, onClose }: Props) {
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState(0);
  // Snapshot once per open — reordering mid-session would be jarring.
  const [recents] = useState(loadRecents);
  const listRef = useRef<HTMLDivElement>(null);

  const results = useMemo(() => {
    const q = query.trim();
    if (!q) {
      // Recent commands float to the top; stable sort keeps the natural
      // order (issue-scoped first) for everything else.
      const rank = new Map(recents.map((id, i) => [id, i]));
      return [...commands].sort(
        (a, b) => (rank.get(a.id) ?? Infinity) - (rank.get(b.id) ?? Infinity)
      );
    }
    return commands
      .map((c) => ({ c, score: fuzzyMatch(q, c.label, c.keywords) }))
      .filter((r): r is { c: PaletteCommand; score: number } => r.score !== null)
      .sort((a, b) => b.score - a.score)
      .map((r) => r.c);
  }, [commands, query, recents]);

  // New query → start from the top; shrinking results → stay in range.
  useEffect(() => setSelected(0), [query]);
  useEffect(() => {
    setSelected((s) => Math.min(s, Math.max(results.length - 1, 0)));
  }, [results.length]);

  useEffect(() => {
    listRef.current?.children[selected]?.scrollIntoView({ block: "nearest" });
  }, [selected, results]);

  function run(cmd: PaletteCommand) {
    const next = [cmd.id, ...loadRecents().filter((id) => id !== cmd.id)].slice(0, MAX_RECENTS);
    localStorage.setItem(RECENTS_KEY, JSON.stringify(next));
    onClose();
    cmd.run();
  }

  function onKeyDown(e: React.KeyboardEvent) {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setSelected((s) => Math.min(s + 1, Math.max(results.length - 1, 0)));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelected((s) => Math.max(s - 1, 0));
    } else if (e.key === "Enter") {
      e.preventDefault();
      const cmd = results[selected];
      if (cmd) run(cmd);
    }
  }

  return (
    <div className="palette-overlay" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className="palette">
        <input
          className="palette-input"
          autoFocus
          placeholder="Type a command or search…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={onKeyDown}
        />
        <div className="palette-list" ref={listRef}>
          {results.length === 0 && <div className="palette-empty muted">No matching commands</div>}
          {results.map((c, i) => (
            <div
              key={c.id}
              className={`palette-item ${i === selected ? "selected" : ""}`}
              onMouseEnter={() => setSelected(i)}
              onMouseDown={(e) => e.preventDefault() /* keep input focus */}
              onClick={() => run(c)}
            >
              <span className="palette-label">{c.label}</span>
              {c.hint && <span className="palette-hint">{c.hint}</span>}
            </div>
          ))}
        </div>
        <div className="palette-footer muted">↑↓ navigate · ↵ run · esc close</div>
      </div>
    </div>
  );
}
