# go-metrics

Репозиторий для трека «Сервер сбора метрик и алертинга».

## Итерации

- **Iter7.** Пакет encoding. Сериализация и десериализация данных.
   - [x] Отправлять с агента метрики в формате json на новый маршрут `/update`
   - [x] Реализовать на сервере новый маршрут `/update` который будет принимать json с метрикой и парсить его в структуру.
   - [x] Получать значение метрик с помощью `POST /value`, в ответе такой же json только с заполненными значениями.
   - [x] Проверить тесты.
- **Iter8.** Пакет compress. Сжатие данных.
   - [ ] Агент передавать данные в формате gzip (добавить Content-Encoding: gzip в заголовок запроса и сжать данные).
   - [ ] Сервер опционально принимать запросы в сжатом формате (при наличии соответствующего HTTP-заголовка Content-Encoding).
   - [x] Отдавать сжатый ответ клиенту, который поддерживает обработку сжатых ответов (с HTTP-заголовком Accept-Encoding).

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
```

## Запуск линтеров

```bash
clear && golangci-lint run
```