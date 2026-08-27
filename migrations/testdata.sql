-- ============================================================
-- AGENTS
-- ============================================================

INSERT INTO agents (
    agent_id,
    hostname,
    os,
    os_version,
    agent_version,
    ip_address,
    metadata,
    status,
    registered_at,
    last_seen
)
VALUES
(
    'demo-agent-win-001',
    'WIN-CLIENT-01',
    'Windows',
    '10 Pro 22H2',
    '1.0.0',
    '192.168.1.101',
    '{
        "collectors": [
            "Security",
            "Sysmon"
        ],
        "architecture": "amd64"
    }'::jsonb,
    'online',
    NOW() - INTERVAL '7 days',
    NOW() - INTERVAL '10 seconds'
),
(
    'demo-agent-linux-001',
    'UBUNTU-SERVER-01',
    'Ubuntu',
    '24.04 LTS',
    '1.0.0',
    '192.168.1.102',
    '{
        "collectors": [
            "syslog",
            "auth.log",
            "Suricata"
        ],
        "architecture": "amd64"
    }'::jsonb,
    'online',
    NOW() - INTERVAL '5 days',
    NOW() - INTERVAL '20 seconds'
),
(
    'demo-agent-win-002',
    'WIN-WORKSTATION-02',
    'Windows',
    '10 Pro 22H2',
    '1.0.0',
    '192.168.1.103',
    '{
        "collectors": [
            "Security",
            "Sysmon"
        ],
        "architecture": "amd64"
    }'::jsonb,
    'offline',
    NOW() - INTERVAL '14 days',
    NOW() - INTERVAL '3 hours'
);


-- ============================================================
-- RULES
-- ============================================================

-- ------------------------------------------------------------
-- 1. COUNT
-- SSH brute force:
-- 5 failed SSH authentications from one host within 2 minutes.
-- ------------------------------------------------------------

INSERT INTO rules (
    id,
    name,
    enabled,
    severity,
    rule_definition,
    tags,
    created_by,
    created_at,
    updated_by,
    updated_at,
    trigger_count
)
VALUES
(
    'demo-rule-ssh-bruteforce',
    'SSH Brute Force',
    true,
    'high',
    '{
        "id": "demo-rule-ssh-bruteforce",
        "name": "SSH Brute Force",
        "os": "linux",
        "enabled": true,
        "severity": "high",
        "conditions": [
            {
                "field": "event_category",
                "operator": "equals",
                "value": "SSH Auth Failure"
            }
        ],
        "aggregation": {
            "type": "count",
            "field": "pc_name",
            "time_window": "2m",
            "threshold": 5
        },
        "actions": [],
        "tags": [
            "ssh",
            "brute-force",
            "authentication",
            "linux"
        ],
        "created_by": "demo",
        "created_at": "2026-08-20T10:00:00Z",
        "updated_by": "demo",
        "updated_at": "2026-08-20T10:00:00Z",
        "trigger_count": 0
    }'::jsonb,
    ARRAY[
        'ssh',
        'brute-force',
        'authentication',
        'linux'
    ],
    'demo',
    NOW() - INTERVAL '7 days',
    'demo',
    NOW() - INTERVAL '1 day',
    0
);


-- ------------------------------------------------------------
-- 2. SEQUENCE
-- Failed authentication followed by successful authentication.
-- ------------------------------------------------------------

INSERT INTO rules (
    id,
    name,
    enabled,
    severity,
    rule_definition,
    tags,
    created_by,
    created_at,
    updated_by,
    updated_at,
    trigger_count
)
VALUES
(
    'demo-rule-auth-sequence',
    'Failed Login Followed By Successful Login',
    true,
    'high',
    '{
        "id": "demo-rule-auth-sequence",
        "name": "Failed Login Followed By Successful Login",
        "os": "linux",
        "enabled": true,
        "severity": "high",
        "conditions": [
            {
                "field": "event_category",
                "operator": "equals",
                "value": "SSH Auth Failure"
            }
        ],
        "aggregation": {
            "type": "sequence",
            "field": "pc_name",
            "time_window": "5m",
            "steps": [
                [
                    {
                        "field": "event_category",
                        "operator": "equals",
                        "value": "SSH Auth Success"
                    }
                }
            ]
        },
        "actions": [],
        "tags": [
            "ssh",
            "authentication",
            "sequence",
            "linux"
        ],
        "created_by": "demo",
        "created_at": "2026-08-20T10:00:00Z",
        "updated_by": "demo",
        "updated_at": "2026-08-20T10:00:00Z",
        "trigger_count": 0
    }'::jsonb,
    ARRAY[
        'ssh',
        'authentication',
        'sequence',
        'linux'
    ],
    'demo',
    NOW() - INTERVAL '6 days',
    'demo',
    NOW() - INTERVAL '1 day',
    0
);


