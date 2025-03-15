package random_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/maynagashev/go-metrics/pkg/random"
)

func TestRandomFloat64(t *testing.T) {
	// Test that the function returns a value within the expected range
	minVal := 0.0
	maxVal := 1.0

	for range 1000 {
		value := random.GenerateRandomFloat64()
		assert.GreaterOrEqual(t, value, minVal)
		assert.LessOrEqual(t, value, maxVal)
	}
}

func TestRandomFloat64_EqualMinMax(t *testing.T) {
	// Test that the function returns a value between 0 and 1
	minVal := 0.0
	maxVal := 1.0

	value := random.GenerateRandomFloat64()
	assert.GreaterOrEqual(t, value, minVal)
	assert.LessOrEqual(t, value, maxVal)
}

func TestRandomFloat64_MinGreaterThanMax(t *testing.T) {
	// Test that the function returns a value between 0 and 1
	minVal := 0.0
	maxVal := 1.0

	for range 100 {
		value := random.GenerateRandomFloat64()
		assert.GreaterOrEqual(t, value, minVal)
		assert.LessOrEqual(t, value, maxVal)
	}
}
