"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { errorText } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useI18n } from "@/lib/i18n";
import { Button, ErrorBanner, Field, inputClass } from "@/components/ui";
import { Mascot } from "@/components/icons";

export default function LoginPage() {
  const { t } = useI18n();
  const { login } = useAuth();
  const router = useRouter();
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(event: React.FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await login(password);
      router.replace("/start");
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex min-h-dvh items-center justify-center bg-paper px-4 py-10">
      <main className="w-full max-w-sm rounded-panel border border-line bg-panel p-7">
        <div className="flex flex-col items-center gap-2 border-b border-line pb-5 text-center">
          <Mascot size={84} className="bob" />
          <span className="font-display text-xl font-bold tracking-tight text-ink">{t("appName")}</span>
          <span className="text-sm text-ink-2">{t("tagline")}</span>
        </div>
        <h1 className="mt-6 font-display text-[1.375rem] font-semibold leading-7 text-ink">{t("signIn")}</h1>
        <p className="mt-1 text-sm text-ink-2">{t("signInSubtitle")}</p>

        {error && (
          <div className="mt-5">
            <ErrorBanner message={error} />
          </div>
        )}

        <form onSubmit={onSubmit} className="mt-6 space-y-4" aria-busy={busy || undefined}>
          <Field label={t("password")}>
            <input
              type="password"
              required
              autoFocus
              autoComplete="current-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              className={inputClass}
              aria-invalid={Boolean(error) || undefined}
            />
          </Field>
          <Button type="submit" disabled={busy} loading={busy} className="w-full">
            {busy ? t("signingIn") : t("signIn")}
          </Button>
        </form>
      </main>
    </div>
  );
}
