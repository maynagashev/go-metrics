package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/maynagashev/go-metrics/internal/server/app"
)

func TestConfig_IsTrustedSubnetEnabled(t *testing.T) {
	testCases := []struct {
		name           string
		trustedSubnet  string
		expectedResult bool
	}{
		{
			name:           "Trusted subnet is specified",
			trustedSubnet:  "192.168.0.0/24",
			expectedResult: true,
		},
		{
			name:           "Trusted subnet is empty",
			trustedSubnet:  "",
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flags := &app.Flags{}
			flags.Server.TrustedSubnet = tc.trustedSubnet
			config := app.NewConfig(flags)

			assert.Equal(t, tc.expectedResult, config.IsTrustedSubnetEnabled())
		})
	}
}

func TestConfig_IsGRPCEnabled(t *testing.T) {
	testCases := []struct {
		name           string
		grpcEnabled    bool
		expectedResult bool
	}{
		{
			name:           "GRPC server is enabled",
			grpcEnabled:    true,
			expectedResult: true,
		},
		{
			name:           "GRPC server is disabled",
			grpcEnabled:    false,
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flags := &app.Flags{}
			flags.GRPC.Enabled = tc.grpcEnabled
			config := app.NewConfig(flags)

			assert.Equal(t, tc.expectedResult, config.IsGRPCEnabled())
		})
	}
}

func TestConfig_GetCryptoKeyPath(t *testing.T) {
	testCases := []struct {
		name           string
		cryptoKeyPath  string
		expectedResult string
	}{
		{
			name:           "Crypto key path is specified",
			cryptoKeyPath:  "/path/to/key.pem",
			expectedResult: "/path/to/key.pem",
		},
		{
			name:           "Crypto key path is empty",
			cryptoKeyPath:  "",
			expectedResult: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Создаем конфигурацию напрямую, без использования NewConfig
			config := &app.Config{
				CryptoKey: tc.cryptoKeyPath,
			}

			assert.Equal(t, tc.expectedResult, config.GetCryptoKeyPath())
		})
	}
}
