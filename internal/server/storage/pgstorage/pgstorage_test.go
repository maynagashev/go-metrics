package pgstorage

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// Mock for pgxpool.Pool.
type MockPgxPool struct {
	mock.Mock
}

// Убедимся, что MockPgxPool реализует интерфейс PgxPoolInterface.
var _ PgxPoolInterface = (*MockPgxPool)(nil)

func (m *MockPgxPool) Close() {
	m.Called()
}

func (m *MockPgxPool) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	args := m.Called(ctx, sql, arguments)
	return args.Get(0).(pgconn.CommandTag), args.Error(1)
}

func (m *MockPgxPool) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	mockArgs := m.Called(ctx, sql, args)
	return mockArgs.Get(0).(pgx.Rows), mockArgs.Error(1)
}

func (m *MockPgxPool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	mockArgs := m.Called(ctx, sql, args)
	return mockArgs.Get(0).(pgx.Row)
}

func (m *MockPgxPool) Begin(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	return args.Get(0).(pgx.Tx), args.Error(1)
}

// Mock for pgx.Rows.
type MockPgxRows struct {
	mock.Mock
}

func (m *MockPgxRows) Next() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockPgxRows) Scan(dest ...interface{}) error {
	args := m.Called(dest...)
	return args.Error(0)
}

func (m *MockPgxRows) Close() {
	m.Called()
}

func (m *MockPgxRows) CommandTag() pgconn.CommandTag {
	args := m.Called()
	return args.Get(0).(pgconn.CommandTag)
}

func (m *MockPgxRows) Err() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockPgxRows) FieldDescriptions() []pgconn.FieldDescription {
	args := m.Called()
	return args.Get(0).([]pgconn.FieldDescription)
}

func (m *MockPgxRows) Values() ([]interface{}, error) {
	args := m.Called()
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *MockPgxRows) RawValues() [][]byte {
	args := m.Called()
	return args.Get(0).([][]byte)
}

func (m *MockPgxRows) Conn() *pgx.Conn {
	args := m.Called()
	return args.Get(0).(*pgx.Conn)
}

// Mock for pgx.Row.
type MockPgxRow struct {
	mock.Mock
}

func (m *MockPgxRow) Scan(dest ...interface{}) error {
	args := m.Called(dest)
	return args.Error(0)
}

// Mock for pgx.Tx.
type MockPgxTx struct {
	mock.Mock
}

func (m *MockPgxTx) Begin(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	return args.Get(0).(pgx.Tx), args.Error(1)
}

func (m *MockPgxTx) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockPgxTx) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockPgxTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	args := m.Called(ctx, tableName, columnNames, rowSrc)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPgxTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	args := m.Called(ctx, b)
	return args.Get(0).(pgx.BatchResults)
}

func (m *MockPgxTx) LargeObjects() pgx.LargeObjects {
	args := m.Called()
	return args.Get(0).(pgx.LargeObjects)
}

func (m *MockPgxTx) Prepare(ctx context.Context, name, sql string) (pgconn.StatementDescription, error) {
	args := m.Called(ctx, name, sql)
	return args.Get(0).(pgconn.StatementDescription), args.Error(1)
}

func (m *MockPgxTx) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	args := m.Called(ctx, sql, arguments)
	return args.Get(0).(pgconn.CommandTag), args.Error(1)
}

func (m *MockPgxTx) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	mockArgs := m.Called(ctx, sql, args)
	return mockArgs.Get(0).(pgx.Rows), mockArgs.Error(1)
}

func (m *MockPgxTx) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	mockArgs := m.Called(ctx, sql, args)
	return mockArgs.Get(0).(pgx.Row)
}

// Mock for pgx.BatchResults.
type MockBatchResults struct {
	mock.Mock
}

func (m *MockBatchResults) Exec() (pgconn.CommandTag, error) {
	args := m.Called()
	return args.Get(0).(pgconn.CommandTag), args.Error(1)
}

func (m *MockBatchResults) Query() (pgx.Rows, error) {
	args := m.Called()
	return args.Get(0).(pgx.Rows), args.Error(1)
}

func (m *MockBatchResults) QueryRow() pgx.Row {
	args := m.Called()
	return args.Get(0).(pgx.Row)
}

