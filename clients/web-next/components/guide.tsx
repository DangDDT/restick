"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Icon, IconTile, TONE, type IconName, type Tone } from "@/components/icons";
import { cx } from "@/components/ui";
import { api } from "@/lib/api";
import { useI18n, type MessageKey } from "@/lib/i18n";
import type { AdminPackPage, CharacterList, DesignPage, KeyList } from "@/lib/types";

export interface Step {
  n: number;
  href: string;
  icon: IconName;
  tone: Tone;
  title: MessageKey;
  what: MessageKey;
  how: MessageKey;
  cta: MessageKey;
  optional?: boolean;
  outside?: boolean;
  /** Routes that belong to this step, used by the progress strip. */
  routes: string[];
}

export const STEPS: Step[] = [
  { n: 1, href: "/characters", icon: "character", tone: "peach", title: "s1Title", what: "s1What", how: "s1How", cta: "s1Cta", optional: true, routes: ["/characters"] },
  { n: 2, href: "/factory", icon: "factory", tone: "pink", title: "s2Title", what: "s2What", how: "s2How", cta: "s2Cta", routes: ["/factory", "/library"] },
  { n: 3, href: "/factory", icon: "sparkle", tone: "lilac", title: "s3Title", what: "s3What", how: "s3How", cta: "s3Cta", outside: true, routes: [] },
  { n: 4, href: "/packs/new", icon: "upload", tone: "sky", title: "s4Title", what: "s4What", how: "s4How", cta: "s4Cta", routes: ["/packs/new", "/packs/detail"] },
  { n: 5, href: "/packs", icon: "pack", tone: "mint", title: "s5Title", what: "s5What", how: "s5How", cta: "s5Cta", routes: ["/packs"] },
  { n: 6, href: "/keys", icon: "key", tone: "butter", title: "s6Title", what: "s6What", how: "s6How", cta: "s6Cta", routes: ["/keys", "/api-docs"] },
];

/** Progress is read from real data, so the guide can never disagree with the studio. */
export function useProgress() {
  const characters = useQuery({ queryKey: ["guide-characters"], queryFn: () => api<CharacterList>("/api/v1/admin/characters") });
  const designs = useQuery({ queryKey: ["guide-designs"], queryFn: () => api<DesignPage>("/api/v1/admin/designs?page=1&page_size=1") });
  const packs = useQuery({ queryKey: ["guide-packs"], queryFn: () => api<AdminPackPage>("/api/v1/admin/packs?page=1&page_size=1") });
  const ready = useQuery({ queryKey: ["guide-ready"], queryFn: () => api<AdminPackPage>("/api/v1/admin/packs?status=ready&page=1&page_size=1") });
  const keys = useQuery({ queryKey: ["guide-keys"], queryFn: () => api<KeyList>("/api/v1/admin/api-keys") });

  const loaded = [characters, designs, packs, ready, keys].every((q) => q.isSuccess);
  const hasPack = (packs.data?.total ?? 0) > 0;
  const done: Record<number, boolean> = {
    1: (characters.data?.items.length ?? 0) > 0,
    2: (designs.data?.total ?? 0) > 0,
    3: hasPack, // the outside-AI step is implied once a sheet has been uploaded into a pack
    4: hasPack,
    5: (ready.data?.total ?? 0) > 0,
    6: (keys.data?.items.some((k) => k.active) ?? false),
  };
  const current = loaded ? STEPS.find((s) => !done[s.n] && !s.optional) ?? null : null;
  return { loaded, done, current };
}

