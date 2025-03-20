# Переменные окружения для подключения к базе данных и пути к миграциям
DB_DSN = "postgres://metrics:password@localhost:5432/metrics?sslmode=disable"
MIGRATIONS_DIR = "migrations/server"

# Цель по умолчанию: прогнать миграции и запустить сервер вместе с агентом
all: migrate server_with_agent

# Объединённая директива .PHONY
.PHONY: migrate test bench lint test-coverage fmt docs staticcheck staticlint proto agent server

# Установка версий для сборки
set-versions:
	$(eval BUILD_VERSION := $(shell git describe --tags --always))
	$(eval BUILD_DATE := $(shell date "+%Y-%m-%d_%H:%M:%S"))
	$(eval BUILD_COMMIT := $(shell git rev-parse HEAD))
	@echo "Build version: $(BUILD_VERSION)"
	@echo "Build date: $(BUILD_DATE)"
	@echo "Build commit: $(BUILD_COMMIT)"


# Сборка всех необходимых бинарных файлов
build:
	@echo "Сборка серверной части, агента и миграций..."
	@go build -o ./bin/server ./cmd/server/.
	@go build -o ./bin/agent ./cmd/agent/.
	@go build -o ./bin/migrate ./cmd/migrate/.

# Генерация кода из proto-файлов
proto:
	@echo "Генерация кода из proto-файлов..."
	@mkdir -p internal/grpc/pb
	@protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/metrics.proto
	@mv proto/metrics.pb.go proto/metrics_grpc.pb.go internal/grpc/pb/
	@echo "Код успешно сгенерирован в директории internal/grpc/pb/"

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

# Запуск агента с логированием
agent-with-logging:
	@echo "Запуск агента с логированием..."
	@go run ./cmd/agent/. -k="private_key_example" | tee logs/agent-with-logging.log
# Запуск агента с коротким интервалом отправки метрик (пример для отладки)
fast-agent:
	@echo "Запуск агента (быстрый режим отправки метрик)..."
	@go run ./cmd/agent/. -k="private_key_example" -r 0.0001

# Запуск сервера и агента вместе
server-with-agent:
	@echo "Запуск сервера и агента вместе..."
	@go run ./cmd/server/. -d $(DB_DSN) & go run ./cmd/agent/.


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

# Запуск сервера с указанием версий (iter20)
server-with-version: set-versions
	@echo "Запуск сервера с указанием версий..."
	@go run -ldflags="-X 'main.BuildVersion=$(BUILD_VERSION)' -X 'main.BuildDate=$(BUILD_DATE)' -X 'main.BuildCommit=$(BUILD_COMMIT)'" ./cmd/server/. -d $(DB_DSN) -k="private_key_example" 

# Запуск агента с указанием версий (iter20)
agent-with-version: set-versions
	@echo "Запуск агента с указанием версий..."
	@go run -ldflags="-X 'main.BuildVersion=$(BUILD_VERSION)' -X 'main.BuildDate=$(BUILD_DATE)' -X 'main.BuildCommit=$(BUILD_COMMIT)'" ./cmd/agent/. -k="private_key_example"

# Запуск сервера с шифрованием (iter21)
server-with-encryption:
	@echo "Запуск сервера с шифрованием..."
	@go run ./cmd/server/. -d $(DB_DSN) -k="private_key_example" -crypto-key=server.key 2>&1 | tee logs/server-with-encryption.log

# Запуск агента с шифрованием (iter21)
agent-with-encryption:
	@echo "Запуск агента с шифрованием..."
	@go run ./cmd/agent/. -k="private_key_example" -crypto-key=server.crt 2>&1 | tee logs/agent-with-encryption.log

# Запуск сервера с конфигурационным файлом (iter22)
server-with-config:
	@echo "Запуск сервера с конфигурационным файлом..."
	@go run ./cmd/server/. -d $(DB_DSN) -k="private_key_example" -config=examples/server-config.json 2>&1 | tee logs/server-with-config.log

# Запуск агента с конфигурационным файлом (iter22)
agent-with-config:
	@echo "Запуск агента с конфигурационным файлом..."
	@go run ./cmd/agent/. -k="private_key_example" -config=examples/agent-config.json 2>&1 | tee logs/agent-with-config.log

# Запуск сервера с логированием для сохранения graceful shutdown лога (iter23)
server-with-graceful-shutdown:
	@echo "Запуск сервера с логированием для сохранения graceful shutdown лога..."
	@go run ./cmd/server/. -d $(DB_DSN) -k="private_key_example" >logs/server-graceful-shutdown.log 2>&1

