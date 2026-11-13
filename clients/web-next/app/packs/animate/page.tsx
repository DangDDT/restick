"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Suspense, useEffect, useRef, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import { Icon } from "@/components/icons";
import {
  ArtFrame,
  Button,
  CopyButton,
  EmptyState,
  ErrorBanner,
  Field,
  PageHeader,
  Section,
  SkeletonRows,
  Spinner,
  Stamp,
  StatusPill,
  SuccessBanner,
  Switch,
  WarnBanner,
  inputClass,
} from "@/components/ui";
import { ApiError, api, errorText } from "@/lib/api";
import { VIDEO_PROVIDERS } from "@/lib/ai-providers";
import { copyImageURL } from "@/lib/copy-reference-image";
import { formatBytes } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import type {
  AdminPackDetail,
  ApiErrorBody,
  CompiledAnimation,
  PackAnimation,
  PackAnimationList,
} from "@/lib/types";

const POLL_MS = 2500;
// The worker has no progress feed; stop watching a stuck draft instead of
// polling forever. Reloading picks the durable state back up.
const POLL_LIMIT_MS = 120_000;
const MAX_CLIP_BYTES = 20 * 1024 * 1024;

function uploadError(xhr: XMLHttpRequest): ApiError {
  try {
    const body = JSON.parse(xhr.responseText) as ApiErrorBody;
    return new ApiError(xhr.status, body.error?.message ?? `HTTP ${xhr.status}`, body.error?.code ?? "");
  } catch {
    return new ApiError(xhr.status, `HTTP ${xhr.status}`, "");
  }
}

