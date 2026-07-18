package rules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ruleSelectColumns = `
    id,
    name,
    enabled,
    severity,
    rule_definition,
    tags,
    created_by,
    created_at,
    COALESCE(updated_by, '') AS updated_by,
    updated_at,
    last_triggered,
    trigger_count
`

type RuleStorage struct {
	pool *pgxpool.Pool
}

// конструктор
func NewRuleStorage(pool *pgxpool.Pool) *RuleStorage {
	return &RuleStorage{pool: pool}
}

type RuleDefinition struct {
	Conditions  []Condition  `json:"conditions"`
	Aggregation *Aggregation `json:"aggregation,omitempty"`
	Actions     []Action     `json:"actions"`
}

// SaveRule сохраняет новое правило или обновляет существующее
func (rst *RuleStorage) SaveRule(ctx context.Context, rule *Rule) error {
	if rst.pool == nil {
		return fmt.Errorf("database pool is nil")
	}
	// Сериализуем определение правила в JSONB
	definition := RuleDefinition{
		Conditions:  rule.Conditions,
		Aggregation: rule.Aggregation,
		Actions:     rule.Actions,
	}

	definitionJSON, err := json.Marshal(definition)
	if err != nil {
		return fmt.Errorf("failed to marshal rule definition: %w", err)
	}

	query := `
		INSERT INTO rules (
			id, name, enabled, severity,
			rule_definition, tags, created_by,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			enabled = EXCLUDED.enabled,
			severity = EXCLUDED.severity,
			rule_definition = EXCLUDED.rule_definition,
			tags = EXCLUDED.tags,
			updated_at = EXCLUDED.updated_at
		WHERE rules.name IS DISTINCT FROM EXCLUDED.name
		OR rules.enabled IS DISTINCT FROM EXCLUDED.enabled
		OR rules.severity IS DISTINCT FROM EXCLUDED.severity
		OR rules.rule_definition IS DISTINCT FROM EXCLUDED.rule_definition
		OR rules.tags IS DISTINCT FROM EXCLUDED.tags
	`
	createdAt := rule.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	_, err = rst.pool.Exec(ctx, query,
		rule.ID,
		rule.Name,
		rule.Enabled,
		rule.Severity,
		definitionJSON,
		rule.Tags,
		rule.CreatedBy,
		createdAt,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to save rule: %w", err)
	}

	return nil
}

// функция удаления правила
// принимает контекст и id правила
// возвращает ошибку
func (rst *RuleStorage) RemoveRule(ctx context.Context, id string) error {
	//проверяем наличие соединений с БД
	if rst.pool == nil {
		slog.Error("database pool is nil")
		return fmt.Errorf("database pool is nil")
	}

	query := `DELETE FROM rules WHERE id = $1`
	_, err := rst.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("Unable to delete %s, %w", id, err)
	}
	return nil
}

// загружает все правила из БД
// принимает контекст
// возвращает массив правил и ошибку
func (rst *RuleStorage) GetAllRules(ctx context.Context) ([]*Rule, error) {
	//проверяем наличие соединений с БД
	if rst.pool == nil {
		slog.Error("database pool is nil")
		return nil, fmt.Errorf("database pool is nil")
	}
	//запрос к бд
	query := `
        SELECT ` + ruleSelectColumns +
		` FROM rules
        ORDER BY created_at DESC
    `
	//получаем строки из бд согласно запросу
	rows, err := rst.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("Failed to query rules: %w", err)
	}
	defer rows.Close()

	//создаём массив объектов *Rule
	var rulesList []*Rule

	//для каждой строки
	for rows.Next() {
		rule, err := rst.ScanRule(rows)
		if err != nil {
			slog.Error("Failed to scan rule", "error", err)
			continue
		}

		rulesList = append(rulesList, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Error iterating rules: %w", err)
	}

	return rulesList, nil
}

// функция получения одного правила
// принимает контекст и id правила
// возвращает правило
func (rst *RuleStorage) GetRule(ctx context.Context, id string) (*Rule, error) {
	if rst.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	//запрос к бд
	query := `
        SELECT ` + ruleSelectColumns +
		` FROM rules
        WHERE id = $1`
	row := rst.pool.QueryRow(ctx, query, id)
	rule, err := rst.ScanRule(row)
	if err != nil {
		return nil, fmt.Errorf("Failed to scan rule: %w", err)
	}
	return rule, nil
}

