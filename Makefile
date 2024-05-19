DB_DSN = "postgres://metrics:password@localhost:5432/metrics?sslmode=disable"
MIGRATIONS_DIR = "migrations/server"

all: migrate server_with_agent

migrate:
	@echo "Running migrations..."
	@go run ./cmd/migrate/main.go -d $(DB_DSN) -migrations-path $(MIGRATIONS_DIR)

server:
	@echo "Starting server..."
	@go run ./cmd/server/. -d $(DB_DSN)

agent:
	@echo "Starting agent..."
	@go run ./cmd/agent/.

server_with_agent:
	@echo "Starting server with agent..."
	@go run ./cmd/server/. -d $(DB_DSN) & go run ./cmd/agent/.

lint :
	@echo "Running linter..."
	@golangci-lint run