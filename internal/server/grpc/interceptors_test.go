//nolint:testpackage // используется для тестирования внутреннего API
package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/maynagashev/go-metrics/internal/grpc/pb"
	"github.com/maynagashev/go-metrics/pkg/sign"
)

// MockUnaryHandler is a mock for grpc.UnaryHandler.
type MockUnaryHandler struct {
	mock.Mock
}

func (m *MockUnaryHandler) Handle(ctx context.Context, req interface{}) (interface{}, error) {
	args := m.Called(ctx, req)
	return args.Get(0), args.Error(1)
}

// MockServerStream is a mock for grpc.ServerStream.
type MockServerStream struct {
	mock.Mock
	ctx context.Context
	grpc.ServerStream
}

func (m *MockServerStream) Context() context.Context {
	return m.ctx
}

// MockStreamHandler is a mock for grpc.StreamHandler.
type MockStreamHandler struct {
	mock.Mock
}

func (m *MockStreamHandler) Handle(srv interface{}, stream grpc.ServerStream) error {
	args := m.Called(srv, stream)
	return args.Error(0)
}

// TestSignatureValidatorInterceptor tests the SignatureValidatorInterceptor function.
func TestSignatureValidatorInterceptor(t *testing.T) {
	logger := zap.NewNop()

	t.Run("private key empty", func(t *testing.T) {
		// Create a mock handler
		mockHandler := &MockUnaryHandler{}
		mockHandler.On("Handle", mock.Anything, mock.Anything).Return("response", nil)

		// Create the interceptor with empty key
		interceptor := SignatureValidatorInterceptor(logger, "")

		// Create a test request
		req := &pb.Metric{Name: "metric1", Type: pb.MetricType_GAUGE}
		ctx := context.Background()

		// Call the interceptor
		resp, err := interceptor(ctx, req, nil, mockHandler.Handle)

		// Verify results
		require.NoError(t, err)
		assert.Equal(t, "response", resp)
		mockHandler.AssertExpectations(t)
	})

	t.Run("no metadata", func(t *testing.T) {
		// Create a mock handler
		mockHandler := &MockUnaryHandler{}

		// Create the interceptor with a key
		interceptor := SignatureValidatorInterceptor(logger, "test-key")

		// Create a test request without metadata
		req := &pb.Metric{Name: "metric1", Type: pb.MetricType_GAUGE}
		ctx := context.Background()

		// Call the interceptor
		resp, err := interceptor(ctx, req, nil, mockHandler.Handle)

		// Verify results
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Nil(t, resp)
		mockHandler.AssertNotCalled(t, "Handle")
	})

	t.Run("no signature", func(t *testing.T) {
		// Create a mock handler
		mockHandler := &MockUnaryHandler{}

		// Create the interceptor with a key
		interceptor := SignatureValidatorInterceptor(logger, "test-key")

		// Create a test request with metadata but no signature
		req := &pb.Metric{Name: "metric1", Type: pb.MetricType_GAUGE}
		md := metadata.New(map[string]string{
			"other-header": "value",
		})
		ctx := metadata.NewIncomingContext(context.Background(), md)

		// Call the interceptor
		resp, err := interceptor(ctx, req, nil, mockHandler.Handle)

		// Verify results
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
		assert.Nil(t, resp)
		mockHandler.AssertNotCalled(t, "Handle")
	})

	t.Run("invalid signature", func(t *testing.T) {
		// Create a mock handler
		mockHandler := &MockUnaryHandler{}

		// Create the interceptor with a key
		interceptor := SignatureValidatorInterceptor(logger, "test-key")

		// Create a test request with metadata and an invalid signature
		req := &pb.Metric{Name: "metric1", Type: pb.MetricType_GAUGE}
		md := metadata.New(map[string]string{
			string(SignatureHeader): "invalid-signature",
		})
		ctx := metadata.NewIncomingContext(context.Background(), md)

		// Call the interceptor
		resp, err := interceptor(ctx, req, nil, mockHandler.Handle)

		// Verify results
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
		assert.Nil(t, resp)
		mockHandler.AssertNotCalled(t, "Handle")
	})

	t.Run("valid signature", func(t *testing.T) {
		// Create a mock handler
		mockHandler := &MockUnaryHandler{}
		mockHandler.On("Handle", mock.Anything, mock.Anything).Return("response", nil)

		// Test key
		testKey := "test-secret-key"

		// Create the interceptor with a key
		interceptor := SignatureValidatorInterceptor(logger, testKey)

		// Create a test request
		req := &pb.Metric{Name: "metric1", Type: pb.MetricType_GAUGE}

		// Marshal the request to generate a valid signature
		reqBytes, err := proto.Marshal(req)
		require.NoError(t, err)

		// Generate a valid signature
		validSignature := sign.ComputeHMACSHA256(reqBytes, testKey)

		// Create context with valid signature
		md := metadata.New(map[string]string{
			string(SignatureHeader): validSignature,
		})
		ctx := metadata.NewIncomingContext(context.Background(), md)

		// Call the interceptor
		resp, err := interceptor(ctx, req, nil, mockHandler.Handle)

		// Verify results
		require.NoError(t, err)
		assert.Equal(t, "response", resp)
		mockHandler.AssertExpectations(t)
	})

	t.Run("non-proto message", func(t *testing.T) {
		// Create a mock handler
		mockHandler := &MockUnaryHandler{}

		// Create the interceptor with a key
		interceptor := SignatureValidatorInterceptor(logger, "test-key")

		// Create a non-proto message request
		req := "not a proto message"

		// Create context with signature
		md := metadata.New(map[string]string{
			string(SignatureHeader): "some-signature",
		})
		ctx := metadata.NewIncomingContext(context.Background(), md)

		// Call the interceptor
		resp, err := interceptor(ctx, req, nil, mockHandler.Handle)

		// Verify results
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, s.Code())
		assert.Nil(t, resp)
		mockHandler.AssertNotCalled(t, "Handle")
	})
}

