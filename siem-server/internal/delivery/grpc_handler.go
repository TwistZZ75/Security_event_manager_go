package delivery

import (
	"context"
	"fmt"
	logstructure "siem-server/internal/logsstructure"
	processor "siem-server/internal/processor"
	"siem-server/proto/server/pkg/pb"
	"time"
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
