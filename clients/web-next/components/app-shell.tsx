"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth";
import { t } from "@/lib/i18n";

const NAV = [
  { href: "/factory", label: "navFactory" },
  { href: "/library", label: "navLibrary" },
  { href: "/characters", label: "navCharacters" },
  { href: "/packs", label: "navPacks" },
  { href: "/keys", label: "navKeys" },
  { href: "/api-docs", label: "navDocs" },
] as const;

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { username, logout } = useAuth();

  const isActive = (href: string) => pathname === href || pathname.startsWith(`${href}/`);

  return (
    <div className="min-h-screen md:flex">
      <aside className="border-b border-zinc-200 bg-white md:w-60 md:shrink-0 md:border-b-0 md:border-r">
        <div className="flex items-center justify-between gap-3 px-5 py-4">
          <Link href="/factory" className="flex items-center gap-3">
            <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-600 text-lg leading-none text-white">
              ▦
            </span>
            <span className="text-sm font-bold tracking-widest text-zinc-900">{t("appName")}</span>
          </Link>
          <button
            onClick={async () => {
              await logout();
              router.replace("/login");
            }}
            className="rounded-lg border border-zinc-200 px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-zinc-50"
          >
            {t("signOut")}
          </button>
        </div>

        <nav aria-label={t("navStudio")} className="px-3 pb-4">
          <p className="hidden px-3 pb-1 pt-2 text-[11px] font-semibold uppercase tracking-wider text-zinc-400 md:block">
            {t("navStudio")}
          </p>
          <ul className="flex gap-1 overflow-x-auto md:flex-col md:gap-1">
            {NAV.map((item) => (
              <li key={item.href}>
                <Link
                  href={item.href}
                  aria-current={isActive(item.href) ? "page" : undefined}
                  className={`flex items-center rounded-lg px-3 py-2 text-sm font-medium whitespace-nowrap transition-colors ${
                    isActive(item.href)
                      ? "bg-indigo-600 text-white"
                      : "text-zinc-600 hover:bg-zinc-100 hover:text-zinc-900"
                  }`}
                >
                  {t(item.label)}
                </Link>
              </li>
            ))}
          </ul>
          <p className="mt-4 hidden truncate px-3 text-xs text-zinc-500 md:block">
            {username ? `owner: ${username}` : null}
          </p>
        </nav>
      </aside>

      <main className="min-w-0 flex-1 p-4 lg:p-8">{children}</main>
    </div>
  );
}
