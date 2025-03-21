// Package pgstorage реализует хранилище метрик на основе PostgreSQL.
// Предоставляет методы для сохранения и получения метрик из базы данных.
package pgstorage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage"
	"github.com/maynagashev/go-metrics/internal/server/storage/pgstorage/migration"
)

const maxRetries = 3

// PgxPoolInterface определяет интерфейс для pgxpool.Pool, чтобы можно было использовать мок в тестах.
type PgxPoolInterface interface {
	Close()
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Убедимся, что pgxpool.Pool реализует наш интерфейс.
var _ PgxPoolInterface = (*pgxpool.Pool)(nil)

type PgStorage struct {
	conn PgxPoolInterface
	cfg  *app.Config
	log  *zap.Logger
}

// New создает новое подключение к базе данных, накатывает миграции и возвращает экземпляр хранилища.
func New(ctx context.Context, config *app.Config, log *zap.Logger) (*PgStorage, error) {
	conn, err := pgxpool.New(ctx, config.Database.DSN)
	log.Debug(fmt.Sprintf("Connecting to database: %s\n", config.Database.DSN))

	if err != nil {
		log.Error(fmt.Sprintf("Unable to connect to database: %v\n", err))
		return nil, err
	}

	p := &PgStorage{
		conn: conn,
		cfg:  config,
		log:  log,
	}

	// Автоматически накатываем миграции при создании экземпляра хранилища.
	if migrateErr := migration.Up(config.Database.MigrationsPath, config.Database.DSN); migrateErr != nil {
		return nil, fmt.Errorf("failed to apply migrations: %w", migrateErr)
	}
	return p, nil
}

// Close закрывает соединение с базой данных.
// Метод всегда возвращает nil, так как pgxpool.Close() не возвращает ошибку,
// но интерфейс Repository требует возврата error.
func (p *PgStorage) Close() error {
	if p.conn != nil {
		p.conn.Close()
	}
	return nil
}

func (p *PgStorage) Count(ctx context.Context) int {
	var count int
	err := p.conn.QueryRow(ctx, `SELECT count(*) FROM metrics`).Scan(&count)
	if err != nil {
		p.log.Error(err.Error())
	}
	return count
}

func (p *PgStorage) GetMetrics(ctx context.Context) []metrics.Metric {
	var items []metrics.Metric
	rows, err := p.conn.Query(ctx, `SELECT name, type, value, delta FROM metrics ORDER BY name`)
	if err != nil {
		p.log.Error(err.Error())
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var metric metrics.Metric
		err = rows.Scan(&metric.Name, &metric.MType, &metric.Value, &metric.Delta)
		if err != nil {
			p.log.Error(err.Error())
			return nil
		}
		items = append(items, metric)
	}

	return items
}

// GetMetric получение значения метрики указанного типа в виде универсальной структуры.
func (p *PgStorage) GetMetric(
	ctx context.Context,
	mType metrics.MetricType,
	name string,
) (metrics.Metric, bool) {
	q := `SELECT name, type, value, delta FROM public.metrics WHERE name = $1 AND type = $2`

	var metric metrics.Metric
	var err error
	for i := 0; i <= maxRetries; i++ {
		row := p.conn.QueryRow(ctx, q, name, mType)
		err = row.Scan(&metric.Name, &metric.MType, &metric.Value, &metric.Delta)

		if err == nil {
			return metric, true
		}

		// Проверяем, является ли ошибка retriable
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if isRetriableError(pgErr) {
				p.log.Error(fmt.Sprintf("Attempt %d: Retriable error getting metric: %v", i+1, err))
				time.Sleep(time.Duration((i+1)*2-1) * time.Second)
				continue
			}
		}

		// Если ошибка не retriable, выходим из цикла
		break
	}

	// Логируем и возвращаем ошибку, если не удалось получить метрику
	if err != nil {
		p.log.Error(fmt.Sprintf("Failed to get metric after %d tries: %v", maxRetries+1, err))
	}
	return metrics.Metric{}, false
}

// GetCounter возвращает счетчик по имени.
func (p *PgStorage) GetCounter(ctx context.Context, name string) (storage.Counter, bool) {
	m, ok := p.GetMetric(ctx, metrics.TypeCounter, name)
	if !ok {
		return 0, false
	}
	return storage.Counter(*m.Delta), true
}

// GetGauge возвращает измерение по имени.
func (p *PgStorage) GetGauge(ctx context.Context, name string) (storage.Gauge, bool) {
	m, ok := p.GetMetric(ctx, metrics.TypeGauge, name)
	if !ok {
		return 0, false
	}
	return storage.Gauge(*m.Value), true
}

func (p *PgStorage) UpdateMetric(ctx context.Context, metric metrics.Metric) error {
	var q string

	// Если метрика существует, то обновляем, иначе создаем новую.
	_, ok := p.GetMetric(ctx, metric.MType, metric.Name)
	if ok {
		q = `UPDATE metrics SET value = $3, delta = delta + $4 WHERE name = $1 AND type = $2`
	} else {
		q = `INSERT INTO metrics (name, type, value, delta) VALUES ($1, $2, $3, $4)`
	}

	// Выполнение запроса
	_, err := p.conn.Exec(ctx, q, metric.Name, metric.MType, metric.Value, metric.Delta)
	if err != nil {
		p.log.Error(fmt.Sprintf("Failed to update metric: %v", err))
		return err
	}
	return nil
}

// UpdateMetrics пакетно обновляет метрики в хранилище.
func (p *PgStorage) UpdateMetrics(ctx context.Context, items []metrics.Metric) error {
	var err error
	q := `INSERT INTO metrics (name, type, value, delta) 
          VALUES ($1, $2, $3, $4)
          ON CONFLICT (name, type) 
          DO UPDATE SET value = EXCLUDED.value, delta = metrics.delta + EXCLUDED.delta`

	// Начало транзакции
	tx, err := p.conn.Begin(ctx)
	if err != nil {
		return err
	}

	// Откатываем транзакцию в случае ошибки
	defer func() {
		if err != nil {
			if rErr := tx.Rollback(ctx); rErr != nil && !errors.Is(rErr, pgx.ErrTxClosed) {
				p.log.Error(fmt.Sprintf("Failed to rollback transaction: %v", rErr))
			}
		}
	}()

	batch := &pgx.Batch{}
	for _, item := range items {
		batch.Queue(q, item.Name, item.MType, item.Value, item.Delta)
	}

	// Выполнение батч-запроса
	br := tx.SendBatch(ctx, batch)
	_, err = br.Exec()
	if errClose := br.Close(); errClose != nil {
		p.log.Error(fmt.Sprintf("Failed to close batch: %v", errClose))
		return errClose
	}

	if err != nil {
		p.log.Error(fmt.Sprintf("Failed to update metrics: %v", err))
		return err
	}

	// Подтверждаем транзакцию
	if err = tx.Commit(ctx); err != nil {
		p.log.Error(fmt.Sprintf("Failed to commit transaction: %v", err))
		return err
	}

	return nil
}

// Проверка, является ли ошибка retriable.
func isRetriableError(err *pgconn.PgError) bool {
	switch err.Code {
	case pgerrcode.ConnectionException,
		pgerrcode.ConnectionDoesNotExist,
		pgerrcode.ConnectionFailure,
		pgerrcode.DiskFull:
		return true
	default:
		return false
	}
}