-- ------------------------------------------------------------
-- 3. THRESHOLD / SUM
-- Sum of transferred bytes exceeds 10000 within 5 minutes.
--
-- The numeric value is taken from:
-- raw_log.bytes_toserver
-- ------------------------------------------------------------

INSERT INTO rules (
    id,
    name,
    enabled,
    severity,
    rule_definition,
    tags,
    created_by,
    created_at,
    updated_by,
    updated_at,
    trigger_count
)
VALUES
(
    'demo-rule-network-bytes',
    'High Network Traffic',
    true,
    'medium',
    '{
        "id": "demo-rule-network-bytes",
        "name": "High Network Traffic",
        "os": "linux",
        "enabled": true,
        "severity": "medium",
        "conditions": [
            {
                "field": "event_category",
                "operator": "equals",
                "value": "Network Traffic"
            }
        ],
        "aggregation": {
            "type": "threshold",
            "field": "pc_name",
            "time_window": "5m",
            "value_field": "raw_log.bytes_toserver",
            "operator": "sum",
            "threshold_value": 10000
        },
        "actions": [],
        "tags": [
            "network",
            "traffic",
            "threshold",
            "linux"
        ],
        "created_by": "demo",
        "created_at": "2026-08-20T10:00:00Z",
        "updated_by": "demo",
        "updated_at": "2026-08-20T10:00:00Z",
        "trigger_count": 0
    }'::jsonb,
    ARRAY[
        'network',
        'traffic',
        'threshold',
        'linux'
    ],
    'demo',
    NOW() - INTERVAL '5 days',
    'demo',
    NOW() - INTERVAL '1 day',
    0
);


-- ------------------------------------------------------------
-- 4. SIMPLE CONDITION
-- Suspicious PowerShell execution.
-- No aggregation: every matching event creates an alert.
-- ------------------------------------------------------------

INSERT INTO rules (
    id,
    name,
    enabled,
    severity,
    rule_definition,
    tags,
    created_by,
    created_at,
    updated_by,
    updated_at,
    trigger_count
)
VALUES
(
    'demo-rule-powershell',
    'Suspicious PowerShell Execution',
    true,
    'critical',
    '{
        "id": "demo-rule-powershell",
        "name": "Suspicious PowerShell Execution",
        "os": "windows",
        "enabled": true,
        "severity": "critical",
        "conditions": [
            {
                "field": "process_name",
                "operator": "equals",
                "value": "powershell.exe"
            }
        ],
        "actions": [],
        "tags": [
            "windows",
            "powershell",
            "process",
            "execution"
        ],
        "created_by": "demo",
        "created_at": "2026-08-20T10:00:00Z",
        "updated_by": "demo",
        "updated_at": "2026-08-20T10:00:00Z",
        "trigger_count": 0
    }'::jsonb,
    ARRAY[
        'windows',
        'powershell',
        'process',
        'execution'
    ],
    'demo',
    NOW() - INTERVAL '4 days',
    'demo',
    NOW() - INTERVAL '1 day',
    0
);


-- ------------------------------------------------------------
-- 5. DISTINCT COUNT
-- More than 3 different usernames observed on one host.
-- ------------------------------------------------------------

