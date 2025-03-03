package random_test

import (
	"testing"

	"github.com/maynagashev/go-metrics/pkg/random"
	"github.com/stretchr/testify/assert"
)

func TestRandomFloat64(t *testing.T) {
	// Test that the function returns a value within the expected range
	min := 0.0
	max := 1.0

	for i := 0; i < 1000; i++ {
		value := random.GenerateRandomFloat64()
		assert.GreaterOrEqual(t, value, min)
		assert.LessOrEqual(t, value, max)
	}
}

func TestRandomFloat64_EqualMinMax(t *testing.T) {
	// Test that the function returns a value between 0 and 1
	min := 0.0
	max := 1.0

	value := random.GenerateRandomFloat64()
	assert.GreaterOrEqual(t, value, min)
	assert.LessOrEqual(t, value, max)
}

func TestRandomFloat64_MinGreaterThanMax(t *testing.T) {
	// Test that the function returns a value between 0 and 1
	min := 0.0
	max := 1.0

	for i := 0; i < 100; i++ {
		value := random.GenerateRandomFloat64()
		assert.GreaterOrEqual(t, value, min)
		assert.LessOrEqual(t, value, max)
	}
}
