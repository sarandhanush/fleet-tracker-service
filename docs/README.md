# Fleet Tracker Service
   Backend fleet tracking with Go, Gin, PostgreSQL, and Redis.
![Uploading image.png…](https://github.com/sarandhanush/fleet-tracker-service/blob/3db445f294dcf2a7a8a93c279996d24d6e0db17f/Fleet-tracker-service.png)
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
```
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
```

## Quick start (using Docker)

1. Copy `.env` to `.env` and edit if needed.
2. Build & run:
   ```
   docker-compose up --build
   ```
    ![Uploading image.png…](https://github.com/sarandhanush/fleet-tracker-service/blob/2a0d52dc70da8d4de6efd746f27c5c8ab1843be1/docker_up.png)

    # Docker Container Running
   
    ![Uploading image.png…](https://github.com/sarandhanush/fleet-tracker-service/blob/539f456c398f8f8b85b9905ab742565e7fe46428/docker_image.png)
   
   # Swagger Image Attachment
    
    ![Uploading image.png…](https://github.com/sarandhanush/fleet-tracker-service/blob/539f456c398f8f8b85b9905ab742565e7fe46428/swagger_page.png)

4. The app will be available at `http://localhost:8080`.
   - POST `/login` with `{"username":"admin","password":"admin@2025"}` to get a JWT.
   - Use `Authorization: Bearer <token>` for protected endpoints.

## Endpoints

- `POST /login` — get JWT (admin/admin@2025)
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

   Migrate up
   ```
migrate -path ./migrations -database "$DB_URL" up
 ```
   Migrate down
 ```
migrate -path ./migrations -database "$DB_URL" down 1
 ```
## Tests

Run unit tests:
```
go test ./...
```
