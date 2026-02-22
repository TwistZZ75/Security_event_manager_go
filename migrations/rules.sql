CREATE TABLE IF NOT EXISTS rules (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(500) NOT NULL,
    enabled BOOLEAN DEFAULT FALSE,
    severity VARCHAR(50) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    rule_definition JSONB NOT NULL,  -- Полное определение правила в JSON
    tags TEXT[],
    created_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(255),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_triggered TIMESTAMP,
    trigger_count BIGINT DEFAULT 0
);
CREATE INDEX idx_rules_enabled ON rules(enabled);
CREATE INDEX idx_rules_severity ON rules(severity);
CREATE INDEX idx_rules_tags ON rules USING GIN(tags);
CREATE INDEX idx_rules_created_at ON rules(created_at DESC);

-- Таблица алертов
CREATE TABLE IF NOT EXISTS alerts (
    id SERIAL PRIMARY KEY,
    rule_id VARCHAR(255) NOT NULL,
    rule_name VARCHAR(500),
    severity VARCHAR(50) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    title VARCHAR(500) NOT NULL,
    description TEXT,
    event_data JSONB,
    status VARCHAR(50) DEFAULT 'open' CHECK (status IN ('open', 'acknowledged', 'resolved', 'false_positive')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    acknowledged_at TIMESTAMP,
    acknowledged_by VARCHAR(255),
    resolved_at TIMESTAMP,
    resolved_by VARCHAR(255),
    notes TEXT,
    FOREIGN KEY (rule_id) REFERENCES rules(id) ON DELETE CASCADE
);

-- Индексы для таблицы alerts
CREATE INDEX idx_alerts_rule_id ON alerts(rule_id);
CREATE INDEX idx_alerts_status ON alerts(status);
CREATE INDEX idx_alerts_severity ON alerts(severity);
CREATE INDEX idx_alerts_created_at ON alerts(created_at DESC);
CREATE INDEX idx_alerts_status_created ON alerts(status, created_at DESC);

-- Таблица логов действий
CREATE TABLE IF NOT EXISTS actions_log (
    id SERIAL PRIMARY KEY,
    alert_id INTEGER NOT NULL,
    action_type VARCHAR(100) NOT NULL,
    target VARCHAR(500),
    parameters JSONB,
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'success', 'failed')),
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    result TEXT,
    error TEXT,
    FOREIGN KEY (alert_id) REFERENCES alerts(id) ON DELETE CASCADE
);

-- Индексы для таблицы actions_log
CREATE INDEX idx_actions_log_alert_id ON actions_log(alert_id);
CREATE INDEX idx_actions_log_status ON actions_log(status);
CREATE INDEX idx_actions_log_action_type ON actions_log(action_type);
CREATE INDEX idx_actions_log_executed_at ON actions_log(executed_at DESC);

-- Таблица состояний правил (для агрегации)
CREATE TABLE IF NOT EXISTS rule_state (
    id SERIAL PRIMARY KEY,
    rule_id VARCHAR(255) NOT NULL,
    group_key VARCHAR(500) NOT NULL,
    counter INTEGER DEFAULT 0,
    first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    state_data JSONB,
    expires_at TIMESTAMP NOT NULL,
    FOREIGN KEY (rule_id) REFERENCES rules(id) ON DELETE CASCADE,
    UNIQUE(rule_id, group_key)
);

-- Индексы для таблицы rule_state
CREATE INDEX idx_rule_state_rule_id ON rule_state(rule_id);
CREATE INDEX idx_rule_state_group_key ON rule_state(group_key);
CREATE INDEX idx_rule_state_expires_at ON rule_state(expires_at);
CREATE INDEX idx_rule_state_rule_group ON rule_state(rule_id, group_key);

-- Функция для очистки устаревших состояний
CREATE OR REPLACE FUNCTION cleanup_expired_rule_states()
RETURNS void AS $$
BEGIN
    DELETE FROM rule_state WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- Триггер для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_rules_updated_at
    BEFORE UPDATE ON rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Функция для получения статистики по алертам
CREATE OR REPLACE FUNCTION get_alert_stats(
    from_date TIMESTAMP DEFAULT NULL,
    to_date TIMESTAMP DEFAULT NULL
)
RETURNS TABLE (
    total BIGINT,
    severity_critical BIGINT,
    severity_high BIGINT,
    severity_medium BIGINT,
    severity_low BIGINT,
    status_open BIGINT,
    status_acknowledged BIGINT,
    status_resolved BIGINT,
    status_false_positive BIGINT,
    avg_resolution_hours NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*)::BIGINT as total,
        COUNT(*) FILTER (WHERE severity = 'critical')::BIGINT as severity_critical,
        COUNT(*) FILTER (WHERE severity = 'high')::BIGINT as severity_high,
        COUNT(*) FILTER (WHERE severity = 'medium')::BIGINT as severity_medium,
        COUNT(*) FILTER (WHERE severity = 'low')::BIGINT as severity_low,
        COUNT(*) FILTER (WHERE status = 'open')::BIGINT as status_open,
        COUNT(*) FILTER (WHERE status = 'acknowledged')::BIGINT as status_acknowledged,
        COUNT(*) FILTER (WHERE status = 'resolved')::BIGINT as status_resolved,
        COUNT(*) FILTER (WHERE status = 'false_positive')::BIGINT as status_false_positive,
        AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600) as avg_resolution_hours
    FROM alerts
    WHERE (from_date IS NULL OR created_at >= from_date)
      AND (to_date IS NULL OR created_at <= to_date);
END;
$$ LANGUAGE plpgsql;

-- Представление для активных правил
CREATE OR REPLACE VIEW active_rules AS
SELECT 
    id,
    name,
    severity,
    enabled,
    trigger_count,
    last_triggered,
    created_at
FROM rules
WHERE enabled = TRUE
ORDER BY trigger_count DESC, last_triggered DESC;

-- Представление для открытых алертов
CREATE OR REPLACE VIEW open_alerts AS
SELECT 
    a.id,
    a.rule_id,
    a.rule_name,
    a.severity,
    a.title,
    a.created_at,
    a.event_data,
    COUNT(al.id) as action_count
FROM alerts a
LEFT JOIN actions_log al ON a.id = al.alert_id
WHERE a.status = 'open'
GROUP BY a.id
ORDER BY a.created_at DESC;

-- Комментарии к таблицам
COMMENT ON TABLE rules IS 'Security rules for event detection and response';
COMMENT ON TABLE alerts IS 'Security alerts generated by rules';
COMMENT ON TABLE actions_log IS 'Log of all actions executed in response to alerts';
COMMENT ON TABLE rule_state IS 'State tracking for stateful rules (aggregation, correlation)';

COMMENT ON COLUMN rules.rule_definition IS 'Complete rule definition in JSON format including conditions, aggregation, and actions';
COMMENT ON COLUMN alerts.event_data IS 'Event data that triggered the alert in JSON format';
COMMENT ON COLUMN rule_state.expires_at IS 'Timestamp when this state should be cleaned up';
COMMENT ON COLUMN rule_state.state_data IS 'Additional state data for complex rules';