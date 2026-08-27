"use client";

import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import { PageHeader } from "@/components/ui";

export default function Page() {
  return (
    <RequireAuth>
      <AppShell>
        <PageHeader
          title="API Docs"
          subtitle="Interactive OpenAPI documentation"
        />
        <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white">
          <iframe
            src="/docs"
            title="API Docs"
            className="h-[calc(100vh-12rem)] w-full"
          />
        </div>
      </AppShell>
    </RequireAuth>
  );
}
