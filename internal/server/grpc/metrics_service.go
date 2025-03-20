package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/maynagashev/go-metrics/internal/contracts/metrics"
	"github.com/maynagashev/go-metrics/internal/grpc/pb"
	"github.com/maynagashev/go-metrics/internal/server/app"
	"github.com/maynagashev/go-metrics/internal/server/storage"
)

// MetricsService реализует интерфейс MetricsServiceServer из прото-файла.
type MetricsService struct {
	pb.UnimplementedMetricsServiceServer
	log     *zap.Logger
	cfg     *app.Config
	storage storage.Repository
}

// NewMetricsService создает новый сервис метрик для gRPC.
func NewMetricsService(
	log *zap.Logger,
	cfg *app.Config,
	storage storage.Repository,
) *MetricsService {
	return &MetricsService{
		log:     log,
		cfg:     cfg,
		storage: storage,
	}
}

// Update обрабатывает запрос на обновление одной метрики.
func (s *MetricsService) Update(
	ctx context.Context,
	req *pb.UpdateRequest,
) (*pb.MetricResponse, error) {
	if req == nil || req.GetMetric() == nil {
		return nil, status.Errorf(codes.InvalidArgument, "metric is required")
	}

	// Преобразуем метрику из protobuf в доменную модель
	metric, err := s.protoToMetric(req.GetMetric())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid metric: %v", err)
	}

	// Сохраняем метрику в хранилище
	updateErr := s.storage.UpdateMetric(ctx, *metric)
	if updateErr != nil {
		s.log.Error(
			"failed to update metric",
			zap.Error(updateErr),
			zap.String("name", metric.Name),
		)
		return nil, status.Errorf(codes.Internal, "failed to update metric: %v", updateErr)
	}

	s.log.Debug("metric updated via gRPC",
		zap.String("name", metric.Name),
		zap.String("type", string(metric.MType)))

	// Преобразуем метрику из доменной модели обратно в protobuf
	response := &pb.MetricResponse{
		Metric: req.GetMetric(), // отправляем обратно полученную метрику
	}

	return response, nil
}

// UpdateBatch обрабатывает запрос на пакетное обновление метрик.
func (s *MetricsService) UpdateBatch(
	ctx context.Context,
	req *pb.UpdateBatchRequest,
) (*pb.UpdateBatchResponse, error) {
	if req == nil || len(req.GetMetrics()) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "metrics are required")
	}

	// Преобразуем метрики из protobuf в доменную модель
	metricsToUpdate := make([]metrics.Metric, 0, len(req.GetMetrics()))
	for _, protoMetric := range req.GetMetrics() {
		metric, err := s.protoToMetric(protoMetric)
		if err != nil {
			s.log.Error("invalid metric", zap.Error(err), zap.String("name", protoMetric.GetName()))
			continue // Пропускаем невалидные метрики
		}
		metricsToUpdate = append(metricsToUpdate, *metric)
	}

	// Если все метрики оказались невалидными
	if len(metricsToUpdate) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "no valid metrics provided")
	}

	// Сохраняем метрики в хранилище
	if err := s.storage.UpdateMetrics(ctx, metricsToUpdate); err != nil {
		s.log.Error("failed to update metrics batch", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to update metrics batch: %v", err)
	}

	s.log.Debug("metrics batch updated via gRPC", zap.Int("count", len(metricsToUpdate)))

	return &pb.UpdateBatchResponse{
		Success: true,
	}, nil
}

// GetValue получает значение метрики по имени и типу.
func (s *MetricsService) GetValue(
	ctx context.Context,
	req *pb.GetValueRequest,
) (*pb.MetricResponse, error) {
	if req == nil || req.GetName() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "metric name is required")
	}

	// Преобразуем тип метрики из protobuf в доменную модель
	var metricType metrics.MetricType
	switch req.GetType() {
	case pb.MetricType_GAUGE:
		metricType = metrics.TypeGauge
	case pb.MetricType_COUNTER:
		metricType = metrics.TypeCounter
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid metric type")
	}

	// Получаем метрику из хранилища
	metric, found := s.storage.GetMetric(ctx, metricType, req.GetName())
	if !found {
		return nil, status.Errorf(codes.NotFound, "metric not found")
	}

	// Преобразуем метрику из доменной модели в protobuf
	protoMetric := s.metricToProto(&metric)

	return &pb.MetricResponse{
		Metric: protoMetric,
	}, nil
}

// Ping проверяет соединение с базой данных.
func (s *MetricsService) Ping(ctx context.Context, _ *pb.PingRequest) (*pb.PingResponse, error) {
	// Проверяем, включена ли база данных
	if !s.cfg.IsDatabaseEnabled() {
		return &pb.PingResponse{
			Success: false,
			Error:   "database not configured",
		}, nil
	}

	// Для проверки соединения с базой данных не используем Ping, так как его нет в интерфейсе
	// Вместо этого просто попробуем получить количество метрик
	_ = s.storage.Count(ctx)

	return &pb.PingResponse{
		Success: true,
	}, nil
}

