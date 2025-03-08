package app

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	// Test with default values
	flags := &Flags{}
	flags.Server.Addr = "localhost:8080"
	flags.Server.StoreInterval = 300
	flags.Server.FileStoragePath = "/tmp/metrics-db.json"
	flags.Server.Restore = true
	flags.Database.DSN = ""
	flags.PrivateKey = ""
	flags.CryptoKey = ""

	config := NewConfig(flags)

	// Verify the config was created correctly
	assert.Equal(t, "localhost:8080", config.Addr)
	assert.Equal(t, 300, config.StoreInterval)
	assert.Equal(t, "/tmp/metrics-db.json", config.FileStoragePath)
	assert.True(t, config.Restore)
	assert.Equal(t, "", config.Database.DSN)
	assert.Equal(t, "", config.PrivateKey)
	assert.Nil(t, config.PrivateRSAKey)
}

func TestConfig_IsStoreEnabled(t *testing.T) {
	// Test with store enabled
	config := &Config{
		FileStoragePath: "/tmp/metrics-db.json",
	}
	assert.True(t, config.IsStoreEnabled())

	// Test with store disabled
	config = &Config{
		FileStoragePath: "",
	}
	assert.False(t, config.IsStoreEnabled())
}

func TestConfig_IsRestoreEnabled(t *testing.T) {
	// Test with restore enabled
	config := &Config{
		Restore:         true,
		FileStoragePath: "/tmp/metrics-db.json",
	}
	assert.True(t, config.IsRestoreEnabled())

	// Test with restore disabled
	config = &Config{
		Restore: false,
	}
	assert.False(t, config.IsRestoreEnabled())
}

func TestConfig_GetStorePath(t *testing.T) {
	config := &Config{
		FileStoragePath: "/tmp/metrics-db.json",
	}
	assert.Equal(t, "/tmp/metrics-db.json", config.GetStorePath())
}

func TestConfig_IsSyncStore(t *testing.T) {
	// Test with sync store
	config := &Config{
		StoreInterval: 0,
	}
	assert.True(t, config.IsSyncStore())

	// Test with async store
	config = &Config{
		StoreInterval: 300,
	}
	assert.False(t, config.IsSyncStore())
}

func TestConfig_GetStoreInterval(t *testing.T) {
	config := &Config{
		StoreInterval: 300,
	}
	assert.Equal(t, 300, config.GetStoreInterval())
}

func TestConfig_IsDatabaseEnabled(t *testing.T) {
	// Test with database enabled
	config := &Config{
		Database: DatabaseConfig{
			DSN: "postgres://user:password@localhost:5432/metrics",
		},
	}
	assert.True(t, config.IsDatabaseEnabled())

	// Test with database disabled
	config = &Config{
		Database: DatabaseConfig{
			DSN: "",
		},
	}
	assert.False(t, config.IsDatabaseEnabled())
}

func TestConfig_IsRequestSigningEnabled(t *testing.T) {
	// Test with request signing enabled
	config := &Config{
		PrivateKey: "test-key",
	}
	assert.True(t, config.IsRequestSigningEnabled())

	// Test with request signing disabled
	config = &Config{
		PrivateKey: "",
	}
	assert.False(t, config.IsRequestSigningEnabled())
}

func TestConfig_IsEncryptionEnabled(t *testing.T) {
	// Generate a real RSA key for testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Test with encryption enabled
	config := &Config{
		PrivateRSAKey: privateKey,
	}
	assert.True(t, config.IsEncryptionEnabled())

	// Test with encryption disabled
	config = &Config{
		PrivateRSAKey: nil,
	}
	assert.False(t, config.IsEncryptionEnabled())
}

func TestNew(t *testing.T) {
	config := &Config{
		Addr:            "localhost:8080",
		StoreInterval:   300,
		FileStoragePath: "/tmp/metrics-db.json",
		Restore:         true,
	}

	server := New(config)

	assert.NotNil(t, server)
	assert.Equal(t, config, server.cfg)
}

func TestServer_GetStoreInterval(t *testing.T) {
	config := &Config{
		StoreInterval: 300,
	}

	server := New(config)

	assert.Equal(t, 300, server.GetStoreInterval())
}

func TestServer_Start(t *testing.T) {
	// Skip this test as it requires complex mocking of HTTP server and signal handling
	t.Skip("Skipping test that requires complex mocking of HTTP server")
}
