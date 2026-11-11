"use client";

import { useEffect, useState } from "react";
import { useI18n, type MessageKey } from "@/lib/i18n";

const TONES: Record<string, string> = {
  green: "bg-emerald-100 text-emerald-700",
  amber: "bg-amber-100 text-amber-700",
  red: "bg-red-100 text-red-700",
  gray: "bg-zinc-100 text-zinc-600",
  blue: "bg-blue-100 text-blue-700",
};

const STATUS: Record<string, { tone: string; label: MessageKey }> = {
  draft: { tone: "amber", label: "statusDraft" },
  ready: { tone: "green", label: "statusReady" },
  revoked: { tone: "gray", label: "statusRevoked" },
  queued: { tone: "blue", label: "statusQueued" },
  running: { tone: "blue", label: "statusRunning" },
  succeeded: { tone: "green", label: "statusSucceeded" },
  failed: { tone: "red", label: "statusFailed" },
  interrupted: { tone: "red", label: "statusInterrupted" },
  active: { tone: "green", label: "statusActive" },
  inactive: { tone: "gray", label: "statusInactive" },
};

export function StatusPill({ status }: { status: string }) {
  const { t } = useI18n();
  const known = STATUS[status];
  const tone = TONES[known?.tone ?? "gray"];
  return (
    <span
      className={`inline-flex shrink-0 items-center rounded-full px-2.5 py-0.5 text-xs font-semibold ${tone}`}
    >
      {known ? t(known.label) : status}
    </span>
  );
}

export function Spinner() {
  const { t } = useI18n();
  return (
    <div className="flex items-center justify-center py-12" role="status" aria-label={t("loading")}>
      <div className="h-6 w-6 animate-spin rounded-full border-2 border-zinc-300 border-t-indigo-600" />
    </div>
  );
}

export function ErrorBanner({ message }: { message: string }) {
  return (
    <div role="alert" className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
      {message}
    </div>
  );
}

export function SuccessBanner({ message }: { message: string }) {
  return (
    <div role="status" className="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800">
      {message}
    </div>
  );
}

export function EmptyState({ title, hint }: { title: string; hint?: string }) {
  return (
    <div className="rounded-lg border border-dashed border-zinc-300 bg-white px-6 py-12 text-center">
      <p className="text-sm font-medium text-zinc-700">{title}</p>
      {hint && <p className="mt-1 text-xs text-zinc-500">{hint}</p>}
    </div>
  );
}

export function PageHeader({
  title,
  subtitle,
  actions,
}: {
  title: string;
  subtitle?: string;
  actions?: React.ReactNode;
}) {
  return (
    <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div className="min-w-0">
        <h1 className="text-xl font-bold text-zinc-900">{title}</h1>
        {subtitle && <p className="mt-1 text-sm text-zinc-500">{subtitle}</p>}
      </div>
      {actions && <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div>}
    </div>
  );
}

export function Button({
  children,
  onClick,
  variant = "primary",
  type = "button",
  disabled,
  className = "",
}: {
  children: React.ReactNode;
  onClick?: () => void;
  variant?: "primary" | "secondary" | "danger" | "ghost";
  type?: "button" | "submit";
  disabled?: boolean;
  className?: string;
}) {
  const base =
    "inline-flex items-center justify-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50";
  const variants: Record<string, string> = {
    primary: "bg-indigo-600 text-white hover:bg-indigo-700",
    secondary: "border border-zinc-300 bg-white text-zinc-700 hover:bg-zinc-50",
    danger: "bg-red-600 text-white hover:bg-red-700",
    ghost: "text-zinc-600 hover:bg-zinc-100",
  };
  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled}
      className={`${base} ${variants[variant]} ${className}`}
    >
      {children}
    </button>
  );
}

export function Field({
  label,
  children,
  hint,
}: {
  label: string;
  children: React.ReactNode;
  hint?: string;
}) {
  return (
    <label className="block">
      <span className="mb-1 block text-sm font-medium text-zinc-700">{label}</span>
      {children}
      {hint && <span className="mt-1 block text-xs text-zinc-500">{hint}</span>}
    </label>
  );
}

export const inputClass =
  "w-full rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm text-zinc-900 placeholder:text-zinc-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500";

// Clipboard API needs a secure context and permission; when either is absent the
// caller's textarea is selected so the operator can copy by hand.
export function CopyButton({
  text,
  selectRef,
}: {
  text: string;
  selectRef?: React.RefObject<HTMLTextAreaElement | null>;
}) {
  const { t } = useI18n();
  const [state, setState] = useState<"idle" | "copied" | "manual">("idle");

  async function copy() {
    try {
      await navigator.clipboard.writeText(text);
      setState("copied");
    } catch {
      selectRef?.current?.focus();
      selectRef?.current?.select();
      setState("manual");
    }
  }

  return (
    <div>
      <Button variant="secondary" onClick={copy}>
        {state === "copied" ? t("copied") : t("copy")}
      </Button>
      {state === "manual" && <p className="mt-1 text-xs text-amber-700">{t("copyManual")}</p>}
    </div>
  );
}

export function Dialog({
  title,
  onClose,
  children,
}: {
  title: string;
  onClose: () => void;
  children: React.ReactNode;
}) {
  const { t } = useI18n();
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape") onClose();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-4 sm:items-center">
      <div
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className="max-h-[85vh] w-full max-w-2xl overflow-y-auto rounded-xl bg-white p-6 shadow-xl"
      >
        <div className="mb-4 flex items-start justify-between gap-4">
          <h2 className="text-lg font-bold text-zinc-900">{title}</h2>
          <Button variant="ghost" onClick={onClose}>
            {t("cancel")}
          </Button>
        </div>
        {children}
      </div>
    </div>
  );
}

export function Pager({
  page,
  pageSize,
  total,
  onChange,
}: {
  page: number;
  pageSize: number;
  total: number;
  onChange: (page: number) => void;
}) {
  const { t } = useI18n();
  const pages = Math.max(1, Math.ceil(total / pageSize));
  return (
    <div className="mt-4 flex flex-wrap items-center justify-between gap-3 text-sm text-zinc-600">
      <span>
        {t("page")} {page}/{pages} · {t("totalRows")} {total}
      </span>
      <div className="flex gap-2">
        <Button variant="secondary" disabled={page <= 1} onClick={() => onChange(page - 1)}>
          {t("previousPage")}
        </Button>
        <Button variant="secondary" disabled={page >= pages} onClick={() => onChange(page + 1)}>
          {t("nextPage")}
        </Button>
      </div>
    </div>
  );
}
