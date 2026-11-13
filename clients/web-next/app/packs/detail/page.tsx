"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Suspense, useEffect, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { DesignPanel } from "@/components/design-panel";
import { SheetPicker, sheetImageURL } from "@/components/design-sheets";
import { RequireAuth } from "@/components/guards";
import { Icon } from "@/components/icons";
import {
  ArtFrame,
  Button,
  ConfirmDialog,
  Dialog,
  EmptyState,
  ErrorBanner,
  Field,
  JobTag,
  PageHeader,
  Section,
  SkeletonRows,
  Spinner,
  Stamp,
  StatusPill,
  SuccessBanner,
  cx,
  inputClass,
} from "@/components/ui";
import { api, errorText } from "@/lib/api";
import { providerName } from "@/lib/ai-providers";
import { formatBytes, formatDate } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import type { AdminPack, AdminPackDetail, AdminPackSticker, Job, JobStatus, PackUpdate } from "@/lib/types";

const POLL_MS = 3000;
// Job timeout is 300 s server-side (RESTICK_PROCESS_TIMEOUT); stop the browser
// poll a little after that instead of polling a terminal-or-abandoned job forever.
const POLL_LIMIT_MS = 330_000;

function isTerminal(status: JobStatus): boolean {
  return status === "succeeded" || status === "failed" || status === "interrupted";
}

type VariantChange = { index: number; filename: string; kind: "animated" | "static" };

// The effective draft variant is the ready, enabled, non-stale animation; it is
// compared with the committed checksum so the dialog lists real changes only.
function unpublishedChanges(stickers: AdminPackSticker[]): VariantChange[] {
  const changes: VariantChange[] = [];
  stickers.forEach((sticker, index) => {
    const draft = sticker.animation;
    if (!draft) return;
    const active = draft.status === "ready" && draft.use_animated && !draft.stale;
    if (active) {
      if (draft.animated_checksum_sha256 !== sticker.animated?.checksum_sha256) {
        changes.push({ index, filename: sticker.filename, kind: "animated" });
      }
      return;
    }
    if (!draft.use_animated && sticker.animated) {
      changes.push({ index, filename: sticker.filename, kind: "static" });
    }
  });
  return changes;
}

function CatalogForm({ pack, onDone }: { pack: AdminPackDetail; onDone: (message: string | null) => void }) {
  const { t } = useI18n();
  const qc = useQueryClient();
  const [name, setName] = useState(pack.name);
  const [author, setAuthor] = useState(pack.author);
  const [description, setDescription] = useState(pack.description);
  const [tags, setTags] = useState((pack.tags ?? []).join(", "));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const body: PackUpdate = {
        name: name.trim(),
        author: author.trim(),
        description: description.trim(),
        tags: tags
          .split(",")
          .map((tag) => tag.trim())
          .filter((tag) => tag.length > 0),
        design_id: pack.design_id,
        sheet_id: pack.sheet_id,
      };
      await api<AdminPack>(`/api/v1/admin/packs/${pack.id}`, { method: "PUT", json: body });
      qc.invalidateQueries({ queryKey: ["admin-pack", pack.id] });
      qc.invalidateQueries({ queryKey: ["admin-packs"] });
      onDone(t("catalogSaved"));
    } catch (err) {
      setError(errorText(err));
      setBusy(false);
    }
  }

  return (
    <form onSubmit={submit} className="space-y-3" aria-busy={busy || undefined}>
      {error && <ErrorBanner message={error} />}
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label={t("packName")}>
          <input value={name} onChange={(e) => setName(e.target.value)} className={inputClass} />
        </Field>
        <Field label={t("packAuthor")}>
          <input value={author} onChange={(e) => setAuthor(e.target.value)} className={inputClass} />
        </Field>
      </div>
      <Field label={t("packDescription")}>
        <textarea value={description} onChange={(e) => setDescription(e.target.value)} rows={2} className={inputClass} />
      </Field>
      <Field label={t("packTags")}>
        <input value={tags} onChange={(e) => setTags(e.target.value)} className={inputClass} />
      </Field>
      <p className="text-xs text-ink-2">{t("catalogHint")}</p>
      <div className="flex justify-end">
        <Button type="submit" disabled={busy} loading={busy}>
          {busy ? t("saving") : t("saveCatalog")}
        </Button>
      </div>
    </form>
  );
}

