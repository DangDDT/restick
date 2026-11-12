"use client";

// Proof Table primitives. Every page composes these; no page-local colours.
// Colour comes only from semantic tokens (paper, panel, sunk, line, ink, cut,
// ok, warn, danger) defined in app/globals.css.

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useId, useRef, useState } from "react";
import { useI18n, type MessageKey } from "@/lib/i18n";
import { Icon, IconTile, Mascot, STATIONS, TONE, type IconName } from "@/components/icons";

export function cx(...parts: Array<string | false | null | undefined>): string {
  return parts.filter(Boolean).join(" ");
}

/* ---------- Status stamp ---------- */

type Tone = "ok" | "warn" | "danger" | "neutral" | "active";

const TONE_CLASS: Record<Tone, string> = {
  ok: "text-ok bg-ok-soft border-ok/40",
  warn: "text-warn bg-warn-soft border-warn/40",
  danger: "text-danger bg-danger-soft border-danger/40",
  neutral: "text-ink-2 bg-sunk border-line-strong",
  active: "text-cut-ink bg-cut-soft border-cut/50",
};

const STATUS: Record<string, { tone: Tone; label: MessageKey }> = {
  draft: { tone: "warn", label: "statusDraft" },
  ready: { tone: "ok", label: "statusReady" },
  revoked: { tone: "neutral", label: "statusRevoked" },
  queued: { tone: "active", label: "statusQueued" },
  running: { tone: "active", label: "statusRunning" },
  succeeded: { tone: "ok", label: "statusSucceeded" },
  failed: { tone: "danger", label: "statusFailed" },
  interrupted: { tone: "danger", label: "statusInterrupted" },
  active: { tone: "ok", label: "statusActive" },
  inactive: { tone: "neutral", label: "statusInactive" },
};

/** Inked rectangular stamp. Use for any state label; never a pastel pill. */
export function Stamp({ tone = "neutral", children, pulse }: { tone?: Tone; children: React.ReactNode; pulse?: boolean }) {
  return (
    <span
      className={cx(
        "inline-flex shrink-0 items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px] font-semibold leading-4",
        TONE_CLASS[tone],
      )}
    >
      {pulse && <span className="size-1.5 animate-pulse rounded-full bg-current" aria-hidden="true" />}
      {children}
    </span>
  );
}

export function StatusPill({ status }: { status: string }) {
  const { t } = useI18n();
  const known = STATUS[status];
  return (
    <Stamp tone={known?.tone ?? "neutral"} pulse={status === "running"}>
      {known ? t(known.label) : status}
    </Stamp>
  );
}

/** Short mono job number, e.g. № a1b2c3. */
export function JobTag({ id, className }: { id: string; className?: string }) {
  return (
    <span className={cx("mono text-[11px] text-ink-3", className)} title={id}>
      № {id.replace(/-/g, "").slice(0, 6)}
    </span>
  );
}

/* ---------- Feedback ---------- */

export function Spinner({ inline, className }: { inline?: boolean; className?: string }) {
  const { t } = useI18n();
  const ring = (
    <span
      className={cx("inline-block size-4 animate-spin rounded-full border-2 border-line-strong border-t-cut", className)}
      aria-hidden="true"
    />
  );
  if (inline) return ring;
  return (
    <div className="flex items-center justify-center gap-3 py-16 text-ink-2" role="status">
      <span className="inline-block size-5 animate-spin rounded-full border-2 border-line-strong border-t-cut" aria-hidden="true" />
      <span>{t("loading")}</span>
    </div>
  );
}

export function Skeleton({ className }: { className?: string }) {
  return <div className={cx("skeleton", className)} aria-hidden="true" />;
}

export function SkeletonRows({ rows = 5, className }: { rows?: number; className?: string }) {
  const { t } = useI18n();
  return (
    <div className={cx("space-y-3", className)} role="status" aria-label={t("loading")}>
      {Array.from({ length: rows }, (_, i) => (
        <Skeleton key={i} className="h-11 w-full" />
      ))}
    </div>
  );
}

