//nolint:testpackage // используется для тестирования внутреннего API
package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	"github.com/maynagashev/go-metrics/internal/grpc/pb"
	"github.com/maynagashev/go-metrics/pkg/sign"
)

// mockUnaryInvoker используется для тестирования UnaryClientInterceptor.
type mockUnaryInvoker struct {
	mock.Mock
}

func (m *mockUnaryInvoker) Invoke(
	ctx context.Context,
	method string,
	req, reply interface{},
	cc *grpc.ClientConn,
	opts ...grpc.CallOption,
) error {
	args := m.Called(ctx, method, req, reply, cc, opts)
	return args.Error(0)
}

// mockStreamer используется для тестирования StreamClientInterceptor.
type mockStreamer struct {
	mock.Mock
}

func (m *mockStreamer) NewStream(
	ctx context.Context,
	desc *grpc.StreamDesc,
	cc *grpc.ClientConn,
	method string,
	opts ...grpc.CallOption,
) (grpc.ClientStream, error) {
	args := m.Called(ctx, desc, cc, method, opts)
	stream, ok := args.Get(0).(grpc.ClientStream)
	if !ok && args.Get(0) != nil {
		// Возвращаем nil-стрим и ошибку приведения типа
		return nil, assert.AnError
	}
	return stream, args.Error(1)
}

// mockClientStream используется как возвращаемое значение из mockStreamer.
type mockClientStream struct {
	mock.Mock
	grpc.ClientStream
}

// TestSigningInterceptor проверяет функцию SigningInterceptor.
func TestSigningInterceptor(t *testing.T) {
	t.Run("with private key", func(t *testing.T) {
		// Создаем перехватчик с ключом
		privateKey := "test-key"
		interceptor := SigningInterceptor(privateKey)
		require.NotNil(t, interceptor)

		// Создаем объект запроса
		req := &pb.UpdateRequest{
			Metric: &pb.Metric{
				Name:  "test-metric",
				Type:  pb.MetricType_GAUGE,
				Value: proto.Float64(123.45),
			},
		}

		// Создаем мок для invoker
		mockInvoker := &mockUnaryInvoker{}
		mockInvoker.On("Invoke", mock.Anything, "TestMethod", req, mock.Anything, mock.Anything, mock.Anything).
			Return(nil)

		// Выполняем перехватчик
		ctx := context.Background()
		err := interceptor(ctx, "TestMethod", req, nil, nil, mockInvoker.Invoke)
		require.NoError(t, err)

		// Проверяем, что invoker был вызван
		mockInvoker.AssertExpectations(t)

		// Проверяем, что в контексте есть подпись
		ctxArg, ok := mockInvoker.Calls[0].Arguments[0].(context.Context)
		require.True(t, ok, "First argument should be a context.Context")
		md, ok := metadata.FromOutgoingContext(ctxArg)
		require.True(t, ok, "Metadata should be present in context")

		signatures := md.Get(SignatureHeader)
		require.NotEmpty(t, signatures, "Signature header should be present")

		// Проверяем, что подпись была создана корректно
		reqBytes, err := proto.Marshal(req)
		require.NoError(t, err)
		expectedSignature := sign.ComputeHMACSHA256(reqBytes, privateKey)
		assert.Equal(t, expectedSignature, signatures[0])
	})

	t.Run("without private key", func(t *testing.T) {
		// Создаем перехватчик без ключа
		interceptor := SigningInterceptor("")
		require.NotNil(t, interceptor)

		// Создаем объект запроса
		req := &pb.UpdateRequest{
			Metric: &pb.Metric{
				Name:  "test-metric",
				Type:  pb.MetricType_GAUGE,
				Value: proto.Float64(123.45),
			},
		}

		// Создаем мок для invoker
		mockInvoker := &mockUnaryInvoker{}
		mockInvoker.On("Invoke", mock.Anything, "TestMethod", req, mock.Anything, mock.Anything, mock.Anything).
			Return(nil)

		// Выполняем перехватчик
		ctx := context.Background()
		err := interceptor(ctx, "TestMethod", req, nil, nil, mockInvoker.Invoke)
		require.NoError(t, err)

		// Проверяем, что invoker был вызван
		mockInvoker.AssertExpectations(t)

		// Проверяем, что в контексте нет подписи
		ctxArg, ok := mockInvoker.Calls[0].Arguments[0].(context.Context)
		require.True(t, ok, "First argument should be a context.Context")
		md, ok := metadata.FromOutgoingContext(ctxArg)

		if ok {
			signatures := md.Get(SignatureHeader)
			assert.Empty(t, signatures, "Signature header should not be present")
		}
	})

	t.Run("with non-proto message", func(t *testing.T) {
		// Создаем перехватчик с ключом
		privateKey := "test-key"
		interceptor := SigningInterceptor(privateKey)

		// Создаем объект, который не является proto.Message
		req := struct {
			Name string
		}{
			Name: "not a proto message",
		}

		// Создаем мок для invoker
		mockInvoker := &mockUnaryInvoker{}
		mockInvoker.On("Invoke", mock.Anything, "TestMethod", req, mock.Anything, mock.Anything, mock.Anything).
			Return(nil)

		// Выполняем перехватчик
		ctx := context.Background()
		err := interceptor(ctx, "TestMethod", req, nil, nil, mockInvoker.Invoke)
		require.NoError(t, err)

		// Проверяем, что invoker был вызван
		mockInvoker.AssertExpectations(t)
	})
}

