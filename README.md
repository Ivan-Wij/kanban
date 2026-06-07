# Kanban

A multi-board kanban app built with Go, PostgreSQL, HTMX, and Tailwind CSS. Manage projects, stories, and tasks on drag-and-drop boards with filtering, search, and archiving.

## Features

- **Multiple boards** — create, search, and rename boards
- **Kanban columns** — To Do, In Progress, and Done per board
- **Card hierarchy** — projects contain stories; stories contain tasks
- **Drag and drop** — move cards between columns (SortableJS)
- **Card detail modal** — edit title, description, and status; manage child items
- **Project filter** — multi-select filter persisted per board in `localStorage`
- **Board search** — filter visible cards by title or description
- **Archiving** — archive done items, browse archived stories with pagination and search
- **Parent grouping** — related cards grouped in the same column

## Tech stack

| Layer | Technology |
|-------|------------|
| Backend | Go, Echo v5 |
| Database | PostgreSQL (pgx, sqlx) |
| Frontend | HTMX, Tailwind CSS (CDN), SortableJS |
| Templates | Go `html/template` (embedded) |

## Prerequisites

- Go 1.26+
- PostgreSQL

## Getting started

### 1. Configure the app

Copy the example config and set your database connection:

```bash
cp config/config.example.jsonc config/config.jsonc
```

Edit `config/config.jsonc`:

```jsonc
{
  "port": "8080",
  "database_url": "postgres://user:pass@localhost:5432/kanban?sslmode=disable"
}
```

You can override the config path with the `CONFIG_PATH` environment variable.

### 2. Create the database

```bash
createdb kanban
```

Migrations run automatically on startup from the `migrations/` directory. A seed board ("My Board") is created by `002_seed_board.sql` if it does not already exist.

### 3. Install dependencies and run

```bash
go mod download
go run .
```

Open [http://localhost:8080](http://localhost:8080) (or the port set in your config).

## Project structure

```
kanban/
├── main.go                 # Entry point
├── config/                 # App configuration (gitignored)
├── migrations/             # SQL migrations
├── internal/
│   ├── config/             # Config loading (JSONC)
│   ├── domain/             # Domain types and helpers
│   ├── handler/            # HTTP handlers and renderer
│   ├── migrate/            # Migration runner
│   ├── repository/         # PostgreSQL data access
│   └── usecase/            # Business logic
└── web/templates/          # HTML templates and partials
```

## Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Board list |
| GET | `/boards/new` | Create board modal |
| POST | `/boards` | Create board |
| GET | `/boards/:boardID` | Board view |
| PUT | `/boards/:boardID` | Update board name |
| GET | `/boards/:boardID/cards/new` | Create card modal |
| POST | `/boards/:boardID/cards` | Create card |
| GET | `/boards/:boardID/archived/stories` | Archived stories |
| GET | `/cards/:id/detail` | Card detail modal |
| PUT | `/cards/:id` | Update card |
| PUT | `/cards/:id/move` | Move card |
| PUT | `/cards/:id/status` | Change card status |
| DELETE | `/cards/:id` | Delete card |
| POST | `/columns/:id/archive-done` | Archive all done cards in column |

## Card types

| Type | Parent | Description |
|------|--------|-------------|
| Project | — | Top-level work item |
| Story | Project | Feature or user story |
| Task | Story | Concrete work item |

New cards are placed in the board's **To Do** column.

## License

Private / internal use.