// функция сканирования правила в структуру БД
// принимает объект, имеющий функцию Scan (в нашем случае pgx.Row,
// но это также может быть и *sql.Row (результат QueryRow), и *sql.Rows (после вызова Next()))
// возвращает правило и ошибку
func (rst *RuleStorage) ScanRule(scanner interface {
	Scan(dest ...interface{}) error
}) (*Rule, error) {
	//структура получаемого правила
	var (
		ruleID         string
		name           string
		enabled        bool
		severity       string
		definitionJSON []byte
		tags           []string
		createdBy      string
		createdAt      time.Time
		updatedBy      string
		updatedAt      *time.Time
		lastTriggered  *time.Time
		triggerCount   int64
	)

	//сохраняем считанные из строки значения по адресам структуры выше
	err := scanner.Scan(
		&ruleID,
		&name,
		&enabled,
		&severity,
		&definitionJSON,
		&tags,
		&createdBy,
		&createdAt,
		&updatedBy,
		&updatedAt,
		&lastTriggered,
		&triggerCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // правило не найдено
		}
		return nil, fmt.Errorf("failed to scan rule row: %w", err)
	}

	// Десериализуем rule_definition из JSONB
	var definition Rule
	if err := json.Unmarshal(definitionJSON, &definition); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rule definition for rule %s: %w", ruleID, err)
	}

	// Создаем объект Rule
	rule := &Rule{
		ID:            ruleID,
		Name:          name,
		OS:            definition.OS,
		Enabled:       enabled,
		Severity:      severity,
		Conditions:    definition.Conditions,
		Aggregation:   definition.Aggregation,
		Actions:       definition.Actions,
		Tags:          tags,
		CreatedBy:     createdBy,
		CreatedAt:     createdAt,
		UpdatedBy:     updatedBy,
		UpdatedAt:     updatedAt,
		LastTriggered: lastTriggered,
		TriggerCount:  triggerCount,
	}
	return rule, nil
}

// функция получения числа правил
// принимает контекст
// возвращает число правил и ошибку
func (rst *RuleStorage) GetRulesCount(ctx context.Context) (int, error) {
	//проверяем наличие соединений с БД
	if rst.pool == nil {
		return 0, fmt.Errorf("database pool is nil")
	}

	//запрос к бд
	query := `SELECT COUNT(*) FROM rules`

	var count int
	err := rst.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count rules: %w", err)
	}

	return count, nil
}

func (rst *RuleStorage) GetEnabledRulesCount(ctx context.Context) (int, error) {
	//проверяем наличие соединений с БД
	if rst.pool == nil {
		return 0, fmt.Errorf("database pool is nil")
	}

	//запрос к бд
	query := `SELECT COUNT(*) FROM rules WHERE enabled = true`

	var count int
	err := rst.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count enabled rules: %w", err)
	}

	return count, nil
}

func (rst *RuleStorage) LoadEnabledRules(ctx context.Context) ([]*Rule, error) {
	if rst.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}
	query := `
		SELECT ` + ruleSelectColumns +
		` FROM rules
		WHERE enabled = true
		ORDER BY created_at DESC`

	rows, err := rst.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query enabled rules: %w", err)
	}
	defer rows.Close()

	var rulesList []*Rule

	for rows.Next() {
		rule, err := rst.ScanRule(rows)
		if err != nil {
			slog.Warn("Failed to scan rule", "error", err)
			continue
		}
		rulesList = append(rulesList, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rules: %w", err)
	}

	return rulesList, nil
}

func (rst *RuleStorage) ListRules(ctx context.Context, filter RuleFilter) ([]*Rule, int64, error) {
	if rst.pool == nil {
		return nil, 0, fmt.Errorf("database pool is nil")
	}
	// Строим динамический запрос с фильтрами
	query := `
		SELECT ` + ruleSelectColumns +
		` FROM rules
		WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM rules WHERE 1=1`

	args := []interface{}{}
	argPos := 1

	// Добавляем фильтры
	if filter.Severity != "" {
		query += fmt.Sprintf(" AND severity = $%d", argPos)
		countQuery += fmt.Sprintf(" AND severity = $%d", argPos)
		args = append(args, filter.Severity)
		argPos++
	}

	if filter.EnabledSet {
		query += fmt.Sprintf(" AND enabled = $%d", argPos)
		countQuery += fmt.Sprintf(" AND enabled = $%d", argPos)
		args = append(args, filter.Enabled)
		argPos++
	}

	if len(filter.Tags) > 0 {
		query += fmt.Sprintf(" AND tags && $%d", argPos)
		countQuery += fmt.Sprintf(" AND tags && $%d", argPos)
		args = append(args, filter.Tags)
		argPos++
	}

	// Получаем общее количество
	var total int64
	err := rst.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count rules: %w", err)
	}

	// Добавляем сортировку и лимиты
	query += " ORDER BY created_at DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filter.Limit)
		argPos++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filter.Offset)
	}

	// Выполняем запрос
	rows, err := rst.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()

	var rulesList []*Rule
	for rows.Next() {
		rule, err := rst.ScanRule(rows)
		if err != nil {
			slog.Warn("failed to scan rule", "error", err)
			continue
		}
		rulesList = append(rulesList, rule)
	}

	return rulesList, total, nil
}

// UpdateRuleTrigger обновляет статистику срабатывания правила
func (rst *RuleStorage) UpdateRuleTrigger(ctx context.Context, id string) error {
	if rst.pool == nil {
		return fmt.Errorf("database pool is nil")
	}
	query := `
		UPDATE rules 
		SET last_triggered = $1,
		    trigger_count = trigger_count + 1
		WHERE id = $2
	`

	_, err := rst.pool.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update rule trigger: %w", err)
	}

	return nil
}

// SetRuleEnabled включает или отключает правило
func (rst *RuleStorage) SetRuleEnabled(ctx context.Context, id string, enabled bool) error {
	if rst.pool == nil {
		return fmt.Errorf("database pool is nil")
	}
	query := `
		UPDATE rules 
		SET enabled = $1,
		    updated_at = $2
		WHERE id = $3
	`

	result, err := rst.pool.Exec(ctx, query, enabled, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to set rule enabled status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("rule not found: %s", id)
	}

	return nil
}
