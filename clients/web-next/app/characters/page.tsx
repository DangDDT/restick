"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import {
  ArtFrame,
  Button,
  ConfirmDialog,
  Dialog,
  EmptyState,
  ErrorBanner,
  Field,
  PageHeader,
  SkeletonRows,
  SuccessBanner,
  WarnBanner,
  inputClass,
} from "@/components/ui";
import { api, errorText } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import type {
  Character,
  CharacterDraft,
  CharacterList,
  ReferenceAsset,
} from "@/lib/types";

function ReferencePreview({ assetId, className }: { assetId: string; className?: string }) {
  const { t } = useI18n();
  return (
    <ArtFrame className={className}>
      {/* The private route answers only with the session cookie; no key in the URL. */}
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src={`/api/v1/admin/reference-assets/${assetId}`}
        alt={t("referenceImage")}
        className="h-full w-full object-contain"
      />
    </ArtFrame>
  );
}

function CharacterForm({
  character,
  onClose,
  onSaved,
}: {
  character: Character | null;
  onClose: () => void;
  onSaved: (message: string) => void;
}) {
  const { t } = useI18n();
  const qc = useQueryClient();
  const fileInput = useRef<HTMLInputElement>(null);
  const [name, setName] = useState(character?.name ?? "");
  const [description, setDescription] = useState(character?.description ?? "");
  const [identityNotes, setIdentityNotes] = useState(character?.identity_notes ?? "");
  const [rightsNotes, setRightsNotes] = useState(character?.rights_notes ?? "");
  const [assetId, setAssetId] = useState(character?.reference_asset_id ?? "");
  const [asset, setAsset] = useState<ReferenceAsset | null>(null);
  const [nameError, setNameError] = useState<string | undefined>();
  const [error, setError] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);

  const save = useMutation({
    mutationFn: (body: CharacterDraft) =>
      character
        ? api<Character>(`/api/v1/admin/characters/${character.id}`, { method: "PUT", json: body })
        : api<Character>("/api/v1/admin/characters", { method: "POST", json: body }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["characters"] });
      onSaved(t("characterSaved"));
      onClose();
    },
    onError: (err) => setError(errorText(err)),
  });

  async function upload(file: File | undefined) {
    if (!file) return;
    setUploading(true);
    setError(null);
    try {
      const form = new FormData();
      form.append("file", file);
      const created = await api<ReferenceAsset>("/api/v1/admin/reference-assets", { form });
      setAsset(created);
      setAssetId(created.id);
      if (fileInput.current) fileInput.current.value = "";
    } catch (err) {
      setError(errorText(err));
    } finally {
      setUploading(false);
    }
  }

  function submit(event: React.FormEvent) {
    event.preventDefault();
    if (!name.trim()) {
      setNameError(t("nameRequired"));
      return;
    }
    setNameError(undefined);
    setError(null);
    save.mutate({
      name: name.trim(),
      description: description.trim(),
      identity_notes: identityNotes.trim(),
      rights_notes: rightsNotes.trim(),
      reference_asset_id: assetId,
    });
  }

  return (
    <Dialog title={character ? t("edit") : t("newCharacter")} onClose={onClose} variant="drawer">
      <form onSubmit={submit} className="space-y-4" noValidate>
        {error && <ErrorBanner message={error} />}

        <Field label={t("characterName")} error={nameError}>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            aria-invalid={Boolean(nameError) || undefined}
            className={inputClass}
          />
        </Field>
        <Field label={t("characterDescription")}>
          <input value={description} onChange={(e) => setDescription(e.target.value)} className={inputClass} />
        </Field>
        <Field label={t("identityNotes")}>
          <textarea
            value={identityNotes}
            onChange={(e) => setIdentityNotes(e.target.value)}
            rows={3}
            className={inputClass}
          />
        </Field>
        <Field label={t("rightsNotes")} hint={t("rightsHint")}>
          <textarea
            value={rightsNotes}
            onChange={(e) => setRightsNotes(e.target.value)}
            rows={2}
            className={inputClass}
          />
        </Field>

        <fieldset className="space-y-3 border-t border-line pt-4">
          <legend className="mb-2 text-[13px] font-medium text-ink">{t("referenceImage")}</legend>
          <WarnBanner message={t("referenceHint")} />
          <div className="flex flex-col gap-4 sm:flex-row sm:items-start">
            {assetId ? (
              <ReferencePreview assetId={assetId} className="size-40 shrink-0" />
            ) : (
              <ArtFrame className="size-40 shrink-0">
                <span className="px-3 text-center text-xs text-ink-2">{t("referenceMissing")}</span>
              </ArtFrame>
            )}
            <div className="min-w-0 flex-1 space-y-2">
              <span className="block text-[13px] text-ink-2">{t("referenceUpload")}</span>
              <input
                ref={fileInput}
                type="file"
                accept="image/png,image/jpeg,image/webp"
                onChange={(e) => void upload(e.target.files?.[0])}
                aria-label={t("referenceUpload")}
                disabled={uploading}
                className="block w-full text-sm text-ink-2 file:mr-3 file:min-h-9 file:cursor-pointer file:rounded-ctl file:border file:border-line-strong file:bg-panel file:px-3 file:text-sm file:font-medium file:text-ink hover:file:bg-sunk disabled:opacity-45 max-md:file:min-h-11"
              />
              <p role="status" aria-live="polite" className="text-xs text-ink-2">
                {uploading
                  ? t("uploading")
                  : asset
                    ? `${asset.filename} · ${asset.width}×${asset.height}`
                    : ""}
              </p>
              {asset && <p className="text-xs font-medium text-ok">{t("referenceUploaded")}</p>}
            </div>
          </div>
        </fieldset>

        <div className="flex flex-wrap justify-end gap-2 border-t border-line pt-4">
          <Button variant="secondary" onClick={onClose} disabled={save.isPending}>
            {t("cancel")}
          </Button>
          <Button type="submit" loading={save.isPending} disabled={uploading}>
            {save.isPending ? t("saving") : t("save")}
          </Button>
        </div>
      </form>
    </Dialog>
  );
}