function Banner({ tone, icon, message, role, onDismiss }: { tone: "ok" | "danger" | "warn"; icon: IconName; message: string; role: "alert" | "status"; onDismiss?: () => void }) {
  const { t } = useI18n();
  return (
    <div
      role={role}
      className={cx(
        "flex items-start gap-2.5 rounded-panel border px-3.5 py-2.5 text-sm",
        tone === "ok" && "border-ok/40 bg-ok-soft text-ok",
        tone === "danger" && "border-danger/40 bg-danger-soft text-danger",
        tone === "warn" && "border-warn/40 bg-warn-soft text-warn",
      )}
    >
      <Icon name={icon} size={16} className="mt-0.5 shrink-0" />
      <span className="min-w-0 flex-1 break-words">{message}</span>
      {onDismiss && (
        <button type="button" onClick={onDismiss} aria-label={t("close")} className="-m-1 shrink-0 rounded p-1 opacity-70 hover:opacity-100">
          <Icon name="close" size={14} />
        </button>
      )}
    </div>
  );
}

export function ErrorBanner({ message, onDismiss }: { message: string; onDismiss?: () => void }) {
  return <Banner tone="danger" icon="alert" message={message} role="alert" onDismiss={onDismiss} />;
}

export function SuccessBanner({ message, onDismiss }: { message: string; onDismiss?: () => void }) {
  return <Banner tone="ok" icon="check" message={message} role="status" onDismiss={onDismiss} />;
}

export function WarnBanner({ message }: { message: string }) {
  return <Banner tone="warn" icon="info" message={message} role="status" />;
}

export function EmptyState({
  title,
  hint,
  icon = "image",
  children,
}: {
  title: string;
  hint?: string;
  icon?: IconName;
  children?: React.ReactNode;
}) {
  return (
    <div className="crop flex flex-col items-center gap-3 rounded-panel px-6 py-14 text-center">
      <span className="relative">
        <Mascot size={64} className="bob" />
        <span className="absolute -bottom-1 -right-2 flex size-7 items-center justify-center rounded-full border-2 border-panel bg-butter-soft text-butter-ink">
          <Icon name={icon} size={14} />
        </span>
      </span>
      <div>
        <p className="font-display text-base font-semibold text-ink">{title}</p>
        {hint && <p className="mx-auto mt-1 max-w-[44ch] text-sm text-ink-2">{hint}</p>}
      </div>
      {children}
    </div>
  );
}

/* ---------- Structure ---------- */

export function PageHeader({
  title,
  subtitle,
  actions,
  back,
}: {
  title: string;
  subtitle?: string;
  actions?: React.ReactNode;
  back?: { href: string; label: string };
}) {
  const pathname = usePathname();
  const station = STATIONS.find((item) => pathname === item.href || pathname.startsWith(`${item.href}/`));
  return (
    <header className="mb-6 flex flex-col gap-3 border-b border-line pb-5 md:flex-row md:items-end md:justify-between">
      <div className="flex min-w-0 items-start gap-3.5">
        {station && <IconTile name={station.icon} tone={station.tone} size={46} className="mt-0.5 rounded-[14px] max-md:hidden" />}
        <div className="min-w-0">
        {back && (
          <Link href={back.href} className="mb-2 inline-flex items-center gap-1 text-sm text-ink-2 hover:text-ink">
            <Icon name="chevronRight" size={14} className="rotate-180" />
            {back.label}
          </Link>
        )}
        <h1 className="font-display text-[1.625rem] font-semibold leading-8 text-ink">{title}</h1>
        {subtitle && <p className="mt-1 max-w-[68ch] text-sm text-ink-2">{subtitle}</p>}
        </div>
      </div>
      {actions && <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div>}
    </header>
  );
}

