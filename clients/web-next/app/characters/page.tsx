"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import {
  Button,
  EmptyState,
  ErrorBanner,
  Field,
  PageHeader,
  Spinner,
  SuccessBanner,
  inputClass,
} from "@/components/ui";
import { api, errorText } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { t } from "@/lib/i18n";
import type {
  Character,
  CharacterDraft,
  CharacterList,
  ReferenceAsset,
} from "@/lib/types";

function ReferencePreview({ assetId }: { assetId: string }) {
  return (
    // The private route answers only with the session cookie; no key in the URL.
    // eslint-disable-next-line @next/next/no-img-element
    <img
      src={`/api/v1/admin/reference-assets/${assetId}`}
      alt={t("referenceImage")}
      className="h-24 w-24 rounded-lg border border-zinc-200 object-cover"
    />
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
  const qc = useQueryClient();
  const fileInput = useRef<HTMLInputElement>(null);
  const [name, setName] = useState(character?.name ?? "");
  const [description, setDescription] = useState(character?.description ?? "");
  const [identityNotes, setIdentityNotes] = useState(character?.identity_notes ?? "");
  const [rightsNotes, setRightsNotes] = useState(character?.rights_notes ?? "");
  const [assetId, setAssetId] = useState(character?.reference_asset_id ?? "");
  const [asset, setAsset] = useState<ReferenceAsset | null>(null);
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
    if (!name.trim()) return setError(t("nameRequired"));
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
    <form onSubmit={submit} className="space-y-3 rounded-xl border border-zinc-200 bg-white p-4">
      <h3 className="text-sm font-semibold text-zinc-700">
        {character ? t("edit") : t("newCharacter")}
      </h3>
      {error && <ErrorBanner message={error} />}
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label={t("characterName")}>
          <input value={name} onChange={(e) => setName(e.target.value)} className={inputClass} />
        </Field>
        <Field label={t("characterDescription")}>
          <input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className={inputClass}
          />
        </Field>
      </div>
      <Field label={t("identityNotes")}>
        <textarea
          value={identityNotes}
          onChange={(e) => setIdentityNotes(e.target.value)}
          rows={2}
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

      <div className="rounded-lg border border-zinc-200 p-3">
        <p className="mb-2 text-sm font-medium text-zinc-700">{t("referenceImage")}</p>
        <div className="flex flex-wrap items-center gap-3">
          {assetId ? (
            <ReferencePreview assetId={assetId} />
          ) : (
            <span className="text-xs text-zinc-500">{t("referenceMissing")}</span>
          )}
          <div className="min-w-0">
            <input
              ref={fileInput}
              type="file"
              accept="image/png,image/jpeg,image/webp"
              onChange={(e) => void upload(e.target.files?.[0])}
              aria-label={t("referenceUpload")}
              className="block w-full text-sm text-zinc-600 file:mr-3 file:rounded-lg file:border-0 file:bg-indigo-50 file:px-3 file:py-2 file:text-sm file:font-medium file:text-indigo-700 hover:file:bg-indigo-100"
            />
            <p className="mt-1 text-xs text-zinc-500">
              {uploading ? t("uploading") : asset ? `${asset.filename} · ${asset.width}×${asset.height}` : t("referenceHint")}
            </p>
          </div>
        </div>
        {asset && <p className="mt-2 text-xs text-emerald-700">{t("referenceUploaded")}</p>}
      </div>

      <div className="flex justify-end gap-2">
        <Button variant="secondary" onClick={onClose}>
          {t("cancel")}
        </Button>
        <Button type="submit" disabled={save.isPending || uploading}>
          {save.isPending ? t("saving") : t("save")}
        </Button>
      </div>
    </form>
  );
}

function Characters() {
  const qc = useQueryClient();
  const [form, setForm] = useState<{ open: boolean; character: Character | null }>({
    open: false,
    character: null,
  });
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const characters = useQuery({
    queryKey: ["characters"],
    queryFn: () => api<CharacterList>("/api/v1/admin/characters"),
  });

  const archive = useMutation({
    mutationFn: (id: string) => api<void>(`/api/v1/admin/characters/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      setError(null);
      setMessage(t("characterArchived"));
      qc.invalidateQueries({ queryKey: ["characters"] });
    },
    onError: (err) => setError(errorText(err)),
  });

  return (
    <>
      <PageHeader
        title={t("charactersTitle")}
        subtitle={t("charactersSubtitle")}
        actions={
          <Button onClick={() => setForm({ open: true, character: null })}>
            {t("newCharacter")}
          </Button>
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

      {form.open && (
        <div className="mb-6">
          <CharacterForm
            character={form.character}
            onClose={() => setForm({ open: false, character: null })}
            onSaved={setMessage}
          />
        </div>
      )}

      {characters.isLoading ? (
        <Spinner />
      ) : characters.isError ? (
        <ErrorBanner message={errorText(characters.error)} />
      ) : characters.data && characters.data.items.length > 0 ? (
        <ul className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {characters.data.items.map((character) => (
            <li key={character.id} className="rounded-xl border border-zinc-200 bg-white p-4">
              <div className="flex items-start gap-3">
                {character.reference_asset_id ? (
                  <ReferencePreview assetId={character.reference_asset_id} />
                ) : (
                  <div className="flex h-24 w-24 items-center justify-center rounded-lg border border-dashed border-zinc-300 text-xs text-zinc-400">
                    {t("referenceMissing")}
                  </div>
                )}
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-semibold text-zinc-900">{character.name}</p>
                  <p className="mt-0.5 line-clamp-2 text-xs text-zinc-500">
                    {character.description || t("none")}
                  </p>
                  <p className="mt-1 text-[11px] text-zinc-400">
                    {t("updatedAt")} {formatDate(character.updated_at)}
                  </p>
                </div>
              </div>
              {character.identity_notes && (
                <p className="mt-3 text-xs text-zinc-600">
                  <span className="font-medium">{t("identityNotes")}: </span>
                  {character.identity_notes}
                </p>
              )}
              {character.rights_notes && (
                <p className="mt-1 text-xs text-zinc-600">
                  <span className="font-medium">{t("rightsNotes")}: </span>
                  {character.rights_notes}
                </p>
              )}
              <p className="mt-2 text-xs text-zinc-500">{t("referenceHint")}</p>
              <div className="mt-3 flex gap-2">
                <Button
                  variant="secondary"
                  onClick={() => setForm({ open: true, character })}
                >
                  {t("edit")}
                </Button>
                <Button
                  variant="danger"
                  onClick={() => {
                    if (window.confirm(t("characterArchiveConfirm"))) archive.mutate(character.id);
                  }}
                  disabled={archive.isPending}
                >
                  {t("archive")}
                </Button>
              </div>
            </li>
          ))}
        </ul>
      ) : (
        <EmptyState title={t("noCharacters")} />
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