function AnimateView({ id, stickerIndex }: { id: string; stickerIndex: number }) {
  const { t } = useI18n();
  const qc = useQueryClient();
  // null means untouched: the uploaded draft's motion text seeds the field
  // without a state-copying effect.
  const [motion, setMotion] = useState<string | null>(null);
  const [prompt, setPrompt] = useState<{ text: string; emoji: string; motion: string } | null>(null);
  const promptRef = useRef<HTMLTextAreaElement>(null);
  const [compiling, setCompiling] = useState(false);
  const [compileError, setCompileError] = useState<string | null>(null);
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [percent, setPercent] = useState(0);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [toggling, setToggling] = useState(false);
  const [copyState, setCopyState] = useState<"copying" | "success" | "error" | null>(null);
  const [pollExpired, setPollExpired] = useState(false);
  const [pollCycle, setPollCycle] = useState(0);
  const xhrRef = useRef<XMLHttpRequest | null>(null);

  const pack = useQuery({
    queryKey: ["admin-pack", id],
    queryFn: () => api<AdminPackDetail>(`/api/v1/admin/packs/${encodeURIComponent(id)}`),
  });

  const animations = useQuery({
    queryKey: ["pack-animations", id],
    queryFn: () => api<PackAnimationList>(`/api/v1/admin/packs/${encodeURIComponent(id)}/animations`),
    refetchInterval: (query) => {
      if (pollExpired) return false;
      return (query.state.data?.items ?? []).some(
        (item) => item.status === "queued" || item.status === "running",
      )
        ? POLL_MS
        : false;
    },
  });

  useEffect(() => {
    const timer = window.setTimeout(() => setPollExpired(true), POLL_LIMIT_MS);
    return () => window.clearTimeout(timer);
  }, [id, stickerIndex, pollCycle]);

  const animationMotion = animations.data?.items.find((item) => item.sticker_index === stickerIndex)?.motion_text ?? "";

  useEffect(() => () => xhrRef.current?.abort(), []);

  const animation = animations.data?.items.find((item) => item.sticker_index === stickerIndex);
  const sticker = pack.data?.stickers[stickerIndex];
  const base = `/api/v1/admin/packs/${encodeURIComponent(id)}/stickers/${stickerIndex}`;
  const previewURL = `${base}/animation/preview`;
  const onWhiteURL = `${base}/on-white.png`;
  const motionValue = (motion ?? animationMotion).trim();
  const promptStale = Boolean(prompt && prompt.motion !== motionValue);

  async function compile() {
    if (!motionValue) {
      setCompileError(t("animateMotionRequired"));
      return;
    }
    setCompiling(true);
    setCompileError(null);
    try {
      const result = await api<CompiledAnimation>("/api/v1/admin/animations/compile", {
        json: { pack_id: id, sticker_index: stickerIndex, motion_text: motionValue },
      });
      setPrompt({ text: result.prompt, emoji: result.emoji_hint, motion: motionValue });
    } catch (err) {
      setCompileError(errorText(err));
    } finally {
      setCompiling(false);
    }
  }

  async function copyReference() {
    if (copyState === "copying") return;
    setCopyState("copying");
    try {
      await copyImageURL(onWhiteURL, "White reference");
      setCopyState("success");
    } catch {
      setCopyState("error");
    }
  }

  function pickFile(next: File | null) {
    setError(null);
    setMessage(null);
    if (next && next.size > MAX_CLIP_BYTES) {
      setFile(null);
      setError(t("animateClipTooLarge"));
      return;
    }
    setFile(next);
  }

  function upload() {
    if (!file || uploading) return;
    if (!motionValue) {
      setError(t("animateMotionRequired"));
      return;
    }
    const form = new FormData();
    form.append("clip", file);
    form.append("motion_text", motionValue);
    const xhr = new XMLHttpRequest();
    xhrRef.current = xhr;
    setUploading(true);
    setPercent(0);
    setError(null);
    setMessage(null);
    setPollExpired(false);
    xhr.open("POST", `${base}/animation`);
    xhr.upload.onprogress = (event) => {
      if (event.lengthComputable) setPercent(Math.round((event.loaded / event.total) * 100));
    };
    xhr.onerror = () => {
      setUploading(false);
      setError(t("animateUploadFailed"));
    };
    xhr.onabort = () => setUploading(false);
    xhr.onload = () => {
      setUploading(false);
      xhrRef.current = null;
      if (xhr.status >= 200 && xhr.status < 300) {
        setFile(null);
        setMessage(t("animateUploadDone"));
        setPollCycle((cycle) => cycle + 1);
        void qc.invalidateQueries({ queryKey: ["pack-animations", id] });
        void qc.invalidateQueries({ queryKey: ["admin-pack", id] });
      } else {
        setError(errorText(uploadError(xhr)));
      }
    };
    xhr.send(form);
  }

  async function setUse(next: boolean) {
    setToggling(true);
    setError(null);
    try {
      await api<PackAnimation>(`${base}/animation`, { method: "PATCH", json: { use_animated: next } });
      void qc.invalidateQueries({ queryKey: ["pack-animations", id] });
      void qc.invalidateQueries({ queryKey: ["admin-pack", id] });
    } catch (err) {
      setError(errorText(err));
    } finally {
      setToggling(false);
    }
  }

  if (pack.isLoading || animations.isLoading) {
    return (
      <div className="space-y-5">
        <SkeletonRows rows={2} />
        <SkeletonRows rows={4} />
      </div>
    );
  }
  if (pack.isError || !pack.data) {
    return (
      <>
        <PageHeader
          title={t("animateTitle")}
          back={{ href: `/packs/detail?id=${encodeURIComponent(id)}`, label: t("back") }}
        />
        <div className="space-y-3">
          <ErrorBanner message={pack.error ? errorText(pack.error) : t("loadFailed")} />
          <Button variant="secondary" icon="refresh" onClick={() => void pack.refetch()}>
            {t("retry")}
          </Button>
        </div>
      </>
    );
  }
  if (!sticker) {
    return (
      <>
        <PageHeader
          title={t("animateTitle")}
          back={{ href: `/packs/detail?id=${encodeURIComponent(id)}`, label: t("back") }}
        />
        <EmptyState title={t("animateMissingParams")} icon="image" />
      </>
    );
  }

  // The preview route keeps one URL per sticker; a fresh checksum busts the
  // browser's copy after a re-upload.
  const animatedSrc =
    animation?.status === "ready"
      ? `${previewURL}?v=${encodeURIComponent(animation.animated_checksum_sha256 ?? "")}`
      : sticker.animated?.object_url ?? null;
  const committed = Boolean(sticker.animated);
  const active = animation?.status === "queued" || animation?.status === "running";
  const pollStopped = pollExpired && active;

  return (
    <>
      <PageHeader
        title={`${t("animateTitle")} · ${sticker.filename}`}
        subtitle={t("animateSubtitle")}
        back={{ href: `/packs/detail?id=${encodeURIComponent(id)}`, label: t("back") }}
        actions={animation && <StatusPill status={animation.status} />}
      />

      {message && (
        <div className="mb-5">
          <SuccessBanner message={message} onDismiss={() => setMessage(null)} />
        </div>
      )}
      {error && (
        <div className="mb-5">
          <ErrorBanner message={error} onDismiss={() => setError(null)} />
        </div>
      )}
      {animations.isError && (
        <div className="mb-5">
          <ErrorBanner message={errorText(animations.error)} />
        </div>
      )}

      <Section title={t("animateCompare")}>
        <div className="grid gap-4 sm:grid-cols-2">
          <figure className="min-w-0">
            <ArtFrame className="aspect-square w-full">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={sticker.thumb_url || sticker.object_url}
                alt={`${t("animateStatic")}: ${sticker.filename}`}
                className="size-full object-contain p-2"
              />
            </ArtFrame>
            <figcaption className="mono mt-2 truncate text-xs text-ink-2">
              {t("animateStatic")} · {sticker.filename}
            </figcaption>
          </figure>
          <figure className="min-w-0">
            {animatedSrc ? (
              <ArtFrame className="aspect-square w-full">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  key={animation?.id ?? "committed"}
                  src={animatedSrc}
                  alt={`${t("animateAnimated")}: ${sticker.filename}`}
                  className="size-full object-contain p-2"
                />
              </ArtFrame>
            ) : (
              <ArtFrame className="flex aspect-square w-full items-center justify-center">
                <p className="px-4 text-center text-xs text-ink-3">{t("animateNoAnimation")}</p>
              </ArtFrame>
            )}
            <figcaption className="mt-2 flex flex-wrap items-center gap-2 text-xs text-ink-2">
              {t("animateAnimated")}
              {animation?.status === "ready" && <Stamp tone="active">{t("animateDraftBadge")}</Stamp>}
              {committed && <Stamp tone="ok">{t("animateCommitted")}</Stamp>}
            </figcaption>
          </figure>
        </div>
      </Section>

      <Section title={t("animateStatus")} actions={animation && <StatusPill status={animation.status} />}>
        {!animation ? (
          <p className="text-sm text-ink-2">{t("animateNoAnimation")}</p>
        ) : (
          <div className="space-y-3" aria-live="polite">
            {animation.stale && <WarnBanner message={t("animateStale")} />}
            {animation.error_code && (
              <ErrorBanner
                message={`${animation.error_code}${animation.error_message ? ` — ${animation.error_message}` : ""}`}
              />
            )}
            {active && (
              <p className="flex items-center gap-2 text-sm text-ink-2">
                <Spinner inline />
                {t("jobNoProgress")}
              </p>
            )}
            {pollStopped && <p className="text-xs text-warn">{t("animatePollingStopped")}</p>}
            <dl className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm sm:grid-cols-4">
              {animation.frames !== undefined && (
                <>
                  <dt className="text-ink-3">{t("animateFrameInfo")}</dt>
                  <dd className="num text-ink">{animation.frames}</dd>
                </>
              )}
              {animation.fps !== undefined && (
                <>
                  <dt className="text-ink-3">fps</dt>
                  <dd className="num text-ink">{animation.fps}</dd>
                </>
              )}
              {animation.duration_ms !== undefined && (
                <>
                  <dt className="text-ink-3">ms</dt>
                  <dd className="num text-ink">{animation.duration_ms}</dd>
                </>
              )}
              {animation.animated_byte_size !== undefined && (
                <>
                  <dt className="text-ink-3">{t("dimensions")}</dt>
                  <dd className="num text-ink">{formatBytes(animation.animated_byte_size)}</dd>
                </>
              )}
            </dl>
            <div className="flex flex-wrap items-center gap-3 border-t border-line pt-3">
              <div data-testid="animate-toggle" className="flex items-center gap-2">
                <Switch
                  checked={animation.use_animated}
                  onChange={(next) => void setUse(next)}
                  disabled={toggling || animation.status !== "ready" || animation.stale}
                  label={t("animateUse")}
                />
                <span className="text-sm text-ink">{t("animateUse")}</span>
                {toggling && <Spinner inline />}
              </div>
            </div>
            <p className="text-xs text-ink-2">{t("animateUseHint")}</p>
          </div>
        )}
      </Section>

      <Section title={t("animatePrompt")}>
        <Field label={t("animateMotion")} hint={t("animateMotionHint")}>
          <textarea
            rows={3}
            maxLength={500}
            value={motion ?? animationMotion}
            onChange={(event) => setMotion(event.target.value)}
            className={inputClass}
          />
        </Field>
        <div className="mt-3 flex flex-wrap items-center gap-3">
          <Button icon="sparkle" loading={compiling} disabled={!motionValue} onClick={() => void compile()}>
            {compiling ? t("animateCompiling") : t("animateCompile")}
          </Button>
          {prompt && (
            <div data-testid="animate-prompt-copy">
              <CopyButton text={prompt.text} selectRef={promptRef} label={t("copy")} />
            </div>
          )}
        </div>
        {promptStale && <p className="mt-2 text-xs text-warn">{t("animatePromptStale")}</p>}
        {compileError && (
          <div className="mt-3">
            <ErrorBanner message={compileError} />
          </div>
        )}
        {prompt && (
          <div className="mt-4 space-y-2">
            <textarea
              ref={promptRef}
              readOnly
              rows={6}
              value={prompt.text}
              aria-label={t("animatePrompt")}
              className={`${inputClass} mono text-xs`}
            />
            {prompt.emoji && (
              <p className="text-xs text-ink-2">
                {t("animateEmojiHint")}: <span className="mono">{prompt.emoji}</span>
              </p>
            )}
          </div>
        )}
        <p className="mt-2 text-xs text-ink-2">{t("animatePromptHint")}</p>
      </Section>

      <Section
        title={t("animateReference")}
        actions={
          <a
            href={onWhiteURL}
            download={`${stickerIndex}-white.png`}
            className="inline-flex min-h-8 items-center gap-1.5 rounded-ctl px-2 text-sm font-medium text-cut-ink hover:bg-cut-soft max-md:min-h-11"
          >
            <Icon name="download" size={15} />
            {t("animateDownloadReference")}
          </a>
        }
      >
        <div className="grid gap-4 sm:grid-cols-[minmax(0,16rem)_minmax(0,1fr)]">
          <ArtFrame className="aspect-square w-full">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={onWhiteURL}
              alt={t("animateReference")}
              loading="lazy"
              className="size-full object-contain p-1"
            />
          </ArtFrame>
          <div className="space-y-3">
            <p className="text-xs text-ink-2">{t("animateReferenceHint")}</p>
            <div className="flex flex-wrap items-center gap-3">
              <Button
                variant="secondary"
                icon="copy"
                loading={copyState === "copying"}
                onClick={() => void copyReference()}
              >
                {t("animateCopyReference")}
              </Button>
              <a
                href={onWhiteURL}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex min-h-11 items-center text-sm text-cut-ink underline underline-offset-4"
              >
                {t("animateOpenReference")}
              </a>
            </div>
            {copyState === "success" && <SuccessBanner message={t("animateReferenceCopied")} />}
            {copyState === "error" && <ErrorBanner message={t("animateReferenceCopyFailed")} />}
          </div>
        </div>
      </Section>

      <Section title={t("animateProviders")}>
        <p className="text-xs text-ink-2">{t("animateProvidersHint")}</p>
        <ul className="mt-3 flex flex-wrap gap-2">
          {VIDEO_PROVIDERS.map((provider) => (
            <li key={provider.id}>
              <a
                href={provider.url}
                target="_blank"
                rel="noopener noreferrer"
                title={provider.hint}
                className="inline-flex min-h-9 items-center gap-1.5 rounded-ctl border border-line-strong bg-panel px-2.5 text-sm text-ink hover:border-ink-3 hover:bg-sunk max-md:min-h-11"
              >
                {provider.name}
                <Icon name="external" size={13} />
              </a>
            </li>
          ))}
        </ul>
      </Section>

      <Section title={t("animateClip")}>
        <p className="text-xs text-ink-2">{t("animateClipHint")}</p>
        <div className="mt-3 grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end">
          <Field label={t("animateClip")}>
            <input
              type="file"
              accept="video/mp4,video/webm,video/quicktime"
              onChange={(event) => pickFile(event.target.files?.[0] ?? null)}
              className={inputClass}
              disabled={uploading}
            />
          </Field>
          <div data-testid="animate-upload">
            <Button icon="upload" loading={uploading} disabled={!file || !motionValue || uploading} onClick={upload}>
              {uploading ? `${t("animateUploading")} ${percent}%` : t("animateUpload")}
            </Button>
          </div>
        </div>
        {file && (
          <p className="mono mt-2 text-xs text-ink-2">
            {file.name} · {formatBytes(file.size)}
          </p>
        )}
        {uploading && (
          <div
            role="progressbar"
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={percent}
            aria-label={t("animateUploading")}
            className="mt-3 h-2 w-full overflow-hidden rounded-full bg-sunk"
          >
            <span className="block h-full bg-cut transition-[width] duration-150" style={{ width: `${percent}%` }} />
          </div>
        )}
        {!motionValue && <p className="mt-2 text-xs text-warn">{t("animateMotionRequired")}</p>}
      </Section>

      <p className="border-t border-line py-5 text-xs text-ink-3">
        <Link
          href={`/packs/detail?id=${encodeURIComponent(id)}`}
          className="inline-flex min-h-9 items-center font-medium text-cut-ink hover:underline"
        >
          {t("back")}
        </Link>
      </p>
    </>
  );
}

function AnimatePage() {
  const { t } = useI18n();
  const params = useSearchParams();
  const id = params.get("id");
  const rawIndex = params.get("index");
  const stickerIndex = rawIndex !== null && /^\d+$/.test(rawIndex) ? Number(rawIndex) : NaN;
  const valid = Boolean(id) && Number.isInteger(stickerIndex);

  return (
    <RequireAuth>
      <AppShell>
        {valid && id ? (
          <AnimateView key={`${id}:${stickerIndex}`} id={id} stickerIndex={stickerIndex} />
        ) : (
          <>
            <PageHeader title={t("animateTitle")} back={{ href: "/packs", label: t("openPackList") }} />
            <EmptyState title={t("animateMissingParams")} icon="image">
              <Link
                href="/packs"
                className="inline-flex min-h-9 items-center text-sm font-medium text-cut-ink hover:underline"
              >
                {t("openPackList")}
              </Link>
            </EmptyState>
          </>
        )}
      </AppShell>
    </RequireAuth>
  );
}

export default function Page() {
  return (
    <Suspense fallback={<Spinner />}>
      <AnimatePage />
    </Suspense>
  );
}
