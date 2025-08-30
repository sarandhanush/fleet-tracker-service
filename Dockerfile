# ---------- Build stage ----------
FROM golang:1.23-bullseye AS builder

# Install swag CLI
RUN go install github.com/swaggo/swag/cmd/swag@latest

WORKDIR /app

# Copy Go modules manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Generate Swagger docs
RUN swag init -g ./cmd/main.go

# Build the binary
RUN go build -o fleet-tracker ./cmd/main.go


# ---------- Runtime stage ----------
FROM debian:bullseye-slim

# Add CA certs for HTTPS calls (important for Go apps)
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary + Swagger docs from builder
COPY --from=builder /app/fleet-tracker /fleet-tracker
COPY --from=builder /app/docs ./docs
COPY --from=builder /app/migrations ./migrations

# Expose port
EXPOSE 8080

# Run the server
CMD ["/fleet-tracker"]
