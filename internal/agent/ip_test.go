package agent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maynagashev/go-metrics/internal/agent"
)

func TestGetOutboundIP(t *testing.T) {
	// Вызываем функцию для получения исходящего IP
	ip, err := agent.GetOutboundIP()

	// Проверяем, что ошибки нет
	require.NoError(t, err)

	// Проверяем, что IP не nil
	assert.NotNil(t, ip)

	// Проверяем, что IP не равен 0.0.0.0
	assert.False(t, ip.IsUnspecified())

	// Проверяем, что IP является валидным IPv4 или IPv6 адресом
	assert.True(t, ip.To4() != nil || ip.To16() != nil)
}