func (m *MockBatchResults) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Create a simple test for isRetriableError function.
func Test_isRetriableError(t *testing.T) {
	// Test cases
	tests := []struct {
		name      string
		errorCode string
		expected  bool
	}{
		{
			name:      "Connection exception",
			errorCode: "08000", // pgerrcode.ConnectionException
			expected:  true,
		},
		{
			name:      "Connection does not exist",
			errorCode: "08003", // pgerrcode.ConnectionDoesNotExist
			expected:  true,
		},
		{
			name:      "Connection failure",
			errorCode: "08006", // pgerrcode.ConnectionFailure
			expected:  true,
		},
		{
			name:      "Disk full",
			errorCode: "53100", // pgerrcode.DiskFull
			expected:  true,
		},
		{
			name:      "Non-retriable error",
			errorCode: "42P01", // pgerrcode.UndefinedTable
			expected:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pgErr := &pgconn.PgError{
				Code: tc.errorCode,
			}
			result := isRetriableError(pgErr)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Test for Close method.
func TestPgStorage_Close(t *testing.T) {
	// Создаем тестовый экземпляр PgStorage с nil-пулом
	storage := &PgStorage{
		conn: nil, // Устанавливаем nil, чтобы проверить безопасность Close()
		log:  zap.NewNop(),
	}

	// Проверяем, что Close() не вызывает панику даже с nil conn
	err := storage.Close()
	assert.NoError(t, err)

	// Теперь проверим с реальным пулом
	// Но поскольку мы не можем создать реальный пул без подключения к БД,
	// мы просто проверим, что метод не паникует
	t.Skip("Skipping full Close() test as it requires a real database connection")
}

// Test for Count method with a simple mock.
func TestPgStorage_Count(t *testing.T) {
	// Создаем моки
	mockPool := new(MockPgxPool)
	mockRow := new(MockPgxRow)

	// Создаем контекст
	ctx := context.Background()

	// Настраиваем ожидаемое поведение для запроса
	mockPool.On("QueryRow", ctx, "SELECT count(*) FROM metrics", []interface{}(nil)).Return(mockRow)

	// Настраиваем ожидаемое поведение для сканирования результата
	mockRow.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		// Получаем слайс аргументов
		destSlice := args.Get(0).([]interface{})
		// Устанавливаем значение счетчика
		*destSlice[0].(*int) = 42
	}).Return(nil)

	// Создаем тестовый экземпляр PgStorage с моком
	storage := &PgStorage{
		conn: mockPool,
		log:  zap.NewNop(),
	}

	// Вызываем тестируемый метод
	count := storage.Count(ctx)

	// Проверяем результат
	assert.Equal(t, 42, count)

	// Проверяем, что моки были вызваны с ожидаемыми параметрами
	mockPool.AssertExpectations(t)
	mockRow.AssertExpectations(t)

	// Тест на обработку ошибки
	mockPool2 := new(MockPgxPool)
	mockRow2 := new(MockPgxRow)

	// Настраиваем ожидаемое поведение для запроса
	mockPool2.On("QueryRow", ctx, "SELECT count(*) FROM metrics", []interface{}(nil)).Return(mockRow2)

	// Настраиваем ожидаемое поведение для сканирования результата с ошибкой
	mockRow2.On("Scan", mock.Anything).Return(errors.New("database error"))

	// Создаем тестовый экземпляр PgStorage с моком
	storage2 := &PgStorage{
		conn: mockPool2,
		log:  zap.NewNop(),
	}

	// Вызываем тестируемый метод
	count = storage2.Count(ctx)

	// Проверяем результат (должен быть 0 при ошибке)
	assert.Equal(t, 0, count)

	// Проверяем, что моки были вызваны с ожидаемыми параметрами
	mockPool2.AssertExpectations(t)
	mockRow2.AssertExpectations(t)
}

// Test for GetMetrics method.
func TestPgStorage_GetMetrics(t *testing.T) {
	// Пропускаем тест, так как он требует сложного мокирования
	t.Skip("Skipping test that requires complex mocking")
}

// Test for GetMetric method.
func TestPgStorage_GetMetric(t *testing.T) {
	// Создаем моки
	mockPool := new(MockPgxPool)
	mockRow := new(MockPgxRow)

	// Создаем контекст
	ctx := context.Background()

	// Тестовые данные
	name := "test_metric"
	mType := metrics.TypeGauge
	value := float64(42.5)

	// Настраиваем ожидаемое поведение для запроса
	mockPool.On("QueryRow", ctx,
		"SELECT name, type, value, delta FROM public.metrics WHERE name = $1 AND type = $2",
		mock.Anything, mock.Anything).Return(mockRow)

	// Настраиваем ожидаемое поведение для сканирования результата
	mockRow.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		// Получаем слайс аргументов
		destSlice := args.Get(0).([]interface{})
		// Устанавливаем значения
		*destSlice[0].(*string) = name
		*destSlice[1].(*metrics.MetricType) = mType
		*destSlice[2].(**float64) = &value
		*destSlice[3].(**int64) = nil
	}).Return(nil)

	// Создаем тестовый экземпляр PgStorage с моком
	storage := &PgStorage{
		conn: mockPool,
		log:  zap.NewNop(),
	}

	// Вызываем тестируемый метод
	metric, ok := storage.GetMetric(ctx, mType, name)

	// Проверяем результат
	assert.True(t, ok)
	assert.Equal(t, name, metric.Name)
	assert.Equal(t, mType, metric.MType)
	assert.Equal(t, &value, metric.Value)
	assert.Nil(t, metric.Delta)

	// Проверяем, что моки были вызваны с ожидаемыми параметрами
	mockPool.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

