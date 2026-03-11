// ============================
// Agent Types
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
// Alert Types
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
// Action Types — matches server ActionLog struct
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
// Event Types (normalized_events)
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
// Rule Types
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

// ============================
// Auth Types
// ============================
export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  success: boolean;
  token?: string;
  message?: string;
}
