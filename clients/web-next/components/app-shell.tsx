"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth";
import { useI18n } from "@/lib/i18n";

function NavItem({
  href,
  label,
  icon,
  active,
}: {
  href: string;
  label: string;
  icon: string;
  active: boolean;
}) {
  return (
    <Link
      href={href}
      className={`flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
        active
          ? "bg-indigo-600 text-white"
          : "text-zinc-600 hover:bg-zinc-100 hover:text-zinc-900"
      }`}
    >
      <span className="text-base leading-none">{icon}</span>
      {label}
    </Link>
  );
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { user, isAdmin, logout } = useAuth();
  const { t, locale, setLocale } = useI18n();

  const isActive = (p: string) => pathname === p || pathname.startsWith(p + "/");

  return (
    <div className="flex min-h-screen">
      <aside className="flex w-64 shrink-0 flex-col border-r border-zinc-200 bg-white">
        <div className="flex items-center justify-between px-5 py-5">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-600 text-white">
              <span className="text-lg leading-none">▦</span>
            </div>
            <span className="text-sm font-bold tracking-widest text-zinc-900">
              {t("appName")}
            </span>
          </div>
          <div className="flex overflow-hidden rounded-md border border-zinc-200 text-xs font-semibold">
            {(["vi", "en"] as const).map((loc) => (
              <button
                key={loc}
                onClick={() => setLocale(loc)}
                className={`px-2 py-1 uppercase ${
                  locale === loc
                    ? "bg-indigo-600 text-white"
                    : "bg-white text-zinc-500 hover:bg-zinc-50"
                }`}
              >
                {loc}
              </button>
            ))}
          </div>
        </div>

        <nav className="flex-1 space-y-1 overflow-y-auto px-3 pb-4">
          <p className="px-3 pb-1 pt-2 text-[11px] font-semibold uppercase tracking-wider text-zinc-400">
            {t("navContributor")}
          </p>
          <NavItem href="/packs" label={t("navMyPacks")} icon="▤" active={isActive("/packs")} />
          <NavItem href="/packs/new" label={t("navUploadPack")} icon="＋" active={isActive("/packs/new")} />
          <NavItem href="/keys" label={t("navApiKeys")} icon="🔑" active={isActive("/keys")} />
          <NavItem href="/api-docs" label={t("navApiDocs")} icon="📖" active={isActive("/api-docs")} />

          {isAdmin && (
            <>
              <p className="px-3 pb-1 pt-5 text-[11px] font-semibold uppercase tracking-wider text-zinc-400">
                {t("navAdmin")}
              </p>
              <NavItem href="/admin/queue" label={t("navQueue")} icon="✓" active={isActive("/admin/queue")} />
              <NavItem href="/admin/published" label={t("navPublished")} icon="🌐" active={isActive("/admin/published")} />
              <NavItem href="/admin/analytics" label={t("navAnalytics")} icon="📊" active={isActive("/admin/analytics")} />
            </>
          )}
        </nav>

        <div className="border-t border-zinc-200 p-4">
          <div className="mb-3">
            <p className="truncate text-sm font-semibold text-zinc-900">
              {user?.display_name ?? user?.email}
            </p>
            <p className="truncate text-xs text-zinc-500">{user?.email}</p>
            {isAdmin && (
              <span className="mt-1 inline-block rounded bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase text-amber-700">
                Admin
              </span>
            )}
          </div>
          <button
            onClick={() => {
              logout();
              router.replace("/login");
            }}
            className="w-full rounded-lg border border-zinc-200 px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-zinc-50"
          >
            {t("signOut")}
          </button>
        </div>
      </aside>

      <main className="min-w-0 flex-1 overflow-x-hidden p-6 lg:p-8">{children}</main>
    </div>
  );
}
