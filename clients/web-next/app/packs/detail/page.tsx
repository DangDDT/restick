"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Suspense, useEffect, useRef, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import {
  Button,
  EmptyState,
  ErrorBanner,
  Field,
  PageHeader,
  Spinner,
  StatusPill,
  SuccessBanner,
  inputClass,
} from "@/components/ui";
import { api, errorText } from "@/lib/api";
import { formatBytes, formatDate } from "@/lib/format";
import { t } from "@/lib/i18n";
import type { AdminPack, AdminPackDetail, Job, JobStatus, PackUpdate } from "@/lib/types";

const POLL_MS = 3000;
// Job timeout is 300 s server-side (RESTICK_PROCESS_TIMEOUT); stop the browser
// poll a little after that instead of polling a terminal-or-abandoned job forever.
const POLL_LIMIT_MS = 330_000;

function isTerminal(status: JobStatus): boolean {
  return status === "succeeded" || status === "failed" || status === "interrupted";
}

function CatalogForm({ pack, onDone }: { pack: AdminPackDetail; onDone: (message: string | null) => void }) {
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
    <form onSubmit={submit} className="space-y-3 rounded-xl border border-zinc-200 bg-white p-4">
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
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={2}
          className={inputClass}
        />
      </Field>
      <Field label={t("packTags")}>
        <input value={tags} onChange={(e) => setTags(e.target.value)} className={inputClass} />
      </Field>
      <p className="text-xs text-zinc-500">{t("catalogHint")}</p>
      <div className="flex justify-end">
        <Button type="submit" disabled={busy}>
          {busy ? t("saving") : t("saveCatalog")}
        </Button>
      </div>
    </form>
  );
}

