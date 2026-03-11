import type {
  Agent,
  Alert,
  Action,
  NormalizedEvent,
  Rule,
  LoginRequest,
  LoginResponse,
} from '../types';

const BASE_URL = '/api';

function getToken(): string | null {
  return localStorage.getItem('siem_token');
}

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const res = await fetch(`${BASE_URL}${path}`, { ...options, headers });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }

  const data = await res.json();
  return (data ?? []) as T;
}

// Auth
export async function login(data: LoginRequest): Promise<LoginResponse> {
  return request<LoginResponse>('/auth/login', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

// Agents
export async function getAgents(): Promise<Agent[]> {
  const data = await request<Agent[] | null>('/agents');
  return data ?? [];
}

export async function getAgent(id: string): Promise<Agent> {
  return request<Agent>(`/agents/${id}`);
}

// Alerts
export async function getAlerts(): Promise<Alert[]> {
  const data = await request<Alert[] | null>('/alerts');
  return data ?? [];
}

export async function getAlert(id: string): Promise<Alert> {
  return request<Alert>(`/alerts/${id}`);
}

export async function updateAlertStatus(
  id: number,
  status: string,
  notes: string
): Promise<void> {
  return request<void>(`/alerts/${id}/status`, {
    method: 'PATCH',
    body: JSON.stringify({ status, notes }),
  });
}

// Actions
export async function getActions(): Promise<Action[]> {
  const data = await request<Action[] | null>('/actions');
  return data ?? [];
}

export async function getAction(id: string): Promise<Action> {
  return request<Action>(`/actions/${id}`);
}

// Events
export async function getEvents(): Promise<NormalizedEvent[]> {
  const data = await request<NormalizedEvent[] | null>('/events');
  return data ?? [];
}

// Rules
export async function getRules(): Promise<Rule[]> {
  const data = await request<Rule[] | null>('/rules');
  return data ?? [];
}

export async function getRule(id: string): Promise<Rule> {
  return request<Rule>(`/rules/${id}`);
}

export async function createRule(rule: Partial<Rule>): Promise<Rule> {
  return request<Rule>('/rules', {
    method: 'POST',
    body: JSON.stringify(rule),
  });
}

export async function updateRule(id: string, rule: Partial<Rule>): Promise<Rule> {
  return request<Rule>(`/rules/${id}`, {
    method: 'PUT',
    body: JSON.stringify(rule),
  });
}

export async function setRuleEnabled(id: string, enabled: boolean): Promise<void> {
  return request<void>(`/rules/${id}/enabled`, {
    method: 'PATCH',
    body: JSON.stringify({ enabled }),
  });
}

export async function deleteRule(id: string): Promise<void> {
  return request<void>(`/rules/${id}`, { method: 'DELETE' });
}