/** Station flow: Sheet -> Cắt -> Sẵn sàng. Derived only from data already on the page. */
function StationFlow({
  sheetName,
  hasSheet,
  jobStatus,
  packStatus,
}: {
  sheetName?: string;
  hasSheet: boolean;
  jobStatus?: JobStatus;
  packStatus: string;
}) {
  const { t } = useI18n();
  const jobActive = jobStatus === "queued" || jobStatus === "running";
  const current = !hasSheet ? 0 : jobActive || packStatus === "draft" ? 1 : 2;

  const stations = [
    {
      key: "sheet",
      label: t("stationSheet"),
      body: hasSheet ? (
        <span className="mono flex min-w-0 items-center gap-1.5 text-xs text-ok">
          <Icon name="check" size={14} className="shrink-0" />
          <span className="truncate">{sheetName}</span>
        </span>
      ) : (
        <span className="text-xs text-ink-2">{t("noSheet")}</span>
      ),
    },
    {
      key: "cut",
      label: t("stationCut"),
      body: jobStatus ? <StatusPill status={jobStatus} /> : <Stamp tone="neutral">{t("none")}</Stamp>,
    },
    {
      key: "ready",
      label: t("statusReady"),
      body: <StatusPill status={packStatus} />,
    },
  ];

  return (
    <ol aria-label={t("pkStationFlow")} className="grid grid-cols-3 border-y border-line">
      {stations.map((station, index) => {
        const active = index === current;
        const done = index < current;
        return (
          <li
            key={station.key}
            aria-current={active ? "step" : undefined}
            className={cx(
              "flex min-w-0 flex-col gap-2 border-t-2 px-3 py-3 first:pl-0 last:pr-0 sm:px-4",
              active ? "border-cut" : done ? "border-ok/60" : "border-transparent",
            )}
          >
            <span className={cx("flex items-center gap-1.5 text-sm font-medium", active ? "text-cut-ink" : "text-ink-2")}>
              <span className="num text-xs text-ink-3">{index + 1}</span>
              {station.label}
            </span>
            <div className="min-w-0">{station.body}</div>
          </li>
        );
      })}
    </ol>
  );
}

type Confirm = "revoke" | "delete";

