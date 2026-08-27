"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { useAuth } from "@/lib/auth";

export function RequireAuth({ children }: { children: React.ReactNode }) {
  const { isReady, token } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (isReady && !token) router.replace("/login");
  }, [isReady, token, router]);

  if (!isReady || !token) {
    return (
      <div className="flex min-h-screen items-center justify-center text-sm text-zinc-500">
        Loading…
      </div>
    );
  }
  return <>{children}</>;
}

export function RequireAdmin({ children }: { children: React.ReactNode }) {
  const { isReady, isAdmin } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (isReady && !isAdmin) router.replace("/packs");
  }, [isReady, isAdmin, router]);

  if (!isReady || !isAdmin) {
    return (
      <div className="flex min-h-screen items-center justify-center text-sm text-zinc-500">
        Loading…
      </div>
    );
  }
  return <>{children}</>;
}
