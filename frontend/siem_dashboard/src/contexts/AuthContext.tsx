import React, { createContext, useContext, useState, useCallback } from 'react';
import { login as apiLogin } from '../api/api';

interface AuthContextType {
  isAuthenticated: boolean;
  token: string | null;
  username: string;           
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [token, setToken] = useState<string | null>(() =>
    localStorage.getItem('siem_token')
  );
  // Сохраняем username отдельно, чтобы передавать на бэк при изменении статуса алёрта
  const [username, setUsername] = useState<string>(
    () => localStorage.getItem('siem_username') ?? ''
  );

  const login = useCallback(async (uname: string, password: string) => {
    const res = await apiLogin({ username: uname, password });
    if (!res.success || !res.token) {
      throw new Error(res.message || 'Login failed');
    }
    localStorage.setItem('siem_token', res.token);
    localStorage.setItem('siem_username', uname);
    setToken(res.token);
    setUsername(uname);
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('siem_token');
    localStorage.removeItem('siem_username');
    setToken(null);
    setUsername('');
  }, []);

  return (
    <AuthContext.Provider value={{ isAuthenticated: !!token, token, username, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
