package grpc

import (
	"context"

	"github.com/maynagashev/go-metrics/pkg/sign"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

// Константа для имени заголовка с подписью, совпадающего с HTTP.
const SignatureHeader = sign.HeaderKey

// SigningInterceptor создает перехватчик для подписи исходящих gRPC запросов.
func SigningInterceptor(privateKey string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Если ключ не указан, не подписываем запрос
		if privateKey == "" {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		// Сериализуем запрос в байты
		reqMsg, ok := req.(proto.Message)
		if !ok {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		reqBytes, err := proto.Marshal(reqMsg)
		if err != nil {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		// Вычисляем HMAC-SHA256 с использованием пакета sign
		signature := sign.ComputeHMACSHA256(reqBytes, privateKey)

		// Добавляем подпись в метаданные запроса
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}
		md.Set(SignatureHeader, signature)
		newCtx := metadata.NewOutgoingContext(ctx, md)

		// Вызываем следующий обработчик с обновленным контекстом
		return invoker(newCtx, method, req, reply, cc, opts...)
	}
}

// StreamSigningInterceptor создает перехватчик для подписи потоковых gRPC запросов.
func StreamSigningInterceptor(privateKey string) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		// Если ключ не указан, не подписываем запрос
		if privateKey == "" {
			return streamer(ctx, desc, cc, method, opts...)
		}

		// Для потоковых запросов подписываем метод
		methodBytes := []byte(method)
		signature := sign.ComputeHMACSHA256(methodBytes, privateKey)

		// Добавляем подпись в метаданные потока
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}
		md.Set(SignatureHeader, signature)
		newCtx := metadata.NewOutgoingContext(ctx, md)

		// Создаем поток с обновленным контекстом
		return streamer(newCtx, desc, cc, method, opts...)
	}
}
