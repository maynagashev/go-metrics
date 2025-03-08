package index

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository - мок для интерфейса storage.Repository.
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRepository) Count(ctx context.Context) int {
	args := m.Called(ctx)
	return args.Int(0)
}

func (m *MockRepository) GetMetrics(ctx context.Context) []metrics.Metric {
	args := m.Called(ctx)
	return args.Get(0).([]metrics.Metric)
}

func (m *MockRepository) GetMetric(ctx context.Context, mType metrics.MetricType, name string) (metrics.Metric, bool) {
	args := m.Called(ctx, mType, name)
	return args.Get(0).(metrics.Metric), args.Bool(1)
}

func (m *MockRepository) GetCounter(ctx context.Context, name string) (storage.Counter, bool) {
	args := m.Called(ctx, name)
	return args.Get(0).(storage.Counter), args.Bool(1)
}

func (m *MockRepository) GetGauge(ctx context.Context, name string) (storage.Gauge, bool) {
	args := m.Called(ctx, name)
	return args.Get(0).(storage.Gauge), args.Bool(1)
}

func (m *MockRepository) UpdateMetric(ctx context.Context, metric metrics.Metric) error {
	args := m.Called(ctx, metric)
	return args.Error(0)
}

func (m *MockRepository) UpdateMetrics(ctx context.Context, metrics []metrics.Metric) error {
	args := m.Called(ctx, metrics)
	return args.Error(0)
}

func TestNew(t *testing.T) {
	// Создаем тестовые случаи
	tests := []struct {
		name           string
		setupMock      func(*MockRepository)
		expectedStatus int
		expectedBody   []string
	}{
		{
			name: "Empty metrics list",
			setupMock: func(m *MockRepository) {
				m.On("GetMetrics", mock.Anything).Return([]metrics.Metric{})
				m.On("Count", mock.Anything).Return(0)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   []string{},
		},
		{
			name: "With gauge metrics",
			setupMock: func(m *MockRepository) {
				value := 42.5
				metrics := []metrics.Metric{
					{
						Name:  "test_gauge",
						MType: "gauge",
						Value: &value,
					},
				}
				m.On("GetMetrics", mock.Anything).Return(metrics)
				m.On("Count", mock.Anything).Return(1)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   []string{"gauge/test_gauge: 42.5"},
		},
		{
			name: "With counter metrics",
			setupMock: func(m *MockRepository) {
				var delta int64 = 10
				metrics := []metrics.Metric{
					{
						Name:  "test_counter",
						MType: "counter",
						Delta: &delta,
					},
				}
				m.On("GetMetrics", mock.Anything).Return(metrics)
				m.On("Count", mock.Anything).Return(1)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   []string{"counter/test_counter: 10"},
		},
		{
			name: "With unknown metric type",
			setupMock: func(m *MockRepository) {
				metrics := []metrics.Metric{
					{
						Name:  "test_unknown",
						MType: "unknown",
					},
				}
				m.On("GetMetrics", mock.Anything).Return(metrics)
				m.On("Count", mock.Anything).Return(1)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   []string{"unknown/test_unknown"},
		},
		{
			name: "With multiple metrics",
			setupMock: func(m *MockRepository) {
				value := 42.5
				var delta int64 = 10
				metrics := []metrics.Metric{
					{
						Name:  "test_gauge",
						MType: "gauge",
						Value: &value,
					},
					{
						Name:  "test_counter",
						MType: "counter",
						Delta: &delta,
					},
				}
				m.On("GetMetrics", mock.Anything).Return(metrics)
				m.On("Count", mock.Anything).Return(2)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   []string{"gauge/test_gauge: 42.5", "counter/test_counter: 10"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Создаем мок репозитория
			mockRepo := new(MockRepository)

			// Настраиваем мок
			tc.setupMock(mockRepo)

			// Создаем обработчик
			handler := New(mockRepo)

			// Создаем тестовый запрос
			req, err := http.NewRequest(http.MethodGet, "/", nil)
			require.NoError(t, err)

			// Создаем ResponseRecorder для записи ответа
			rr := httptest.NewRecorder()

			// Вызываем обработчик
			handler.ServeHTTP(rr, req)

			// Проверяем статус-код ответа
			assert.Equal(t, tc.expectedStatus, rr.Code)

			// Проверяем заголовок Content-Type
			assert.Equal(t, "text/html", rr.Header().Get("Content-Type"))

			// Проверяем тело ответа
			body := rr.Body.String()
			for _, expectedLine := range tc.expectedBody {
				assert.Contains(t, body, expectedLine)
			}

			// Проверяем, что каждая строка заканчивается переносом строки
			lines := strings.Split(strings.TrimSpace(body), "\n")
			if len(tc.expectedBody) > 0 {
				assert.Equal(t, len(tc.expectedBody), len(lines))
			}

			// Проверяем, что все методы мока были вызваны
			mockRepo.AssertExpectations(t)
		})
	}
}
