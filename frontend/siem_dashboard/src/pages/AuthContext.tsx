import React, { createContext, useContext, useState, useCallback } from 'react';
import { login as apiLogin } from '../api/api';

interface AuthContextType {
  isAuthenticated: boolean;
  token: string | null;
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [token, setToken] = useState<string | null>(() =>
    localStorage.getItem('siem_token')
  );

  const login = useCallback(async (username: string, password: string) => {
    const res = await apiLogin({ username, password });
    if (!res.success || !res.token) {
      throw new Error(res.message || 'Login failed');
    }
    localStorage.setItem('siem_token', res.token);
    setToken(res.token);
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('siem_token');
    setToken(null);
  }, []);

  return (
    <AuthContext.Provider value={{ isAuthenticated: !!token, token, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