/** A bounded work surface: hairline border, optional heading row. Never nest Panels. */
export function Panel({
  title,
  actions,
  children,
  className,
  padded = true,
}: {
  title?: React.ReactNode;
  actions?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
  padded?: boolean;
}) {
  return (
    <section className={cx("soft-shadow rounded-panel border border-line bg-panel", className)}>
      {(title || actions) && (
        <div className="flex min-h-12 items-center justify-between gap-3 border-b border-line px-4 py-2.5">
          {title && <h2 className="font-display text-[0.9375rem] font-semibold text-ink">{title}</h2>}
          {actions && <div className="flex items-center gap-2">{actions}</div>}
        </div>
      )}
      <div className={padded ? "p-4" : undefined}>{children}</div>
    </section>
  );
}

/** Unboxed ruled section: heading above a hairline, for page-level grouping. */
const SECTION_TONES = ["pink", "lilac", "mint", "sky", "butter", "peach"] as const;
const SECTION_ICONS = ["star", "heart", "sparkle"] as const;

export function Section({ title, hint, actions, children, className }: { title: string; hint?: string; actions?: React.ReactNode; children: React.ReactNode; className?: string }) {
  return (
    <section className={cx("py-5", className)}>
      <div className="mb-3 flex items-end justify-between gap-3 border-b border-line pb-2">
        <div className="min-w-0">
          <h2 className="flex items-center gap-2 font-display text-base font-semibold text-ink">
            <span aria-hidden="true" className={cx("inline-flex size-6 items-center justify-center rounded-full", TONE[SECTION_TONES[title.length % SECTION_TONES.length]])}>
              <Icon name={SECTION_ICONS[title.length % SECTION_ICONS.length]} size={13} />
            </span>
            {title}
            {hint && <Hint text={hint} />}
          </h2>
        </div>
        {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
      </div>
      {children}
    </section>
  );
}

/** Artwork frame: crop marks on a proof ground. `checker` shows transparency. */
export function ArtFrame({
  children,
  selected,
  checker = true,
  className,
}: {
  children: React.ReactNode;
  selected?: boolean;
  checker?: boolean;
  className?: string;
}) {
  return (
    <div
      className={cx(
        "relative flex items-center justify-center overflow-hidden rounded-[6px]",
        checker ? "proof-ground" : "crop",
        selected && "cutline",
        className,
      )}
    >
      {children}
    </div>
  );
}

export function Kbd({ children }: { children: React.ReactNode }) {
  return <kbd className="mono rounded-[3px] border border-line-strong bg-sunk px-1.5 py-px text-[11px] text-ink-2">{children}</kbd>;
}

/* ---------- Controls ---------- */

type ButtonVariant = "primary" | "secondary" | "danger" | "ghost";

export const BUTTON_VARIANT: Record<ButtonVariant, string> = {
  primary: "bg-cut text-on-cut shadow-[0_3px_0_var(--cut-hover)] hover:-translate-y-px hover:bg-cut-hover active:translate-y-[2px] active:shadow-[0_1px_0_var(--cut-hover)]",
  secondary: "border border-line-strong bg-panel text-ink hover:border-ink-3 hover:bg-sunk active:translate-y-px",
  danger: "border border-danger/50 bg-panel text-danger hover:bg-danger-soft active:translate-y-px",
  ghost: "text-ink-2 hover:bg-sunk hover:text-ink",
};

export interface ButtonProps {
  children?: React.ReactNode;
  onClick?: () => void;
  variant?: ButtonVariant;
  type?: "button" | "submit";
  disabled?: boolean;
  className?: string;
  size?: "sm" | "md";
  icon?: IconName;
  loading?: boolean;
  href?: string;
  title?: string;
  "aria-label"?: string;
}

export function Button({
  children,
  onClick,
  variant = "primary",
  type = "button",
  disabled,
  className,
  size = "md",
  icon,
  loading,
  href,
  title,
  "aria-label": ariaLabel,
}: ButtonProps) {
  const cls = cx(
    "inline-flex select-none items-center justify-center gap-2 rounded-ctl font-medium whitespace-nowrap transition-[background-color,border-color,color,transform] duration-150 ease-out",
    "disabled:cursor-not-allowed disabled:opacity-45 aria-disabled:cursor-not-allowed aria-disabled:opacity-45",
    size === "sm" ? "min-h-8 px-2.5 text-[13px] max-md:min-h-10" : "min-h-9 px-3.5 text-sm max-md:min-h-11",
    BUTTON_VARIANT[variant],
    className,
  );
  const inner = (
    <>
      {loading ? <Spinner inline /> : icon ? <Icon name={icon} size={size === "sm" ? 15 : 16} /> : null}
      {children}
    </>
  );
  if (href && !disabled) {
    return (
      <Link href={href} className={cls} title={title} aria-label={ariaLabel}>
        {inner}
      </Link>
    );
  }
  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={cls}
      title={title}
      aria-label={ariaLabel}
    >
      {inner}
    </button>
  );
}

