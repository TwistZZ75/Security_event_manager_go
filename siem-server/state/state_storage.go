package state

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StateStorage struct {
	pool *pgxpool.Pool
}

func NewStateStorage(pool *pgxpool.Pool) *StateStorage {
	return &StateStorage{pool: pool}
}

// получает состояние правила по rule_id и group_key
func (ss *StateStorage) GetState(ctx context.Context, ruleID, groupKey string) (*RuleState, error) {
	query := `
		SELECT 
			id,
			rule_id,
			group_key,
			counter,
			first_seen,
			last_seen,
			state_data,
			expires_at
		FROM rule_state
		WHERE rule_id = $1 AND group_key = $2
	`

	row := ss.pool.QueryRow(ctx, query, ruleID, groupKey)
	state, err := ss.scanState(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Состояние не найдено - это нормально
		}
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	// Проверяем, не истекло ли состояние
	if time.Now().After(state.ExpiresAt) {
		return nil, nil
	}

	return state, nil
}

// удаляет состояние правила
func (ss *StateStorage) DeleteState(ctx context.Context, ruleID, groupKey string) error {
	query := `
		DELETE FROM rule_state
		WHERE rule_id = $1 AND group_key = $2
	`

	_, err := ss.pool.Exec(ctx, query, ruleID, groupKey)
	if err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}

	return nil
}

