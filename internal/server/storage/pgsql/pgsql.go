package pgsql

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage"
	"go.uber.org/zap"
)

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

// IncrementCounter увеличивает значение счетчика на указанное значение, если записи нет то создает новую.
func (p *PostgresStorage) IncrementCounter(name string, delta storage.Counter) {
	m := metrics.NewCounter(name, int64(delta))
	err := p.UpdateMetric(*m)
	if err != nil {
		p.log.Error(err.Error())
	}
}

// UpdateMetric универсальный метод обновления метрики: gauge, counter.
// Если метрика существует, то обновляем, иначе создаем новую.
func (p *PostgresStorage) UpdateMetric(metric metrics.Metric) error {
	var q string

	// Если метрика существует, то обновляем, иначе создаем новую.
	_, ok := p.GetMetric(metric.MType, metric.Name)
	if ok {
		q = `UPDATE public.metrics SET value = $3, delta = delta + $4 WHERE name = $1 AND type = $2`
	} else {
		q = `INSERT INTO public.metrics (name, type, value, delta) VALUES ($1, $2, $3, $4)`
	}

	_, err := p.conn.Exec(p.ctx, q, metric.Name, metric.MType, metric.Value, metric.Delta)
	if err != nil {
		p.log.Error(err.Error())
		return err
	}

	return nil
}
