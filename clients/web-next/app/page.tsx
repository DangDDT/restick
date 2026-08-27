"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { useAuth } from "@/lib/auth";

export default function Home() {
  const { isReady, token } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!isReady) return;
    router.replace(token ? "/packs" : "/login");
  }, [isReady, token, router]);

  return (
    <div className="flex min-h-screen items-center justify-center text-sm text-zinc-500">
      Loading…
    </div>
  );
}
