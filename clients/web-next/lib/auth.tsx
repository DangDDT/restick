"use client";

import { createContext, useCallback, useContext, useMemo, useState } from "react";
import type { TokenResponse, User } from "./types";

const TOKEN_KEY = "restick_access_token";
const USER_KEY = "restick_user_json";

interface AuthContextValue {
  token: string | null;
  user: User | null;
  isReady: boolean;
  isAdmin: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, displayName: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

function readStored(): { token: string | null; user: User | null } {
  if (typeof window === "undefined") return { token: null, user: null };
  const token = window.localStorage.getItem(TOKEN_KEY);
  const raw = window.localStorage.getItem(USER_KEY);
  let user: User | null = null;
  if (raw) {
    try {
      user = JSON.parse(raw) as User;
    } catch {
      user = null;
    }
  }
  return { token, user };
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [session] = useState<{ token: string | null; user: User | null }>(() => {
    const stored = readStored();
    return { token: stored.token, user: stored.user };
  });
  const [token, setToken] = useState<string | null>(session.token);
  const [user, setUser] = useState<User | null>(session.user);


  const persist = useCallback((t: string, u: User) => {
    window.localStorage.setItem(TOKEN_KEY, t);
    window.localStorage.setItem(USER_KEY, JSON.stringify(u));
    setToken(t);
    setUser(u);
  }, []);

  const login = useCallback(
    async (email: string, password: string) => {
      const res = await fetch("/api/v1/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });
      if (!res.ok) throw await toError(res);
      const data = (await res.json()) as TokenResponse;
      persist(data.access_token, data.user);
    },
    [persist],
  );

  const register = useCallback(
    async (email: string, password: string, displayName: string) => {
      const res = await fetch("/api/v1/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password, display_name: displayName }),
      });
      if (!res.ok) throw await toError(res);
      const data = (await res.json()) as TokenResponse;
      persist(data.access_token, data.user);
    },
    [persist],
  );

  const logout = useCallback(() => {
    window.localStorage.removeItem(TOKEN_KEY);
    window.localStorage.removeItem(USER_KEY);
    setToken(null);
    setUser(null);
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      token,
      user,
      isReady: true,
      isAdmin: user?.role === "admin",
      login,
      register,
      logout,
    }),
    [token, user, login, register, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}

export async function toError(res: Response): Promise<Error> {
  let message = `Request failed (${res.status})`;
  try {
    const body = (await res.json()) as { error?: { message?: string } };
    if (body?.error?.message) message = body.error.message;
  } catch {
    // non-JSON body; keep default message
  }
  return new Error(message);
}
