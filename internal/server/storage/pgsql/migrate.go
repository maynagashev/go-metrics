package pgsql

// Создание необходимых таблиц в базе данных, для gauge и счетчиков отдельные таблички, с разным типом значения.
func (p *PostgresStorage) createTables() error {
	_, err := p.conn.Exec(p.ctx, `CREATE TABLE IF NOT EXISTS metrics (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		delta BIGINT NULL ,
		value DOUBLE PRECISION NULL
	)`)
	if err != nil {
		return err
	}

	// Создаем индекс по имени метрики
	_, err = p.conn.Exec(p.ctx, `CREATE INDEX IF NOT EXISTS idx_metrics_name ON metrics (name)`)
	if err != nil {
		return err
	}

	// Создаем индекс по типу метрики
	_, err = p.conn.Exec(p.ctx, `CREATE INDEX IF NOT EXISTS idx_metrics_type ON metrics (type)`)
	if err != nil {
		return err
	}

	// Создаем уникальный индекс по имени и типу метрики
	_, err = p.conn.Exec(p.ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_metrics_name_type ON metrics (name, type)`)
	if err != nil {
		return err
	}

	p.log.Debug("created table if not exists: metrics")

	return nil
}
