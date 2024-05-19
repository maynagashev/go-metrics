# go-metrics

Репозиторий для трека «Сервер сбора метрик и алертинга».

## Итерации

- **Iter7.** Пакет encoding. Сериализация и десериализация данных.
    - [x] Отправлять с агента метрики в формате json на новый маршрут `/update`
    - [x] Реализовать на сервере новый маршрут `/update` который будет принимать json с метрикой и парсить его в
      структуру.
    - [x] Получать значение метрик с помощью `POST /value`, в ответе такой же json только с заполненными значениями.
    - [x] Проверить тесты.
- **Iter8.** Пакет compress. Сжатие данных.
    - [x] Агент передавать данные в формате gzip (добавить Content-Encoding: gzip в заголовок запроса и сжать данные).
    - [x] Сервер опционально принимать запросы в сжатом формате (при наличии соответствующего HTTP-заголовка
      Content-Encoding).
    - [x] Отдавать сжатый ответ клиенту, который поддерживает обработку сжатых ответов (с HTTP-заголовком
      Accept-Encoding).
- **Iter9.** Пакет os. Операции с файлами.
    - [x] Доработайте код сервера, чтобы он мог с заданной периодичностью сохранять текущие значения метрик на диск в
      указанный файл, а на старте — опционально загружать сохранённые ранее значения. При штатном завершении сервера все
      накопленные данные должны сохраняться.
    - [x] Флаг `-i`, переменная окружения STORE_INTERVAL — интервал времени в секундах, по истечении которого текущие
      показания сервера сохраняются на диск (по умолчанию 300 секунд, значение 0 делает запись синхронной).
    - [x] Флаг `-f`, переменная окружения FILE_STORAGE_PATH — полное имя файла, куда сохраняются текущие значения (по
      умолчанию /tmp/metrics-db.json, пустое значение отключает функцию записи на диск).
    - [x] Флаг `-r`, переменная окружения RESTORE — булево значение (true/false), определяющее, загружать или нет ранее
      сохранённые значения из указанного файла при старте сервера (по умолчанию true).
- **Iter10.** Пакет database/sql. Взаимодействие с базами данных SQL.
    - [x] Добавьте на сервер функциональность подключения к базе данных. В качестве СУБД используйте PostgreSQL не ниже 10 версии.
    - [x] Строка с адресом подключения к БД должна получаться из переменной окружения DATABASE_DSN или флага командной строки -d.
    - [x] Добавьте в сервер хендлер GET /ping, который при запросе проверяет соединение с базой данных. При успешной проверке хендлер должен вернуть HTTP-статус 200 OK, при неуспешной — 500 Internal Server Error.
- **Iter11.** Пакет database/sql. Взаимодействие с базами данных SQL.
    - [x] Перепишите сервер для сбора метрик таким образом, чтобы СУБД PostgreSQL стала хранилищем метрик вместо текущей реализации.
- **Iter12.** Пакет database/sql. Взаимодействие с базами данных SQL.
    - [x] **Сервер:** Добавьте новый хендлер POST /updates/, принимающий в теле запроса множество метрик в формате: []Metrics (списка метрик).
    - [x] **Агент:** Научите агент работать с использованием нового API (отправлять метрики батчами).
- **Iter13.** Добавьте обработку `retriable`-ошибок.
    - [x] Агент не сумел с первой попытки выгрузить данные на сервер из-за временной невозможности установить соединение с сервером.
    - [x] При обращении к PostgreSQL cервер получил ошибку транспорта (из категории Class 08 — Connection Exception)
    - [x] Ошибка доступа к файлу, который был заблокирован другим процессом.
    - [x] Три повтора (всего 4 попытки). Интервалы между повторами должны увеличиваться: 1s, 3s, 5s.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m main template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/main .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например,
в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте
в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Локальный запуск тестов

```bash
# бинарник для MacOS (intel)
wget https://github.com/Yandex-Practicum/go-autotests/releases/download/v0.10.6/metricstest-darwin-amd64
chmod +x metricstest-darwin-amd64

# запуск тестов
./metricstest-darwin-amd64 -test.v  -binary-path cmd/server/server -agent-binary-path=cmd/agent/agent -source-path . > test.log

# запуск конкретной итерации
./metricstest-darwin-amd64 -test.v -test.run=^TestIteration7$ -binary-path cmd/server/server -agent-binary-path=cmd/agent/agent -source-path . | tee test.log

# запуск конкретной итерации с  -server-port=8080 
 ./metricstest-darwin-amd64 -test.v -test.run=^TestIteration8$ -server-port=8080 -binary-path cmd/server/server -agent-binary-path=cmd/agent/agent -source-path . | tee test.log

# проверка iter12
./metricstest-darwin-amd64 -test.v -test.run=^TestIteration12$ -server-port=8080 -binary-path cmd/server/server -agent-binary-path=cmd/agent/agent -database-dsn=postgres://metrics:password@localhost:5432/metrics -source-path . | tee test.log

# проверка iter13
./metricstest-darwin-amd64 -test.v -test.run=^TestIteration13$ -server-port=8080 -binary-path cmd/server/server -agent-binary-path=cmd/agent/agent -database-dsn=postgres://metrics:password@localhost:5432/metrics -source-path . | tee test.log


# запуск сервера с postgres
go run . -d postgres://metrics:password@localhost:5432/metrics
```

## Запуск линтеров

```bash
clear && golangci-lint run
```