INSERT INTO rules (
    id,
    name,
    enabled,
    severity,
    rule_definition,
    tags,
    created_by,
    created_at,
    updated_by,
    updated_at,
    trigger_count
)
VALUES
(
    'demo-rule-distinct-users',
    'Multiple Users On Host',
    true,
    'medium',
    '{
        "id": "demo-rule-distinct-users",
        "name": "Multiple Users On Host",
        "os": "",
        "enabled": true,
        "severity": "medium",
        "conditions": [
            {
                "field": "event_category",
                "operator": "equals",
                "value": "User Activity"
            }
        ],
        "aggregation": {
            "type": "threshold",
            "field": "pc_name",
            "time_window": "10m",
            "value_field": "username",
            "operator": "distinct_count",
            "threshold_value": 3
        },
        "actions": [],
        "tags": [
            "users",
            "authentication",
            "distinct-count"
        ],
        "created_by": "demo",
        "created_at": "2026-08-20T10:00:00Z",
        "updated_by": "demo",
        "updated_at": "2026-08-20T10:00:00Z",
        "trigger_count": 0
    }'::jsonb,
    ARRAY[
        'users',
        'authentication',
        'distinct-count'
    ],
    'demo',
    NOW() - INTERVAL '3 days',
    'demo',
    NOW() - INTERVAL '1 day',
    0
);


-- ============================================================
-- NORMALIZED EVENTS
-- ============================================================

-- ------------------------------------------------------------
-- SSH brute-force events.
--
-- NOTE:
-- These events are intentionally close together in time.
-- The rule itself is not automatically triggered by inserting
-- rows into the database; it is triggered when the Rule Engine
-- evaluates incoming events.
-- ------------------------------------------------------------

INSERT INTO normalized_events (
    id,
    pc_name,
    username,
    event_description,
    event_category,
    process_name,
    severity,
    timestamp,
    os,
    source,
    raw_log
)
VALUES
(
    'demo-event-ssh-001',
    'UBUNTU-SERVER-01',
    'root',
    'Failed SSH authentication attempt',
    'SSH Auth Failure',
    'sshd',
    'WARNING',
    NOW() - INTERVAL '40 minutes',
    'Ubuntu',
    'auth.log',
    'Failed password for root from 192.168.1.50'
),
(
    'demo-event-ssh-002',
    'UBUNTU-SERVER-01',
    'root',
    'Failed SSH authentication attempt',
    'SSH Auth Failure',
    'sshd',
    'WARNING',
    NOW() - INTERVAL '39 minutes 50 seconds',
    'Ubuntu',
    'auth.log',
    'Failed password for root from 192.168.1.50'
),
(
    'demo-event-ssh-003',
    'UBUNTU-SERVER-01',
    'root',
    'Failed SSH authentication attempt',
    'SSH Auth Failure',
    'sshd',
    'WARNING',
    NOW() - INTERVAL '39 minutes 40 seconds',
    'Ubuntu',
    'auth.log',
    'Failed password for root from 192.168.1.50'
),
(
    'demo-event-ssh-004',
    'UBUNTU-SERVER-01',
    'root',
    'Failed SSH authentication attempt',
    'SSH Auth Failure',
    'sshd',
    'WARNING',
    NOW() - INTERVAL '39 minutes 30 seconds',
    'Ubuntu',
    'auth.log',
    'Failed password for root from 192.168.1.50'
),
(
    'demo-event-ssh-005',
    'UBUNTU-SERVER-01',
    'root',
    'Failed SSH authentication attempt',
    'SSH Auth Failure',
    'sshd',
    'WARNING',
    NOW() - INTERVAL '39 minutes 20 seconds',
    'Ubuntu',
    'auth.log',
    'Failed password for root from 192.168.1.50'
);


-- ------------------------------------------------------------
-- Authentication sequence.
--
-- Failed login followed by successful login.
-- ------------------------------------------------------------

INSERT INTO normalized_events (
    id,
    pc_name,
    username,
    event_description,
    event_category,
    process_name,
    severity,
    timestamp,
    os,
    source,
    raw_log
)
VALUES
(
    'demo-event-sequence-001',
    'UBUNTU-SERVER-01',
    'admin',
    'Failed SSH authentication attempt',
    'SSH Auth Failure',
    'sshd',
    'WARNING',
    NOW() - INTERVAL '25 minutes',
    'Ubuntu',
    'auth.log',
    'Failed password for admin from 192.168.1.60'
),
(
    'demo-event-sequence-002',
    'UBUNTU-SERVER-01',
    'admin',
    'Successful SSH authentication',
    'SSH Auth Success',
    'sshd',
    'INFO',
    NOW() - INTERVAL '24 minutes',
    'Ubuntu',
    'auth.log',
    'Accepted publickey for admin from 192.168.1.60'
);


