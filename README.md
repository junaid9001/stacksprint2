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

## V2 Improvements & Recent Upgrades

- **Deterministic Generator Engine**: Replaced fragile string replacements with a robust, structured `// stacksprint:` marker-based injection system across all 15 architectures.
- **Strategy Pattern Pipeline**: The core generation logic is decoupled via the `GeneratorStrategy` interface, dramatically increasing language extensibility and separating concerns. 
- **Intelligent Frontend UI**: 
  - Live Complexity Tracking & Analyzer built into the interactive interface.
  - Granular dynamic toggles for language, framework, database, and infrastructure.
  - Auto-refreshing Bash script and Project Explorer previews.
- **Deep Architecture Awareness**: True code generation for MVP, Clean, Hexagonal, Modular Monolithic, and Microservices across Go, Node.js, and Python.
- **ORM & Database Flexibility**: Full support for SQLAlchemy, Django ORM, GORM, database/sql, Prisma, and native SQL drivers with environment-driven schemas.
- **Production-Ready Output**: Scaffolds include optional Git configurations, automated `git init`, multi-stage Dockerfiles, Makefile targets, and Docker Compose wiring.

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