package wsevents

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

const (
	EventCreated = "event.created"

	AlertCreated = "alert.created"
	AlertUpdated = "alert.updated"
)

type Event struct { // определяем отправляемые события
	Type      string      `json:"type"`    // тип события "normlog", "alert", "action"
	Payload   interface{} `json:"payload"` // данные о событии
	Timestamp time.Time   `json:"timestamp"`
}

type Subscriber struct { // подписчик
	ID      int64
	Channel chan Event // канал событий на который подписываются логи/алерты/действия
	Filter  func(Event) bool
}

type Bus struct {
	mu          sync.RWMutex
	subscribers map[int64]*Subscriber // мапа подписчиков для того, чтобы отслеживать что вообще сейчас смотрится пользователем
	nextID      int64                 // id следующего подписчика
	closed      bool                  // флаг закрытия
	done        chan struct{}         // сигнальный канал, по которому все горутины связанные с Bus завершатся после его закрытия
}

func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[int64]*Subscriber),
		done:        make(chan struct{}),
	}
}

// подписка
// создаём подписчика и добавляем его в мапу
func (b *Bus) Subscribe(filter func(Event) bool) *Subscriber {
	b.mu.Lock()
	defer b.mu.Unlock()

	// не создём новых подписчиков, если шина закрыта
	if b.closed {
		return nil
	}

	id := b.nextID
	b.nextID++

	sub := &Subscriber{
		ID:      id,
		Channel: make(chan Event, 64),
		Filter:  filter,
	}
	b.subscribers[id] = sub

	slog.Debug("new bus subs added", "subscriber", b.subscribers[id])

	return sub
}

// отписка
// закрываем канал и удаляем подписчика из мапы
func (b *Bus) Unsubscribe(id int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub, ok := b.subscribers[id]
	if !ok {
		return
	}

	close(sub.Channel)
	delete(b.subscribers, id)
	slog.Debug("event bus subscriber removed", "subscriber_id", id)
}

func (b *Bus) Publish(event Event) {
	b.mu.RLock() // лочим в случае, если кто-то подписывается
	defer b.mu.RUnlock()

	if b.closed { //если шина закрыта не отправляем события.
		return
	}

	for _, sub := range b.subscribers { // проходимся по мапе
		if sub.Filter == nil || sub.Filter(event) { // узнаём кто сейчас должен писать в канал исходя из фильтра
			// фильтр получаем с фронта
			// заходим сюда, если фильтр nil (т.е. отправляем все события) или если фильтр возвращает true для такого типа событий

			// допустима потеря части событий в real-time, потому как front может после reconnect получить все данные через REST
			select {
			case sub.Channel <- event: // пишем в канал
			default:
				slog.Warn("event bus subscriber channel full, dropping event",
					"subscriber_id", sub.ID,
					"event_type", event.Type) // скипаем, если буфер канала полный и пишем в лог, что скипнули
			}
		}
	}
}

func (b *Bus) Shutdown(ctx context.Context) error { //закрытие всех каналов и очищение мапы subscrubers

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed == true { // проверяем не закрыт ли уже канал
		return errors.New("Bus already closed")
	}

	b.closed = true // отмечаем, что канал закрыт
	close(b.done)   // закрываем канал

	for id, sub := range b.subscribers {
		// чистим мапу и закрываем каналы подписчиков
		close(sub.Channel)
		delete(b.subscribers, id)
	}

	slog.Info("event bus shut down")
	return nil
}

func (b *Bus) Done() <-chan struct{} {
	return b.done
}
