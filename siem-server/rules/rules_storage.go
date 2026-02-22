package rules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RuleStorage struct {
	pool *pgxpool.Pool
}

type RuleStorageInterface interface {
	AddRule(ctx context.Context, rule *Rule) error
	RemoveRule(ctx context.Context, ruleId string) error
	UpdateRule(ctx context.Context, rule *Rule) error
	GetAllRules(ctx context.Context) ([]*Rule, error)
	GetRule(ctx context.Context, ruleId string) (*Rule, error)
	GetRulesCount() (int, error)
}

// конструктор
func NewRuleStorage(pool *pgxpool.Pool) *RuleStorage {
	return &RuleStorage{pool: pool}
}

// функция добавления правил в БД
// принимает контекст и правило
// возвращает ошибку
func (rst *RuleStorage) AddRule(ctx context.Context, rule *Rule) error {

	// if rst.CheckPool() != nil{
	// 	log.Print("database pool is nil")
	// 	return fmt.Errorf("database pool is nil")
	// }

	//проверяем наличие соединений с БД
	if rst.pool == nil {
		log.Print("database pool is nil")
		return fmt.Errorf("database pool is nil")
	}

	ruleJSON, err_json := rst.SerializeStructIntoJson(rule)
	if err_json != nil {
		return fmt.Errorf("json marshal failed: %v", err_json)
	}

	//пишем запрос к БД
	query := `
	INSERT INTO rules (id, name, enabled, severity, 
	rule_definition, tags, created_by, created_at, 
	updated_by, updated_at, last_triggered, trigger_count)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	ON CONFLICT (id) DO NOTHING
	`
	_, err := rst.pool.Exec(ctx, query,
		rule.ID,
		rule.Name,
		rule.Enabled,
		rule.Severity,
		ruleJSON,
		rule.Tags,
		rule.CreatedBy,
		rule.CreatedAt,
		rule.UpdatedBy,
		rule.UpdatedAt,
		rule.LastTriggered,
		rule.TriggerCount,
	)
	//добавляем что нужно сохранить
	if err != nil {
		log.Printf("exec failed: %v", err)
		return fmt.Errorf("exec failed: %v", err)
	}
	return nil
}

// func (rst *RuleStorage) CheckPool() error {
// 	//проверяем наличие соединений с БД
// 	if rst.pool == nil {
// 		log.Print("database pool is nil")
// 		return fmt.Errorf("database pool is nil")
// 	}
// 	return nil
// }

// преобразование структуры Go в структуру json
// принимает структуру Go
// возвращает json и ошибку
func (rst *RuleStorage) SerializeStructIntoJson(rule *Rule) ([]byte, error) {
	ruleJSON, err := json.Marshal(rule) // сериализуем структуру в JSON
	if err != nil {
		return nil, err
	}
	return ruleJSON, nil
}

// функция удаления правила
// принимает контекст и id правила
// возвращает ошибку
func (rst *RuleStorage) RemoveRule(ctx context.Context, id string) error {
	//проверяем наличие соединений с БД
	if rst.pool == nil {
		log.Print("database pool is nil")
		return fmt.Errorf("database pool is nil")
	}

	query := `DELETE FROM rules WHERE id = $1`
	_, err := rst.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("Unable to delete %v, %v", id, err)
	}
	return nil
}

// функиця обновления правила
// принимает контекст и правило
// возвращает ошибку
func (rst *RuleStorage) UpdateRule(ctx context.Context, rule *Rule) error {
	//проверяем наличие соединений с БД
	if rst.pool == nil {
		log.Print("database pool is nil")
		return fmt.Errorf("database pool is nil")
	}

	ruleJSON, err_json := rst.SerializeStructIntoJson(rule)
	if err_json != nil {
		return fmt.Errorf("json marshal failed: %v", err_json)
	}

	query := `UPDATE rule SET 
	name = $2,
    enabled = $3,
    severity = $4,
    rule_definition = $5,
    tags = $6,
    updated_by = $7,
    updated_at = Now() 
	WHERE id = $1
	AND (name, enabled, severity, rule_definition, tags, updated_by)
    IS DISTINCT FROM ($2, $3, $4, $5, $6, $7)`

	_, err := rst.pool.Exec(ctx, query,
		rule.ID,
		rule.Name,
		rule.Enabled,
		rule.Severity,
		ruleJSON,
		rule.Tags,
		rule.UpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("Unable to update: %v", err)
	}
	return nil
}

// загружает все правила из БД
// принимает контекст
// возвращает массив правил и ошибку
func (rst *RuleStorage) GetAllRules(ctx context.Context) ([]*Rule, error) {
	//проверяем наличие соединений с БД
	if rst.pool == nil {
		log.Print("database pool is nil")
		return nil, fmt.Errorf("database pool is nil")
	}
	//запрос к бд
	query := `
        SELECT 
            id,
            name,
            enabled,
            severity,
            rule_definition,
            tags,
            created_by,
            created_at,
			COALESCE(updated_by, 'noone') as updated_by,
            updated_at,
            last_triggered,
            trigger_count
        FROM rules
        ORDER BY created_at DESC
    `
	//получаем строки из бд согласно запросу
	rows, err := rst.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %v", err)
	}
	defer rows.Close()

	//создаём массив объектов *Rule
	var rulesList []*Rule

	//для каждой строки
	for rows.Next() {
		//структура правила
		var (
			id             string
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
		err := rows.Scan(
			&id,
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
			return nil, fmt.Errorf("failed to scan rule row: %v", err)
		}

		// Десериализуем rule_definition из JSONB
		var definition Rule
		if err := json.Unmarshal(definitionJSON, &definition); err != nil {
			// Логируем ошибку, но продолжаем загрузку других правил
			fmt.Printf("Warning: failed to unmarshal rule %s: %v\n", id, err)
			continue
		}

		// Создаем объект Rule
		rule := &Rule{
			ID:            id,
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

		rulesList = append(rulesList, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rules: %v", err)
	}

	return rulesList, nil
}

// функция получения одного правила
// принимает контекст и id правила
// возвращает правило
func (rs *RuleStorage) GetRule(ctx context.Context, id string) (*Rule, error) {

	//запрос к бд
	query := `
        SELECT 
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
        FROM rules
        WHERE id = $1
    `

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
	err := rs.pool.QueryRow(ctx, query, id).Scan(
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
		return nil, fmt.Errorf("failed to scan rule row: %v", err)
	}

	// Десериализуем rule_definition из JSONB
	var definition Rule
	if err := json.Unmarshal(definitionJSON, &definition); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rule definition for rule %s: %v", ruleID, err)
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
func (rs *RuleStorage) GetRulesCount(ctx context.Context) (int, error) {
	//проверяем наличие соединений с БД
	if rs.pool == nil {
		return 0, fmt.Errorf("database pool is nil")
	}

	//запрос к бд
	query := `SELECT COUNT(*) FROM rules`

	var count int
	err := rs.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count rules: %v", err)
	}

	return count, nil
}