# Запуск агента с логированием для сохранения graceful shutdown лога (iter23)
agent-with-graceful-shutdown:
	@echo "Запуск агента с логированием для сохранения graceful shutdown лога..."
	@go run ./cmd/agent/. -k="private_key_example" >logs/agent-graceful-shutdown.log 2>&1

# Запуск сервера с доверенной подсетью (iter24)
server-with-trusted-subnet:
	@echo "Запуск сервера с доверенной подсетью..."
	@go run ./cmd/server/. -d $(DB_DSN) -k="private_key_example" -t="192.168.1.0/24" 2>&1 | tee logs/server-with-trusted-subnet.log

# Запуск агента с заданным IP-адресом из доверенной подсети (iter24)
agent-with-trusted-ip:
	@echo "Запуск агента с заданным IP-адресом из доверенной подсети..."
	@go run ./cmd/agent/. -k="private_key_example" -real-ip="192.168.1.1" 2>&1 | tee logs/agent-with-trusted-subnet.log

# Запуск агента с IP-адресом отличным от доверенного (iter24)
agent-with-other-ip:
	@echo "Запуск агента с IP-адресом отличным от доверенного..."
	@go run ./cmd/agent/. -k="private_key_example" -real-ip="192.168.2.1" 2>&1 | tee logs/agent-with-other-ip.log

# Проверка запросов к серверу с разным X-Real-IP (iter24):
request-from-trusted-subnet:
	@echo "Проверка запросов к серверу с ip из доверенной подсети 192.168.1.1..."
	@curl -v -X POST -H "X-Real-IP: 192.168.1.1" -H "Content-Type: application/json" http://localhost:8080/update -d '{"id":"test","type":"counter","delta":1}' | tee logs/request-from-trusted-subnet.log
request-from-other-subnet:
	@echo "Проверка запросов к серверу с ip из другой подсети 192.168.2.1..."
	@curl -v -X POST -H "X-Real-IP: 192.168.2.1" -H "Content-Type: application/json" http://localhost:8080/update -d '{"id":"test","type":"counter","delta":1}' | tee logs/request-from-other-subnet.log


# Запуск сервера с поддержкой gRPC (iter25)
server-with-grpc:
	@echo "Запуск сервера с поддержкой gRPC..."
	@go run ./cmd/server/. -d $(DB_DSN) -k="private_key_example" -grpc-enabled -grpc-address="localhost:9090" 2>&1 | tee logs/server-with-grpc.log

# Запуск агента с использованием gRPC (iter25)
agent-with-grpc:
	@echo "Запуск агента с использованием gRPC..."
	@go run ./cmd/agent/. -k="private_key_example" -grpc-enabled -grpc-address="localhost:9090" 2>&1 | tee logs/agent-with-grpc.log

# Запуск сервера с шифрованием gRPC (iter25)
server-with-grpc-encryption:
	@echo "Запуск сервера с шифрованием gRPC..."
	@go run ./cmd/server/. -d $(DB_DSN) -k="private_key_example" -crypto-key=server.key -grpc-enabled -grpc-address="localhost:9090" 2>&1 | tee logs/server-with-grpc-encryption.log

# Запуск агента с шифрованием gRPC (iter25)	
agent-with-grpc-encryption:
	@echo "Запуск агента с шифрованием gRPC..."
	@go run ./cmd/agent/. -k="private_key_example" -crypto-key=server.crt -grpc-enabled -grpc-address="localhost:9090" 2>&1 | tee logs/agent-with-grpc-encryption.log	

# Запуск всех тестов
test:
	@echo "Запуск всех тестов..."
	@go test -v ./... | tee logs/test.log

# Тест с генерацией отчёта о покрытии
test-coverage:
	@echo "Запуск тестов с генерацией покрытия..."
	go test -coverprofile=logs/coverage.out ./...
	go tool cover -html=logs/coverage.out -o logs/coverage.html
	go tool cover -func=logs/coverage.out | tee logs/coverage.log


# Пример запуска бенчмарков
bench:
	@echo "Запуск бенчмарков..."
	@mkdir -p logs
	go test -bench=. -benchmem ./internal/benchmarks/... | tee logs/benchmarks.log

# Запуск линтера
lint:
	@echo "Запуск линтера..."
	golangci-lint run ./... --fix



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
	goimports -w .
	golines -w .
	./scripts/fix-imports.sh

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
	go run ./cmd/staticlint/ ./... | tee logs/staticlint.log

# Генерация приватного ключа и публичного сертификата x509
gen-keys:
	@echo "Генерация ключей..."
	@go run ./cmd/keygen/main.go -bits=4096 -private=server.key -public=server.crt
