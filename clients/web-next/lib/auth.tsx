"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { api, ApiError } from "./api";
import type { SessionInfo } from "./types";

// The owner session is an HttpOnly cookie; this provider only tracks whether
// the browser currently holds one. Nothing secret is stored client-side.
const LEGACY_KEYS = ["restick_access_token", "restick_user_json"];

interface AuthContextValue {
  username: string | null;
  isReady: boolean;
  login: (password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [username, setUsername] = useState<string | null>(null);
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    // Cutover cleanup: the retired community console kept a bearer token and a
    // user record in localStorage. Drop them once so no stale credential stays.
    for (const key of LEGACY_KEYS) window.localStorage.removeItem(key);

    let active = true;
    api<SessionInfo>("/api/v1/admin/session")
      .then((info) => {
        if (active) setUsername(info.username);
      })
      .catch(() => {
        if (active) setUsername(null);
      })
      .finally(() => {
        if (active) setIsReady(true);
      });
    return () => {
      active = false;
    };
  }, []);

  const login = useCallback(async (password: string) => {
    await api<void>("/api/v1/admin/session", { method: "POST", json: { password } });
    const info = await api<SessionInfo>("/api/v1/admin/session");
    setUsername(info.username);
  }, []);

  const logout = useCallback(async () => {
    try {
      await api<void>("/api/v1/admin/session", { method: "DELETE" });
    } catch (err) {
      if (!(err instanceof ApiError) || err.status !== 401) throw err;
    }
    setUsername(null);
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({ username, isReady, login, logout }),
    [username, isReady, login, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
