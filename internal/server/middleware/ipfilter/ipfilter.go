// Package ipfilter содержит middleware для фильтрации запросов по IP-адресу.
// Проверяет, что IP-адрес клиента входит в доверенную подсеть.
package ipfilter

import (
	"net"
	"net/http"

	"github.com/maynagashev/go-metrics/internal/server/app"
	"go.uber.org/zap"
)

// Middleware представляет middleware для фильтрации запросов по IP-адресу.
type Middleware struct {
	log    *zap.Logger
	config *app.Config
	subnet *net.IPNet
}

// New создает новый middleware для фильтрации запросов по IP-адресу.
func New(config *app.Config, log *zap.Logger) func(http.Handler) http.Handler {
	m := &Middleware{
		log:    log,
		config: config,
	}

	// Если указана доверенная подсеть, парсим её
	if config.IsTrustedSubnetEnabled() {
		_, ipNet, err := net.ParseCIDR(config.TrustedSubnet)
		if err != nil {
			log.Error("failed to parse trusted subnet CIDR",
				zap.String("cidr", config.TrustedSubnet),
				zap.Error(err))
		} else {
			m.subnet = ipNet
			log.Info("trusted subnet enabled",
				zap.String("cidr", config.TrustedSubnet))
		}
	}

	return m.Handler
}

// Handler обрабатывает запрос, проверяя IP-адрес клиента.
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Если доверенная подсеть не указана или не удалось её распарсить, пропускаем запрос
		if m.subnet == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Получаем IP-адрес из заголовка X-Real-IP
		ipStr := r.Header.Get("X-Real-IP")
		if ipStr == "" {
			m.log.Warn("request without X-Real-IP header",
				zap.String("remote_addr", r.RemoteAddr))
			next.ServeHTTP(w, r)
			return
		}

		// Парсим IP-адрес
		ip := net.ParseIP(ipStr)
		if ip == nil {
			m.log.Warn("invalid IP address in X-Real-IP header",
				zap.String("ip", ipStr))
			http.Error(w, "Invalid IP address", http.StatusBadRequest)
			return
		}

		// Проверяем, входит ли IP-адрес в доверенную подсеть
		if !m.subnet.Contains(ip) {
			m.log.Warn("IP address not in trusted subnet",
				zap.String("ip", ipStr),
				zap.String("subnet", m.config.TrustedSubnet))
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// IP-адрес входит в доверенную подсеть, пропускаем запрос
		m.log.Debug("IP address in trusted subnet",
			zap.String("ip", ipStr),
			zap.String("subnet", m.config.TrustedSubnet))
		next.ServeHTTP(w, r)
	})
}
