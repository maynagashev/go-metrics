# Оптимизация обработки сжатых запросов

## Проблема

При профилировании сервера было обнаружено избыточное потребление памяти при обработке сжатых запросов.
Основная причина - создание нового `gzip.Reader` для каждого входящего запроса.

## Решение

Реализован пул объектов для переиспользования `gzip.Reader` и вспомогательных структур в пакете
`internal/server/middleware/decompresspool`.

### Основные изменения

1\. Создана структура `Middleware` для хранения пулов:

```go
type Middleware struct {
    log        *zap.Logger
    readerPool sync.Pool
    closerPool sync.Pool
}
```

2\. Реализовано переиспользование объектов через пулы:

```go
reader, ok := m.readerPool.Get().(*gzip.Reader)
if !ok {
    // обработка ошибки
}
defer m.readerPool.Put(reader)
```

3\. Добавлена структура `gzipReadCloser` для корректного освобождения ресурсов:

```go
type gzipReadCloser struct {
    *gzip.Reader
    middleware   *Middleware
    log          *zap.Logger
    originalBody io.ReadCloser
}
```

## Результаты профилирования

Сравнение аллокаций памяти до и после оптимизации:

### Уменьшение аллокаций

- -33.49MB (33.41%) в `compress/flate.NewWriter`
- -11.29MB в `compress/flate.NewReader`
- -11.79MB в `compress/gzip.(*Reader).Reset`
- -9.28MB в `compress/flate.(*dictDecoder).init`
- -8.01MB в `compress/flate.(*compressor).initDeflate`

### Бенчмарки на разных размерах данных

| Размер данных | Стандартная версия | Версия с пулом | Время (standard) | Время (pool) |
|---------------|-------------------|----------------|-----------------|--------------|
| small (100B)  | 50888 B/op       | 50964 B/op     | 22836 ns/op    | 22270 ns/op |
| medium (10KB) | 50864 B/op       | 50933 B/op     | 25983 ns/op    | 26320 ns/op |
| large (1MB)   | 50799 B/op       | 50860 B/op     | 417748 ns/op   | 404718 ns/op |

## Выводы

1. Оптимизация наиболее эффективна для больших запросов (>1MB):
   - Уменьшение времени обработки на ~3%
   - Незначительное увеличение потребления памяти (+61B/op)

2. Для средних и малых запросов:
   - Производительность примерно одинакова
   - Небольшой overhead в памяти из-за поддержки пула

3. Общее улучшение:
   - Значительное уменьшение количества аллокаций
   - Снижение нагрузки на GC
   - Более эффективное использование памяти при высокой нагрузке

## Дальнейшие возможности оптимизации

1. Добавление пула для буферов чтения
2. Оптимизация размера начального буфера gzip.Reader
3. Тонкая настройка размеров пулов под конкретную нагрузку

## Как запустить бенчмарки

```bash
# Запуск бенчмарков
go test -bench=BenchmarkDecompressMiddleware -benchmem ./internal/benchmarks/

# Сравнение результатов с помощью benchstat
go test -bench=BenchmarkDecompressMiddleware -benchmem -count=5 ./internal/benchmarks/ > results.txt
benchstat results.txt
```