// TestStreamSignatureValidatorInterceptor tests the StreamSignatureValidatorInterceptor function.
func TestStreamSignatureValidatorInterceptor(t *testing.T) {
	logger := zap.NewNop()

	t.Run("private key empty", func(t *testing.T) {
		// Create a mock stream handler
		mockHandler := &MockStreamHandler{}
		mockHandler.On("Handle", mock.Anything, mock.Anything).Return(nil)

		// Create stream info
		streamInfo := &grpc.StreamServerInfo{
			FullMethod: "/test.Service/TestStream",
		}

		// Create a mock stream
		mockStream := &MockServerStream{
			ctx: context.Background(),
		}

		// Create the interceptor with empty key
		interceptor := StreamSignatureValidatorInterceptor(logger, "")

		// Call the interceptor
		err := interceptor(nil, mockStream, streamInfo, mockHandler.Handle)

		// Verify results
		require.NoError(t, err)
		mockHandler.AssertExpectations(t)
	})

	t.Run("no metadata", func(t *testing.T) {
		// Create a mock stream handler
		mockHandler := &MockStreamHandler{}

		// Create stream info
		streamInfo := &grpc.StreamServerInfo{
			FullMethod: "/test.Service/TestStream",
		}

		// Create a mock stream without metadata
		mockStream := &MockServerStream{
			ctx: context.Background(),
		}

		// Create the interceptor with a key
		interceptor := StreamSignatureValidatorInterceptor(logger, "test-key")

		// Call the interceptor
		err := interceptor(nil, mockStream, streamInfo, mockHandler.Handle)

		// Verify results
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		mockHandler.AssertNotCalled(t, "Handle")
	})

	t.Run("no signature", func(t *testing.T) {
		// Create a mock stream handler
		mockHandler := &MockStreamHandler{}

		// Create stream info
		streamInfo := &grpc.StreamServerInfo{
			FullMethod: "/test.Service/TestStream",
		}

		// Create a mock stream with metadata but no signature
		md := metadata.New(map[string]string{
			"other-header": "value",
		})
		ctx := metadata.NewIncomingContext(context.Background(), md)
		mockStream := &MockServerStream{
			ctx: ctx,
		}

		// Create the interceptor with a key
		interceptor := StreamSignatureValidatorInterceptor(logger, "test-key")

		// Call the interceptor
		err := interceptor(nil, mockStream, streamInfo, mockHandler.Handle)

		// Verify results
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
		mockHandler.AssertNotCalled(t, "Handle")
	})

	t.Run("invalid signature", func(t *testing.T) {
		// Create a mock stream handler
		mockHandler := &MockStreamHandler{}

		// Create stream info
		streamInfo := &grpc.StreamServerInfo{
			FullMethod: "/test.Service/TestStream",
		}

		// Create a mock stream with metadata and an invalid signature
		md := metadata.New(map[string]string{
			string(SignatureHeader): "invalid-signature",
		})
		ctx := metadata.NewIncomingContext(context.Background(), md)
		mockStream := &MockServerStream{
			ctx: ctx,
		}

		// Create the interceptor with a key
		interceptor := StreamSignatureValidatorInterceptor(logger, "test-key")

		// Call the interceptor
		err := interceptor(nil, mockStream, streamInfo, mockHandler.Handle)

		// Verify results
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
		mockHandler.AssertNotCalled(t, "Handle")
	})

	t.Run("valid signature", func(t *testing.T) {
		// Create a mock stream handler
		mockHandler := &MockStreamHandler{}
		mockHandler.On("Handle", mock.Anything, mock.Anything).Return(nil)

		// Test key
		testKey := "test-secret-key"

		// Create stream info
		method := "/test.Service/TestStream"
		streamInfo := &grpc.StreamServerInfo{
			FullMethod: method,
		}

		// Generate a valid signature for the method name
		methodBytes := []byte(method)
		validSignature := sign.ComputeHMACSHA256(methodBytes, testKey)

		// Create a mock stream with valid signature
		md := metadata.New(map[string]string{
			string(SignatureHeader): validSignature,
		})
		ctx := metadata.NewIncomingContext(context.Background(), md)
		mockStream := &MockServerStream{
			ctx: ctx,
		}

		// Create the interceptor with a key
		interceptor := StreamSignatureValidatorInterceptor(logger, testKey)

		// Call the interceptor
		err := interceptor(nil, mockStream, streamInfo, mockHandler.Handle)

		// Verify results
		require.NoError(t, err)
		mockHandler.AssertExpectations(t)
	})
}
