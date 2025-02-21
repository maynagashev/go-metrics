# Переменные окружения для подключения к базе данных и пути к миграциям
DB_DSN = "postgres://metrics:password@localhost:5432/metrics?sslmode=disable"
MIGRATIONS_DIR = "migrations/server"

# Цель по умолчанию: прогнать миграции и запустить сервер вместе с агентом
all: migrate server_with_agent

# Объединённая директива .PHONY
.PHONY: migrate test bench lint test-coverage fmt docs staticcheck

# Сборка всех необходимых бинарных файлов
build:
	@echo "Сборка серверной части, агента и миграций..."
	@go build -o ./bin/server ./cmd/server/.
	@go build -o ./bin/agent ./cmd/agent/.
	@go build -o ./bin/migrate ./cmd/migrate/.

# Прогон миграций
migrate:
	@echo "Запуск миграций..."
	@go run ./cmd/migrate/main.go -d $(DB_DSN) -migrations-path $(MIGRATIONS_DIR)

# Запуск сервера
server:
	@echo "Запуск сервера..."
	@go run ./cmd/server/. -d $(DB_DSN) -k="private_key_example"

# Запуск агента
agent:
	@echo "Запуск агента..."
	@go run ./cmd/agent/. -k="private_key_example"

# Запуск агента с коротким интервалом отправки метрик (пример для отладки)
fast_agent:
	@echo "Запуск агента (быстрый режим отправки метрик)..."
	@go run ./cmd/agent/. -k="private_key_example" -r 0.0001

# Запуск сервера и агента вместе
server_with_agent:
	@echo "Запуск сервера и агента вместе..."
	@go run ./cmd/server/. -d $(DB_DSN) & go run ./cmd/agent/.

# Запуск всех тестов
test:
	@echo "Запуск всех тестов..."
	@go test -v ./... | tee logs/test.log

# Пример запуска бенчмарков
bench:
	@echo "Запуск бенчмарков..."
	@mkdir -p logs
	go test -bench=. -benchmem ./internal/benchmarks/... | tee logs/benchmarks.log

# Запуск линтера
lint:
	@echo "Запуск линтера..."
	golangci-lint run ./...

# Пример запуска автотеста для итерации 10
iter10: build
	@echo "Запуск тестов для итерации 10..."
	./bin/metricstest-darwin-amd64 -test.v -test.run=^TestIteration10$  \
									-server-port=8080 -binary-path=bin/server -agent-binary-path=bin/agent \
									-database-dsn=$(DB_DSN) \
									-source-path . \
									-key=iter10 \
									| tee logs/iter10.log

# Пример запуска автотеста для итерации 14
iter14: build
	@echo "Запуск тестов для итерации 14..."
	./bin/metricstest-darwin-amd64 -test.v -test.run=^TestIteration14$  \
									-server-port=8080 -binary-path=bin/server -agent-binary-path=bin/agent \
									-database-dsn=$(DB_DSN) \
									-source-path . \
									-key=iter14 \
									| tee logs/iter14.log

# Тест с генерацией отчёта о покрытии
test-coverage:
	@echo "Запуск тестов с генерацией покрытия..."
	go test -coverprofile=logs/coverage.out ./...
	go tool cover -html=logs/coverage.out -o logs/coverage.html
	go tool cover -func=logs/coverage.out | tee logs/coverage.log

# Запуск всех типов профилирования
save-all-profiles: profile-benchmarks profile-server-memory profile-agent-memory

# Запуск сервера с профилированием
profile-server:
	@echo "Запуск сервера с профилированием..."
	@echo "Откройте http://localhost:8080/debug/pprof для просмотра профилей"
	@go run ./cmd/server/. -d $(DB_DSN) -k="private_key_example" -pprof

# Запуск агента с профилированием
profile-agent:
	@echo "Запуск агента с профилированием..."
	@go run ./cmd/agent/. -k="private_key_example" -pprof

# Сохранение профиля памяти сервера (heap, allocs)
profile-server-memory:
	@mkdir -p profiles
	$(eval DATE := $(shell date '+%Y%m%d_%H%M%S'))
	curl -s http://localhost:8080/debug/pprof/heap > profiles/server_heap_$(DATE).pprof
	curl -s http://localhost:8080/debug/pprof/allocs > profiles/server_allocs_$(DATE).pprof

# Сохранение профиля памяти агента (heap, allocs)
profile-agent-memory:
	@mkdir -p profiles
	$(eval DATE := $(shell date '+%Y%m%d_%H%M%S'))
	curl -s http://localhost:6060/debug/pprof/heap > profiles/agent_heap_$(DATE).pprof
	curl -s http://localhost:6060/debug/pprof/allocs > profiles/agent_allocs_$(DATE).pprof

# Сохранение профиля памяти бенчмарков
profile-benchmarks:
	@echo "Запуск бенчмарков с профилированием памяти..."
	@mkdir -p profiles
	$(eval DATE := $(shell date '+%Y%m%d_%H%M%S'))
	go test -bench=. -benchmem -memprofile=profiles/bench_mem_$(DATE).pprof ./internal/benchmarks/...

# Сравнение профилей памяти
compare-profiles:
	@echo "Сравнение профилей памяти..."
	@go tool pprof -top -diff_base=profiles/base_server_allocs_20250215_230049.pprof profiles/server_allocs_$(shell date '+%Y%m%d_%H%M%S').pprof | tee logs/compare-profiles.log

# Форматирование кода
fmt:
	gofmt -s -w .
	goimports --local -w .
	golines -w .

# Запуск сервера с документацией
docs:
	godoc -http=:8888 -play

# Запуск staticcheck
staticcheck:
	@echo "Запуск staticcheck..."
	staticcheck ./... | tee logs/staticcheck.log

# Запуск кастомного мультичекера
staticlint:
	@echo "Запуск кастомного мультичекера staticlint..."
	go run ./cmd/staticlint/ ./...