/** Full guide for /start: numbered steps, the current one expanded. */
export function StepGuide() {
  const { t } = useI18n();
  const { loaded, done, current } = useProgress();
  const finished = loaded && !current;

  return (
    <div>
      {finished && (
        <p className="mb-5 flex items-center gap-2 rounded-panel bg-mint-soft px-4 py-3 text-sm font-medium text-mint-ink">
          <Icon name="heart" size={16} />
          {t("startAllDone")}
        </p>
      )}
      <ol className="space-y-3">
        {STEPS.map((step) => {
          const isDone = done[step.n];
          const isCurrent = current?.n === step.n;
          return (
            <li
              key={step.n}
              aria-current={isCurrent ? "step" : undefined}
              className={cx(
                "soft-shadow rounded-panel border bg-panel p-4 md:p-5",
                isCurrent ? "border-cut" : "border-line",
              )}
            >
              <div className="flex items-start gap-3.5">
                <span
                  className={cx(
                    "flex size-9 shrink-0 items-center justify-center rounded-full text-sm font-bold",
                    isDone ? "bg-mint-soft text-mint-ink" : isCurrent ? "bg-cut text-on-cut" : "bg-sunk text-ink-2",
                  )}
                >
                  {isDone ? <Icon name="check" size={16} strokeWidth={2.2} /> : step.n}
                </span>
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <h2 className="font-display text-base font-semibold text-ink">{t(step.title)}</h2>
                    {isDone && <span className="rounded-full bg-mint-soft px-2 py-0.5 text-[11px] font-semibold text-mint-ink">{t("stepDone")}</span>}
                    {step.optional && !isDone && <span className="rounded-full bg-sunk px-2 py-0.5 text-[11px] font-medium text-ink-2">{t("stepOptional")}</span>}
                    {step.outside && <span className="rounded-full bg-lilac-soft px-2 py-0.5 text-[11px] font-medium text-lilac-ink">{t("stepOutside")}</span>}
                  </div>
                  <p className="mt-1 text-sm text-ink-2">{t(step.what)}</p>
                  {(isCurrent || !isDone) && (
                    <p className="mt-2 rounded-ctl bg-sunk px-3 py-2 text-sm text-ink">
                      <Icon name="info" size={14} className="mr-1.5 inline -translate-y-px text-ink-3" />
                      {t(step.how)}
                    </p>
                  )}
                  <Link
                    href={step.href}
                    className={cx(
                      "mt-3 inline-flex min-h-9 items-center gap-2 rounded-ctl px-3.5 text-sm font-medium max-md:min-h-11",
                      isCurrent
                        ? "bg-cut text-on-cut shadow-[0_3px_0_var(--cut-hover)] hover:bg-cut-hover"
                        : "border border-line-strong bg-panel text-ink hover:bg-sunk",
                    )}
                  >
                    <IconTile name={step.icon} tone={step.tone} size={22} className="!rounded-full" />
                    {isCurrent ? t("stepNow") : t(step.cta)}
                    <Icon name="chevronRight" size={14} />
                  </Link>
                </div>
              </div>
            </li>
          );
        })}
      </ol>
    </div>
  );
}

/** Slim "you are here" bar shown above every page except /start. */
export function StepStrip() {
  const { t } = useI18n();
  const pathname = usePathname();
  const { loaded, done, current } = useProgress();
  if (pathname === "/start" || !loaded) return null;

  const here = STEPS.find((s) => s.routes.some((r) => pathname === r || pathname.startsWith(`${r}/`)));
  const focus = here ?? current;
  if (!focus) return null;

  return (
    <div className="mb-5 flex flex-wrap items-center gap-x-4 gap-y-2 rounded-panel border border-line bg-panel px-3.5 py-2.5 text-sm soft-shadow">
      <ol className="flex items-center gap-1" aria-label={t("startTitle")}>
        {STEPS.map((s) => (
          <li key={s.n}>
            <Link
              href={s.href}
              aria-label={`${t("stepWord")} ${s.n}: ${t(s.title)}`}
              aria-current={s.n === focus.n ? "step" : undefined}
              className={cx(
                "flex size-6 items-center justify-center rounded-full text-[11px] font-bold max-md:size-7",
                done[s.n] ? "bg-mint-soft text-mint-ink" : s.n === focus.n ? "bg-cut text-on-cut" : "bg-sunk text-ink-2",
              )}
            >
              {done[s.n] ? <Icon name="check" size={12} strokeWidth={2.4} /> : s.n}
            </Link>
          </li>
        ))}
      </ol>
      <p className="min-w-0 flex-1 text-ink-2">
        <span className={cx("mr-1.5 rounded-full px-2 py-0.5 text-xs font-semibold", TONE[focus.tone])}>
          {t("stepWord")} {focus.n}
        </span>
        <span className="font-medium text-ink">{t(focus.title)}</span>
        <span className="hidden lg:inline"> · {t(focus.what)}</span>
      </p>
      <Link href="/start" className="text-sm font-medium text-cut-ink underline-offset-2 hover:underline">
        {t("guideShow")}
      </Link>
    </div>
  );
}
