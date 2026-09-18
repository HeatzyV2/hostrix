# Hostrix — Plan technique

**Slogan:** Your Infrastructure. Simplified.

Hostrix est un panel d’hébergement open-source, entièrement auto-hébergé, permettant de gérer des workloads Linux isolés dans des containers LXC via Incus. Alternative moderne et simplifiée à Pterodactyl (Docker/Wings).

---

## 1. Architecture

```
Browser
   │
   ▼
Hostrix Panel (Next.js)
   │
   ▼
Hostrix API (Go, REST + WebSocket)
   │
   ├── MariaDB
   │
   └── Hostrix Agent(s) ──► Incus ──► LXC containers
```

### Modes de déploiement

| Mode | Description |
|------|-------------|
| **Mono-node** | Panel + API + Agent + MariaDB + Incus sur la même machine |
| **Multi-node** | Panel/API/MariaDB centraux ; un Agent par Node distant |

Le backend central reste toujours auto-hébergé par l’utilisateur. Aucun SaaS Hostrix.

### Composants

| Composant | Rôle |
|-----------|------|
| `panel/` | UI Next.js (dashboard, servers, admin) |
| `api/` | API Go REST + WebSocket, auth, permissions, orchestration |
| `agent/` | Daemon Node : Incus, console, fichiers, métriques |
| `templates/` | Définitions YAML des stacks (Minecraft, Node.js, …) |
| `installer/` | Scripts d’installation Linux one-liner |
| `docker/` | Uniquement pour développement local (MariaDB) |

---

## 2. Décisions techniques

| Décision | Choix | Justification |
|----------|-------|---------------|
| Base de données | **MariaDB** + GORM | Exigence produit ; ORM pour parameterization SQL |
| Conteneurs | **Incus / LXC** | Différence clé vs Pterodactyl/Docker |
| Auth sessions | Cookies HTTP-only + sessions en DB | Simple, compatible SSR/panel |
| Password hashing | **bcrypt** (cost ≥ 12) | Standard éprouvé |
| IDs publics | **UUID v4** | Pas d’énumération d’IDs numériques |
| Abstraction runtime | Interface `ContainerManager` | Découplage Incus ; évolutivité future |
| Message queue | **Aucune** | Simplicité ; pas de Redis/NATS/Kafka |
| Microservices | **Non** | Monolithe modulaire API + Agent |
| Reverse proxy | Nginx ou Caddy (prod) | TLS, servir le panel |

### Ce que nous n’introduisons pas

Kubernetes, Redis, RabbitMQ, NATS, Kafka, PostgreSQL, microservices — sauf justification technique réelle plus tard.

### Environnement de développement (constaté le 2026-09-18)

| Outil | État sur la machine de dév |
|-------|----------------------------|
| OS | Windows 10 (dev) — **cible prod = Linux** |
| Node.js | v22.20.0 |
| npm | 10.9.3 |
| Go | Installé localement pour builds (toolchain portable) |
| MariaDB | Absent en local → via Docker compose **ou** machine Linux |
| Incus | Absent (Phase 2, Linux uniquement) |
| WSL / Docker | Non disponibles actuellement |

Phase 1 est développable sans Incus. L’API doit démarrer avec MariaDB accessible.

---

## 3. Phases

### Phase 1 — Fondations *(terminée côté code)*

- [x] Monorepo + PLAN.md
- [x] API Go : config, DB, migrations, health
- [x] Auth : login / logout / me, sessions, bcrypt
- [x] Panel Next.js : branding, login, shell navigation
- [x] Schéma MariaDB minimal
- [x] Installer one-liner (skeleton fonctionnel)
- [x] README + `.env.example`

### Phase 2 — Incus + Agent *(terminée côté code)*

- [x] Interface `ContainerManager` + `IncusContainerManager`
- [x] Agent authentifié (token Node)
- [x] Create / delete / start / stop / restart / kill
- [x] Heartbeat Node → API
- [x] API nodes + servers lifecycle
- [x] Panel Nodes / Servers UI
### Phase 3 — Servers UI + Console + Metrics *(terminée côté code)*

- [x] Servers UI (liste + détail)
- [x] Console WebSocket (ticket → API → Agent → Incus exec)
- [x] Métriques réelles Incus (CPU delta, RAM, disk, network)
- [x] Dashboard alimenté par heartbeats Nodes (pas de fausses stats)

