# Hostrix

**Your Infrastructure. Simplified.**

Hostrix is an open-source, self-hosted hosting panel for managing Linux workloads in LXC containers via [Incus](https://linuxcontainers.org/incus/). It is a modern, simplified alternative to Pterodactyl — without Docker/Wings and without any Hostrix SaaS dependency.

> Status: **Phase 3** — console WebSocket, live Incus metrics, server detail UI.

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
5. Open a server detail page for live metrics and an interactive console.

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
├── agent/        # Node agent (Phase 2)
├── templates/    # YAML service templates
├── installer/    # Linux install / upgrade / systemd
├── docker/       # Local MariaDB for development
├── docs/
└── PLAN.md       # Full technical plan
```

## Development (Phase 1)

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

## Phases

1. **Foundations** — API, panel, auth, MariaDB *(current)*
2. Incus + Agent + container lifecycle
3. Servers UI, WebSocket console, metrics
4. File manager
5. Templates (Minecraft, Node.js, Python, …)
6. Backups, multi-node, advanced permissions

See [PLAN.md](./PLAN.md) for details.

## Security notes

- Passwords are bcrypt-hashed; never stored in plaintext
- Sessions use opaque tokens stored as SHA-256 hashes
- Login is rate-limited per IP
- No arbitrary shell execution from user input (validated internal operations only)

## License

Apache-2.0 (planned) — see LICENSE when published.
