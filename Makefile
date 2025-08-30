build:
	go build -o fleet-tracker ./cmd/main.go

run:
	go run ./cmd/main.go

test:
	go test ./...

migrate-up:
	migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/fleet?sslmode=disable" up

migrate-down:
	migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/fleet?sslmode=disable" down 1
