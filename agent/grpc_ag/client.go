package grpc_ag

import (
	"context"
	"fmt"
	"siem-agent/proto/pkg/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type LogClient struct {
	conn   *grpc.ClientConn
	client pb.LogServiceClient
}

// функция создания нового клиента
// принимает адрес сервера
// возвращает ссылку на структуру клиента и ошибку, есил она есть
func NewLogClient(adr string) (*LogClient, error) {
	conn, err := grpc.NewClient(
		adr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %v", err)
	}
	client := pb.NewLogServiceClient(conn)

	return &LogClient{
		conn:   conn,
		client: client,
	}, nil
}

// функция закрытия соединения
func (c *LogClient) Close() error {
	return c.conn.Close()
}

// функция отправки логов по одному
// принимает контекст и отправляемый лог
// возвращает ошибку при наличии таковой
func (c *LogClient) SendLog(ctx context.Context, log *pb.RequestRawLog) error {
	responce, err := c.client.SendRawLog(ctx, log)
	if err != nil {
		return err
	}

	if responce.Response != true {
		return fmt.Errorf("Server error %v", responce.Response)
	}

	return nil
}

// функция отправки логов пачкой
// принимает контекст и slice отправляемых логов
// возвращает ошибку при наличии таковой
func (c *LogClient) SendLogBatch(ctx context.Context, logs []*pb.RequestRawLog) error {
	stream, err := c.client.SendRawLogStream(ctx)
	if err != nil {
		return fmt.Errorf("Failed to create stream, %v", err)
	}

	for _, log := range logs {
		if err := stream.Send(log); err != nil {
			return fmt.Errorf("Failed to send log in batch, %v", err)
		}
	}

	responce, err := stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("Failed to close stream, %v", err)
	}

	if responce.Response != true {
		return fmt.Errorf("Batch proccessing had errors, %v", err)
	}
	return nil
}
