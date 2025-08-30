# Fleet Tracker Service
   Backend fleet tracking with Go, Gin, PostgreSQL, and Redis.
Features :
- Backend: Go + Gin
- Database: PostgreSQL for vehicles & trips
- Cache: Redis caching /api/vehicle/status (5 minutes)
- Simulated Sensor Stream: Goroutines & channels
- Authentication: JWT middleware
- API Documentation: OpenAPI (docs/swagger)
- Docker: docker-compose for easy setup
- Database Migrations: /migrations folder
- Unit Tests: Minimal table-driven tests

## Project Structure
`
cmd/               → Application entry (main.go)
docs/              → API docs (Postman, curl, Swagger)
internal/
 ├─ auth/          → JWT authentication
 ├─ handlers/      → HTTP handlers
 ├─ model/         → Data models
 ├─ repository/    → DB access
 └─ service/       → Business logic & tests
migrations/        → DB schema & index SQL
server/            → DB migration & server setup
.env           → Environment variables
docker-compose.yml → Sets up and links multiple services (app, database,        cache) for easy startup
Dockerfile         → To build Docker image of app
go.mod             → To defines app dependencies
go.sum             → To keep the checksums of module dependencies
Makefile           → Quick commands to build, run, test, and manage migrations.
`

## Quick start (using Docker)

1. Copy `.env` to `.env` and edit if needed.
2. Build & run:
   ```
   docker-compose up --build
   ```
3. The app will be available at `http://localhost:8080`.
   - POST `/login` with `{"username":"demo-user","password":"demo-pass"}` to get a JWT.
   - Use `Authorization: Bearer <token>` for protected endpoints.

## Endpoints

- `POST /login` — get JWT (demo-user/demo-pass)
- `POST /api/vehicle/ingest` — ingest sensor payload (protected)
- `GET /api/vehicle/status?vehicle_id=<uuid>` — cached status (protected)
- `GET /api/vehicle/trips?vehicle_id=<uuid>` — trips past 24 hours (protected)

## Design notes

- Cache-aside strategy used: reader checks Redis first, on miss reads PostgreSQL and sets Redis with 5m TTL.
- On ingest, write-through: update DB and immediately update Redis to keep cache fresh.
- Simulated stream runs as a goroutine and pushes status updates every 2 seconds.

## DB Migrations
Initial schema & indexes in /migrations:
001_init.up.sql / 001_init.down.sql
002_index.up.sql / 002_index.down.sql

# Migrate up
migrate -path ./migrations -database "$DB_URL" up

# Migrate down
migrate -path ./migrations -database "$DB_URL" down 1

## Tests

Run unit tests:
```
go test ./...
```