-- ------------------------------------------------------------
-- Network traffic events for threshold/sum aggregation.
-- Total bytes_toserver > 10000.
-- ------------------------------------------------------------

INSERT INTO normalized_events (
    id,
    pc_name,
    username,
    event_description,
    event_category,
    process_name,
    severity,
    timestamp,
    os,
    source,
    raw_log
)
VALUES
(
    'demo-event-network-001',
    'UBUNTU-SERVER-01',
    'root',
    'Network traffic detected',
    'Network Traffic',
    'suricata',
    'INFO',
    NOW() - INTERVAL '15 minutes',
    'Ubuntu',
    'Suricata',
    '{
        "bytes_toserver": 4000,
        "bytes_toclient": 1200
    }'
),
(
    'demo-event-network-002',
    'UBUNTU-SERVER-01',
    'root',
    'Network traffic detected',
    'Network Traffic',
    'suricata',
    'INFO',
    NOW() - INTERVAL '14 minutes',
    'Ubuntu',
    'Suricata',
    '{
        "bytes_toserver": 3500,
        "bytes_toclient": 1000
    }'
),
(
    'demo-event-network-003',
    'UBUNTU-SERVER-01',
    'root',
    'Network traffic detected',
    'Network Traffic',
    'suricata',
    'INFO',
    NOW() - INTERVAL '13 minutes',
    'Ubuntu',
    'Suricata',
    '{
        "bytes_toserver": 5000,
        "bytes_toclient": 800
    }'
);


-- ------------------------------------------------------------
-- PowerShell events.
-- ------------------------------------------------------------

INSERT INTO normalized_events (
    id,
    pc_name,
    username,
    event_description,
    event_category,
    process_name,
    severity,
    timestamp,
    os,
    source,
    raw_log
)
VALUES
(
    'demo-event-powershell-001',
    'WIN-CLIENT-01',
    'john.doe',
    'PowerShell process started',
    'Process Creation',
    'powershell.exe',
    'CRITICAL',
    NOW() - INTERVAL '8 minutes',
    'Windows',
    'Sysmon',
    'EventID=1 Image=C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe'
),
(
    'demo-event-powershell-002',
    'WIN-CLIENT-01',
    'john.doe',
    'PowerShell process started',
    'Process Creation',
    'powershell.exe',
    'CRITICAL',
    NOW() - INTERVAL '7 minutes',
    'Windows',
    'Sysmon',
    'EventID=1 Image=C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe'
);


-- ------------------------------------------------------------
-- Events for distinct_count.
-- Three different usernames on one host.
-- ------------------------------------------------------------

INSERT INTO normalized_events (
    id,
    pc_name,
    username,
    event_description,
    event_category,
    process_name,
    severity,
    timestamp,
    os,
    source,
    raw_log
)
VALUES
(
    'demo-event-users-001',
    'WIN-WORKSTATION-02',
    'alice',
    'User activity detected',
    'User Activity',
    NULL,
    'INFO',
    NOW() - INTERVAL '5 minutes',
    'Windows',
    'Security',
    'User activity event'
),
(
    'demo-event-users-002',
    'WIN-WORKSTATION-02',
    'bob',
    'User activity detected',
    'User Activity',
    NULL,
    'INFO',
    NOW() - INTERVAL '4 minutes',
    'Windows',
    'Security',
    'User activity event'
),
(
    'demo-event-users-003',
    'WIN-WORKSTATION-02',
    'charlie',
    'User activity detected',
    'User Activity',
    NULL,
    'INFO',
    NOW() - INTERVAL '3 minutes',
    'Windows',
    'Security',
    'User activity event'
);


-- ============================================================
-- DEMONSTRATION ALERTS
-- ============================================================
--
-- Эти записи нужны для демонстрации интерфейса.
-- Они НЕ являются результатом выполнения Rule Engine
-- во время INSERT.
-- ============================================================

