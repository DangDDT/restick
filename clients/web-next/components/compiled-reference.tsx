"use client";

import { useState } from "react";
import { ReferencePreview } from "@/components/reference-preview";
import { Button, ErrorBanner, SuccessBanner } from "@/components/ui";
import { copyReferenceImage } from "@/lib/copy-reference-image";
import { useI18n } from "@/lib/i18n";
import type { Snapshot } from "@/lib/types";

function FrozenReference({ assetId, name }: { assetId: string; name: string }) {
  const { t } = useI18n();
  const [status, setStatus] = useState<"copying" | "success" | "error" | null>(null);

  async function copyImage() {
    if (status === "copying") return;
    setStatus("copying");
    try {
      await copyReferenceImage(assetId);
      setStatus("success");
    } catch {
      setStatus("error");
    }
  }

  return (
    <section aria-label={t("compiledReference")} className="mt-4 space-y-3 border-t border-line pt-4">
      <h3 className="text-sm font-medium text-ink">{t("compiledReference")} · {name}</h3>
      <ReferencePreview assetId={assetId} alt={`${t("referenceImage")}: ${name}`} className="aspect-square w-full max-w-64" />
      <p className="text-xs text-ink-2">{t("compiledReferenceHint")}</p>
      <div className="flex flex-wrap items-center gap-3">
        <Button variant="secondary" icon="copy" loading={status === "copying"} onClick={() => void copyImage()}>
          {t("copyReferenceImage")}
        </Button>
        <a
          href={`/api/v1/admin/reference-assets/${encodeURIComponent(assetId)}`}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex min-h-11 items-center text-sm text-cut-ink underline underline-offset-4"
        >
          {t("openReferenceImage")}
        </a>
      </div>
      {status === "success" && <SuccessBanner message={t("referenceImageCopied")} />}
      {status === "error" && <ErrorBanner message={t("referenceImageCopyFailed")} />}
    </section>
  );
}

export function CompiledReference({ character }: { character: Snapshot["character"] }) {
  return character?.reference_asset_id
    ? <FrozenReference key={character.reference_asset_id} assetId={character.reference_asset_id} name={character.name} />
    : null;
}