function Characters() {
  const { t } = useI18n();
  const qc = useQueryClient();
  const [form, setForm] = useState<{ open: boolean; character: Character | null }>({
    open: false,
    character: null,
  });
  const [archiveTarget, setArchiveTarget] = useState<Character | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  const characters = useQuery({
    queryKey: ["characters"],
    queryFn: () => api<CharacterList>("/api/v1/admin/characters"),
  });

  const archive = useMutation({
    mutationFn: (id: string) => api<void>(`/api/v1/admin/characters/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      setArchiveTarget(null);
      setMessage(t("characterArchived"));
      qc.invalidateQueries({ queryKey: ["characters"] });
    },
  });

  function closeArchive() {
    archive.reset();
    setArchiveTarget(null);
  }

  return (
    <>
      <PageHeader
        title={t("charactersTitle")}
        subtitle={t("charactersSubtitle")}
        actions={
          <Button icon="plus" onClick={() => setForm({ open: true, character: null })}>
            {t("newCharacter")}
          </Button>
        }
      />

      {message && (
        <div className="mb-5">
          <SuccessBanner message={message} onDismiss={() => setMessage(null)} />
        </div>
      )}

      {form.open && (
        <CharacterForm
          key={form.character?.id ?? "create-character"}
          character={form.character}
          onClose={() => setForm({ open: false, character: null })}
          onSaved={setMessage}
        />
      )}

      {characters.isLoading ? (
        <SkeletonRows rows={3} />
      ) : characters.isError ? (
        <div className="space-y-3">
          <ErrorBanner message={errorText(characters.error)} />
          <Button variant="secondary" icon="refresh" onClick={() => characters.refetch()}>
            {t("retry")}
          </Button>
        </div>
      ) : characters.data && characters.data.items.length > 0 ? (
        <section className="space-y-4">
          <WarnBanner message={t("referenceHint")} />
          <ul aria-label={t("charactersTitle")} className="divide-y divide-line border-y border-line">
            {characters.data.items.map((character) => (
              <li key={character.id} className="flex flex-col gap-4 py-4 sm:flex-row sm:items-start">
                {character.reference_asset_id ? (
                  <ReferencePreview assetId={character.reference_asset_id} className="size-20 shrink-0" />
                ) : (
                  <ArtFrame className="size-20 shrink-0">
                    <span className="px-2 text-center text-[11px] leading-4 text-ink-2">{t("referenceMissing")}</span>
                  </ArtFrame>
                )}
                <div className="min-w-0 flex-1 space-y-1.5">
                  <div className="flex min-w-0 flex-wrap items-baseline gap-x-3 gap-y-1">
                    <h3 className="min-w-0 break-words text-sm font-semibold text-ink">{character.name}</h3>
                    <span className="mono text-[11px] text-ink-3">
                      {t("updatedAt")} {formatDate(character.updated_at)}
                    </span>
                  </div>
                  <p className="break-words text-sm text-ink-2">{character.description || t("none")}</p>
                  {character.identity_notes && (
                    <p className="break-words text-xs text-ink-2">
                      <span className="font-medium text-ink">{t("identityNotes")}: </span>
                      {character.identity_notes}
                    </p>
                  )}
                  {character.rights_notes && (
                    <p className="break-words text-xs text-ink-2">
                      <span className="font-medium text-ink">{t("rightsNotes")}: </span>
                      {character.rights_notes}
                    </p>
                  )}
                  <div className="flex flex-wrap gap-2 pt-2">
                    <Button
                      variant="secondary"
                      icon="edit"
                      onClick={() => setForm({ open: true, character })}
                    >
                      {t("edit")}
                    </Button>
                    <Button
                      variant="danger"
                      onClick={() => {
                        archive.reset();
                        setArchiveTarget(character);
                      }}
                    >
                      {t("archive")}
                    </Button>
                  </div>
                </div>
              </li>
            ))}
          </ul>
        </section>
      ) : (
        <EmptyState title={t("noCharacters")} icon="character">
          <Button icon="plus" onClick={() => setForm({ open: true, character: null })}>
            {t("newCharacter")}
          </Button>
        </EmptyState>
      )}

      {archiveTarget && (
        <ConfirmDialog
          title={t("archive")}
          body={t("characterArchiveConfirm")}
          confirmLabel={t("archive")}
          busy={archive.isPending}
          error={archive.isError ? errorText(archive.error) : null}
          onConfirm={() => archive.mutate(archiveTarget.id)}
          onClose={closeArchive}
        />
      )}
    </>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <AppShell>
        <Characters />
      </AppShell>
    </RequireAuth>
  );
}
