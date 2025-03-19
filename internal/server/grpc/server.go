// Package grpc реализует gRPC-сервер для сбора метрик.
package grpc

import (
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip" // Автоматически регистрирует gzip компрессор при импорте
	"google.golang.org/grpc/keepalive"

	"github.com/maynagashev/go-metrics/internal/grpc/pb"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage"
)

// Константы для параметров keepalive.
const (
	// Keepalive enforcement policy.
	MinPingTime         = 5 * time.Second // минимальное время между ping от клиента
	PermitWithoutStream = true            // разрешить keepalive без активных потоков

	// Keepalive server parameters.
	MaxConnectionIdleTime     = 15 * time.Second // максимальное время простоя соединения
	MaxConnectionAgeTime      = 30 * time.Second // максимальное время жизни соединения
	MaxConnectionAgeGraceTime = 5 * time.Second  // grace период перед принудительным закрытием
	PingTime                  = 5 * time.Second  // интервал для ping от сервера
	PingTimeout               = 1 * time.Second  // таймаут для ping

	// Default maximum number of concurrent streams.
	DefaultMaxConcurrentStreams uint32 = 100 // значение по умолчанию для максимального количества одновременных потоков
)

// Server представляет gRPC сервер.
type Server struct {
	cfg         *app.Config
	log         *zap.Logger
	storage     storage.Repository
	grpcServer  *grpc.Server
	metricsServ *MetricsService
}

// New создает новый экземпляр gRPC сервера.
func New(log *zap.Logger, cfg *app.Config, storage storage.Repository) *Server {
	metricsService := NewMetricsService(log, cfg, storage)

	return &Server{
		cfg:         cfg,
		log:         log,
		storage:     storage,
		metricsServ: metricsService,
	}
}

// Start запускает gRPC сервер.
func (s *Server) Start() error {
	// Проверяем, включен ли gRPC
	if !s.cfg.IsGRPCEnabled() {
		s.log.Info("gRPC server disabled, skip start")
		return nil
	}

	// Создаем слушатель TCP
	addr := s.cfg.GRPC.Addr
	s.log.Info("starting gRPC server", zap.String("address", addr))
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// Создаем параметры keepalive
	kaep := keepalive.EnforcementPolicy{
		MinTime:             MinPingTime,
		PermitWithoutStream: PermitWithoutStream,
	}

	kasp := keepalive.ServerParameters{
		MaxConnectionIdle:     MaxConnectionIdleTime,
		MaxConnectionAge:      MaxConnectionAgeTime,
		MaxConnectionAgeGrace: MaxConnectionAgeGraceTime,
		Time:                  PingTime,
		Timeout:               PingTimeout,
	}

	// Определяем максимальное количество одновременных потоков
	maxConnections := DefaultMaxConcurrentStreams

	// Безопасное использование значения из конфигурации, если оно положительное и в пределах uint32
	if s.cfg.GRPC.MaxConn > 0 && s.cfg.GRPC.MaxConn <= int(DefaultMaxConcurrentStreams) {
		// Просто копируем значение - безопасно, т.к. уже проверили, что значение в допустимых пределах
		//nolint:gosec // G115: проверка на допустимые значения выполнена выше
		maxConnections = uint32(s.cfg.GRPC.MaxConn)
	} else if s.cfg.GRPC.MaxConn > int(DefaultMaxConcurrentStreams) {
		s.log.Warn("MaxConn value exceeds safe limit, using default",
			zap.Int("configured", s.cfg.GRPC.MaxConn),
			zap.Uint32("using", DefaultMaxConcurrentStreams))
	}

	// Настраиваем опции сервера
	opts := []grpc.ServerOption{
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),
		grpc.MaxConcurrentStreams(maxConnections),
	}

	// Создаем gRPC сервер
	s.grpcServer = grpc.NewServer(opts...)

	// Регистрируем сервис метрик
	pb.RegisterMetricsServiceServer(s.grpcServer, s.metricsServ)

	// Запускаем сервер в отдельной горутине
	go func() {
		serveErr := s.grpcServer.Serve(lis)
		if serveErr != nil {
			s.log.Error("failed to serve gRPC", zap.Error(serveErr))
		}
	}()

	s.log.Info("gRPC server started", zap.String("address", addr))
	return nil
}

// Stop останавливает gRPC сервер.
func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.log.Info("stopping gRPC server")
		s.grpcServer.GracefulStop()
		s.log.Info("gRPC server stopped")
	}
}
