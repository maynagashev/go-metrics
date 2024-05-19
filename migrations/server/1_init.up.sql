CREATE TABLE IF NOT EXISTS metrics
(
    id    SERIAL PRIMARY KEY,
    name  VARCHAR(250)     NOT NULL,
    type  VARCHAR(50)      NOT NULL,
    delta BIGINT           NULL,
    value DOUBLE PRECISION NULL
);

/* Индексы для ускорения выборки метрик по имени и типу. */
CREATE INDEX IF NOT EXISTS idx_metrics_name ON metrics (name);
CREATE INDEX IF NOT EXISTS idx_metrics_type ON metrics (type);

/* Комбинации имени и типа метрики в БД должны быть уникальны. */
CREATE UNIQUE INDEX IF NOT EXISTS idx_metrics_name_type ON metrics (name, type);