### Phase 4 — Files manager *(terminée côté code)*

- [x] list / upload / download / write / delete / rename / move
- [x] Protection path traversal stricte (`SanitizeContainerPath`)
- [x] Agent file routes + Incus SFTP / file API
- [x] API proxy `/api/v1/servers/{id}/files...` (auth + CanAccess)
- [x] Panel file manager (breadcrumb, editor, extract)

### Phase 5 — Templates *(terminée côté code)*

- [x] YAML templates + Minecraft / Node.js / Python / PHP / Nginx
- [x] Variables `{{RAM}}`, `{{SERVER_PORT}}`, etc.
- [x] Sync `templates/` → MariaDB (upsert by slug)
- [x] API CRUD `/api/v1/templates`
- [x] Server create uses `template_uuid` / `template_slug`
- [x] Panel `/templates` + create-form dropdown

### Phase 6 — Backups + multi-node + permissions avancées *(terminée côté code)*

- [x] Backups Incus (create / list / delete / download / restore)
- [x] API CRUD backups + panel `/backups` + section serveur
- [x] `server_permissions` (grant/revoke) + enforcement power/console/metrics/files
- [x] Multi-node polish : node info dans list servers, refuse create si OFFLINE

**Règle :** ne pas démarrer la phase N+1 tant que N n’est pas fonctionnelle.

---

## 4. API (préfixe `/api/v1`)

### Phase 1

```
POST /auth/login
POST /auth/logout
GET  /auth/me
GET  /health
```

### Phases suivantes (contrat)

```
GET|POST          /servers
GET|DELETE        /servers/:id
POST              /servers/:id/{start|stop|restart|kill}
GET|POST|DELETE   /servers/:id/backups...
POST              /servers/:id/backups/:id/restore
GET|POST|DELETE   /servers/:id/permissions...
GET               /backups
GET               /users
GET|POST|DELETE   /nodes...
GET|POST|PUT|DELETE /templates...
```

Console WebSocket : `/api/v1/servers/:uuid/console/ws`

---

## 5. Base de données (MariaDB)

Tables Phase 1 :

- `users` — id, uuid, username, email, password_hash, is_admin, timestamps
- `sessions` — id, user_id, token_hash, expires_at, ip, user_agent, timestamps
- `settings` — key/value pour config runtime

Tables créées (schéma prêt, usage Phase 2+) :

- `nodes`, `servers`, `server_templates`, `allocations`, `backups`, `server_permissions`

Pas de sur-normalisation.

---

## 6. Sécurité

- Jamais `exec(userInput)` — opérations via fonctions internes validées
- Path traversal bloqué côté Agent (Phase 4)
- Permissions toujours vérifiées côté API
- Rate limiting login
- Tokens Agent longs / rotatifs (Phase 2)
- TLS en production (reverse proxy)
- Logs d’actions admin (Phase 2+)
- Ne jamais faire confiance au frontend

---

## 7. Installation one-liner (cible)

```bash
curl -fsSL https://raw.githubusercontent.com/<org>/hostrix/main/installer/install.sh | bash
```

L’installateur Linux doit :

1. Vérifier OS / arch  
2. Installer dépendances + Incus + MariaDB si besoin  
3. Créer DB / user  
4. Installer binaires Hostrix  
5. systemd (`hostrix-api`, `hostrix-agent`)  
6. Afficher l’URL du panel  

Modes interactif et non-interactif (`HOSTRIX_NONINTERACTIVE=1`).

---

## 8. Prochaines étapes immédiates

1. Finaliser Phase 1 (API + Panel + migrations + installer skeleton)
2. Valider `go build` + `npm run build`
3. Publier le dépôt GitHub
4. Tester l’installateur sur une VM Linux
5. Enchaîner Phase 2 (Incus + Agent)

---

## 9. Identité produit

- **Nom :** Hostrix  
- **Slogan :** Your Infrastructure. Simplified.  
- **UI :** dark-first, premium, minimaliste, spacieuse — pas un clone visuel de Pterodactyl  
- **Navigation admin :** Dashboard, Servers, Nodes, Templates, Users, Backups, Settings  
- **Navigation user :** Dashboard, Servers, Account  
