# Сжатие запросов обновления метрик агентом

По умолчанию, агент должен сжимать запросы обновления метрик с использованием алгоритма GZIP.

## Пример лога входящего запроса
```log
 2024-05-04T17:31:19.467+0700    INFO    logger/logger.go:38     request completed       
```
```json
{
  "method": "POST",
  "path": "/update",
  "remote_addr": "127.0.0.1:51223",
  "user_agent": "go-resty/2.12.0 (https://github.com/go-resty/resty)",
  "request_id": "localhost.local/COHM144SnI-000029",
  "headers": {
    "Accept": [
      "application/json"
    ],
    "Accept-Encoding": [
      "gzip"
    ],
    "Content-Encoding": [
      "gzip"
    ],
    "Content-Length": [
      "67"
    ],
    "Content-Type": [
      "application/json"
    ],
    "User-Agent": [
      "go-resty/2.12.0 (https://github.com/go-resty/resty)"
    ]
  },
  "request_body": "\u001f\ufffd\u0008\u0000\u0000\u0000\u0000\u0000\u0002\ufffd\ufffdV\ufffdLQ\ufffdR\n\ufffd\ufffd\ufffdq\ufffd/\ufffd+Q\ufffdQ*\ufffd,HU\ufffdRJ\u0006qS\ufffd\ufffdt\ufffdRRsJ\u0012\ufffd\ufffdLk\u0001\u0001\u0000\u0000\ufffd\ufffd\ufffd^+\ufffd-\u0000\u0000\u0000",
  "status": 200,
  "response_bytes": 109,
  "duration": "461.999µs",
  "response_body": "\u001f\ufffd\u0008\u0000\u0000\u0000\u0000\u0000\u0000\ufffd\ufffd\ufffdRP\ufffdM-.NLOU\ufffd\u00021K\ufffd2\ufffd\u0015|\ufffdT\ufffd\ufffd\ufffd\ufffdB@~N\ufffds~i^\ufffd\ufffdBHeA\ufffd\ufffdB2\ufffd\ufffdZ\ufffd\ufffd\ufffd\ufffd\ufffdS\ufffdh\ufffd`Z\ufffdPZ\ufffd\ufffdX\ufffd\ufffd\ufffd\ufffdP\ufffdZ\\\ufffdSb\ufffd`fhb\ufffd\ufffdU\u000b\u0008\u0000\u0000\ufffd\ufffd\ufffd\ufffd\ufffdx]\u0000\u0000\u0000"
}
```


## Обработка ошибок
Если агент отправляет `Content-Encoding: gzip` в заголовке запроса, но данные не сжаты, сервер вернет ошибку:
`Ошибка при декомпрессии содержимого запроса gzip`.

Проверка:
```log
2024-05-04T09:02:10.822+0700    INFO    logger/logger.go:38     request completed
```
```json
{
  "method": "POST",
  "path": "/update",
  "remote_addr": "127.0.0.1:54881",
  "user_agent": "go-resty/2.12.0 (https://github.com/go-resty/resty)",
  "request_id": "localhost.local/C8Lj1xURpn-000003",
  "headers": {
    "Accept": [
      "application/json"
    ],
    "Accept-Encoding": [
      "gzip"
    ],
    "Content-Encoding": [
      "gzip"
    ],
    "Content-Length": [
      "70"
    ],
    "Content-Type": [
      "application/json"
    ],
    "User-Agent": [
      "go-resty/2.12.0 (https://github.com/go-resty/resty)"
    ]
  },
  "request_body": "{\"id\":\"GCCPUFraction\",\"type\":\"gauge\",\"value\":0.0000037169578824393963}",
  "status": 400,
  "response_bytes": 88,
  "duration": "673.125µs",
  "response_body": "Ошибка при декомпрессии содержимого запроса gzip\n"
}
```