export function IconButton({
  icon,
  label,
  onClick,
  variant = "ghost",
  disabled,
  className,
}: {
  icon: IconName;
  label: string;
  onClick?: () => void;
  variant?: ButtonVariant;
  disabled?: boolean;
  className?: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={label}
      title={label}
      className={cx(
        "inline-flex size-9 shrink-0 items-center justify-center rounded-ctl transition-colors duration-150 disabled:cursor-not-allowed disabled:opacity-45 max-md:size-11",
        BUTTON_VARIANT[variant],
        className,
      )}
    >
      <Icon name={icon} size={18} />
    </button>
  );
}

export function Field({
  label,
  children,
  hint,
  error,
}: {
  label: string;
  children: React.ReactNode;
  hint?: string;
  error?: string;
}) {
  return (
    <label className="block">
      <span className="mb-1.5 flex items-center gap-1.5 text-[13px] font-medium text-ink">
        {label}
        {hint && <Hint text={hint} />}
      </span>
      {children}
      {error && (
        <span role="alert" className="mt-1.5 block text-xs text-danger">
          {error}
        </span>
      )}
    </label>
  );
}

/** Round "!" next to a label; the hint text lives in a hover/focus tooltip so
 * forms keep one label baseline whether or not they have a hint. */
export function Hint({ text }: { text: string }) {
  return (
    <span
      className="group relative inline-flex shrink-0 items-center align-middle"
      tabIndex={0}
      role="note"
      aria-label={text}
      data-qa="hint"
    >
      <span
        aria-hidden="true"
        className="flex size-4 items-center justify-center rounded-full border border-line-strong text-[10px] font-semibold leading-none text-ink-2"
      >
        !
      </span>
      <span
        role="tooltip"
        className="pointer-events-none invisible absolute left-1/2 top-full z-30 mt-1.5 w-max max-w-[16rem] -translate-x-1/2 rounded-ctl border border-line bg-panel px-2.5 py-1.5 text-xs font-normal leading-5 text-ink-2 opacity-0 shadow-[var(--shadow-pop)] transition-opacity duration-150 group-focus-visible:visible group-focus-visible:opacity-100 group-hover:visible group-hover:opacity-100"
      >
        {text}
      </span>
    </span>
  );
}

export const inputClass =
  "block w-full min-h-9 rounded-ctl border border-line-strong bg-panel px-3 py-1.5 text-sm text-ink placeholder:text-ink-3 transition-colors hover:border-ink-3 focus:border-cut focus:outline-none focus:ring-2 focus:ring-cut/25 disabled:cursor-not-allowed disabled:bg-sunk disabled:text-ink-3 aria-[invalid=true]:border-danger max-md:min-h-11";

/** Native select with a drawn chevron; pair with <select className={selectClass}>. */
export const selectClass = `${inputClass} ctl-select appearance-none pr-8`;

