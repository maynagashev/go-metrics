package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentEncryption(t *testing.T) {
	// Создаем мок для агента
	mockAgent := new(MockAgent)
	mockAgent.On("IsEncryptionEnabled").Return(false)

	// Проверяем метод IsEncryptionEnabled
	result := mockAgent.IsEncryptionEnabled()
	assert.False(t, result)

	// Проверяем, что метод был вызван
	mockAgent.AssertExpectations(t)
}
