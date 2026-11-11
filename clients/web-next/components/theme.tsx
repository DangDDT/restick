"use client";

import { useCallback, useSyncExternalStore } from "react";

type Theme = "light" | "dark";
const KEY = "restick_theme";
const EVENT = "restick-theme-change";

function resolve(): Theme {
  const explicit = document.documentElement.dataset.theme;
  if (explicit === "light" || explicit === "dark") return explicit;
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function subscribe(onChange: () => void) {
  const media = window.matchMedia("(prefers-color-scheme: dark)");
  media.addEventListener("change", onChange);
  window.addEventListener(EVENT, onChange);
  return () => {
    media.removeEventListener("change", onChange);
    window.removeEventListener(EVENT, onChange);
  };
}

/** Current resolved theme plus a toggle; the choice is persisted per browser. */
export function useTheme() {
  const theme = useSyncExternalStore(subscribe, resolve, () => "light" as Theme);
  const toggle = useCallback(() => {
    const next: Theme = resolve() === "dark" ? "light" : "dark";
    document.documentElement.dataset.theme = next;
    try {
      window.localStorage.setItem(KEY, next);
    } catch {
      // Storage blocked: the choice still applies for this page view.
    }
    window.dispatchEvent(new Event(EVENT));
  }, []);
  return { theme, toggle };
}
