"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useAuth } from "@/lib/auth";
import { useI18n } from "@/lib/i18n";
import { Button, ErrorBanner, Field, inputClass } from "@/components/ui";

export default function LoginPage() {
  const { login } = useAuth();
  const { t } = useI18n();
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await login(email.trim(), password);
      router.replace("/packs");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-50 px-4">
      <div className="w-full max-w-sm rounded-xl border border-zinc-200 bg-white p-8 shadow-sm">
        <div className="mb-6 flex items-center gap-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-600 text-white">
            <span className="text-lg leading-none">▦</span>
          </div>
          <span className="text-sm font-bold tracking-widest text-zinc-900">{t("appName")}</span>
        </div>
        <h1 className="text-lg font-bold text-zinc-900">{t("signIn")}</h1>
        <p className="mb-6 mt-1 text-sm text-zinc-500">{t("loginSubtitle")}</p>

        {error && <div className="mb-4"><ErrorBanner message={error} /></div>}

        <form onSubmit={onSubmit} className="space-y-4">
          <Field label={t("email")}>
            <input
              type="email"
              required
              autoComplete="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className={inputClass}
              placeholder="you@example.com"
            />
          </Field>
          <Field label={t("password")}>
            <input
              type="password"
              required
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={inputClass}
              placeholder="••••••••"
            />
          </Field>
          <Button type="submit" disabled={busy} className="w-full">
            {busy ? t("signingIn") : t("signIn")}
          </Button>
        </form>

        <p className="mt-6 text-center text-sm text-zinc-500">
          {t("noAccount")}{" "}
          <Link href="/register" className="font-medium text-indigo-600 hover:underline">
            {t("register")}
          </Link>
        </p>
      </div>
    </div>
  );
}
