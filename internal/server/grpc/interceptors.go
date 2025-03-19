package grpc

import (
	"context"
	"errors"

	"github.com/maynagashev/go-metrics/pkg/sign"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	// ErrNoSignature ошибка при отсутствии подписи в запросе.
	ErrNoSignature = errors.New("no signature in request metadata")
	// ErrInvalidSignature ошибка при невалидной подписи.
	ErrInvalidSignature = errors.New("invalid signature")
)

// Константа для имени заголовка с подписью, совпадающего с HTTP.
const SignatureHeader = sign.HeaderKey

// SignatureValidatorInterceptor создает перехватчик для проверки подписи входящих gRPC запросов.
func SignatureValidatorInterceptor(log *zap.Logger, privateKey string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Если ключ не указан, не проверяем подпись
		if privateKey == "" {
			return handler(ctx, req)
		}

		// Извлекаем метаданные запроса
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Debug("no metadata in request")
			return nil, status.Errorf(codes.InvalidArgument, "missing metadata")
		}

		// Получаем подпись из метаданных
		signatures := md.Get(SignatureHeader)
		if len(signatures) == 0 {
			log.Debug("no signature in metadata")
			return nil, status.Errorf(codes.Unauthenticated, "missing signature")
		}
		receivedSignature := signatures[0]

		// Сериализуем запрос в байты
		reqMsg, ok := req.(proto.Message)
		if !ok {
			log.Error("request is not a proto message")
			return nil, status.Errorf(codes.Internal, "invalid request format")
		}

		reqBytes, err := proto.Marshal(reqMsg)
		if err != nil {
			log.Error("failed to marshal request", zap.Error(err))
			return nil, status.Errorf(codes.Internal, "failed to process request")
		}

		// Проверяем подпись с использованием пакета sign
		expectedSignature, err := sign.VerifyHMACSHA256(reqBytes, privateKey, receivedSignature)
		if err != nil {
			log.Warn("invalid signature",
				zap.String("received", receivedSignature),
				zap.String("expected", expectedSignature),
				zap.Error(err))
			return nil, status.Errorf(codes.Unauthenticated, "invalid signature")
		}

		// Если подпись валидна, продолжаем обработку запроса
		return handler(ctx, req)
	}
}

// StreamSignatureValidatorInterceptor создает перехватчик для проверки подписи входящих потоковых gRPC запросов.
func StreamSignatureValidatorInterceptor(log *zap.Logger, privateKey string) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Если ключ не указан, не проверяем подпись
		if privateKey == "" {
			return handler(srv, ss)
		}

		// Извлекаем метаданные запроса
		ctx := ss.Context()
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Debug("no metadata in stream request")
			return status.Errorf(codes.InvalidArgument, "missing metadata")
		}

		// Получаем подпись из метаданных
		signatures := md.Get(SignatureHeader)
		if len(signatures) == 0 {
			log.Debug("no signature in stream metadata")
			return status.Errorf(codes.Unauthenticated, "missing signature")
		}
		receivedSignature := signatures[0]

		// Для потоковых запросов подписываем метод
		methodBytes := []byte(info.FullMethod)

		// Проверяем подпись метода
		expectedSignature, err := sign.VerifyHMACSHA256(methodBytes, privateKey, receivedSignature)
		if err != nil {
			log.Warn("invalid stream signature",
				zap.String("method", info.FullMethod),
				zap.String("received", receivedSignature),
				zap.String("expected", expectedSignature),
				zap.Error(err))
			return status.Errorf(codes.Unauthenticated, "invalid signature")
		}

		// Если подпись валидна, продолжаем обработку потока
		return handler(srv, ss)
	}
}