function PackDetailView({ id, initialJobId }: { id: string; initialJobId: string | null }) {
  const { t } = useI18n();
  const router = useRouter();
  const qc = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [starting, setStarting] = useState(false);
  const [sheetOpen, setSheetOpen] = useState(false);
  const [pickedSheet, setPickedSheet] = useState("");
  const [sheetBusy, setSheetBusy] = useState(false);
  const [sheetError, setSheetError] = useState<string | null>(null);
  const [requestedJobId, setJobId] = useState<string | null>(null);
  const [pollExpired, setPollExpired] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [confirm, setConfirm] = useState<Confirm | null>(null);
  const [confirmBusy, setConfirmBusy] = useState(false);
  const [confirmError, setConfirmError] = useState<string | null>(null);
  const [publishOpen, setPublishOpen] = useState(false);
  const [publishBusy, setPublishBusy] = useState(false);
  const [publishError, setPublishError] = useState<string | null>(null);

  const pack = useQuery({
    queryKey: ["admin-pack", id],
    queryFn: () => api<AdminPackDetail>(`/api/v1/admin/packs/${id}`),
    refetchOnMount: "always",
  });
  const jobId = requestedJobId ?? pack.data?.latest_job?.id ?? initialJobId;

  const job = useQuery({
    queryKey: ["job", jobId],
    queryFn: () => api<Job>(`/api/v1/admin/jobs/${jobId}`),
    enabled: Boolean(jobId),
    initialData: () => (pack.data?.latest_job?.id === jobId ? pack.data.latest_job : undefined),
    refetchInterval: (query) => {
      const data = query.state.data;
      if (data && isTerminal(data.status)) return false;
      if (pollExpired) return false;
      return POLL_MS;
    },
  });

  // Stop polling at the server job timeout, measured from the moment this view
  // starts watching the job (including after a reload with ?job=...).
  useEffect(() => {
    if (!jobId) return;
    const timer = window.setTimeout(() => setPollExpired(true), POLL_LIMIT_MS);
    return () => window.clearTimeout(timer);
  }, [jobId]);

  const jobStatus = job.data?.status;
  useEffect(() => {
    if (!jobStatus || !isTerminal(jobStatus)) return;
    qc.invalidateQueries({ queryKey: ["admin-pack", id] });
    qc.invalidateQueries({ queryKey: ["admin-packs"] });
  }, [jobStatus, id, qc]);

  async function changeSheet() {
    if (!pack.data || !pickedSheet || pickedSheet === pack.data.sheet_id) return;
    setSheetBusy(true);
    setSheetError(null);
    setMessage(null);
    try {
      const body: PackUpdate = {
        name: pack.data.name,
        author: pack.data.author,
        description: pack.data.description,
        tags: pack.data.tags ?? [],
        design_id: pack.data.design_id,
        sheet_id: pickedSheet,
      };
      await api<AdminPack>(`/api/v1/admin/packs/${id}`, { method: "PUT", json: body });
      setSheetOpen(false);
      setMessage(t("sheetChanged"));
      qc.invalidateQueries({ queryKey: ["admin-pack", id] });
      qc.invalidateQueries({ queryKey: ["admin-packs"] });
    } catch (err) {
      // A running job keeps the sheet fixed: the 409 job_active message names the wait.
      setSheetError(errorText(err));
    } finally {
      setSheetBusy(false);
    }
  }

  async function startProcessing() {
    setStarting(true);
    setError(null);
    setMessage(null);
    try {
      const created = await api<Job>(`/api/v1/admin/packs/${id}/process`, { method: "POST" });
      setJobId(created.id);
      setPollExpired(false);
      // Keep explicit job links reloadable; ordinary navigation discovers the
      // latest durable attempt through the admin detail response.
      router.replace(`/packs/detail?id=${encodeURIComponent(id)}&job=${encodeURIComponent(created.id)}`);
      setMessage(t("processStarted"));
      qc.invalidateQueries({ queryKey: ["admin-pack", id] });
    } catch (err) {
      setError(errorText(err));
    } finally {
      setStarting(false);
    }
  }

  async function runConfirm() {
    if (!confirm) return;
    setConfirmBusy(true);
    setConfirmError(null);
    try {
      if (confirm === "revoke") {
        await api<AdminPack>(`/api/v1/admin/packs/${id}/revoke`, { method: "POST" });
        setConfirm(null);
        setMessage(t("revokedDone"));
        qc.invalidateQueries({ queryKey: ["admin-pack", id] });
        qc.invalidateQueries({ queryKey: ["admin-packs"] });
      } else {
        await api<void>(`/api/v1/admin/packs/${id}`, { method: "DELETE" });
        qc.invalidateQueries({ queryKey: ["admin-packs"] });
        setConfirm(null);
        router.replace("/packs");
      }
    } catch (err) {
      setConfirmError(errorText(err));
    } finally {
      setConfirmBusy(false);
    }
  }

  async function publishAnimations() {
    setPublishBusy(true);
    setPublishError(null);
    setError(null);
    setMessage(null);
    try {
      const created = await api<Job>(`/api/v1/admin/packs/${id}/publish-animations`, { method: "POST" });
      setJobId(created.id);
      setPollExpired(false);
      router.replace(`/packs/detail?id=${encodeURIComponent(id)}&job=${encodeURIComponent(created.id)}`);
      setPublishOpen(false);
      setMessage(t("animatePublishStarted"));
      void qc.invalidateQueries({ queryKey: ["admin-pack", id] });
      void qc.invalidateQueries({ queryKey: ["admin-packs"] });
    } catch (err) {
      // 409 no_changes / job_active stay next to the changes they refer to.
      setPublishError(errorText(err));
    } finally {
      setPublishBusy(false);
    }
  }

  if (pack.isLoading) {
    return (
      <div className="space-y-5">
        <SkeletonRows rows={2} />
        <SkeletonRows rows={5} />
      </div>
    );
  }
  if (pack.isError || !pack.data) {
    return (
      <>
        <PageHeader title={t("detailTitle")} back={{ href: "/packs", label: t("openPackList") }} />
        <div className="space-y-3">
          <ErrorBanner message={pack.error ? errorText(pack.error) : t("loadFailed")} />
          <Button variant="secondary" icon="refresh" onClick={() => void pack.refetch()}>
            {t("retry")}
          </Button>
        </div>
      </>
    );
  }

  const data = pack.data;
  const changes = unpublishedChanges(data.stickers);
  const designId = data.design_id ?? "";
  const jobRunning = job.data ? !isTerminal(job.data.status) : false;
  const pollStopped = pollExpired && Boolean(job.data && !isTerminal(job.data.status));

  return (
    <>
      <PageHeader
        title={data.name}
        subtitle={`${data.slug} · ${data.author} · ${t("currentVersion")} ${data.current_version}`}
        back={{ href: "/packs", label: t("openPackList") }}
        actions={
          <>
            <JobTag id={data.id} />
            <StatusPill status={data.status} />
          </>
        }
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

      <div className="py-5">
        <StationFlow
          hasSheet={Boolean(data.sheet)}
          sheetName={data.sheet?.filename}
          jobStatus={job.data?.status}
          packStatus={data.status}
        />
      </div>

      <div className="grid gap-x-8 lg:grid-cols-2">
        <Section
          title={t("sheetLabel")}
          actions={
            <>
              <Button
                variant="secondary"
                size="sm"
                onClick={() => {
                  setPickedSheet(data.sheet_id);
                  setSheetError(null);
                  setSheetOpen(true);
                }}
                disabled={!designId || data.status === "revoked"}
              >
                {t("changeSheet")}
              </Button>
              <Button
                onClick={() => void startProcessing()}
                disabled={starting || jobRunning || !data.sheet || data.status === "revoked"}
                loading={starting}
                icon="scissors"
                size="sm"
              >
                {starting ? t("processing") : t("process")}
              </Button>
            </>
          }
        >
          {data.sheet ? (
            <div className="space-y-3">
              <ArtFrame className="aspect-[4/3] w-full max-w-xs">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  data-qa="pack-sheet-image"
                  src={sheetImageURL(designId, data.sheet.id)}
                  alt={data.sheet.filename}
                  className="size-full object-contain p-1"
                />
              </ArtFrame>
              <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
                <dt className="text-ink-3">{t("sheetProvider")}</dt>
                <dd className="text-ink">{providerName(data.sheet.provider)}</dd>
                <dt className="text-ink-3">{t("sheetLabel")}</dt>
                <dd className="mono break-all text-xs text-ink">{data.sheet.filename}</dd>
                <dt className="text-ink-3">{t("dimensions")}</dt>
                <dd className="num text-ink">
                  {data.sheet.width}×{data.sheet.height} · {formatBytes(data.sheet.byte_size)}
                </dd>
                <dt className="text-ink-3">{t("checksum")}</dt>
                <dd className="mono break-all text-[11px] text-ink-2">{data.sheet.checksum_sha256}</dd>
              </dl>
            </div>
          ) : (
            <p className="text-sm text-ink-2">{t("noSheet")}</p>
          )}
          <p className="mt-4 text-xs text-ink-2">{t("processHint")}</p>
          {data.status === "revoked" && <p className="text-xs text-ink-2">{t("revokedHint")}</p>}
        </Section>

        <Section
          title={t("pkCatalogSection")}
          actions={
            <Button variant="secondary" size="sm" icon={editing ? "close" : "edit"} onClick={() => setEditing((value) => !value)}>
              {editing ? t("cancel") : t("editCatalog")}
            </Button>
          }
        >
          {editing ? (
            <CatalogForm
              pack={data}
              onDone={(next) => {
                setEditing(false);
                setMessage(next);
              }}
            />
          ) : (
            <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
              <dt className="text-ink-3">{t("packAuthor")}</dt>
              <dd className="text-ink">{data.author || t("none")}</dd>
              <dt className="text-ink-3">{t("packDescription")}</dt>
              <dd className="text-ink">{data.description || t("none")}</dd>
              <dt className="text-ink-3">{t("packTags")}</dt>
              <dd className="text-ink">{(data.tags ?? []).length ? (data.tags ?? []).join(", ") : t("none")}</dd>
              <dt className="text-ink-3">{t("designLabel")}</dt>
              <dd className="mono break-all text-xs text-ink-2">{data.design_id || t("none")}</dd>
              <dt className="text-ink-3">{t("createdAt")}</dt>
              <dd className="num text-ink-2">{formatDate(data.created_at)}</dd>
              <dt className="text-ink-3">{t("updatedAt")}</dt>
              <dd className="num text-ink-2">{formatDate(data.updated_at)}</dd>
              <dt className="text-ink-3">{t("checksum")}</dt>
              <dd className="mono break-all text-[11px] text-ink-2">{data.checksum_sha256 || t("none")}</dd>
            </dl>
          )}
        </Section>
      </div>

      {data.design && (
        <Section title={t("designLabel")}>
          <DesignPanel design={data.design} />
        </Section>
      )}

      <Section title={t("jobLabel")} actions={job.data && <StatusPill status={job.data.status} />}>
        {/* Live region: job status changes are announced without moving focus. */}
        <div aria-live="polite" className="space-y-2 text-sm">
          {!jobId && <p className="text-ink-2">{t("pkNoJob")}</p>}
          {jobId && job.isLoading && <SkeletonRows rows={2} />}
          {jobId && job.isError && <ErrorBanner message={errorText(job.error)} />}
          {jobId && job.data && (
            <>
              <p className="num text-xs text-ink-2">
                {t("jobVersion")} {job.data.version} · {formatDate(job.data.created_at)}
              </p>
              {job.data.status === "running" && (
                <p className="flex items-center gap-2 text-ink-2">
                  <Spinner inline />
                  {t("jobNoProgress")}
                </p>
              )}
              {job.data.status === "queued" && <p className="text-ink-2">{t("jobNoProgress")}</p>}
              {job.data.error_code && (
                <p className="text-danger">
                  {t("jobError")}: <code className="mono">{job.data.error_code}</code>
                  {job.data.error_message ? ` — ${job.data.error_message}` : ""}
                </p>
              )}
              {job.data.diagnostics && (
                <div>
                  <p className="text-xs font-medium text-ink-2">{t("jobDiagnostics")}</p>
                  <pre className="mono mt-1 max-h-40 overflow-auto rounded-ctl bg-sunk p-2.5 text-xs text-ink-2">
                    {job.data.diagnostics}
                  </pre>
                </div>
              )}
            </>
          )}
          {pollStopped && <p className="text-xs text-warn">{t("jobPollingStopped")}</p>}
        </div>
      </Section>

      <Section
        title={t("animateUnpublished")}
        hint={t("animatePublishHint")}
        actions={
          <span data-testid="publish-animations">
            <Button
              icon="sparkle"
              size="sm"
              disabled={data.status === "revoked" || jobRunning}
              onClick={() => {
                setPublishError(null);
                setPublishOpen(true);
              }}
            >
              {t("animatePublish")}
            </Button>
          </span>
        }
      >
        <p className="flex flex-wrap items-center gap-2 text-sm text-ink-2">
          <span data-qa="unpublished-count" className="num text-ink">
            {changes.length}
          </span>
          {t("animateUnpublished")}
        </p>
        {changes.length === 0 ? (
          <p className="mt-2 text-xs text-ink-2">{t("animateNoneUnpublished")}</p>
        ) : (
          <ul className="mt-3 flex flex-wrap gap-2">
            {changes.map((change) => (
              <li key={change.index} className="flex min-w-0 items-center gap-2">
                <span className="mono truncate text-xs text-ink-2">{change.filename}</span>
                <Stamp tone={change.kind === "animated" ? "active" : "neutral"}>
                  {t(change.kind === "animated" ? "animateChangeAnimated" : "animateChangeStatic")}
                </Stamp>
              </li>
            ))}
          </ul>
        )}
      </Section>

      <Section
        title={t("stickersTitle")}
        actions={
          data.bundle_url ? (
            <a
              href={data.bundle_url}
              className="inline-flex min-h-8 items-center gap-1.5 rounded-ctl px-2 text-sm font-medium text-cut-ink hover:bg-cut-soft max-md:min-h-11"
            >
              <Icon name="download" size={15} />
              {t("downloadBundle")}
            </a>
          ) : undefined
        }
      >
        {data.stickers.length === 0 ? (
          <EmptyState title={t("noStickers")} icon="image" />
        ) : (
          <ul className="grid grid-cols-2 gap-px overflow-hidden rounded-panel border border-line bg-line sm:grid-cols-3 md:grid-cols-4 xl:grid-cols-6">
            {data.stickers.map((sticker, index) => (
              <li
                key={sticker.filename}
                className="peel flex min-w-0 flex-col gap-2 bg-panel p-2"
                style={{ "--i": index } as React.CSSProperties}
              >
                <a href={sticker.object_url} className="block">
                  <ArtFrame className="aspect-square w-full rounded-[4px]">
                    {/* eslint-disable-next-line @next/next/no-img-element */}
                    <img
                      src={sticker.thumb_url || sticker.object_url}
                      alt={sticker.filename}
                      loading="lazy"
                      className="size-full object-contain p-1"
                    />
                  </ArtFrame>
                </a>
                <div className="min-w-0 space-y-0.5 text-xs text-ink-2">
                  <p className="mono truncate text-ink">{sticker.filename}</p>
                  <p className="num">
                    {sticker.width}×{sticker.height}
                  </p>
                  {sticker.emoji_hint && <p className="truncate">{sticker.emoji_hint}</p>}
                  {(sticker.keywords ?? []).length > 0 && <p className="truncate text-ink-3">{(sticker.keywords ?? []).join(", ")}</p>}
                  {(sticker.animated || sticker.animation) && (
                    <div className="flex flex-wrap items-center gap-1 pt-0.5">
                      {sticker.animated && <Stamp tone="ok">{t("animateCommitted")}</Stamp>}
                      {sticker.animation && <StatusPill status={sticker.animation.status} />}
                      {sticker.animated && sticker.animation && !sticker.animation.use_animated && (
                        <Stamp tone="warn">{t("animatePendingStatic")}</Stamp>
                      )}
                    </div>
                  )}
                  <div className="flex flex-wrap items-center gap-x-3">
                    <a
                      href={sticker.object_url}
                      download
                      className="inline-flex min-h-8 items-center gap-1 font-medium text-cut-ink hover:underline max-md:min-h-11"
                    >
                      <Icon name="download" size={13} />
                      {t("downloadFile")}
                    </a>
                    <Link
                      data-testid={`animate-button-${index}`}
                      href={`/packs/animate?id=${encodeURIComponent(data.id)}&index=${index}`}
                      className="inline-flex min-h-8 items-center gap-1 font-medium text-cut-ink hover:underline max-md:min-h-11"
                    >
                      <Icon name="sparkle" size={13} />
                      {t("animateLink")}
                    </Link>
                  </div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </Section>

      {(data.status === "draft" || data.status === "ready") && (
        <div className="mt-6 flex flex-wrap gap-2 border-t border-line pt-5">
          {data.status === "draft" && (
            <Button variant="danger" icon="trash" onClick={() => setConfirm("delete")}>
              {t("deletePack")}
            </Button>
          )}
          {data.status === "ready" && (
            <Button variant="danger" icon="lock" onClick={() => setConfirm("revoke")}>
              {t("revoke")}
            </Button>
          )}
        </div>
      )}

      {sheetOpen && designId && (
        <Dialog
          title={t("changeSheet")}
          onClose={() => {
            if (sheetBusy) return;
            setSheetOpen(false);
          }}
          footer={
            <>
              <Button variant="secondary" onClick={() => setSheetOpen(false)} disabled={sheetBusy}>
                {t("cancel")}
              </Button>
              <Button
                onClick={() => void changeSheet()}
                disabled={!pickedSheet || pickedSheet === data.sheet_id}
                loading={sheetBusy}
              >
                {sheetBusy ? t("saving") : t("save")}
              </Button>
            </>
          }
        >
          {sheetError && (
            <div className="mb-3">
              <ErrorBanner message={sheetError} />
            </div>
          )}
          <SheetPicker designId={designId} value={pickedSheet} onChange={setPickedSheet} />
        </Dialog>
      )}

      {publishOpen && (
        <Dialog
          title={t("animatePublishTitle")}
          onClose={() => {
            if (publishBusy) return;
            setPublishOpen(false);
            setPublishError(null);
          }}
          footer={
            <>
              <Button
                variant="secondary"
                onClick={() => {
                  setPublishOpen(false);
                  setPublishError(null);
                }}
                disabled={publishBusy}
              >
                {t("cancel")}
              </Button>
              <Button icon="sparkle" onClick={() => void publishAnimations()} loading={publishBusy}>
                {publishBusy ? t("saving") : t("animatePublish")}
              </Button>
            </>
          }
        >
          <div className="space-y-3 text-sm text-ink-2">
            <p>{t("animatePublishHint")}</p>
            {changes.length === 0 ? (
              <p>{t("animatePublishEmpty")}</p>
            ) : (
              <ul className="space-y-1">
                {changes.map((change) => (
                  <li key={change.index} className="flex min-w-0 flex-wrap items-center gap-2">
                    <span className="mono truncate text-xs text-ink-2">{change.filename}</span>
                    <span className="text-ink">
                      {t(change.kind === "animated" ? "animateChangeAnimated" : "animateChangeStatic")}
                    </span>
                  </li>
                ))}
              </ul>
            )}
            {publishError && <ErrorBanner message={publishError} />}
          </div>
        </Dialog>
      )}

      {confirm && (
        <ConfirmDialog
          title={t(confirm === "revoke" ? "pkRevokeTitle" : "pkDeleteTitle")}
          body={t(confirm === "revoke" ? "revokeConfirm" : "deletePackConfirm")}
          confirmLabel={t(confirm === "revoke" ? "revoke" : "deletePack")}
          onConfirm={() => void runConfirm()}
          onClose={() => {
            if (confirmBusy) return;
            setConfirm(null);
            setConfirmError(null);
          }}
          busy={confirmBusy}
          error={confirmError}
        />
      )}
    </>
  );
}

function DetailPage() {
  const { t } = useI18n();
  const params = useSearchParams();
  const id = params.get("id");
  const jobId = params.get("job");

  return (
    <RequireAuth>
      <AppShell>
        {id ? (
          <PackDetailView key={id} id={id} initialJobId={jobId} />
        ) : (
          <>
            <PageHeader title={t("detailTitle")} back={{ href: "/packs", label: t("openPackList") }} />
            <EmptyState title={t("detailMissingId")} icon="pack">
              <Link href="/packs" className="inline-flex min-h-9 items-center text-sm font-medium text-cut-ink hover:underline">
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
      <DetailPage />
    </Suspense>
  );
}
