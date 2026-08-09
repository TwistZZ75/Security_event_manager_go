package alerts

import (
	"context"
	"fmt"
	"log"
	wsevents "siem-server/internal/events"
	"strconv"
	"time"
)

// AlertManager управляет созданием и обработкой алертов
type AlertManager struct {
	storage  *AlertStorage
	notifier Notifier
	eventBus *wsevents.Bus
}

// Notifier интерфейс для отправки уведомлений
type Notifier interface {
	SendNotification(ctx context.Context, alert *Alert, channels []string) error
}

// NewAlertManager создает новый менеджер алертов
func NewAlertManager(storage *AlertStorage, notifier Notifier, eventBus *wsevents.Bus) *AlertManager {
	return &AlertManager{
		storage:  storage,
		notifier: notifier,
		eventBus: eventBus,
	}
}

// CreateAlert создает новый алерт с валидацией и уведомлениями
func (am *AlertManager) CreateAlert(ctx context.Context, alert *Alert) error {
	// Валидация
	if err := am.validateAlert(alert); err != nil {
		return fmt.Errorf("invalid alert: %v", err)
	}

	// Устанавливаем значения по умолчанию
	if alert.Status == "" {
		alert.Status = StatusOpen
	}

	// Сохраняем в БД
	if err := am.storage.CreateAlert(ctx, alert); err != nil {
		return fmt.Errorf("failed to create alert: %v", err)
	}

	// публикуем событие для websocket
	am.eventBus.Publish(wsevents.Event{
		Type:      wsevents.AlertCreated,
		Payload:   alert,
		Timestamp: time.Now(),
	})

	log.Printf("Alert created: ID=%d, Rule=%s, Severity=%s", alert.ID, alert.RuleName, alert.Severity)

	// Отправляем уведомление
	if am.notifier != nil {
		channels := am.getNotificationChannels(alert.Severity)
		if len(channels) > 0 {
			if err := am.notifier.SendNotification(ctx, alert, channels); err != nil {
				log.Printf("Warning: failed to send notification for alert %d: %v", alert.ID, err)
				// Не прерываем создание алерта из-за ошибки уведомления
			}
		}
	}

	return nil
}

// GetAlert получает алерт по ID
func (am *AlertManager) GetAlert(ctx context.Context, id string) (*Alert, error) {
	return am.storage.GetAlert(ctx, id)
}

// UpdateAlertStatus обновляет статус алерта
func (am *AlertManager) UpdateAlertStatus(ctx context.Context, id int64, status, updatedBy, notes string) error {

	if err := am.storage.UpdateAlert(
		ctx,
		id,
		status,
		updatedBy,
		notes,
	); err != nil {
		return fmt.Errorf("update alert status: %w", err)
	}

	// Получаем актуальную версию Alert из БД
	alert, err := am.storage.GetAlert(ctx, strconv.FormatInt(id, 10))
	if err != nil {
		return fmt.Errorf("get updated alert: %w", err)
	}

	am.eventBus.Publish(wsevents.Event{
		Type:      wsevents.AlertUpdated,
		Payload:   alert,
		Timestamp: time.Now(),
	})

	return nil
}

// GetAlerts получает список алертов с фильтром
func (am *AlertManager) GetAlerts(ctx context.Context, filter AlertFilter) ([]*Alert, error) {
	return am.storage.GetAlerts(ctx, filter)
}

// GetAlertStats получает статистику по алертам
func (am *AlertManager) GetAlertStats(ctx context.Context, from, to time.Time) (*AlertStats, error) {
	return am.storage.GetAlertStats(ctx, from, to)
}

// validateAlert проверяет корректность алерта
func (am *AlertManager) validateAlert(alert *Alert) error {
	if alert.RuleID == "" {
		return fmt.Errorf("rule_id is required")
	}
	if alert.Title == "" {
		return fmt.Errorf("title is required")
	}
	if alert.Severity == "" {
		return fmt.Errorf("severity is required")
	}

	// Проверяем корректность severity
	validSeverities := map[string]bool{
		"low":      true,
		"medium":   true,
		"high":     true,
		"critical": true,
	}
	if !validSeverities[alert.Severity] {
		return fmt.Errorf("invalid severity: %s", alert.Severity)
	}

	return nil
}

// getNotificationChannels определяет каналы уведомлений на основе severity
func (am *AlertManager) getNotificationChannels(severity string) []string {
	switch severity {
	case "critical":
		return []string{"email", "telegram"}
	case "high":
		return []string{"telegram"}
	case "medium":
		return []string{"telegram"}
	default:
		return []string{}
	}
}
