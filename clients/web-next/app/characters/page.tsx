"use client";

import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import { ReferencePreview } from "@/components/reference-preview";
import {
  ArtFrame,
  Button,
  ConfirmDialog,
  CopyButton,
  Dialog,
  EmptyState,
  ErrorBanner,
  Field,
  IconButton,
  PageHeader,
  SkeletonRows,
  SuccessBanner,
  WarnBanner,
  cx,
  inputClass,
  selectClass,
} from "@/components/ui";
import { api, errorText } from "@/lib/api";
import { buildCharacterPrompt } from "@/lib/character-prompt";
import { formatDate } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import type {
  Character,
  CharacterDraft,
  CharacterList,
  FragmentPage,
  ReferenceAsset,
} from "@/lib/types";

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
  const promptRef = useRef<HTMLTextAreaElement>(null);
  const [name, setName] = useState(character?.name ?? "");
  const [description, setDescription] = useState(character?.description ?? "");
  const [identityNotes, setIdentityNotes] = useState(character?.identity_notes ?? "");
  const [rightsNotes, setRightsNotes] = useState(character?.rights_notes ?? "");
  const [assetIds, setAssetIds] = useState<string[]>(character?.reference_asset_ids ?? []);
  const [uploadErrors, setUploadErrors] = useState<string[]>([]);
  const [nameError, setNameError] = useState<string | undefined>();
  const [error, setError] = useState<string | null>(null);
  const [uploading, setUploading] = useState<{ done: number; total: number } | null>(null);
  const [identityFragmentId, setIdentityFragmentId] = useState("");
  const identityFragments = useInfiniteQuery({
    queryKey: ["character-identity-fragments"],
    initialPageParam: 1,
    queryFn: ({ pageParam }) => api<FragmentPage>(
      `/api/v1/admin/fragments?scope=identity&page_size=100&page=${pageParam}`,
    ),
    getNextPageParam: (lastPage) =>
      lastPage.page * lastPage.page_size < lastPage.total ? lastPage.page + 1 : undefined,
  });
  const identityChoices = identityFragments.data?.pages.flatMap((page) => page.items) ?? [];
  const prompt = buildCharacterPrompt({ name, description, identity_notes: identityNotes });

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

  // Sequential: successes stay in the gallery even when a later file fails.
  async function upload(files: FileList | null) {
    const selected = files ? Array.from(files) : [];
    if (selected.length === 0) return;
    setUploading({ done: 0, total: selected.length });
    setUploadErrors([]);
    setError(null);
    const failed: string[] = [];
    for (const [index, file] of selected.entries()) {
      try {
        const form = new FormData();
        form.append("file", file);
        const created = await api<ReferenceAsset>("/api/v1/admin/reference-assets", { form });
        setAssetIds((current) => (current.includes(created.id) ? current : [...current, created.id]));
      } catch (err) {
        failed.push(`${file.name} (${errorText(err)})`);
      }
      setUploading({ done: index + 1, total: selected.length });
    }
    setUploading(null);
    setUploadErrors(failed);
    if (fileInput.current) fileInput.current.value = "";
  }

  function submit(event: React.FormEvent) {
    event.preventDefault();
    if (uploading) return;
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
      reference_asset_ids: assetIds,
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
        <div className="space-y-2">
          <Field label={t("identityFragmentSource")} hint={t("identitySnapshotHint")}>
            <select
              value={identityFragmentId}
              onChange={(event) => setIdentityFragmentId(event.target.value)}
              disabled={identityFragments.isPending}
              className={selectClass}
            >
              <option value="">{identityFragments.isPending ? t("loading") : t("chooseFragment")}</option>
              {identityChoices.map((fragment) => (
                <option key={fragment.id} value={fragment.id}>{fragment.name}</option>
              ))}
            </select>
          </Field>
          <div className="flex flex-wrap gap-2">
            <Button
              variant="secondary"
              disabled={!identityFragmentId}
              onClick={() => {
                const fragment = identityChoices.find((item) => item.id === identityFragmentId);
                if (!fragment) return;
                setIdentityNotes((current) => [current.trim(), fragment.prompt].filter(Boolean).join("\n"));
                setIdentityFragmentId("");
              }}
            >
              {t("identityImport")}
            </Button>
            {identityFragments.hasNextPage && (
              <Button
                variant="secondary"
                loading={identityFragments.isFetchingNextPage}
                onClick={() => void identityFragments.fetchNextPage()}
              >
                {t("identityLoadMore")}
              </Button>
            )}
            {identityFragments.isError && (
              <Button variant="secondary" onClick={() => void identityFragments.refetch()}>{t("retry")}</Button>
            )}
          </div>
          {identityFragments.isError && <ErrorBanner message={errorText(identityFragments.error)} />}
          {!identityFragments.isPending && !identityFragments.isError && identityChoices.length === 0 && (
            <p className="text-sm text-ink-2">{t("noFragments")}</p>
          )}
        </div>
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
          <legend className="mb-2 text-[13px] font-medium text-ink">
            {t("referenceImage")}
            <span className="mono ml-2 text-[11px] font-normal text-ink-3">
              {assetIds.length} {t("referenceImageCount")}
            </span>
          </legend>
          <WarnBanner message={t("referenceHint")} />
          <p className="text-xs text-ink-2">{t("referenceGalleryHint")}</p>
          {assetIds.length > 0 ? (
            <ul aria-label={t("referenceGallery")} className="flex flex-wrap gap-3">
              {assetIds.map((id, index) => (
                <li key={id} className="relative">
                  <a
                    href={`/api/v1/admin/reference-assets/${id}`}
                    target="_blank"
                    rel="noreferrer"
                    aria-label={`${t("referenceView")} ${index + 1}`}
                    className="block rounded-[6px]"
                  >
                    <ReferencePreview
                      assetId={id}
                      alt={`${t("referenceImage")} ${index + 1}`}
                      className="size-28"
                    />
                  </a>
                  <IconButton
                    icon="close"
                    label={`${t("referenceRemove")} ${index + 1}`}
                    variant="secondary"
                    onClick={() => setAssetIds((current) => current.filter((item) => item !== id))}
                    className="absolute -right-2 -top-2 shadow"
                  />
                </li>
              ))}
            </ul>
          ) : (
            <ArtFrame className="size-28">
              <span className="px-3 text-center text-xs text-ink-2">{t("referenceMissing")}</span>
            </ArtFrame>
          )}
          <span className="block text-[13px] text-ink-2">{t("referenceUpload")}</span>
          <input
            ref={fileInput}
            type="file"
            accept="image/png,image/jpeg,image/webp"
            multiple
            onChange={(e) => void upload(e.target.files)}
            aria-label={t("referenceUpload")}
            disabled={uploading !== null}
            className="block w-full text-sm text-ink-2 file:mr-3 file:min-h-9 file:cursor-pointer file:rounded-ctl file:border file:border-line-strong file:bg-panel file:px-3 file:text-sm file:font-medium file:text-ink hover:file:bg-sunk disabled:opacity-45 max-md:file:min-h-11"
          />
          <p role="status" aria-live="polite" className="text-xs text-ink-2">
            {uploading ? `${t("uploading")} ${uploading.done}/${uploading.total}` : ""}
          </p>
          {uploadErrors.length > 0 && (
            <ErrorBanner message={`${t("referenceUploadFailed")} ${uploadErrors.join("; ")}`} />
          )}
        </fieldset>

        <fieldset className="space-y-3 border-t border-line pt-4">
          <legend className="mb-2 text-[13px] font-medium text-ink">{t("characterPrompt")}</legend>
          <p className="text-xs text-ink-2">{t("characterPromptHint")}</p>
          <textarea
            ref={promptRef}
            readOnly
            value={prompt}
            rows={10}
            aria-label={t("characterPrompt")}
            className={cx(inputClass, "mono text-xs leading-5")}
          />
          <CopyButton text={prompt} selectRef={promptRef} label={t("copyCharacterPrompt")} />
        </fieldset>

        <div className="flex flex-wrap justify-end gap-2 border-t border-line pt-4">
          <Button variant="secondary" onClick={onClose} disabled={save.isPending}>
            {t("cancel")}
          </Button>
          <Button type="submit" loading={save.isPending} disabled={uploading !== null}>
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
                {character.reference_asset_ids.length > 0 ? (
                  <ReferencePreview assetId={character.reference_asset_ids[0]} className="size-20 shrink-0" />
                ) : (
                  <ArtFrame className="size-20 shrink-0">
                    <span className="px-2 text-center text-[11px] leading-4 text-ink-2">{t("referenceMissing")}</span>
                  </ArtFrame>
                )}
                <div className="min-w-0 flex-1 space-y-1.5">
                  <div className="flex min-w-0 flex-wrap items-baseline gap-x-3 gap-y-1">
                    <h3 className="min-w-0 break-words text-sm font-semibold text-ink">{character.name}</h3>
                    <span className="mono text-[11px] text-ink-3">
                      {character.reference_asset_ids.length} {t("referenceImageCount")}
                    </span>
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
