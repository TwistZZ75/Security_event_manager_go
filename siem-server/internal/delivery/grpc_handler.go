package delivery

import (
	"context"
	"fmt"
	"io"
	logstructure "siem-server/internal/logsstructure"
	processor "siem-server/internal/processor"
	"siem-server/proto/server/pkg/pb"
	"time"

	"google.golang.org/grpc"
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
		fmt.Printf("%v", err.Error())
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
	if err := handler.processor.ProcessRawLog(raw); err != nil {
		//присылаем о ошибке при обработке или сохранении лога
		return &pb.ResponseRawLog{
			Response: false,
			Message:  err.Error(),
		}, nil
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
func (handler *LogHandler) SendRawLogStream(stream grpc.ClientStreamingServer[pb.RequestRawLog, pb.ResponseRawLog]) error {
	var processedCount int32
	var failedCount int32

	for {
		// Recv() возвращает (*pb.RequestRawLog, error)
		req, err := stream.Recv()
		if err == io.EOF {
			// Создаём ответ и передаём указатель
			response := &pb.ResponseRawLog{
				Response: true,
				Message:  fmt.Sprintf("Processed %d logs successfully, %d failed", processedCount, failedCount),
			}
			return stream.SendAndClose(response) // отправка и закрытие
		}
		if err != nil {
			return fmt.Errorf("error receiving log: %v", err)
		}

		// req уже является *pb.RequestRawLog, просто используем поля
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

		err = handler.processor.ProcessRawLog(raw)
		if err == nil {
			processedCount++
		} else {
			failedCount++
			fmt.Printf("Failed to process log: %v\n", err)
		}
	}
}
