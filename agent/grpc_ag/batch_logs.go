package grpc_ag

import (
	"context"
	"fmt"
	"siem-agent/collector"
	"siem-agent/proto/pkg/pb"
	"time"
)

// функция объединения логов в буфер для отправки
// принимает контекст, структуру клиента и массив коллекторов
// ничего не возвращает
func BatchLogs(ctx context.Context, client *LogClient, collectors []collector.LogCollector) {

	batchSize := 10 //размер буффера
	batchTimeout := 5 * time.Second

	batch := make([]*pb.RequestRawLog, 0, batchSize)
	ticker := time.NewTicker(batchTimeout)
	defer ticker.Stop()

	logChan := mergeLogs(collectors) //объединяем логи из разных потоков

	for {
		select {
		//отправляем буффер, если закончили работу агента, даже если буфер не полный
		case <-ctx.Done():
			if len(batch) > 0 {
				client.SendLogBatch(context.Background(), batch)
			}
			return
		//отправляем буфер, если он заполнен
		case log := <-logChan:
			batch = append(batch, log)
			if len(batch) >= batchSize {
				if err := client.SendLogBatch(ctx, batch); err != nil {
					fmt.Printf("%v", err)
				}
				batch = make([]*pb.RequestRawLog, 0, batchSize)
			}
		//отправляем буфер по таймеру, даже если он не заполнен полностью
		case <-ticker.C:
			if len(batch) > 0 {
				if err := client.SendLogBatch(ctx, batch); err != nil {
					fmt.Printf("%v", err)
				}
				batch = make([]*pb.RequestRawLog, 0, batchSize)
			}
		}
	}
}

// функция объединения логов из разных потоков
// принимает массив коллекторов
// возвращает канал логов
func mergeLogs(collectors []collector.LogCollector) <-chan *pb.RequestRawLog {
	out := make(chan *pb.RequestRawLog, 100)

	//для каждого из коллекторов запускаем GetLogs() и записываем полученную информацию в канал
	for _, c := range collectors {
		go func(collector collector.LogCollector) {
			for log := range collector.GetLogs() {
				out <- log
			}
		}(c)
	}
	return out
}
