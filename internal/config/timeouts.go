// Package config предоставляет конфигурационные параметры для компонентов системы.
// Содержит настройки таймаутов, интервалов и других временных параметров.
package config

import "time"

const (
	// HTTP server timeouts.
	DefaultReadTimeout   = 10 * time.Second
	DefaultWriteTimeout  = 10 * time.Second
	DefaultIdleTimeout   = 60 * time.Second
	DefaultHeaderTimeout = 5 * time.Second
)
