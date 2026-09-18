# Hostrix

**Your Infrastructure. Simplified.**

Hostrix is an open-source, self-hosted hosting panel for managing Linux workloads in LXC containers via [Incus](https://linuxcontainers.org/incus/). It is a modern, simplified alternative to Pterodactyl — without Docker/Wings and without any Hostrix SaaS dependency.

> Status: **Phase 6** — backups, advanced server permissions, multi-node polish.

## One-liner install (Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/HeatzyV2/hostrix/main/installer/install.sh | bash
```

## Phase 2 — Agent quick start

1. Create a Node in the panel (`/nodes`) and copy the one-time token.
2. On the node host, write `/etc/hostrix/agent.env`:

```bash
HOSTRIX_AGENT_ADDR=:8081
HOSTRIX_API_URL=http://<panel-ip>:8080
HOSTRIX_NODE_UUID=<node-uuid>
HOSTRIX_NODE_TOKEN=<token>
```

3. Enable the agent:

```bash
systemctl enable --now hostrix-agent
```

4. Confirm the node shows **ONLINE**, then create a server from `/servers`.
5. Open a server detail page for live metrics, console, backups, and access shares.

For the console WebSocket when the panel and API are on different origins, set:

```bash
NEXT_PUBLIC_HOSTRIX_API_URL=http://<api-host>:8080
```

Non-interactive:

```bash
HOSTRIX_NONINTERACTIVE=1 \
HOSTRIX_DB_PASSWORD='strong-db-pass' \
HOSTRIX_BOOTSTRAP_ADMIN_PASSWORD='strong-admin-pass' \
curl -fsSL https://raw.githubusercontent.com/HeatzyV2/hostrix/main/installer/install.sh | bash
```

Supported install targets: Ubuntu / Debian (amd64, arm64).

## Architecture

```
Browser → Hostrix Panel (Next.js)
              ↓
         Hostrix API (Go)
          ├── MariaDB
          └── Agent(s) → Incus → LXC
```

## Stack

| Layer | Tech |
|-------|------|
| Panel | Next.js, React, TypeScript, Tailwind |
| API | Go, REST, GORM, MariaDB |
| Runtime | Incus / LXC (Phase 2+) |

Explicitly **not** in scope: Kubernetes, Redis, RabbitMQ, NATS, Kafka, PostgreSQL, microservices.

## Repository layout

```
hostrix/
├── panel/        # Next.js UI
├── api/          # Go API
├── agent/        # Node agent
├── templates/    # YAML service templates
├── installer/    # Linux install / upgrade / systemd
├── docker/       # Local MariaDB for development
├── docs/
└── PLAN.md       # Full technical plan
```

## Development

### Prerequisites

- Go 1.22+
- Node.js 20+
- MariaDB 10.11+ / 11.x

### Database

```bash
# optional local DB
docker compose -f docker/docker-compose.yml up -d
```

Copy `.env.example` and export variables (or use a dotenv tool).

### API

```bash
cd api
go test ./...
go run ./cmd/hostrix-api
```

### Agent

```bash
cd agent
go test ./...
go run ./cmd/hostrix-agent
```

### Panel

```bash
cd panel
npm install
npm run dev
```

Open http://localhost:3000 — default bootstrap admin is created on first API start (`admin` / value of `HOSTRIX_BOOTSTRAP_ADMIN_PASSWORD`).

### Auth endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/login` | Create session cookie |
| POST | `/api/v1/auth/logout` | Destroy session |
| GET | `/api/v1/auth/me` | Current user |
| GET | `/api/v1/health` | Health check |

### Phase 6 endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET/POST | `/api/v1/servers/:id/backups` | List / create backups |
| GET/DELETE | `/api/v1/servers/:id/backups/:backupId` | Get / delete |
| POST | `/api/v1/servers/:id/backups/:backupId/restore` | Restore container from backup |
| GET | `/api/v1/servers/:id/backups/:backupId/download` | Download backup archive |
| GET | `/api/v1/backups` | All accessible backups |
| GET/POST | `/api/v1/servers/:id/permissions` | List / grant shares |
| DELETE | `/api/v1/servers/:id/permissions/:userId` | Revoke share |
| GET | `/api/v1/users` | Admin user list |

Server list/detail JSON includes `node_uuid`, `node_name`, and `node_status`. Creating a server on an **OFFLINE** node is refused.

## Phases

1. **Foundations** — API, panel, auth, MariaDB
2. Incus + Agent + container lifecycle
3. Servers UI, WebSocket console, metrics
4. File manager
5. Templates (Minecraft, Node.js, Python, …)
6. **Backups, multi-node, advanced permissions** *(current)*

See [PLAN.md](./PLAN.md) for details.

## Security notes

- Passwords are bcrypt-hashed; never stored in plaintext
- Sessions use opaque tokens stored as SHA-256 hashes
- Login is rate-limited per IP
- No arbitrary shell execution from user input (validated internal operations only)
- Server shares enforce `can_start` / `can_stop` / `can_files` / `can_console` on the API

## License

Apache-2.0 (planned) — see LICENSE when published.