function PackDetailView({ id, initialJobId }: { id: string; initialJobId: string | null }) {
  const router = useRouter();
  const qc = useQueryClient();
  const fileInput = useRef<HTMLInputElement>(null);
  const [editing, setEditing] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [starting, setStarting] = useState(false);
  const [jobId, setJobId] = useState<string | null>(initialJobId);
  const [pollExpired, setPollExpired] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const pack = useQuery({
    queryKey: ["admin-pack", id],
    queryFn: () => api<AdminPackDetail>(`/api/v1/admin/packs/${id}`),
  });

  const job = useQuery({
    queryKey: ["job", jobId],
    queryFn: () => api<Job>(`/api/v1/admin/jobs/${jobId}`),
    enabled: Boolean(jobId),
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

  async function uploadSource() {
    if (!file) return;
    setUploading(true);
    setError(null);
    setMessage(null);
    try {
      const form = new FormData();
      form.append("file", file);
      await api<AdminPack>(`/api/v1/admin/packs/${id}/source`, { form });
      if (fileInput.current) fileInput.current.value = "";
      setFile(null);
      setMessage(t("sourceUploaded"));
      qc.invalidateQueries({ queryKey: ["admin-pack", id] });
      qc.invalidateQueries({ queryKey: ["admin-packs"] });
    } catch (err) {
      setError(errorText(err));
    } finally {
      setUploading(false);
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
      // Keep the job id in the URL so a reload resumes polling (there is no
      // "list jobs for pack" endpoint).
      router.replace(`/packs/detail?id=${encodeURIComponent(id)}&job=${encodeURIComponent(created.id)}`);
      setMessage(t("processStarted"));
      qc.invalidateQueries({ queryKey: ["admin-pack", id] });
    } catch (err) {
      setError(errorText(err));
    } finally {
      setStarting(false);
    }
  }

  async function revoke() {
    if (!window.confirm(t("revokeConfirm"))) return;
    setError(null);
    setMessage(null);
    try {
      await api<AdminPack>(`/api/v1/admin/packs/${id}/revoke`, { method: "POST" });
      setMessage(t("revokedDone"));
      qc.invalidateQueries({ queryKey: ["admin-pack", id] });
      qc.invalidateQueries({ queryKey: ["admin-packs"] });
    } catch (err) {
      setError(errorText(err));
    }
  }

  async function deletePack() {
    if (!window.confirm(t("deletePackConfirm"))) return;
    setError(null);
    try {
      await api<void>(`/api/v1/admin/packs/${id}`, { method: "DELETE" });
      qc.invalidateQueries({ queryKey: ["admin-packs"] });
      router.replace("/packs");
    } catch (err) {
      setError(errorText(err));
    }
  }

  if (pack.isLoading) return <Spinner />;
  if (pack.isError || !pack.data) {
    return (
      <>
        <PageHeader title={t("detailTitle")} />
        <ErrorBanner message={pack.error ? errorText(pack.error) : t("loadFailed")} />
        <div className="mt-4">
          <Link href="/packs" className="text-sm font-medium text-indigo-600 hover:underline">
            {t("openPackList")}
          </Link>
        </div>
      </>
    );
  }

  const data = pack.data;
  const jobRunning = job.data ? !isTerminal(job.data.status) : false;
  const pollStopped = pollExpired && Boolean(job.data && !isTerminal(job.data.status));

  return (
    <>
      <PageHeader
        title={data.name}
        subtitle={`${data.slug} · ${data.author} · ${t("currentVersion")} ${data.current_version}`}
        actions={
          <>
            <StatusPill status={data.status} />
            <Link
              href="/packs"
              className="rounded-lg border border-zinc-300 bg-white px-4 py-2 text-sm font-medium text-zinc-700 hover:bg-zinc-50"
            >
              {t("openPackList")}
            </Link>
          </>
        }
      />

      {message && (
        <div className="mb-4">
          <SuccessBanner message={message} />
        </div>
      )}
      {error && (
        <div className="mb-4">
          <ErrorBanner message={error} />
        </div>
      )}

      <div className="grid gap-5 lg:grid-cols-2">
        <section className="space-y-3 rounded-xl border border-zinc-200 bg-white p-4">
          <div className="flex items-start justify-between gap-2">
            <h2 className="text-sm font-semibold text-zinc-700">{t("editCatalog")}</h2>
            <Button variant="secondary" onClick={() => setEditing((value) => !value)}>
              {editing ? t("cancel") : t("editCatalog")}
            </Button>
          </div>
          {editing ? (
            <CatalogForm pack={data} onDone={(next) => { setEditing(false); setMessage(next); }} />
          ) : (
            <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
              <dt className="text-zinc-500">{t("packDescription")}</dt>
              <dd className="text-zinc-800">{data.description || t("none")}</dd>
              <dt className="text-zinc-500">{t("packTags")}</dt>
              <dd className="text-zinc-800">{(data.tags ?? []).length ? (data.tags ?? []).join(", ") : t("none")}</dd>
              <dt className="text-zinc-500">{t("designLabel")}</dt>
              <dd className="break-all text-zinc-800">{data.design_id || t("none")}</dd>
              <dt className="text-zinc-500">{t("createdAt")}</dt>
              <dd className="text-zinc-800">{formatDate(data.created_at)}</dd>
              <dt className="text-zinc-500">{t("updatedAt")}</dt>
              <dd className="text-zinc-800">{formatDate(data.updated_at)}</dd>
              <dt className="text-zinc-500">{t("checksum")}</dt>
              <dd className="break-all text-xs text-zinc-600">{data.checksum_sha256 || t("none")}</dd>
            </dl>
          )}
        </section>

        <section className="space-y-3 rounded-xl border border-zinc-200 bg-white p-4">
          <h2 className="text-sm font-semibold text-zinc-700">{t("sourceLabel")}</h2>
          {data.source ? (
            <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
              <dt className="text-zinc-500">{t("sourceLabel")}</dt>
              <dd className="break-all text-zinc-800">{data.source.filename}</dd>
              <dt className="text-zinc-500">{t("dimensions")}</dt>
              <dd className="text-zinc-800">
                {data.source.width}×{data.source.height} · {formatBytes(data.source.byte_size)}
              </dd>
              <dt className="text-zinc-500">{t("checksum")}</dt>
              <dd className="break-all text-xs text-zinc-600">{data.source.checksum_sha256}</dd>
            </dl>
          ) : (
            <p className="text-sm text-zinc-500">{t("noSource")}</p>
          )}
          <Field label={t("uploadSource")} hint={t("uploadSourceHint")}>
            <input
              ref={fileInput}
              type="file"
              accept="image/png,image/jpeg,image/webp"
              onChange={(event) => setFile(event.target.files?.[0] ?? null)}
              className="block w-full text-sm text-zinc-600 file:mr-3 file:rounded-lg file:border-0 file:bg-indigo-50 file:px-3 file:py-2 file:text-sm file:font-medium file:text-indigo-700 hover:file:bg-indigo-100"
            />
          </Field>
          <div className="flex flex-wrap gap-2">
            <Button variant="secondary" onClick={() => void uploadSource()} disabled={!file || uploading}>
              {uploading ? t("uploading") : t("uploadSource")}
            </Button>
            <Button
              onClick={() => void startProcessing()}
              disabled={starting || jobRunning || !data.source || data.status === "revoked"}
            >
              {starting ? t("processing") : t("process")}
            </Button>
          </div>
          <p className="text-xs text-zinc-500">{t("processHint")}</p>
          {data.status === "revoked" && <p className="text-xs text-zinc-500">{t("revokedHint")}</p>}
        </section>
      </div>

      {jobId && (
        <section className="mt-5 rounded-xl border border-zinc-200 bg-white p-4">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="text-sm font-semibold text-zinc-700">{t("jobLabel")}</h2>
            {job.data && <StatusPill status={job.data.status} />}
            {job.data && (
              <span className="text-xs text-zinc-500">
                {t("jobVersion")} {job.data.version} · {formatDate(job.data.created_at)}
              </span>
            )}
          </div>
          {job.isLoading && <Spinner />}
          {job.isError && <ErrorBanner message={errorText(job.error)} />}
          {job.data && (
            <div className="mt-2 space-y-1 text-sm">
              {job.data.status === "queued" || job.data.status === "running" ? (
                <p className="text-zinc-600">{t("jobNoProgress")}</p>
              ) : null}
              {job.data.error_code && (
                <p className="text-red-700">
                  {t("jobError")}: <code>{job.data.error_code}</code>
                  {job.data.error_message ? ` — ${job.data.error_message}` : ""}
                </p>
              )}
              {job.data.diagnostics && (
                <div>
                  <p className="text-xs font-medium text-zinc-600">{t("jobDiagnostics")}</p>
                  <pre className="mt-1 max-h-40 overflow-auto rounded bg-zinc-50 p-2 text-xs text-zinc-700">
                    {job.data.diagnostics}
                  </pre>
                </div>
              )}
            </div>
          )}
          {pollStopped && <p className="mt-2 text-xs text-amber-700">{t("jobPollingStopped")}</p>}
        </section>
      )}

      <section className="mt-5">
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
          <h2 className="text-sm font-semibold text-zinc-700">{t("stickersTitle")}</h2>
          {data.bundle_url && (
            <a
              href={data.bundle_url}
              className="text-sm font-medium text-indigo-600 hover:underline"
            >
              {t("downloadBundle")}
            </a>
          )}
        </div>
        {data.stickers.length === 0 ? (
          <EmptyState title={t("noStickers")} />
        ) : (
          <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 xl:grid-cols-6">
            {data.stickers.map((sticker) => (
              <li
                key={sticker.filename}
                className="overflow-hidden rounded-xl border border-zinc-200 bg-white"
              >
                <a href={sticker.object_url} className="block">
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={sticker.thumb_url || sticker.object_url}
                    alt={sticker.filename}
                    loading="lazy"
                    className="aspect-square w-full bg-zinc-50 object-contain"
                  />
                </a>
                <div className="space-y-0.5 p-2 text-xs text-zinc-600">
                  <p className="truncate font-medium text-zinc-800">{sticker.filename}</p>
                  <p>
                    {sticker.width}×{sticker.height}
                  </p>
                  {sticker.emoji_hint && <p className="truncate">{sticker.emoji_hint}</p>}
                  {(sticker.keywords ?? []).length > 0 && (
                    <p className="truncate text-zinc-400">{(sticker.keywords ?? []).join(", ")}</p>
                  )}
                  <a
                    href={sticker.object_url}
                    download
                    className="inline-block font-medium text-indigo-600 hover:underline"
                  >
                    {t("downloadFile")}
                  </a>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      <div className="mt-6 flex flex-wrap gap-2 border-t border-zinc-200 pt-5">
        {data.status === "draft" && (
          <Button variant="danger" onClick={() => void deletePack()}>
            {t("deletePack")}
          </Button>
        )}
        {data.status === "ready" && (
          <Button variant="danger" onClick={() => void revoke()}>
            {t("revoke")}
          </Button>
        )}
      </div>
    </>
  );
}

function DetailPage() {
  const params = useSearchParams();
  const id = params.get("id");
  const jobId = params.get("job");

  return (
    <RequireAuth>
      <AppShell>
        {id ? (
          <PackDetailView id={id} initialJobId={jobId} />
        ) : (
          <>
            <PageHeader title={t("detailTitle")} />
            <EmptyState title={t("detailMissingId")} />
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
