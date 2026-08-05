package events

import (
	"sync"
	"time"
)

type Event struct { // определяем отправляемые события
	Type      string      // тип события "normlog", "alert", "action"
	Payload   interface{} // данные о событии
	Timestamp time.Time
}

type Subscruber struct { // подписчик
	ID      int64
	Channel chan Event       //канал событий на который подписываются логи/алерты/действия
	Filter  func(Event) bool // фильтр, чтобы не отправлять логи по websocket, пока пользователь на странице alerts
}

type Bus struct {
	mu          sync.RWMutex
	subscrubers map[int64]*Subscruber // мапа подписчиков для того, чтобы отслеживать что вообще сейчас смотрится пользователями
	nextID      int64                 // id следующего подписчика
}

func NewBus() *Bus {
	return &Bus{
		subscrubers: make(map[int64]*Subscruber),
		nextID:      0,
	}
}

// подписка
// создаём подписчика и добавляем его в мапу
func (b *Bus) Subscrube(filter func(Event) bool) *Subscruber {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := b.nextID
	b.nextID++

	sub := &Subscruber{
		ID:      id,
		Channel: make(chan Event, 64),
		Filter:  filter,
	}
	b.subscrubers[id] = sub
	return sub
}

// отписка
// закрываем канал и удаляем подписчика из мапы
func (b *Bus) Unsubscrube(id int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if sub, ok := b.subscrubers[id]; ok {
		close(sub.Channel)
		delete(b.subscrubers, id)
	}
}

func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, sub := range b.subscrubers {
		if sub != nil { // TODO: условие сравнивающее тип события с фильтром
			select {
			case sub.Channel <- event: // пишем в канал

			default: // скипаем, если буфер канала полный
			}
		}
	}
}

func (b *Bus) Shutdown() { //закрытие всех каналов и очищение мапы subscrubers

}
