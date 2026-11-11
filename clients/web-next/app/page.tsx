"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { Icon } from "@/components/icons";
import { Spinner } from "@/components/ui";
import { useI18n } from "@/lib/i18n";

export default function Home() {
  const { t } = useI18n();
  const router = useRouter();

  useEffect(() => {
    router.replace("/factory");
  }, [router]);

  return (
    <div className="flex min-h-dvh flex-col items-center justify-center gap-5">
      <span className="flex size-10 items-center justify-center rounded-[6px] bg-ink text-paper" aria-hidden="true">
        <Icon name="scissors" size={18} />
      </span>
      <Spinner />
      <span className="sr-only">{t("appName")}</span>
    </div>
  );
}
