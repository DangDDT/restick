"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useAuth } from "@/lib/auth";
import { useI18n, type MessageKey } from "@/lib/i18n";
import { errorText } from "@/lib/api";
import { Dialog, ErrorBanner, IconButton, Kbd, cx } from "@/components/ui";
import { Icon, IconTile, Mascot, TONE, type IconName, type Tone } from "@/components/icons";
import { CommandPalette, type Command } from "@/components/command-palette";
import { useTheme } from "@/components/theme";
import { StepStrip } from "@/components/guide";

interface NavItem {
  href: string;
  label: MessageKey;
  icon: IconName;
  tone: Tone;
}

// Order follows the production line: compose → release.
const START: NavItem = { href: "/start", tone: "butter", label: "navStart", icon: "sparkle" };
const PRODUCE: NavItem[] = [
  { href: "/factory", tone: "pink", label: "navFactory", icon: "factory" },
  { href: "/library", tone: "lilac", label: "navLibrary", icon: "library" },
  { href: "/characters", tone: "peach", label: "navCharacters", icon: "character" },
];
const RELEASE: NavItem[] = [
  { href: "/packs", tone: "mint", label: "navPacks", icon: "pack" },
  { href: "/keys", tone: "butter", label: "navKeys", icon: "key" },
  { href: "/api-docs", tone: "sky", label: "navDocs", icon: "docs" },
];
const ALL = [START, ...PRODUCE, ...RELEASE];
const TABS = [...PRODUCE, RELEASE[0]];

