package delivery

import (
	"context"
	logstructure "siem-server/internal/logsstructure"
	processor "siem-server/internal/processor"
	"siem-server/proto/server/pkg/pb"
	"time"
)

type LogHandler struct {
	processor *processor.LogProc
	pb.UnimplementedLogServiceServer
}

func NewLogHandler(proc *processor.LogProc) *LogHandler {
	return &LogHandler{
		processor: proc,
	}
}

func (handler *LogHandler) SendRawLog(ctx context.Context, req *pb.RequestRawLog) (*pb.ResponseRawLog, error) {
	raw := &logstructure.RawLog{
		Username:        req.Username,
		PC_name:         req.PcName,
		OS:              req.Os,
		Log_source:      req.LogSource,
		Event_timestamp: time.Unix(0, req.Timestamp),
		Format:          req.LogFormat,
		Raw_data:        req.Content,
	}

	if error := handler.processor.ProcessRawLog(raw); error != nil {
		return &pb.ResponseRawLog{
			Response: false,
			Message:  error.Error(),
		}, nil
	}

	return &pb.ResponseRawLog{
		Response: true,
		Message:  "Log processed successfully",
	}, nil
}