INSERT INTO alerts (
    rule_id,
    rule_name,
    severity,
    title,
    description,
    event_data,
    status,
    created_at,
    acknowledged_at,
    acknowledged_by,
    resolved_at,
    resolved_by,
    notes
)
VALUES
(
    'demo-rule-ssh-bruteforce',
    'SSH Brute Force',
    'high',
    'Rule triggered: SSH Brute Force',
    'Rule ''SSH Brute Force'' triggered. User: root, PC: UBUNTU-SERVER-01, Event: Failed SSH authentication attempt',
    '{
        "id": "demo-event-ssh-005",
        "pc_name": "UBUNTU-SERVER-01",
        "username": "root",
        "event_description": "Failed SSH authentication attempt",
        "event_category": "SSH Auth Failure",
        "process_name": "sshd",
        "severity": "WARNING",
        "os": "Ubuntu",
        "source": "auth.log",
        "aggregation_type": "count",
        "counter": 5,
        "threshold": 5,
        "group_key": "pc_name:UBUNTU-SERVER-01",
        "field": "pc_name",
        "window": "2m"
    }'::jsonb,
    'open',
    NOW() - INTERVAL '35 minutes',
    NULL,
    NULL,
    NULL,
    NULL,
    NULL
),
(
    'demo-rule-powershell',
    'Suspicious PowerShell Execution',
    'critical',
    'Rule triggered: Suspicious PowerShell Execution',
    'Rule ''Suspicious PowerShell Execution'' triggered. User: john.doe, PC: WIN-CLIENT-01, Event: PowerShell process started',
    '{
        "id": "demo-event-powershell-001",
        "pc_name": "WIN-CLIENT-01",
        "username": "john.doe",
        "event_description": "PowerShell process started",
        "event_category": "Process Creation",
        "process_name": "powershell.exe",
        "severity": "CRITICAL",
        "os": "Windows",
        "source": "Sysmon"
    }'::jsonb,
    'acknowledged',
    NOW() - INTERVAL '20 minutes',
    NOW() - INTERVAL '15 minutes',
    NULL,
    NULL,
    NULL,
    'Investigation in progress.'
),
(
    'demo-rule-auth-sequence',
    'Failed Login Followed By Successful Login',
    'high',
    'Rule triggered: Failed Login Followed By Successful Login',
    'Rule ''Failed Login Followed By Successful Login'' triggered. User: admin, PC: UBUNTU-SERVER-01, Event: Successful SSH authentication',
    '{
        "id": "demo-event-sequence-002",
        "pc_name": "UBUNTU-SERVER-01",
        "username": "admin",
        "event_description": "Successful SSH authentication",
        "event_category": "SSH Auth Success",
        "process_name": "sshd",
        "severity": "INFO",
        "os": "Ubuntu",
        "source": "auth.log",
        "aggregation_type": "sequence",
        "steps_total": 2,
        "group_key": "pc_name:UBUNTU-SERVER-01",
        "window": "5m"
    }'::jsonb,
    'resolved',
    NOW() - INTERVAL '2 days',
    NOW() - INTERVAL '2 days' + INTERVAL '10 minutes',
    NULL,
    NOW() - INTERVAL '2 days' + INTERVAL '30 minutes',
    NULL,
    'Demonstration alert.'
),
(
    'demo-rule-network-bytes',
    'High Network Traffic',
    'medium',
    'Rule triggered: High Network Traffic',
    'Rule ''High Network Traffic'' triggered. Network traffic exceeded the configured threshold.',
    '{
        "pc_name": "UBUNTU-SERVER-01",
        "username": "root",
        "event_category": "Network Traffic",
        "source": "Suricata",
        "aggregation_type": "threshold",
        "operator": "sum",
        "value_field": "raw_log.bytes_toserver",
        "current_value": 12500,
        "threshold_value": 10000,
        "group_key": "pc_name:UBUNTU-SERVER-01",
        "window": "5m"
    }'::jsonb,
    'false_positive',
    NOW() - INTERVAL '1 day',
    NOW() - INTERVAL '23 hours 50 minutes',
    NULL,
    NOW() - INTERVAL '23 hours 30 minutes',
    NULL,
    'Demonstration alert.'
);
