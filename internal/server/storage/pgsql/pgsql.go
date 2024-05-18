package pgsql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jackc/pgx/v5"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage"
	"go.uber.org/zap"
)

const maxRetries = 3

type PostgresStorage struct {
	conn *pgx.Conn
	cfg  *app.Config
	log  *zap.Logger
	ctx  context.Context
}

func New(ctx context.Context, config *app.Config, log *zap.Logger) (*PostgresStorage, error) {
	conn, err := pgx.Connect(ctx, config.Database.DSN)
	log.Debug(fmt.Sprintf("Connecting to database: %s\n", config.Database.DSN))

	if err != nil {
		log.Error(fmt.Sprintf("Unable to connect to database: %v\n", err))
		return nil, err
	}

	p := &PostgresStorage{
		conn: conn,
		cfg:  config,
		log:  log,
		ctx:  ctx,
	}

	// Создание необходимых таблиц в базе данных.
	err = p.createTables()
	if err != nil {
		log.Error(fmt.Sprintf("Unable to create tables: %v\n", err))
		return p, err
	}

	return p, nil
}

func (p *PostgresStorage) Close() error {
	return p.conn.Close(p.ctx)
}

func (p *PostgresStorage) Count() int {
	var count int
	err := p.conn.QueryRow(p.ctx, `SELECT count(*) FROM metrics`).Scan(&count)
	if err != nil {
		p.log.Error(err.Error())
	}
	return count
}

func (p *PostgresStorage) GetMetrics() []metrics.Metric {
	var items []metrics.Metric
	rows, err := p.conn.Query(p.ctx, `SELECT name, type, value, delta FROM metrics ORDER BY name`)
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
func (p *PostgresStorage) GetMetric(mType metrics.MetricType, name string) (metrics.Metric, bool) {
	q := `SELECT name, type, value, delta FROM public.metrics WHERE name = $1 AND type = $2`
	row := p.conn.QueryRow(p.ctx, q, name, mType)

	var metric metrics.Metric
	err := row.Scan(&metric.Name, &metric.MType, &metric.Value, &metric.Delta)
	if err != nil {
		return metrics.Metric{}, false
	}
	return metric, true
}

// GetCounter возвращает счетчик по имени.
func (p *PostgresStorage) GetCounter(name string) (storage.Counter, bool) {
	m, ok := p.GetMetric(metrics.TypeCounter, name)
	if !ok {
		return 0, false
	}
	return storage.Counter(*m.Delta), true
}

// GetGauge возвращает измерение по имени.
func (p *PostgresStorage) GetGauge(name string) (storage.Gauge, bool) {
	m, ok := p.GetMetric(metrics.TypeGauge, name)
	if !ok {
		return 0, false
	}
	return storage.Gauge(*m.Value), true
}

// UpdateMetric универсальный метод обновления метрики: gauge, counter.
// Если метрика существует, то обновляем, иначе создаем новую.
func (p *PostgresStorage) UpdateMetric(metric metrics.Metric) error {
	var q string

	// Если метрика существует, то обновляем, иначе создаем новую.
	_, ok := p.GetMetric(metric.MType, metric.Name)
	if ok {
		q = `UPDATE metrics SET value = $3, delta = delta + $4 WHERE name = $1 AND type = $2`
	} else {
		q = `INSERT INTO metrics (name, type, value, delta) VALUES ($1, $2, $3, $4)`
	}

	// Попытка выполнения запроса с обработкой retriable-ошибок
	var err error
	for i := 0; i <= maxRetries; i++ {
		_, err = p.conn.Exec(p.ctx, q, metric.Name, metric.MType, metric.Value, metric.Delta)

		// Если нет ошибок выходим из цикла и функции.
		if err == nil {
			return nil
		}

		// Проверяем, является ли ошибка retriable
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if isRetriableError(pgErr) {
				p.log.Error(fmt.Sprintf("Attempt %d: Retriable error updating metric: %v", i+1, err))
				time.Sleep(time.Duration((i+1)*2-1) * time.Second)
				continue
			}
		}

		// Если ошибка не retriable, выходим из цикла
		break
	}

	// Логируем и возвращаем ошибку, если не удалось обновить метрику
	if err != nil {
		p.log.Error(fmt.Sprintf("Failed to update metric: %v", err))
	}
	return err
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
