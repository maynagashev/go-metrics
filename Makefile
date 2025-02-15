DB_DSN = "postgres://metrics:password@localhost:5432/metrics?sslmode=disable"
MIGRATIONS_DIR = "migrations/server"

all: migrate server_with_agent

build:
	@echo "Building: server, agent, migrate..."
	@go build -o ./bin/server ./cmd/server/.
	@go build -o ./bin/agent ./cmd/agent/.
	@go build -o ./bin/migrate ./cmd/migrate/.

migrate:
	@echo "Running migrations..."
	@go run ./cmd/migrate/main.go -d $(DB_DSN) -migrations-path $(MIGRATIONS_DIR)

server:
	@echo "Starting server..."
	@go run ./cmd/server/. -d $(DB_DSN) -k="private_key_example"

agent:
	@echo "Starting agent..."
	@go run ./cmd/agent/. -k="private_key_example"

fast_agent:
	@echo "Starting agent with short report interval..."
	@go run ./cmd/agent/. -k="private_key_example" -r 0.0001


server_with_agent:
	@echo "Starting server with agent..."
	@go run ./cmd/server/. -d $(DB_DSN) & go run ./cmd/agent/.

lint :
	@echo "Running linter..."
	golangci-lint run ./...

iter10: build
	@echo "Running iteration 14 tests ..."
	./bin/metricstest-darwin-amd64 -test.v -test.run=^TestIteration10$  \
									-server-port=8080 -binary-path=bin/server -agent-binary-path=bin/agent \
									-database-dsn=$(DB_DSN) \
									-source-path . \
									-key=iter10 \
									| tee logs/iter10.log

iter14: build
	@echo "Running iteration 14 tests ..."
	./bin/metricstest-darwin-amd64 -test.v -test.run=^TestIteration14$  \
									-server-port=8080 -binary-path=bin/server -agent-binary-path=bin/agent \
									-database-dsn=$(DB_DSN) \
									-source-path . \
									-key=iter14 \
									| tee logs/iter14.log

.PHONY: test-coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

.PHONY: bench
bench:
	@echo "Running benchmarks..."
	@mkdir -p logs
	go test -bench=. -benchmem ./internal/benchmarks/... | tee logs/benchmarks.log


# Профилирование различных компонентов системы
.PHONY: save-all-profiles
save-all-profiles: profile-benchmarks profile-server-memory profile-agent-memory

# Запуск сервера с профилированием
.PHONY: profile-server
profile-server:
	@echo "Running server with profiling enabled..."
	@echo "Open http://localhost:8080/debug/pprof to view profiles"
	@go run ./cmd/server/. -d $(DB_DSN) -k="private_key_example" -pprof

# Запуск агента с профилированием
.PHONY: profile-agent
profile-agent:
	@echo "Running agent with profiling enabled..."
	@go run ./cmd/agent/. -k="private_key_example" -pprof

# Сохранение профиля memory для сервера
.PHONY: profile-server-memory
profile-server-memory:
	@mkdir -p profiles
	$(eval DATE := $(shell date '+%Y%m%d_%H%M%S'))
	curl -s http://localhost:8080/debug/pprof/heap > profiles/server_heap_$(DATE).pprof
	curl -s http://localhost:8080/debug/pprof/allocs > profiles/server_allocs_$(DATE).pprof
# Сохранение профиля memory для агента
.PHONY: profile-agent-memory
profile-agent-memory:
	@mkdir -p profiles
	$(eval DATE := $(shell date '+%Y%m%d_%H%M%S'))
	curl -s http://localhost:6060/debug/pprof/heap > profiles/agent_heap_$(DATE).pprof
	curl -s http://localhost:6060/debug/pprof/allocs > profiles/agent_allocs_$(DATE).pprof

# Сохранение профиля памяти для бенчмарков
.PHONY: profile-benchmarks
profile-benchmarks:
	@echo "Running benchmarks with memory profiling..."
	@mkdir -p profiles
	$(eval DATE := $(shell date '+%Y%m%d_%H%M%S'))
	go test -bench=. -benchmem -memprofile=profiles/bench_mem_$(DATE).pprof ./internal/benchmarks/...
