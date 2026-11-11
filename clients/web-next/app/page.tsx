"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { Mascot } from "@/components/icons";
import { Spinner } from "@/components/ui";
import { useI18n } from "@/lib/i18n";

export default function Home() {
  const { t } = useI18n();
  const router = useRouter();

  useEffect(() => {
    router.replace("/start");
  }, [router]);

  return (
    <div className="flex min-h-dvh flex-col items-center justify-center gap-5">
      <Mascot size={72} className="bob" />
      <Spinner />
      <span className="sr-only">{t("appName")}</span>
    </div>
  );
}
