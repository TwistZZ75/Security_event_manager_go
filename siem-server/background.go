package main

import (
	"context"
	"log"
	"log/slog"
	webserver "siem-server/web_server"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func StartWebServer(ctx context.Context, cancel func(), webServer *webserver.WebServer, http_port string, wg *sync.WaitGroup) {
	defer wg.Done()
	if err := webServer.Start(http_port); err != nil {
		slog.Error("Web server failed", "error", err)
		cancel()
	}
}

// функция запуска движка правил
// принимает контекст, функцию отмены, объект сервиса, объект хранилища и waitGroup
func StartRuleEngine(ctx context.Context, cancel func(), services *Services, storages *Storages, wg *sync.WaitGroup) {
	defer wg.Done()
	// Загружаем правила из БД
	if err := services.RuleEngine.LoadRules(ctx); err != nil {
		slog.Error("Load rules error", "error", err)
		cancel()
	}

	ruleCount, err := storages.RuleStorage.GetEnabledRulesCount(ctx)
	if err != nil {
		slog.Error("Counting rules error", "error", err)
		cancel()
	}
	log.Printf("Rule Engine initialized with %d enabled rules", ruleCount)

}

// функция удаления просроченных refresh токенов
// удаляет просроченные токены каждый час
// принимает контекст, пул соединений и waitGroup
func DeleteExpiredTokens(ctx context.Context, pool *pgxpool.Pool, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ctxExpired, cancel := context.WithTimeout(ctx, 2*time.Second)
			_, err := pool.Exec(ctxExpired, `DELETE FROM refresh_tokens WHERE expires_at < NOW()`)
			cancel()
			if err != nil {
				slog.Error("Failed to cleanup refresh tokens", "error", err)
			}
		}
	}
}

// функция удаления истёкших состояний правил
// удаляет истёкшие состояния правил каждые 5 минут
// принимает контекст, объект хранилища и waitGroup
func DeleteExpiredStates(ctx context.Context, storages *Storages, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ctxExpired, cancel := context.WithTimeout(ctx, 2*time.Second)
			err := storages.StateStorage.DeleteExpiredStates(ctxExpired)
			cancel()
			if err != nil {
				slog.Error("Failed to cleanup expired states", "error", err)
			}
		}
	}
}

// функция перезагрузки правил
// загружает все правила каждые 5 минут
// принимает контекст, объект хранилища, объект сервиса и waitGroup
func ReloadRules(ctx context.Context, storages *Storages, services *Services, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ctxReload, cancel := context.WithTimeout(ctx, 2*time.Second)
			errReload := services.RuleEngine.ReloadRules(ctxReload)
			cancel()
			if errReload != nil {
				slog.Error("Failed to reload rules", "error", errReload)
				return
			} else {
				ctxCount, cancel := context.WithTimeout(ctx, 2*time.Second)
				ruleCount, errCount := storages.RuleStorage.GetEnabledRulesCount(ctxCount)
				cancel()
				if errCount != nil {
					slog.Error("Cannot count enabled rules", "error", errCount)
				}
				slog.Info("Rules reloaded", "Enabled rule count: ", ruleCount)
			}

		}
	}

}

// функция проверки статуса "в сети" у агентов
// каждую минуту проверяет находится ли агент онлайн
// принимает контекст, объект хранилища, объект сервиса и waitGroup
func CheckAgentStatus(ctx context.Context, storages *Storages, services *Services, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(1 * time.Minute) //все дальнейшие проверки идут по тикеру
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			//проверка статуса агентов при запуске
			if err := storages.AgentStorage.MarkOfflineAgents(ctx); err != nil {
				slog.Error("Failed to mark agent offline", "error", err)
			} else {
				slog.Info("Agent marked offline")
			}
		}
	}

}
