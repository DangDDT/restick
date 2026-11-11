"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { useI18n } from "@/lib/i18n";

export default function Home() {
  const { t } = useI18n();
  const router = useRouter();

  useEffect(() => {
    router.replace("/factory");
  }, [router]);

  return (
    <div className="flex min-h-screen items-center justify-center text-sm text-zinc-500">
      {t("loading")}
    </div>
  );
}