// TestStreamSigningInterceptor проверяет функцию StreamSigningInterceptor.
func TestStreamSigningInterceptor(t *testing.T) {
	t.Run("with private key", func(t *testing.T) {
		// Создаем перехватчик с ключом
		privateKey := "test-key"
		interceptor := StreamSigningInterceptor(privateKey)
		require.NotNil(t, interceptor)

		// Создаем метод
		method := "TestStreamMethod"

		// Создаем мок для streamer
		mockClientStream := &mockClientStream{}
		mockStreamer := &mockStreamer{}
		mockStreamer.On("NewStream", mock.Anything, mock.Anything, mock.Anything, method, mock.Anything).
			Return(mockClientStream, nil)

		// Выполняем перехватчик
		ctx := context.Background()
		_, err := interceptor(ctx, nil, nil, method, mockStreamer.NewStream)
		require.NoError(t, err)

		// Проверяем, что streamer был вызван
		mockStreamer.AssertExpectations(t)

		// Проверяем, что в контексте есть подпись
		ctxArg, ok := mockStreamer.Calls[0].Arguments[0].(context.Context)
		require.True(t, ok, "First argument should be a context.Context")
		md, ok := metadata.FromOutgoingContext(ctxArg)
		require.True(t, ok, "Metadata should be present in context")

		signatures := md.Get(SignatureHeader)
		require.NotEmpty(t, signatures, "Signature header should be present")

		// Проверяем, что подпись была создана корректно
		expectedSignature := sign.ComputeHMACSHA256([]byte(method), privateKey)
		assert.Equal(t, expectedSignature, signatures[0])
	})

	t.Run("without private key", func(t *testing.T) {
		// Создаем перехватчик без ключа
		interceptor := StreamSigningInterceptor("")
		require.NotNil(t, interceptor)

		// Создаем метод
		method := "TestStreamMethod"

		// Создаем мок для streamer
		mockClientStream := &mockClientStream{}
		mockStreamer := &mockStreamer{}
		mockStreamer.On("NewStream", mock.Anything, mock.Anything, mock.Anything, method, mock.Anything).
			Return(mockClientStream, nil)

		// Выполняем перехватчик
		ctx := context.Background()
		_, err := interceptor(ctx, nil, nil, method, mockStreamer.NewStream)
		require.NoError(t, err)

		// Проверяем, что streamer был вызван
		mockStreamer.AssertExpectations(t)

		// Проверяем, что в контексте нет подписи
		ctxArg, ok := mockStreamer.Calls[0].Arguments[0].(context.Context)
		require.True(t, ok, "First argument should be a context.Context")
		md, ok := metadata.FromOutgoingContext(ctxArg)

		if ok {
			signatures := md.Get(SignatureHeader)
			assert.Empty(t, signatures, "Signature header should not be present")
		}
	})
}
