package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maynagashev/go-metrics/internal/server/app"
)

func TestPrintVersion(t *testing.T) {
	// Save the original stdout
	oldStdout := os.Stdout

	// Create a pipe to capture stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)

	// Set stdout to our pipe
	os.Stdout = w

	// Set test values for the build variables
	originalBuildVersion := BuildVersion
	originalBuildDate := BuildDate
	originalBuildCommit := BuildCommit

	// Restore the original values when the test completes
	defer func() {
		BuildVersion = originalBuildVersion
		BuildDate = originalBuildDate
		BuildCommit = originalBuildCommit
		os.Stdout = oldStdout
	}()

	// Set test values
	BuildVersion = "v1.0.0"
	BuildDate = "2023-01-01"
	BuildCommit = "abc123"

	// Call the function
	printVersion()

	// Close the write end of the pipe to flush the buffer
	w.Close()

	// Read the captured output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	// Check the output
	output := buf.String()
	assert.Contains(t, output, "Build version: v1.0.0")
	assert.Contains(t, output, "Build date: 2023-01-01")
	assert.Contains(t, output, "Build commit: abc123")
}

func TestInitLogger(t *testing.T) {
	// Test that the logger is created without panicking
	assert.NotPanics(t, func() {
		logger := initLogger()
		assert.NotNil(t, logger)
	})
}

func TestInitStorage(t *testing.T) {
	// Create a minimal config for testing
	logger := initLogger()

	// Test with database disabled
	t.Run("MemoryStorage", func(t *testing.T) {
		cfg := &app.Config{
			Database: app.DatabaseConfig{
				DSN: "",
			},
		}

		repo, err := initStorage(cfg, logger)
		require.NoError(t, err)
		assert.NotNil(t, repo)

		// Clean up
		err = repo.Close()
		require.NoError(t, err)
	})

	// We can't easily test the PostgreSQL path without a real database,
	// so we'll skip that test case
}
