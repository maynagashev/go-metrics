// Package grpc реализует gRPC-сервер для сбора метрик.
package grpc

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

	// Создаем и настраиваем сервер
	configErr := s.configureAndStartServer(lis, addr)
	if configErr != nil {
		return configErr
	}

	return nil
}

// configureAndStartServer создает и запускает gRPC сервер.
func (s *Server) configureAndStartServer(lis net.Listener, addr string) error {
	// Параметры сервера
	opts := s.createServerOptions()

	// Добавляем TLS если необходимо
	cryptoKeyPath := s.cfg.GetCryptoKeyPath()
	if err := s.configureTLS(cryptoKeyPath, &opts); err != nil {
		return err
	}

	// Создаем gRPC сервер
	s.grpcServer = grpc.NewServer(opts...)

	// Регистрируем сервисы
	pb.RegisterMetricsServiceServer(s.grpcServer, s.metricsServ)

	// Запускаем сервер в отдельной горутине
	go func() {
		if serveErr := s.grpcServer.Serve(lis); serveErr != nil {
			s.log.Error("gRPC server error", zap.Error(serveErr))
		}
	}()

	s.log.Info("gRPC server started successfully",
		zap.String("address", addr),
		zap.Uint32("max_connections", s.getMaxConnections()),
		zap.Bool("request_signing_enabled", s.cfg.IsRequestSigningEnabled()),
		zap.Bool("tls_enabled", cryptoKeyPath != ""))

	return nil
}

// createServerOptions создает и возвращает опции для gRPC сервера.
func (s *Server) createServerOptions() []grpc.ServerOption {
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

	// Создаем перехватчики для проверки безопасности запросов
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		SignatureValidatorInterceptor(
			s.log,
			s.cfg.PrivateKey,
		), // Добавляем перехватчик для проверки подписи
	}

	streamInterceptors := []grpc.StreamServerInterceptor{
		StreamSignatureValidatorInterceptor(
			s.log,
			s.cfg.PrivateKey,
		), // Добавляем перехватчик для проверки подписи потоков
	}

	// Настраиваем опции сервера
	return []grpc.ServerOption{
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),
		grpc.MaxConcurrentStreams(s.getMaxConnections()),
		grpc.ChainUnaryInterceptor(
			unaryInterceptors...), // Добавляем цепочку унарных перехватчиков
		grpc.ChainStreamInterceptor(
			streamInterceptors...), // Добавляем цепочку потоковых перехватчиков
	}
}

// getMaxConnections возвращает максимальное количество одновременных потоков.
func (s *Server) getMaxConnections() uint32 {
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

	return maxConnections
}

// configureTLS настраивает TLS для gRPC сервера.
func (s *Server) configureTLS(cryptoKeyPath string, opts *[]grpc.ServerOption) error {
	if cryptoKeyPath != "" {
		s.log.Info("loading TLS credentials", zap.String("key_path", cryptoKeyPath))
		// Загружаем сертификат и ключ сервера
		creds, tlsErr := loadTLSCredentials(cryptoKeyPath)
		if tlsErr != nil {
			s.log.Error("failed to load TLS credentials", zap.Error(tlsErr))
			return fmt.Errorf("failed to load TLS credentials: %w", tlsErr)
		}

		// Добавляем TLS credentials в опции сервера
		*opts = append(*opts, grpc.Creds(creds))
		s.log.Info("TLS encryption enabled for gRPC server", zap.String("key_path", cryptoKeyPath))
	} else {
		s.log.Warn("TLS encryption disabled for gRPC server, using insecure connection")
	}

	return nil
}

// loadTLSCredentials загружает TLS креды для защищенного соединения..
func loadTLSCredentials(keyFile string) (credentials.TransportCredentials, error) {
	// Создаем сертификат из публичного ключа, извлеченного из приватного ключа
	certFile := "server.crt"

	// Загружаем сертификат и приватный ключ сервера
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate and key: %w", err)
	}

	// Создаем TLS конфигурацию
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.NoClientCert,
		MinVersion:   tls.VersionTLS12,
	}

	return credentials.NewTLS(config), nil
}

// Stop останавливает gRPC сервер.
func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.log.Info("stopping gRPC server")
		s.grpcServer.GracefulStop()
		s.log.Info("gRPC server stopped")
	}
}