// Test for GetCounter method.
func TestPgStorage_GetCounter(t *testing.T) {
	t.Skip("Skipping test that requires complex mocking")
}

// Test for GetGauge method.
func TestPgStorage_GetGauge(t *testing.T) {
	t.Skip("Skipping test that requires complex mocking")
}

// Test for UpdateMetric method.
func TestPgStorage_UpdateMetric(t *testing.T) {
	// Создаем моки
	mockPool := new(MockPgxPool)
	mockRow := new(MockPgxRow)

	// Создаем контекст
	ctx := context.Background()

	// Тестовые данные
	name := "test_metric"
	mType := metrics.TypeGauge
	value := float64(42.5)

	// Создаем тестовую метрику
	metric := metrics.Metric{
		Name:  name,
		MType: mType,
		Value: &value,
		Delta: nil,
	}

	// Настраиваем ожидаемое поведение для запроса GetMetric
	mockPool.On("QueryRow", ctx,
		"SELECT name, type, value, delta FROM public.metrics WHERE name = $1 AND type = $2",
		mock.Anything, mock.Anything).Return(mockRow)

	// Настраиваем ожидаемое поведение для сканирования результата (метрика не найдена)
	mockRow.On("Scan", mock.Anything).Return(pgx.ErrNoRows)

	// Настраиваем ожидаемое поведение для запроса INSERT
	mockPool.On("Exec", ctx,
		"INSERT INTO metrics (name, type, value, delta) VALUES ($1, $2, $3, $4)",
		mock.Anything).Return(pgconn.CommandTag{}, nil)

	// Создаем тестовый экземпляр PgStorage с моком
	storage := &PgStorage{
		conn: mockPool,
		log:  zap.NewNop(),
	}

	// Вызываем тестируемый метод
	err := storage.UpdateMetric(ctx, metric)

	// Проверяем результат
	assert.NoError(t, err)

	// Проверяем, что моки были вызваны с ожидаемыми параметрами
	mockPool.AssertExpectations(t)
	mockRow.AssertExpectations(t)

	// Тест на случай, когда метрика уже существует
	mockPool2 := new(MockPgxPool)
	mockRow2 := new(MockPgxRow)

	// Настраиваем ожидаемое поведение для запроса GetMetric
	mockPool2.On("QueryRow", ctx,
		"SELECT name, type, value, delta FROM public.metrics WHERE name = $1 AND type = $2",
		mock.Anything, mock.Anything).Return(mockRow2)

	// Настраиваем ожидаемое поведение для сканирования результата (метрика найдена)
	mockRow2.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		// Получаем слайс аргументов
		destSlice := args.Get(0).([]interface{})
		// Устанавливаем значения
		*destSlice[0].(*string) = name
		*destSlice[1].(*metrics.MetricType) = mType
		*destSlice[2].(**float64) = &value
		*destSlice[3].(**int64) = nil
	}).Return(nil)

	// Настраиваем ожидаемое поведение для запроса UPDATE
	mockPool2.On("Exec", ctx,
		"UPDATE metrics SET value = $3, delta = delta + $4 WHERE name = $1 AND type = $2",
		mock.Anything).Return(pgconn.CommandTag{}, nil)

	// Создаем тестовый экземпляр PgStorage с моком
	storage2 := &PgStorage{
		conn: mockPool2,
		log:  zap.NewNop(),
	}

	// Вызываем тестируемый метод
	err = storage2.UpdateMetric(ctx, metric)

	// Проверяем результат
	assert.NoError(t, err)

	// Проверяем, что моки были вызваны с ожидаемыми параметрами
	mockPool2.AssertExpectations(t)
	mockRow2.AssertExpectations(t)

	// Тест на случай ошибки при выполнении запроса
	mockPool3 := new(MockPgxPool)
	mockRow3 := new(MockPgxRow)

	// Настраиваем ожидаемое поведение для запроса GetMetric
	mockPool3.On("QueryRow", ctx,
		"SELECT name, type, value, delta FROM public.metrics WHERE name = $1 AND type = $2",
		mock.Anything, mock.Anything).Return(mockRow3)

	// Настраиваем ожидаемое поведение для сканирования результата (метрика не найдена)
	mockRow3.On("Scan", mock.Anything).Return(pgx.ErrNoRows)

	// Настраиваем ожидаемое поведение для запроса INSERT с ошибкой
	mockPool3.On("Exec", ctx,
		"INSERT INTO metrics (name, type, value, delta) VALUES ($1, $2, $3, $4)",
		mock.Anything).Return(pgconn.CommandTag{}, errors.New("database error"))

	// Создаем тестовый экземпляр PgStorage с моком
	storage3 := &PgStorage{
		conn: mockPool3,
		log:  zap.NewNop(),
	}

	// Вызываем тестируемый метод
	err = storage3.UpdateMetric(ctx, metric)

	// Проверяем результат
	assert.Error(t, err)
	assert.Equal(t, "database error", err.Error())

	// Проверяем, что моки были вызваны с ожидаемыми параметрами
	mockPool3.AssertExpectations(t)
	mockRow3.AssertExpectations(t)
}

