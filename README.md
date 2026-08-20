# Compose Dashboard

Lightweight dashboard for Docker Compose projects: list projects and services with
their status, and `up`/`restart` a whole project or a single service.

- Backend: Go stdlib, embeds the built frontend into a single static binary.
- Frontend: Vue 3 + Tailwind CSS v4 + shadcn-vue.

## Run

```sh
make build          # build frontend (dist/) and compile ./compose-dash
./compose-dash      # serve on http://127.0.0.1:8123
```

### Flags

```
--host 127.0.0.1    address to listen on
--port 8123         port to listen on
--timeout 10m       timeout for up/restart commands
--status-timeout 30s  timeout for status listing commands
```

### Development

```sh
make dev            # go run . + vite dev server (proxies /api to :8123)
```

## API

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/projects` | List projects with service states (health, ports) |
| POST | `/api/projects/{name}/up` | `docker compose up -d` for a project |
| POST | `/api/projects/{name}/restart` | `docker compose restart` for a project |
| POST | `/api/projects/{name}/services/{service}/up` | `up -d` a single service |
| POST | `/api/projects/{name}/services/{service}/restart` | restart a single service |

Actions resolve the project's config files/working dir via `docker compose ls`, so
projects anywhere on disk are manageable. If Docker is unreachable, the API returns
the error and the UI shows it as a banner instead of crashing.