// сохраняет или обновляет состояние правила
func (ss *StateStorage) SaveState(ctx context.Context, state *RuleState) error {
	// Сериализуем state_data в JSONB
	stateDataJSON, err := json.Marshal(state.StateData)
	if err != nil {
		return fmt.Errorf("failed to marshal state data: %w", err)
	}

	query := `
		INSERT INTO rule_state (
			rule_id,
			group_key,
			counter,
			first_seen,
			last_seen,
			state_data,
			expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (rule_id, group_key) DO UPDATE SET
			counter = EXCLUDED.counter,
			last_seen = EXCLUDED.last_seen,
			state_data = EXCLUDED.state_data,
			expires_at = EXCLUDED.expires_at
		RETURNING id
	`

	err = ss.pool.QueryRow(ctx, query,
		state.RuleID,
		state.GroupKey,
		state.Counter,
		state.FirstSeen,
		state.LastSeen,
		stateDataJSON,
		state.ExpiresAt,
	).Scan(&state.ID)

	if err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// IncrementCounter атомарно увеличивает счётчик для (ruleID, groupKey).
//
// Логика временного окна:
//   - Если запись не существует — создаётся новая: counter=1, первое окно.
//   - Если запись существует И окно ещё не истекло (expires_at >= NOW) —
//     счётчик увеличивается, expires_at НЕ меняется (окно фиксировано с первого события).
//   - Если запись существует, но окно уже истекло (expires_at < NOW) —
//     счётчик сбрасывается в 1, first_seen и expires_at обновляются (новое окно).
//
// Это реализует «tumbling window»: окно открывается первым событием и закрывается
// через time_window. Все события в этом промежутке считаются вместе.
func (ss *StateStorage) IncrementCounter(ctx context.Context, ruleID, groupKey string, window time.Duration) (*RuleState, error) {
	now := time.Now()
	expiresAt := now.Add(window)

	query := `
		INSERT INTO rule_state (
			rule_id,
			group_key,
			counter,
			first_seen,
			last_seen,
			state_data,
			expires_at
		) VALUES ($1, $2, 1, $3, $3, '{}'::jsonb, $4)
		ON CONFLICT (rule_id, group_key) DO UPDATE SET
			-- Если окно уже истекло — сбрасываем счётчик и открываем новое окно.
			-- Если окно ещё активно — просто инкрементируем счётчик.
			counter = CASE
				WHEN rule_state.expires_at < $3 THEN 1
				ELSE rule_state.counter + 1
			END,
			first_seen = CASE
				WHEN rule_state.expires_at < $3 THEN $3
				ELSE rule_state.first_seen
			END,
			last_seen = $3,
			-- expires_at фиксируется при открытии окна и не сдвигается.
			-- При просроченном окне — открываем новое с текущего момента.
			expires_at = CASE
				WHEN rule_state.expires_at < $3 THEN $4
				ELSE rule_state.expires_at
			END,
			state_data = CASE
				WHEN rule_state.expires_at < $3 THEN '{}'::jsonb
				ELSE rule_state.state_data
			END
		RETURNING id, counter, first_seen, last_seen, expires_at
	`

	var st RuleState
	st.RuleID = ruleID
	st.GroupKey = groupKey
	st.StateData = make(map[string]interface{})

	err := ss.pool.QueryRow(ctx, query, ruleID, groupKey, now, expiresAt).Scan(
		&st.ID,
		&st.Counter,
		&st.FirstSeen,
		&st.LastSeen,
		&st.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to increment counter: %w", err)
	}

	return &st, nil
}

// возвращает все состояния для конкретного правила
func (ss *StateStorage) GetStatesByRule(ctx context.Context, ruleID string) ([]*RuleState, error) {
	query := `
		SELECT 
			id, rule_id, group_key, counter,
			first_seen, last_seen, state_data, expires_at
		FROM rule_state
		WHERE rule_id = $1	
		ORDER BY counter DESC
	`

	rows, err := ss.pool.Query(ctx, query, ruleID)
	if err != nil {
		return nil, fmt.Errorf("failed to query states: %w", err)
	}
	defer rows.Close()

	var states []*RuleState

	for rows.Next() {
		state, err := ss.scanState(rows)
		if err != nil {
			slog.Warn("failed to scan state", "error", err)
			continue
		}
		states = append(states, state)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return states, nil
}

// возвращает все активные (не истекшие) состояния
func (ss *StateStorage) GetActiveStates(ctx context.Context) ([]*RuleState, error) {
	query := `
		SELECT 
			id, rule_id, group_key, counter,
			first_seen, last_seen, state_data, expires_at
		FROM rule_state
		WHERE expires_at > $1
		ORDER BY last_seen DESC
	`

	rows, err := ss.pool.Query(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to query active states: %w", err)
	}
	defer rows.Close()

	var states []*RuleState

	for rows.Next() {
		state, err := ss.scanState(rows)
		if err != nil {
			slog.Error("Scan state error", "error", err)
			continue
		}
		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return states, nil
}

// сбрасывает счетчик состояния на 0
func (ss *StateStorage) ResetState(ctx context.Context, ruleID, groupKey string) error {
	query := `
		UPDATE rule_state
		SET counter = 0,
		    first_seen = $3,
		    last_seen = $3
		WHERE rule_id = $1 AND group_key = $2
	`

	_, err := ss.pool.Exec(ctx, query, ruleID, groupKey, time.Now())
	if err != nil {
		return fmt.Errorf("failed to reset state: %w", err)
	}

	return nil
}

// возвращает количество активных состояний для правила
func (ss *StateStorage) GetStateCount(ctx context.Context, ruleID string) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM rule_state
		WHERE rule_id = $1 AND expires_at > $2
	`

	var count int64
	err := ss.pool.QueryRow(ctx, query, ruleID, time.Now()).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count states: %w", err)
	}

	return count, nil
}

// обновляет поле state_data для состояния
func (ss *StateStorage) UpdateStateData(ctx context.Context, ruleID, groupKey string, stateData map[string]interface{}) error {
	stateDataJSON, err := json.Marshal(stateData)
	if err != nil {
		return fmt.Errorf("failed to marshal state data: %w", err)
	}

	query := `
		UPDATE rule_state
		SET state_data = $3
		WHERE rule_id = $1 AND group_key = $2
	`

	_, err = ss.pool.Exec(ctx, query, ruleID, groupKey, stateDataJSON)
	if err != nil {
		return fmt.Errorf("failed to update state data: %w", err)
	}

	return nil
}

// продлевает срок действия состояния
func (ss *StateStorage) ExtendExpiration(ctx context.Context, ruleID, groupKey string, window time.Duration) error {
	expiresAt := time.Now().Add(window)

	query := `
		UPDATE rule_state
		SET expires_at = $3
		WHERE rule_id = $1 AND group_key = $2
	`

	_, err := ss.pool.Exec(ctx, query, ruleID, groupKey, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to extend expiration: %w", err)
	}

	return nil
}

// удаляет все истекшие состояния
// Вызывается периодически для очистки
func (ss *StateStorage) DeleteExpiredStates(ctx context.Context) error {
	query := `
		DELETE FROM rule_state
		WHERE expires_at < $1
	`

	result, err := ss.pool.Exec(ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete expired states: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected > 0 {
		slog.Info("Deleted expired rule states", "number of deleted states", rowsAffected)
	}

	return nil
}

// возвращает статистику по состояниям конкретного правила
func (ss *StateStorage) GetStatsForRule(ctx context.Context, ruleID string) (*StateStats, error) {
	query := `
		SELECT
			COUNT(*)::BIGINT as total,
			COUNT(*) FILTER (WHERE expires_at > $2)::BIGINT as active,
			COUNT(*) FILTER (WHERE expires_at <= $2)::BIGINT as expired,
			AVG(counter)::FLOAT as avg_counter,
			MAX(counter)::INT as max_counter,
			MIN(counter)::INT as min_counter
		FROM rule_state
		WHERE rule_id = $1
	`

	var stats StateStats
	err := ss.pool.QueryRow(ctx, query, ruleID, time.Now()).Scan(
		&stats.Total,
		&stats.Active,
		&stats.Expired,
		&stats.AvgCounter,
		&stats.MaxCounter,
		&stats.MinCounter,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get state stats: %w", err)
	}

	return &stats, nil
}

// сканирует строку из БД в структуру RuleState
func (ss *StateStorage) scanState(scanner interface {
	Scan(dest ...interface{}) error
}) (*RuleState, error) {
	var (
		id            int64
		ruleID        string
		groupKey      string
		counter       int
		firstSeen     time.Time
		lastSeen      time.Time
		stateDataJSON []byte
		expiresAt     time.Time
	)

	err := scanner.Scan(
		&id,
		&ruleID,
		&groupKey,
		&counter,
		&firstSeen,
		&lastSeen,
		&stateDataJSON,
		&expiresAt,
	)

	if err != nil {
		return nil, fmt.Errorf("Rule state scan error %w", err)
	}

	// Десериализуем state_data
	var stateData map[string]interface{}
	if len(stateDataJSON) > 0 {
		if err := json.Unmarshal(stateDataJSON, &stateData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal state data: %w", err)
		}
	}

	state := &RuleState{
		ID:        id,
		RuleID:    ruleID,
		GroupKey:  groupKey,
		Counter:   counter,
		FirstSeen: firstSeen,
		LastSeen:  lastSeen,
		StateData: stateData,
		ExpiresAt: expiresAt,
	}

	return state, nil
}
