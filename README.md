# StackSprint V2

StackSprint is a backend project initialization engine that generates production-ready starter code and a one-command setup script.

## Highlights

- Multi-language support: Go, Node.js, Python
- Framework support:
  - Go: Gin, Fiber
  - Node.js: Express, Fastify
  - Python: FastAPI, Django (API mode)
- Architecture modes:
  - MVP
  - Clean Architecture
  - Hexagonal
  - Modular Monolith
  - Microservices (2-5 services)
- Database options: PostgreSQL, MySQL, MongoDB, None
- Optional infra/features:
  - Redis, Kafka, NATS
  - JWT auth boilerplate
  - Swagger/OpenAPI
  - GitHub Actions CI
  - Makefile, logger, global error handler, health endpoint, sample tests
- Dynamic customization:
  - Add/remove folders
  - Add/remove files
  - Add/remove services
- Output format:
  - Bash script
  - Live file tree preview paths

## V2 Improvements

- True architecture-aware generation for Go, Node.js, and Python
- ORM toggle (`use_orm`) for SQL stacks:
  - Go: GORM or `database/sql`
  - Node.js: Prisma or SQL driver setup
  - Python: SQLAlchemy (FastAPI) or Django ORM
- Stronger script generation:
  - Empty directory preservation with `.gitkeep`
  - Safer bash heredoc delimiter
- Live frontend preview with debounce:
  - Auto-refresh script preview
  - Project Explorer from backend `file_paths`

## Project Structure

```text
stacksprint/
  frontend/      # Next.js App Router UI
  backend/       # Go Fiber stateless generation API
  templates/     # Architecture + language templates
  docker-compose.yaml
  README.md
```

## Run StackSprint

```bash
docker compose up --build
```

- Frontend: `http://localhost:3000`
- Backend: `http://localhost:8080`

## API

### `POST /generate`

Request body includes:

- `language`, `framework`, `architecture`
- `services` (for microservices)
- `db`, `use_orm`
- `service_communication`
- `infra`, `features`
- `file_toggles`
- `custom` (add/remove folders/files/services)
- `root`

Response:

```json
{
  "bash_script": "...",
  "file_paths": ["..."]
}
```

## Development

Backend:

```bash
cd backend
go test ./...
```

Frontend:

```bash
cd frontend
npm run build
```

## Notes

- Generated projects are designed to run with `docker compose up --build`.
- StackSprint itself is stateless and does not require its own app database.