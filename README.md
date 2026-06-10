# taskdock

Minimal self-hosted Linear alternative — a personal issue tracker with a
first-class MCP server so Claude / Claude Code can manage tickets directly.

Single static Go binary (~30MB RAM idle), SQLite storage, React + Tiptap UI.

## Features

- **Projects** with Linear-style issue keys (`TD-12`)
- **Issues** — markdown descriptions, status, priority, assignee, labels
- **Comments** — markdown thread per issue; Claude logs progress here
- **MCP server** at `POST /mcp` (JSON-RPC 2.0, streamable HTTP) with 7 tools,
  including `taskdock_get_next_task`: the highest-priority open issue
  assigned to `claude` — so a Claude Code session can ask "what's next?"
- **No auth** — users are simple rows (`saurav`, `claude`); run it on
  localhost or your LAN

## Run

```bash
docker compose up --build -d     # http://localhost:8860
```

Dev mode (hot reload):

```bash
go run ./cmd/server              # backend :8860
cd frontend && npm run dev       # frontend :8861, proxies /api + /mcp
```

## Connect Claude Code

```bash
claude mcp add --transport http taskdock http://localhost:8860/mcp
```

Or grab the config snippet from `GET /mcp/config.json`.

### MCP tools

| Tool | Purpose |
|---|---|
| `taskdock_list_projects` | List projects |
| `taskdock_list_issues` | Filter by project / status / assignee / query |
| `taskdock_get_issue` | Full issue + comments as markdown |
| `taskdock_save_issue` | Create (no key) or update (with key) |
| `taskdock_delete_issue` | Delete by key |
| `taskdock_save_comment` | Log progress on a ticket (default author: claude) |
| `taskdock_get_next_task` | Highest-priority open issue for an assignee |

## API

REST under `/api`: `projects`, `users`, `labels`, `issues` (+
`issues/:key/comments`). Issues are addressed by key (`TD-12`), support
partial PATCH updates, and filter via query params (`?project=TD&status=todo&assignee=claude&q=text`).

## Stack

Go (Echo + GORM + SQLite, Goose migrations embedded in the binary) ·
React + Vite + Tiptap · Docker scratch image.
