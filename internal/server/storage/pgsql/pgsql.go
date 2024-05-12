package pgsql

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"go.uber.org/zap"
)

func New(_ context.Context, config *app.Config, log *zap.Logger) (*pgx.Conn, error) {
	// urlExample := "postgres://username:password@localhost:5432/database_name"
	conn, err := pgx.Connect(context.Background(), config.Database.DSN)
	log.Debug(fmt.Sprintf("Connecting to database: %s\n", config.Database.DSN))
	if err != nil {
		log.Error(fmt.Sprintf("Unable to connect to database: %v\n", err))
		return nil, err
	}
	defer func() {
		_ = conn.Close(context.Background())
	}()
	return conn, nil
}
