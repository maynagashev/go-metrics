package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name           string
		dsn            string
		migrationsPath string
		wantErr        bool
		errMsg         string
	}{
		{
			name:           "empty dsn",
			dsn:            "",
			migrationsPath: "../../migrations",
			wantErr:        true,
			errMsg:         "не указаны параметры подключения к БД: -d postgres://user:password@localhost:5432/database",
		},
		{
			name:           "empty migrations path",
			dsn:            "postgres://user:password@localhost:5432/database",
			migrationsPath: "",
			wantErr:        true,
			errMsg:         "не указан путь к директории с миграциями: -migrations-path ../../migrations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(tt.dsn, tt.migrationsPath)
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, tt.errMsg, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMain_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Сохраняем оригинальные аргументы
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no args",
			args:    []string{"app"},
			wantErr: true,
		},
		{
			name: "only dsn",
			args: []string{
				"app",
				"-d", "postgres://user:password@localhost:5432/database",
			},
			wantErr: true,
		},
		{
			name: "only migrations path",
			args: []string{
				"app",
				"-migrations-path", "../../migrations",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем тестовые аргументы
			os.Args = tt.args

			// Проверяем, что программа паникует при неверных аргументах
			if tt.wantErr {
				require.Panics(t, main)
			} else {
				require.NotPanics(t, main)
			}
		})
	}
}
