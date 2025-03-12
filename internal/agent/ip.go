package agent

import (
	"errors"
	"log/slog"
	"net"
)

// ErrNoOutboundIP возникает, когда не удается определить исходящий IP-адрес.
var ErrNoOutboundIP = errors.New("no outbound IP address found")

// GetOutboundIP определяет исходящий IP-адрес, используемый для подключения к внешним ресурсам.
// Функция создает UDP-соединение (которое не устанавливает реальное соединение) с публичным IP-адресом
// и использует локальный адрес этого соединения как исходящий IP.
func GetOutboundIP() (net.IP, error) {
	// Используем 8.8.8.8:80 (Google DNS) как адрес назначения
	// Это не устанавливает реальное соединение, а только определяет, какой сетевой интерфейс будет использован
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		slog.Error("failed to determine outbound IP", "error", err)
		return nil, err
	}
	defer conn.Close()

	// Получаем локальный адрес соединения
	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		slog.Error("failed to convert local address to UDP address")
		return nil, ErrNoOutboundIP
	}

	// Проверяем, что IP-адрес не nil и не равен 0.0.0.0
	if localAddr.IP == nil || localAddr.IP.IsUnspecified() {
		slog.Error("local IP address is nil or unspecified")
		return nil, ErrNoOutboundIP
	}

	slog.Debug("determined outbound IP", "ip", localAddr.IP.String())
	return localAddr.IP, nil
}
