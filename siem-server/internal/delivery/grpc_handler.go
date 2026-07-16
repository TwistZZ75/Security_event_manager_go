package delivery

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	logstructure "siem-server/internal/logsstructure"
	processor "siem-server/internal/processor"
	"siem-server/proto/server/pkg/pb"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LogHandler struct {
	processor *processor.LogProc
	pb.UnimplementedLogServiceServer
}

// конструктор нового обработчика (хендлера)
func NewLogHandler(proc *processor.LogProc) *LogHandler {
	return &LogHandler{
		processor: proc,
	}
}

// функция принимающая запрос на размещение лога от клиента
// принимает контекст и запрос в виде protobuf структуры
// возвращает сообщение о успехе размещения лога или сообшение о провале и ошибку
func (handler *LogHandler) SendRawLog(ctx context.Context, req *pb.RequestRawLog) (*pb.ResponseRawLog, error) {
	timestamp, err := time.Parse(time.RFC3339, req.Timestamp)
	if err != nil {
		slog.Warn("invalid timestamp", "error", err, "timestamp", req.Timestamp)
		return nil, status.Errorf(codes.InvalidArgument, "invalid timestamp: %v", err)
	}
	raw := &logstructure.RawLog{
		Username:        req.Username,
		PC_name:         req.PcName,
		OS:              req.Os,
		Log_source:      req.LogSource,
		Event_timestamp: timestamp,
		Format:          req.LogFormat,
		Raw_data:        req.Content,
	}

	//вызов функции обработки лога (его парсинга и сохранения в бд)
	if err := handler.processor.ProcessLog(ctx, raw); err != nil {
		slog.Error("failed to process log", "error", err)
		return nil, status.Errorf(codes.Internal, "processing failed: %v", err)
	}

	//присылаем ответ об успешной обработке клиенту
	return &pb.ResponseRawLog{
		Response: true,
		Message:  "Log processed successfully",
	}, nil
}

//
//grpc.ClientStreamingServer - Это специальный интерфейс из gRPC для обработки клиентских потоковых вызовов:
// Клиентский поток (Client Streaming) - это когда:
// Клиент отправляет несколько сообщений в потоке (stream)
// Сервер получает их все и отправляет один ответ
//

// функция принимающая запрос на принятие потока данных от клиента
// принимает параметр через который функция принимает логи и отправляет ответ
// возвращает ошибку, если она была
// TODO: возвращать число успешно обработанных событий и слайс ошибок через errors.Join
func (handler *LogHandler) SendRawLogStream(stream grpc.ClientStreamingServer[pb.RequestRawLog, pb.ResponseRawLog]) error {
	var processedCount int32
	var failedCount int32
	ctx := stream.Context()
	const batchSize = 10
	var batch []*logstructure.RawLog

	for {
		req, err := stream.Recv()
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err == io.EOF {
				// Обрабатываем остаток батча
				if len(batch) > 0 {
					if err := handler.processor.ProcessBatch(ctx, batch); err != nil {
						slog.Warn("invalid timestamp", "error", err)
						failedCount += int32(len(batch))
					} else {
						processedCount += int32(len(batch))
					}
				}
				response := &pb.ResponseRawLog{
					Response: true,
					Message:  fmt.Sprintf("Processed %d logs successfully, %d failed", processedCount, failedCount),
				}
				return stream.SendAndClose(response)
			}
			if err != nil {
				return fmt.Errorf("error receiving log: %v", err)
			}

			timestamp, err := time.Parse(time.RFC3339, req.Timestamp)
			if err != nil {
				failedCount++
				continue
			}

			raw := &logstructure.RawLog{
				Username:        req.Username,
				PC_name:         req.PcName,
				OS:              req.Os,
				Log_source:      req.LogSource,
				Event_timestamp: timestamp,
				Format:          req.LogFormat,
				Raw_data:        req.Content,
			}

			batch = append(batch, raw)
			if len(batch) >= batchSize {
				if err := handler.processor.ProcessBatch(ctx, batch); err != nil {
					failedCount += int32(len(batch))
				} else {
					processedCount += int32(len(batch))
				}
				batch = nil // очищаем батч
			}
		}
	}
}
