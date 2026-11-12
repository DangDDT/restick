"use client";

import { ArtFrame } from "@/components/ui";
import { useI18n } from "@/lib/i18n";

/** Private same-origin reference asset shown inside the studio artwork frame. */
export function ReferencePreview({
  assetId,
  className,
  alt,
}: {
  assetId: string;
  className?: string;
  alt?: string;
}) {
  const { t } = useI18n();
  return (
    <ArtFrame className={className}>
      {/* The private route answers only with the session cookie; no key in the URL. */}
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src={`/api/v1/admin/reference-assets/${assetId}`}
        alt={alt ?? t("referenceImage")}
        className="h-full w-full object-contain"
      />
    </ArtFrame>
  );
}
