// ============================
// Пользователи и аутентификация
// ============================
export type UserRole = 'admin' | 'analyst' | 'viewer';

export interface SafeUser {
  id: number;
  username: string;
  email: string;
  role: UserRole;
  is_active: boolean;
  created_at: string;
  last_login_at?: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: SafeUser;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
  role?: UserRole;
}

export interface RefreshRequest {
  refresh_token: string;
}

// ============================
// Агенты
// ============================
export interface Agent {
  ID: number;
  AgentID: string;
  Hostname: string;
  OS: string;
  OSVersion: string;
  AgentVersion: string;
  IPAddress: string;
  Metadata: Record<string, string>;
  Status: 'online' | 'offline' | 'error';
  RegisteredAt: string;
  LastSeen: string;
}

// ============================
// Алерты
// ============================
export type AlertSeverity = 'low' | 'medium' | 'high' | 'critical';
export type AlertStatus = 'open' | 'acknowledged' | 'resolved' | 'false_positive';

export interface Alert {
  id: number;
  rule_id: string;
  rule_name: string;
  severity: AlertSeverity;
  title: string;
  description: string;
  event_data: Record<string, unknown>;
  status: AlertStatus;
  created_at: string;
  updated_at?: string;
  acknowledged_at?: string;
  acknowledged_by?: string;
  resolved_at?: string;
  resolved_by?: string;
  notes?: string;
}

// ============================
// Действия
// ============================
export type ActionStatus = 'pending' | 'success' | 'failed';

export interface Action {
  id: number;
  alert_id: number;
  action_type: string;
  target: string;
  parameters: Record<string, unknown>;
  status: ActionStatus;
  executed_at: string;
  result?: string;
  error?: string;
}

// ============================
// События (normalized_events)
// ============================
export interface NormalizedEvent {
  id: string;
  pc_name: string;
  username: string;
  event_description: string;
  event_category: string;
  process_name: string;
  severity: string;
  timestamp: string;
  os: string;
  source: string;
}

// ============================
// Правила
// ============================
export type RuleSeverity = 'low' | 'medium' | 'high' | 'critical';

export interface RuleCondition {
  field: string;
  operator: string;
  value: string | number | boolean | string[];
}

export interface RuleAggregation {
  type: 'count' | 'sequence' | 'threshold';
  field?: string;
  threshold?: number;
  time_window?: string;
  group_by?: string[];
  operator?: string;
}

export interface RuleAction {
  type: string;
  parameters: Record<string, unknown>;
}

export interface Rule {
  id: string;
  name: string;
  description?: string;
  severity: RuleSeverity;
  enabled: boolean;
  os?: string;
  conditions: RuleCondition[];
  aggregation?: RuleAggregation;
  actions: RuleAction[];
  tags?: string[];
  created_by?: string;
  created_at: string;
  updated_at?: string;
  updated_by?: string;
  trigger_count?: number;
  last_triggered?: string;
}