// Test for UpdateMetrics method.
func TestPgStorage_UpdateMetrics(t *testing.T) {
	t.Skip("Skipping test that requires complex mocking")
}

// Test for UpdateMetrics method with transaction error.
func TestPgStorage_UpdateMetrics_TransactionError(t *testing.T) {
	t.Skip("Skipping test that requires complex mocking")
}

// Test for isRetriableError function.
func TestIsRetriableError(t *testing.T) {
	// Тест на retriable ошибки
	retriableErrors := []string{
		pgerrcode.ConnectionException,
		pgerrcode.ConnectionDoesNotExist,
		pgerrcode.ConnectionFailure,
		pgerrcode.DiskFull,
	}

	for _, code := range retriableErrors {
		pgErr := &pgconn.PgError{
			Code: code,
		}
		assert.True(t, isRetriableError(pgErr), "Error with code %s should be retriable", code)
	}

	// Тест на non-retriable ошибки
	nonRetriableErrors := []string{
		pgerrcode.InvalidCatalogName,
		pgerrcode.InvalidPassword,
		pgerrcode.SyntaxError,
	}

	for _, code := range nonRetriableErrors {
		pgErr := &pgconn.PgError{
			Code: code,
		}
		assert.False(t, isRetriableError(pgErr), "Error with code %s should not be retriable", code)
	}
}

// Test for New function.
func TestNew(t *testing.T) {
	// Пропускаем тест, так как он требует реального подключения к базе данных
	t.Skip("Skipping test that requires real database connection")

	// Создаем контекст
	ctx := context.Background()

	// Создаем мок для конфигурации
	mockConfig := &app.Config{
		Database: app.DatabaseConfig{
			DSN: "postgres://user:password@localhost:5432/testdb",
		},
	}

	// Создаем мок для логгера
	mockLogger := zap.NewNop()

	// Создаем тестовый экземпляр PgStorage
	storage, err := New(ctx, mockConfig, mockLogger)

	// Проверяем, что ошибка не возникла
	assert.NoError(t, err)
	assert.NotNil(t, storage)

	// Проверяем, что поля инициализированы корректно
	assert.Equal(t, mockLogger, storage.log)

	// Закрываем соединение
	storage.Close()
}