// StreamMetrics обрабатывает потоковую отправку метрик от клиента.
// Этот метод реализует потоковую передачу клиент -> сервер (client streaming RPC),
// что позволяет отправлять большие объемы метрик без создания большого JSON-объекта
// в памяти и снижает накладные расходы на обработку HTTP-запросов.
// Метод собирает метрики из потока, буферизует их и затем сохраняет в хранилище.
func (s *MetricsService) StreamMetrics(stream pb.MetricsService_StreamMetricsServer) error {
	ctx := stream.Context()
	// Создаем буфер для метрик
	var metricsBuffer []metrics.Metric

	// Обрабатываем поток метрик
	for {
		// Получаем метрику из потока
		protoMetric, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// Конец потока, сохраняем накопленные метрики
			break
		}
		if err != nil {
			s.log.Error("error receiving stream", zap.Error(err))
			return status.Errorf(codes.Internal, "failed to receive metric: %v", err)
		}

		// Преобразуем метрику из protobuf в доменную модель
		metric, err := s.protoToMetric(protoMetric)
		if err != nil {
			s.log.Error(
				"invalid metric in stream",
				zap.Error(err),
				zap.String("name", protoMetric.GetName()),
			)
			continue // Пропускаем невалидные метрики
		}

		// Добавляем метрику в буфер
		metricsBuffer = append(metricsBuffer, *metric)

		// Если буфер достиг определенного размера, сохраняем метрики
		// Это позволяет сократить количество обращений к хранилищу
		const maxBufferSize = 100
		if len(metricsBuffer) >= maxBufferSize {
			saveErr := s.saveMetricsBuffer(ctx, metricsBuffer)
			if saveErr != nil {
				return status.Errorf(codes.Internal, "failed to save metrics batch: %v", saveErr)
			}
			metricsBuffer = nil // Очищаем буфер
		}
	}

	// Сохраняем оставшиеся метрики
	if len(metricsBuffer) > 0 {
		saveErr := s.saveMetricsBuffer(ctx, metricsBuffer)
		if saveErr != nil {
			return status.Errorf(codes.Internal, "failed to save metrics batch: %v", saveErr)
		}
	}

	// Отправляем ответ клиенту
	return stream.SendAndClose(&pb.UpdateBatchResponse{
		Success: true,
	})
}

// saveMetricsBuffer сохраняет буфер метрик в хранилище.
func (s *MetricsService) saveMetricsBuffer(
	ctx context.Context,
	metricsBuffer []metrics.Metric,
) error {
	if len(metricsBuffer) == 0 {
		return nil
	}

	s.log.Debug("saving metrics from stream buffer", zap.Int("count", len(metricsBuffer)))

	// Сохраняем метрики в хранилище
	if err := s.storage.UpdateMetrics(ctx, metricsBuffer); err != nil {
		s.log.Error("failed to update metrics batch from stream", zap.Error(err))
		return err
	}

	return nil
}

// protoToMetric преобразует метрику из protobuf в доменную модель.
func (s *MetricsService) protoToMetric(protoMetric *pb.Metric) (*metrics.Metric, error) {
	if protoMetric == nil {
		return nil, errors.New("metric is nil")
	}

	// Проверяем тип метрики
	switch protoMetric.GetType() {
	case pb.MetricType_GAUGE:
		if protoMetric.Value == nil {
			return nil, errors.New("gauge value is required")
		}
		return metrics.NewGauge(protoMetric.GetName(), protoMetric.GetValue()), nil
	case pb.MetricType_COUNTER:
		if protoMetric.Delta == nil {
			return nil, errors.New("counter delta is required")
		}
		return metrics.NewCounter(protoMetric.GetName(), protoMetric.GetDelta()), nil
	default:
		return nil, fmt.Errorf("unknown metric type: %v", protoMetric.GetType())
	}
}

// metricToProto преобразует метрику из доменной модели в protobuf.
func (s *MetricsService) metricToProto(metric *metrics.Metric) *pb.Metric {
	protoMetric := &pb.Metric{
		Name: metric.Name,
	}

	switch metric.MType {
	case metrics.TypeGauge:
		value := *metric.Value
		protoMetric.Type = pb.MetricType_GAUGE
		protoMetric.Value = &value
	case metrics.TypeCounter:
		delta := *metric.Delta
		protoMetric.Type = pb.MetricType_COUNTER
		protoMetric.Delta = &delta
	}

	return protoMetric
}
