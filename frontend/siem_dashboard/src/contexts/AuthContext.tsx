import React, { createContext, useContext, useState, useCallback, useEffect, useRef } from 'react';
import { login as apiLogin, register as apiRegister, refreshToken as apiRefresh, logout as apiLogout } from '../api/api';
import type { SafeUser, TokenPair } from '../types';

interface AuthContextType {
  isAuthenticated: boolean;
  token: string | null;
  user: SafeUser | null;
  username: string;
  login: (username: string, password: string) => Promise<void>;
  register: (username: string, email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | null>(null);

const TOKEN_KEY    = 'siem_access_token';
const REFRESH_KEY  = 'siem_refresh_token';
const USER_KEY     = 'siem_user';

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [token, setToken]   = useState<string | null>(() => localStorage.getItem(TOKEN_KEY));
  const [user, setUser]     = useState<SafeUser | null>(() => {
    try { return JSON.parse(localStorage.getItem(USER_KEY) ?? 'null'); } catch { return null; }
  });
  const refreshTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Сохраняем токены и планируем автообновление
  const storeTokens = useCallback((pair: TokenPair) => {
    localStorage.setItem(TOKEN_KEY, pair.access_token);
    localStorage.setItem(REFRESH_KEY, pair.refresh_token);
    if (pair.user) {
      localStorage.setItem(USER_KEY, JSON.stringify(pair.user));
      setUser(pair.user);
    }
    setToken(pair.access_token);

    // Автообновление за 60 секунд до истечения
    if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current);
    const delay = Math.max((pair.expires_in - 60) * 1000, 10_000);
    refreshTimerRef.current = setTimeout(async () => {
      const rt = localStorage.getItem(REFRESH_KEY);
      if (!rt) return;
      try {
        const newPair = await apiRefresh(rt);
        storeTokens(newPair);
      } catch {
        clearAuth();
      }
    }, delay);
  }, []);

  const clearAuth = useCallback(() => {
    if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current);
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(REFRESH_KEY);
    localStorage.removeItem(USER_KEY);
    setToken(null);
    setUser(null);
  }, []);

  // Восстановление сессии при загрузке — пробуем обновить токен
  useEffect(() => {
    const rt = localStorage.getItem(REFRESH_KEY);
    if (!rt || token) return;

    apiRefresh(rt)
      .then(storeTokens)
      .catch(clearAuth);
  }, []);

  const login = useCallback(async (username: string, password: string) => {
    const pair = await apiLogin({ username, password });
    storeTokens(pair);
  }, [storeTokens]);

  const register = useCallback(async (username: string, email: string, password: string) => {
    const pair = await apiRegister({ username, email, password });
    storeTokens(pair);
  }, [storeTokens]);

  const logout = useCallback(async () => {
    const rt = localStorage.getItem(REFRESH_KEY);
    try { if (rt) await apiLogout(rt); } catch { /* игнорируем */ }
    clearAuth();
  }, [clearAuth]);

  return (
    <AuthContext.Provider value={{
      isAuthenticated: !!token,
      token,
      user,
      username: user?.username ?? '',
      login,
      register,
      logout,
    }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