export const checkClass = "size-4 shrink-0 cursor-pointer rounded-[3px] accent-[var(--cut)]";

export function Switch({
  checked,
  onChange,
  label,
  disabled,
}: {
  checked: boolean;
  onChange: (next: boolean) => void;
  label: string;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cx(
        "relative inline-flex h-5 w-9 shrink-0 items-center rounded-full border transition-colors duration-150 disabled:opacity-45 max-md:h-6 max-md:w-11",
        checked ? "border-cut bg-cut" : "border-line-strong bg-sunk",
      )}
    >
      <span
        className={cx(
          "absolute left-0.5 size-3.5 rounded-full bg-panel shadow-sm transition-transform duration-150 ease-out max-md:size-4.5",
          checked ? "translate-x-4 max-md:translate-x-5" : "translate-x-0",
        )}
      />
    </button>
  );
}

export interface TabItem<T extends string> {
  id: T;
  label: string;
  count?: number;
}

export function Tabs<T extends string>({
  items,
  value,
  onChange,
  label,
  variant = "underline",
}: {
  items: TabItem<T>[];
  value: T;
  onChange: (id: T) => void;
  label: string;
  variant?: "underline" | "segmented";
}) {
  function onKey(event: React.KeyboardEvent, index: number) {
    const step = event.key === "ArrowRight" ? 1 : event.key === "ArrowLeft" ? -1 : 0;
    if (!step) return;
    event.preventDefault();
    const next = items[(index + step + items.length) % items.length];
    onChange(next.id);
    (event.currentTarget.parentElement?.children[(index + step + items.length) % items.length] as HTMLElement | undefined)?.focus();
  }
  return (
    <div
      role="tablist"
      aria-label={label}
      className={cx(
        "flex max-w-full overflow-x-auto",
        variant === "underline" ? "gap-1 border-b border-line" : "w-fit gap-0.5 rounded-ctl border border-line bg-sunk p-0.5",
      )}
    >
      {items.map((item, index) => {
        const active = item.id === value;
        return (
          <button
            key={item.id}
            role="tab"
            type="button"
            aria-selected={active}
            tabIndex={active ? 0 : -1}
            onClick={() => onChange(item.id)}
            onKeyDown={(event) => onKey(event, index)}
            className={cx(
              "inline-flex min-h-9 items-center gap-1.5 whitespace-nowrap px-3 text-sm font-medium transition-colors max-md:min-h-11",
              variant === "underline"
                ? cx("-mb-px border-b-2", active ? "border-cut text-ink" : "border-transparent text-ink-2 hover:text-ink")
                : cx("rounded-[3px]", active ? "bg-panel text-ink shadow-sm" : "text-ink-2 hover:text-ink"),
            )}
          >
            {item.label}
            {item.count !== undefined && <span className="nums text-xs text-ink-3">{item.count}</span>}
          </button>
        );
      })}
    </div>
  );
}

/* ---------- Overlays ---------- */