export function Wordmark() {
  const { t } = useI18n();
  return (
    <Link href="/start" className="flex items-center gap-2.5">
      <Mascot size={34} className="bob" />
      <span className="font-display text-[0.9375rem] font-semibold tracking-tight text-ink">{t("appName")}</span>
    </Link>
  );
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const { t } = useI18n();
  const pathname = usePathname();
  const router = useRouter();
  const { username, logout } = useAuth();
  const { theme, toggle } = useTheme();
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [moreOpen, setMoreOpen] = useState(false);
  const [logoutError, setLogoutError] = useState<string | null>(null);

  const isActive = (href: string) => pathname === href || pathname.startsWith(`${href}/`);

  const signOut = useCallback(async () => {
    setLogoutError(null);
    try {
      await logout();
      router.replace("/login");
    } catch (err) {
      setLogoutError(errorText(err));
    }
  }, [logout, router]);

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setPaletteOpen((open) => !open);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const commands = useMemo<Command[]>(
    () => [
      ...ALL.map((item) => ({ id: item.href, label: item.label, group: "go" as const, icon: item.icon, run: () => router.push(item.href) })),
      { id: "new-pack", label: "cmdNewPack", group: "cmd", icon: "plus", run: () => router.push("/packs/new") },
      {
        id: "theme",
        label: theme === "dark" ? "themeSwitchLight" : "themeSwitchDark",
        group: "cmd",
        icon: theme === "dark" ? "sun" : "moon",
        run: toggle,
      },
      { id: "signout", label: "signOut", group: "cmd", icon: "logout", run: () => void signOut() },
    ],
    [router, theme, toggle, signOut],
  );

  const themeButton = (
    <IconButton
      icon={theme === "dark" ? "sun" : "moon"}
      label={theme === "dark" ? t("themeSwitchLight") : t("themeSwitchDark")}
      onClick={toggle}
    />
  );

  return (
    <div className="min-h-dvh md:flex">
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:fixed focus:left-3 focus:top-3 focus:z-[60] focus:rounded-ctl focus:bg-ink focus:px-3 focus:py-2 focus:text-paper"
      >
        {t("skipToContent")}
      </a>

      {/* Desktop station rail */}
      <aside className="sticky top-0 hidden h-dvh w-60 shrink-0 flex-col border-r border-line bg-panel md:flex">
        <div className="px-5 py-4">
          <Wordmark />
        </div>
        <nav aria-label={t("navStudio")} className="flex-1 overflow-y-auto px-3 pb-4">
          <Link
            href="/start"
            aria-current={isActive("/start") ? "page" : undefined}
            className={cx(
              "hover-wiggle mb-4 flex min-h-11 items-center gap-2.5 rounded-ctl px-2.5 text-sm font-semibold transition-colors",
              isActive("/start") ? TONE.butter : "bg-butter-soft/60 text-butter-ink hover:bg-butter-soft",
            )}
          >
            <IconTile name="sparkle" tone="butter" size={28} className="bg-panel/70" />
            {t("navStart")}
          </Link>
          {[
            { title: t("navProduce"), items: PRODUCE },
            { title: t("navRelease"), items: RELEASE },
          ].map((group) => (
            <div key={group.title} className="mb-5">
              <p className="px-3 pb-1.5 text-xs font-medium text-ink-3">{group.title}</p>
              <ul className="ml-3 border-l border-line">
                {group.items.map((item) => {
                  const active = isActive(item.href);
                  return (
                    <li key={item.href} className="relative">
                      <span
                        aria-hidden="true"
                        className={cx(
                          "absolute -left-[4.5px] top-1/2 size-2 -translate-y-1/2 rounded-full border transition-colors",
                          active ? "border-cut bg-cut" : "border-line-strong bg-panel",
                        )}
                      />
                      <Link
                        href={item.href}
                        aria-current={active ? "page" : undefined}
                        className={cx(
                          "ml-3 flex min-h-10 items-center gap-2.5 rounded-ctl px-2 text-sm font-medium transition-colors",
                          "hover-wiggle",
                          active ? `${TONE[item.tone]} font-semibold` : "text-ink-2 hover:bg-sunk hover:text-ink",
                        )}
                      >
                        <IconTile name={item.icon} tone={item.tone} size={28} className={active ? "bg-panel/70" : ""} />
                        {t(item.label)}
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </div>
          ))}
        </nav>
        <div className="space-y-2 border-t border-line p-3">
          <button
            type="button"
            onClick={() => setPaletteOpen(true)}
            className="flex min-h-9 w-full items-center gap-2 rounded-ctl border border-line-strong bg-panel px-2.5 text-sm text-ink-2 hover:border-ink-3 hover:text-ink"
          >
            <Icon name="search" size={15} />
            <span className="flex-1 text-left">{t("cmdOpen")}</span>
            <Kbd>⌘K</Kbd>
          </button>
          {logoutError && <ErrorBanner message={logoutError} onDismiss={() => setLogoutError(null)} />}
          <div className="flex items-center justify-between gap-2 pl-1">
            <span className="min-w-0 truncate text-xs text-ink-2" title={t("ownerLabel")}>
              {username}
            </span>
            <div className="flex shrink-0">
              {themeButton}
              <IconButton icon="logout" label={t("signOut")} onClick={() => void signOut()} />
            </div>
          </div>
        </div>
      </aside>

      {/* Mobile top bar */}
      <header className="sticky top-0 z-30 flex items-center justify-between border-b border-line bg-panel/90 px-4 py-2 backdrop-blur md:hidden">
        <Wordmark />
        <div className="flex">
          <IconButton icon="search" label={t("cmdOpen")} onClick={() => setPaletteOpen(true)} />
          {themeButton}
        </div>
      </header>

      <main id="main" className="min-w-0 flex-1 px-4 pb-28 pt-5 md:px-8 md:pb-10 md:pt-8 xl:px-10">
        <div className="mx-auto w-full max-w-[1400px]">
          <StepStrip />
          {children}
        </div>
      </main>

      {/* Mobile bottom tabs */}
      <nav aria-label={t("navStudio")} className="fixed inset-x-0 bottom-0 z-30 border-t border-line bg-panel pb-[env(safe-area-inset-bottom)] md:hidden">
        <ul className="grid grid-cols-5">
          {TABS.map((item) => {
            const active = isActive(item.href);
            return (
              <li key={item.href}>
                <Link
                  href={item.href}
                  aria-current={active ? "page" : undefined}
                  className={cx("flex min-h-14 flex-col items-center justify-center gap-0.5 text-[11px] font-medium", active ? "text-ink" : "text-ink-2")}
                >
                  <span className={cx("flex h-7 w-11 items-center justify-center rounded-full", active ? TONE[item.tone] : "text-ink-3")}>
                    <Icon name={item.icon} size={18} />
                  </span>
                  {t(item.label)}
                </Link>
              </li>
            );
          })}
          <li>
            <button
              type="button"
              onClick={() => setMoreOpen(true)}
              className={cx(
                "flex min-h-14 w-full flex-col items-center justify-center gap-0.5 text-[11px] font-medium",
                isActive("/keys") || isActive("/api-docs") ? "text-cut-ink" : "text-ink-2",
              )}
            >
              <span className="flex h-6 w-10 items-center justify-center">
                <Icon name="more" size={18} />
              </span>
              {t("navMore")}
            </button>
          </li>
        </ul>
      </nav>

      {moreOpen && (
        <Dialog title={t("navMore")} onClose={() => setMoreOpen(false)} size="sm">
          <ul className="-my-1">
            {[START, ...RELEASE.slice(1)].map((item) => (
              <li key={item.href}>
                <Link
                  href={item.href}
                  onClick={() => setMoreOpen(false)}
                  className="flex min-h-12 items-center gap-3 border-b border-line text-sm font-medium text-ink"
                >
                  <IconTile name={item.icon} tone={item.tone} size={30} />
                  {t(item.label)}
                </Link>
              </li>
            ))}
            <li>
              <button type="button" onClick={() => void signOut()} className="flex min-h-12 w-full items-center gap-3 text-sm font-medium text-ink">
                <Icon name="logout" size={18} />
                {t("signOut")}
              </button>
            </li>
          </ul>
          {logoutError && <ErrorBanner message={logoutError} />}
        </Dialog>
      )}

      {paletteOpen && <CommandPalette commands={commands} onClose={() => setPaletteOpen(false)} />}
    </div>
  );
}
