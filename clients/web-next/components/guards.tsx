"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { useAuth } from "@/lib/auth";
import { useI18n } from "@/lib/i18n";

export function RequireAuth({ children }: { children: React.ReactNode }) {
  const { t } = useI18n();
  const { isReady, username } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (isReady && !username) router.replace("/login");
  }, [isReady, username, router]);

  if (!isReady || !username) {
    return (
      <div className="flex min-h-screen items-center justify-center text-sm text-zinc-500">
        {t("loading")}
      </div>
    );
  }
  return <>{children}</>;
}
