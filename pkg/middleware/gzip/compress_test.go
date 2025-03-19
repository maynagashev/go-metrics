package gzip_test

import (
	"bytes"
	gziplib "compress/gzip"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maynagashev/go-metrics/pkg/middleware/gzip"
)

func TestCompress(t *testing.T) {
	// Create test data
	testData := strings.Repeat("test data", 100) // Make it large enough to benefit from compression

	// Compress the data
	compressed, err := gzip.Compress([]byte(testData))
	require.NoError(t, err)
	require.NotNil(t, compressed)

	// Decompress the data
	reader, err := gziplib.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	require.NoError(t, err)

	// Check that the decompressed data matches the original
	assert.Equal(t, testData, string(decompressed))
}

func TestCompressEmptyData(t *testing.T) {
	// Compress empty data
	compressed, err := gzip.Compress([]byte{})
	require.NoError(t, err)
	require.NotNil(t, compressed)

	// Decompress the data
	reader, err := gziplib.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	require.NoError(t, err)

	// Check that the decompressed data is empty
	assert.Empty(t, string(decompressed))
}

func TestCompressError(t *testing.T) {
	// Create corrupted gzip data
	invalidData := []byte{
		0x1f, 0x8b, // Magic number
		0x08,                   // Compression method (deflate)
		0x00,                   // Flags
		0x00, 0x00, 0x00, 0x00, // Modification time
		0x00,       // Extra flags
		0xff,       // OS (unknown)
		0x01, 0x02, // Corrupted data
	}

	// Try to decompress invalid data
	reader, err := gziplib.NewReader(bytes.NewReader(invalidData))
	if err == nil {
		defer reader.Close()
		_, err = io.ReadAll(reader)
		assert.Error(t, err, "Expected an error when decompressing invalid data")
	} else {
		assert.Error(t, err, "Expected an error when creating reader for invalid data")
	}
}