export function Dialog({
  title,
  onClose,
  children,
  footer,
  variant = "center",
  size = "md",
}: {
  title: string;
  onClose: () => void;
  children: React.ReactNode;
  footer?: React.ReactNode;
  variant?: "center" | "drawer";
  size?: "sm" | "md" | "lg";
}) {
  const { t } = useI18n();
  const ref = useRef<HTMLDialogElement>(null);
  const titleId = useId();

  // Native <dialog>: modal focus trap, Esc handling and focus restore for free.
  useEffect(() => {
    const dialog = ref.current;
    if (dialog && !dialog.open) dialog.showModal();
    // React autoFocus fires before the dialog is open; honour data-autofocus here.
    dialog?.querySelector<HTMLElement>("[data-autofocus]")?.focus();
    return () => {
      if (dialog?.open) dialog.close();
    };
  }, []);

  const width = size === "sm" ? "max-w-md" : size === "lg" ? "max-w-4xl" : "max-w-2xl";
  return (
    <dialog
      ref={ref}
      aria-labelledby={titleId}
      className={cx(
        "sheet fixed inset-0 m-0 h-dvh w-dvw open:flex",
        variant === "drawer" ? "justify-end" : "items-end justify-center p-0 sm:items-center sm:p-4",
      )}
      onCancel={(event) => {
        event.preventDefault();
        onClose();
      }}
      onClick={(event) => {
        if (event.target === ref.current) onClose();
      }}
    >
      <div
        className={cx(
          "sheet-body flex max-h-[92dvh] w-full flex-col border border-line bg-panel text-ink shadow-[var(--shadow-pop)]",
          variant === "drawer" ? "h-dvh max-h-none max-w-md border-y-0 border-r-0" : cx(width, "rounded-t-xl sm:rounded-xl"),
        )}
      >
        <div className="flex items-start justify-between gap-4 border-b border-line px-5 py-3.5">
          <h2 id={titleId} className="font-display text-lg font-semibold">
            {title}
          </h2>
          <IconButton icon="close" label={t("close")} onClick={onClose} className="-mr-2 -mt-1" />
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4">{children}</div>
        {footer && <div className="flex flex-wrap justify-end gap-2 border-t border-line px-5 py-3">{footer}</div>}
      </div>
    </dialog>
  );
}

/** Destructive-action confirmation. The action button is the only red element. */
export function ConfirmDialog({
  title,
  body,
  confirmLabel,
  onConfirm,
  onClose,
  busy,
  error,
}: {
  title: string;
  body: React.ReactNode;
  confirmLabel: string;
  onConfirm: () => void;
  onClose: () => void;
  busy?: boolean;
  error?: string | null;
}) {
  const { t } = useI18n();
  return (
    <Dialog
      title={title}
      onClose={onClose}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={busy}>
            {t("cancel")}
          </Button>
          <Button variant="danger" onClick={onConfirm} loading={busy} icon="trash">
            {confirmLabel}
          </Button>
        </>
      }
    >
      <div className="space-y-3 text-sm text-ink-2">
        <div>{body}</div>
        {error && <ErrorBanner message={error} />}
      </div>
    </Dialog>
  );
}

// Clipboard API needs a secure context and permission; when either is absent the
// caller's textarea is selected so the operator can copy by hand.
export function CopyButton({
  text,
  selectRef,
  variant = "secondary",
  label,
}: {
  text: string;
  selectRef?: React.RefObject<HTMLTextAreaElement | null>;
  variant?: ButtonVariant;
  label?: string;
}) {
  const { t } = useI18n();
  const [state, setState] = useState<"idle" | "copied" | "manual">("idle");

  useEffect(() => {
    if (state !== "copied") return;
    const timer = window.setTimeout(() => setState("idle"), 2400);
    return () => window.clearTimeout(timer);
  }, [state]);

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
      <Button variant={variant} onClick={copy} icon={state === "copied" ? "check" : "copy"}>
        {state === "copied" ? t("copied") : (label ?? t("copy"))}
      </Button>
      <span role="status" className="sr-only">
        {state === "copied" ? t("copied") : ""}
      </span>
      {state === "manual" && <p className="mt-1.5 max-w-[40ch] text-xs text-warn">{t("copyManual")}</p>}
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
    <nav aria-label={t("page")} className="mt-4 flex flex-wrap items-center justify-between gap-3 text-sm text-ink-2">
      <span className="nums">
        {t("page")} {page}/{pages} · {t("totalRows")} {total}
      </span>
      <div className="flex gap-2">
        <Button variant="secondary" size="sm" disabled={page <= 1} onClick={() => onChange(page - 1)}>
          {t("previousPage")}
        </Button>
        <Button variant="secondary" size="sm" disabled={page >= pages} onClick={() => onChange(page + 1)}>
          {t("nextPage")}
        </Button>
      </div>
    </nav>
  );
}
