import type {
  Agent, Alert, Action, NormalizedEvent, Rule,
  LoginRequest, RegisterRequest, TokenPair,
} from '../types';

const BASE_URL = '/api';

function getToken(): string | null {
  return localStorage.getItem('siem_access_token');
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const res = await fetch(`${BASE_URL}${path}`, { ...options, headers });

  if (res.status === 401) {
    // токен истёк — сбрасываем сессию
    localStorage.removeItem('siem_access_token');
    localStorage.removeItem('siem_refresh_token');
    localStorage.removeItem('siem_user');
    window.location.href = '/login';
    throw new Error('Сессия истекла, войдите снова');
  }

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }

  const data = await res.json();
  return (data ?? []) as T;
}

// ── Auth ──────────────────────────────────────────────────────────────────────

export async function login(data: LoginRequest): Promise<TokenPair> {
  return request<TokenPair>('/auth/login', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function register(data: RegisterRequest): Promise<TokenPair> {
  return request<TokenPair>('/auth/register', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function refreshToken(refresh_token: string): Promise<TokenPair> {
  return request<TokenPair>('/auth/refresh', {
    method: 'POST',
    body: JSON.stringify({ refresh_token }),
  });
}

export async function logout(refresh_token: string): Promise<void> {
  return request<void>('/auth/logout', {
    method: 'POST',
    body: JSON.stringify({ refresh_token }),
  });
}

// ── Агенты ────────────────────────────────────────────────────────────────────

export async function getAgents(): Promise<Agent[]> {
  const data = await request<Agent[] | null>('/agents');
  return data ?? [];
}

export async function getAgent(id: string): Promise<Agent> {
  return request<Agent>(`/agents/${encodeURIComponent(id)}`);
}

export async function sendAgentCommand(
  hostname: string,
  commandType: string,
  parameters: Record<string, string> = {},
): Promise<{ id: number; status: string }> {
  return request(`/agents/${encodeURIComponent(hostname)}/command`, {
    method: 'POST',
    body: JSON.stringify({ command_type: commandType, parameters }),
  });
}

// ── Алерты ────────────────────────────────────────────────────────────────────

export async function getAlerts(): Promise<Alert[]> {
  const data = await request<Alert[] | null>('/alerts');
  return data ?? [];
}

export async function getAlert(id: string): Promise<Alert> {
  return request<Alert>(`/alerts/${id}`);
}

export async function updateAlertStatus(id: number, status: string, notes: string): Promise<void> {
  return request<void>(`/alerts/${id}/status`, {
    method: 'PATCH',
    body: JSON.stringify({ status, notes }),
  });
}

// ── Действия ──────────────────────────────────────────────────────────────────

export async function getActions(): Promise<Action[]> {
  const data = await request<Action[] | null>('/actions');
  return data ?? [];
}

export async function getAction(id: string): Promise<Action> {
  return request<Action>(`/actions/${id}`);
}

// ── События ───────────────────────────────────────────────────────────────────

export async function getEvents(limit?: number): Promise<NormalizedEvent[]> {
  const url = limit ? `/events?limit=${limit}` : '/events';
  const data = await request<NormalizedEvent[] | null>(url);
  return data ?? [];
}

// ── Правила ───────────────────────────────────────────────────────────────────

export async function getRules(): Promise<Rule[]> {
  const data = await request<Rule[] | null>('/rules');
  return data ?? [];
}

export async function getRule(id: string): Promise<Rule> {
  return request<Rule>(`/rules/${id}`);
}

export async function createRule(rule: Partial<Rule>): Promise<Rule> {
  return request<Rule>('/rules', { method: 'POST', body: JSON.stringify(rule) });
}

export async function updateRule(id: string, rule: Partial<Rule>): Promise<Rule> {
  return request<Rule>(`/rules/${id}`, { method: 'PUT', body: JSON.stringify(rule) });
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

// ── Пользователи (только admin) ───────────────────────────────────────────────

export async function getUsers(): Promise<import('../types').SafeUser[]> {
  const data = await request<import('../types').SafeUser[] | null>('/users');
  return data ?? [];
}

export async function deleteUser(id: number): Promise<void> {
  return request<void>(`/users/${id}`, { method: 'DELETE' });
}

export async function updateUser(
  id: number,
  input: { email?: string; password?: string; role?: import('../types').UserRole; is_active?: boolean },
): Promise<import('../types').SafeUser> {
  return request<import('../types').SafeUser>(`/users/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  });
}
