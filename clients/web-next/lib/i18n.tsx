"use client";

import { createContext, useCallback, useContext, useMemo, useSyncExternalStore } from "react";
import { LOCALE_STORAGE_KEY, messages, vi, type Locale, type MessageKey } from "./i18n-messages";

export type { Locale, MessageKey };

interface I18nContextValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: MessageKey) => string;
}

const I18nContext = createContext<I18nContextValue | null>(null);
const LOCALE_CHANGED = "restick-locale-change";

function subscribe(onChange: () => void) {
  window.addEventListener("storage", onChange);
  window.addEventListener(LOCALE_CHANGED, onChange);
  return () => {
    window.removeEventListener("storage", onChange);
    window.removeEventListener(LOCALE_CHANGED, onChange);
  };
}

function storedLocale(): Locale {
  return window.localStorage.getItem(LOCALE_STORAGE_KEY) === "en" ? "en" : "vi";
}

export function I18nProvider({ children }: { children: React.ReactNode }) {
  // Static export hydrates in Vietnamese before reading the browser preference.
  const locale = useSyncExternalStore(subscribe, storedLocale, () => "vi" as Locale);
  const setLocale = useCallback((next: Locale) => {
    window.localStorage.setItem(LOCALE_STORAGE_KEY, next);
    window.dispatchEvent(new Event(LOCALE_CHANGED));
  }, []);
  const value = useMemo<I18nContextValue>(
    () => ({ locale, setLocale, t: (key) => messages[locale][key] ?? vi[key] }),
    [locale, setLocale],
  );
  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nContextValue {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used within I18nProvider");
  return ctx;
}
