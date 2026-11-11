"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import {
  Button,
  ConfirmDialog,
  Dialog,
  EmptyState,
  ErrorBanner,
  Field,
  JobTag,
  PageHeader,
  Pager,
  Section,
  SkeletonRows,
  Stamp,
  Tabs,
  checkClass,
  inputClass,
  selectClass,
} from "@/components/ui";
import { api, errorText } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import type {
  Colors,
  Fragment,
  FragmentDraft,
  FragmentPage,
  Group,
  GroupDraft,
  GroupList,
  Scope,
} from "@/lib/types";

const SCOPES: Scope[] = ["identity", "pack", "sticker"];
const PAGE_SIZE = 20;
const HEX = /^#[0-9A-Fa-f]{6}$/;
const EMPTY_COLORS: Colors = {
  primary: "#FFFFFF",
  secondary: "#FFFFFF",
  accent: "#FFFFFF",
  line: "#000000",
};

const ROLES = [
  ["primary", "rolePrimary"],
  ["secondary", "roleSecondary"],
  ["accent", "roleAccent"],
  ["line", "roleLine"],
] as const;

const SCOPE_LABEL: Record<Scope, "scopeIdentity" | "scopePack" | "scopeSticker"> = {
  identity: "scopeIdentity",
  pack: "scopePack",
  sticker: "scopeSticker",
};

type FragmentErrors = { group?: string; name?: string; prompt?: string; colors?: string };

function ErrorRetry({ message, onRetry }: { message: string; onRetry: () => void }) {
  const { t } = useI18n();
  return (
    <div className="space-y-3">
      <ErrorBanner message={message} />
      <Button variant="secondary" icon="refresh" onClick={onRetry}>
        {t("retry")}
      </Button>
    </div>
  );
}

/* ---------- Groups ---------- */

function GroupForm({ group, onClose }: { group: Group | null; onClose: () => void }) {
  const { t } = useI18n();
  const qc = useQueryClient();
  const formRef = useRef<HTMLFormElement>(null);
  const [name, setName] = useState(group?.name ?? "");
  const [description, setDescription] = useState(group?.description ?? "");
  const [sortOrder, setSortOrder] = useState(group ? String(group.sort_order) : "0");
  const [nameError, setNameError] = useState<string | undefined>();
  const [error, setError] = useState<string | null>(null);

  const save = useMutation({
    mutationFn: (body: GroupDraft) =>
      group
        ? api<Group>(`/api/v1/admin/fragment-groups/${group.id}`, { method: "PUT", json: body })
        : api<Group>("/api/v1/admin/fragment-groups", { method: "POST", json: body }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["groups"] });
      qc.invalidateQueries({ queryKey: ["fragments"] });
      onClose();
    },
    onError: (err) => setError(errorText(err)),
  });

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
      sort_order: Number(sortOrder) || 0,
    });
  }

  return (
    <Dialog
      title={group ? t("edit") : t("newGroup")}
      onClose={onClose}
      variant="drawer"
    >
      <form ref={formRef} onSubmit={submit} className="space-y-4" noValidate>
        {error && <ErrorBanner message={error} />}
        <Field label={t("groupName")} error={nameError}>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            aria-invalid={Boolean(nameError) || undefined}
            className={inputClass}
          />
        </Field>
        <Field label={t("groupDescription")}>
          <input value={description} onChange={(e) => setDescription(e.target.value)} className={inputClass} />
        </Field>
        <Field label={t("groupSortOrder")}>
          <input
            type="number"
            inputMode="numeric"
            value={sortOrder}
            onChange={(e) => setSortOrder(e.target.value)}
            className={inputClass}
          />
        </Field>
        <div className="flex flex-wrap justify-end gap-2 border-t border-line pt-4">
          <Button variant="secondary" onClick={onClose} disabled={save.isPending}>
            {t("cancel")}
          </Button>
          <Button type="submit" loading={save.isPending}>
            {save.isPending ? t("saving") : t("save")}
          </Button>
        </div>
      </form>
    </Dialog>
  );
}

function GroupSection({ groups }: { groups: Group[] }) {
  const { t } = useI18n();
  const qc = useQueryClient();
  const [form, setForm] = useState<{ open: boolean; group: Group | null }>({ open: false, group: null });
  const [deleteTarget, setDeleteTarget] = useState<Group | null>(null);
  const deleteGroup = useMutation({
    mutationFn: (id: string) => api<void>(`/api/v1/admin/fragment-groups/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      setDeleteTarget(null);
      qc.invalidateQueries({ queryKey: ["groups"] });
    },
  });

  function closeDelete() {
    deleteGroup.reset();
    setDeleteTarget(null);
  }

  return (
    <Section
      title={t("tabGroups")}
      actions={
        <Button icon="plus" onClick={() => setForm({ open: true, group: null })}>
          {t("newGroup")}
        </Button>
      }
    >
      {form.open && (
        <GroupForm
          key={form.group?.id ?? "create-group"}
          group={form.group}
          onClose={() => setForm({ open: false, group: null })}
        />
      )}

      {groups.length > 0 ? (
        <ul aria-label={t("tabGroups")} className="divide-y divide-line border-y border-line">
          {groups.map((group) => (
            <li key={group.id} className="flex flex-col gap-3 py-3.5 sm:flex-row sm:items-center sm:justify-between">
              <div className="min-w-0">
                <h3 className="break-words text-sm font-semibold text-ink">{group.name}</h3>
                <p className="mt-0.5 break-words text-xs text-ink-2">
                  {group.description || t("none")} · {t("groupSortOrder")} <span className="mono">{group.sort_order}</span>
                </p>
              </div>
              <div className="flex shrink-0 flex-wrap gap-2">
                <Button variant="secondary" icon="edit" onClick={() => setForm({ open: true, group })}>
                  {t("edit")}
                </Button>
                <Button
                  variant="danger"
                  icon="trash"
                  onClick={() => {
                    deleteGroup.reset();
                    setDeleteTarget(group);
                  }}
                >
                  {t("delete")}
                </Button>
              </div>
            </li>
          ))}
        </ul>
      ) : (
        <EmptyState title={t("noGroups")} icon="library">
          <Button icon="plus" onClick={() => setForm({ open: true, group: null })}>
            {t("newGroup")}
          </Button>
        </EmptyState>
      )}

      {deleteTarget && (
        <ConfirmDialog
          title={t("delete")}
          body={t("groupDeleteConfirm")}
          confirmLabel={t("delete")}
          busy={deleteGroup.isPending}
          error={deleteGroup.isError ? errorText(deleteGroup.error) : null}
          onConfirm={() => deleteGroup.mutate(deleteTarget.id)}
          onClose={closeDelete}
        />
      )}
    </Section>
  );
}

/* ---------- Fragments ---------- */

function FragmentForm({
  groups,
  fragment,
  onClose,
}: {
  groups: Group[];
  fragment: Fragment | null;
  onClose: () => void;
}) {
  const { t } = useI18n();
  const qc = useQueryClient();
  const formRef = useRef<HTMLFormElement>(null);
  const [groupId, setGroupId] = useState(fragment?.group_id ?? groups[0]?.id ?? "");
  const [name, setName] = useState(fragment?.name ?? "");
  const [nameEn, setNameEn] = useState(fragment?.name_en ?? "");
  const [prompt, setPrompt] = useState(fragment?.prompt ?? "");
  const [notes, setNotes] = useState(fragment?.notes ?? "");
  const [source, setSource] = useState(fragment?.source ?? "");
  const [scope, setScope] = useState<Scope>(fragment?.scope ?? "pack");
  const [isPalette, setIsPalette] = useState(Boolean(fragment?.colors));
  const [colors, setColors] = useState<Colors>(fragment?.colors ?? EMPTY_COLORS);
  const [fieldErrors, setFieldErrors] = useState<FragmentErrors>({});
  const [error, setError] = useState<string | null>(null);

  const save = useMutation({
    mutationFn: (body: FragmentDraft) =>
      fragment
        ? api<Fragment>(`/api/v1/admin/fragments/${fragment.id}`, { method: "PUT", json: body })
        : api<Fragment>("/api/v1/admin/fragments", { method: "POST", json: body }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["fragments"] });
      onClose();
    },
    onError: (err) => setError(errorText(err)),
  });

  function submit(event: React.FormEvent) {
    event.preventDefault();
    const next: FragmentErrors = {};
    if (!groupId) next.group = t("groupRequired");
    if (!name.trim()) next.name = t("nameRequired");
    if (!prompt.trim()) next.prompt = t("promptRequired");
    if (isPalette && !Object.values(colors).every((value) => HEX.test(value))) {
      next.colors = t("colorsRequired");
    }
    setFieldErrors(next);
    if (Object.keys(next).length > 0) return;
    setError(null);
    save.mutate({
      group_id: groupId,
      name: name.trim(),
      name_en: nameEn.trim(),
      prompt: prompt.trim(),
      notes: notes.trim(),
      source: source.trim(),
      scope,
      colors: isPalette ? colors : undefined,
    });
  }

  return (
    <Dialog title={fragment ? t("edit") : t("newFragment")} onClose={onClose} variant="drawer">
      <form ref={formRef} onSubmit={submit} className="space-y-4" noValidate>
        {error && <ErrorBanner message={error} />}

        <Field label={t("filterGroup")} error={fieldErrors.group}>
          <select
            value={groupId}
            onChange={(e) => setGroupId(e.target.value)}
            aria-invalid={Boolean(fieldErrors.group) || undefined}
            className={selectClass}
          >
            <option value="">{t("chooseFragment")}</option>
            {groups.map((group) => (
              <option key={group.id} value={group.id}>
                {group.name}
              </option>
            ))}
          </select>
        </Field>
        <Field label={t("fragmentScope")}>
          <select value={scope} onChange={(e) => setScope(e.target.value as Scope)} className={selectClass}>
            {SCOPES.map((value) => (
              <option key={value} value={value}>
                {t(SCOPE_LABEL[value])}
              </option>
            ))}
          </select>
        </Field>
        <Field label={t("fragmentName")} error={fieldErrors.name}>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            aria-invalid={Boolean(fieldErrors.name) || undefined}
            className={inputClass}
          />
        </Field>
        <Field label={t("fragmentNameEn")}>
          <input value={nameEn} onChange={(e) => setNameEn(e.target.value)} className={inputClass} />
        </Field>
        <Field label={t("fragmentPrompt")} error={fieldErrors.prompt}>
          <textarea
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            rows={4}
            aria-invalid={Boolean(fieldErrors.prompt) || undefined}
            className={inputClass}
          />
        </Field>
        <Field label={t("fragmentNotes")}>
          <input value={notes} onChange={(e) => setNotes(e.target.value)} className={inputClass} />
        </Field>
        <Field label={t("fragmentSource")}>
          <input value={source} onChange={(e) => setSource(e.target.value)} className={inputClass} />
        </Field>

        <div className="space-y-2 border-t border-line pt-4">
          <label className="inline-flex min-h-11 cursor-pointer items-center gap-2.5 text-sm text-ink">
            <input
              type="checkbox"
              checked={isPalette}
              onChange={(e) => setIsPalette(e.target.checked)}
              className={checkClass}
            />
            {t("paletteToggle")}
          </label>
          <p className="text-xs text-ink-2">{t("paletteToggleHint")}</p>
          {isPalette && (
            <div className="grid grid-cols-2 gap-3 pt-1 sm:grid-cols-4">
              {ROLES.map(([role, label]) => (
                <div key={role} className="space-y-2 rounded-panel border border-line p-2.5">
                  <span
                    aria-hidden="true"
                    className="block h-12 w-full rounded-[4px] border border-line-strong"
                    style={{ backgroundColor: colors[role] }}
                  />
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-[13px] font-medium text-ink">{t(label)}</span>
                    <input
                      type="color"
                      value={colors[role]}
                      onChange={(e) => setColors({ ...colors, [role]: e.target.value.toUpperCase() })}
                      aria-label={t(label)}
                      className="h-9 w-10 cursor-pointer rounded-ctl border border-line-strong bg-panel p-0.5"
                    />
                  </div>
                  <p className="mono text-xs text-ink-2">{colors[role]}</p>
                </div>
              ))}
            </div>
          )}
          {fieldErrors.colors && (
            <p role="alert" className="text-xs text-danger">
              {fieldErrors.colors}
            </p>
          )}
        </div>

        <div className="flex flex-wrap justify-end gap-2 border-t border-line pt-4">
          <Button variant="secondary" onClick={onClose} disabled={save.isPending}>
            {t("cancel")}
          </Button>
          <Button type="submit" loading={save.isPending}>
            {save.isPending ? t("saving") : t("save")}
          </Button>
        </div>
      </form>
    </Dialog>
  );
}

function FragmentSection({ groups }: { groups: Group[] }) {
  const { t } = useI18n();
  const qc = useQueryClient();
  const [query, setQuery] = useState("");
  const [groupFilter, setGroupFilter] = useState("");
  const [scopeFilter, setScopeFilter] = useState("");
  const [includeArchived, setIncludeArchived] = useState(false);
  const [applied, setApplied] = useState({ query: "", group: "", scope: "", page: 1 });
  const [form, setForm] = useState<{ open: boolean; fragment: Fragment | null }>({
    open: false,
    fragment: null,
  });
  const [archiveTarget, setArchiveTarget] = useState<Fragment | null>(null);
  const params = new URLSearchParams();
  if (applied.query) params.set("q", applied.query);
  if (applied.group) params.set("group_id", applied.group);
  if (applied.scope) params.set("scope", applied.scope);
  if (includeArchived) params.set("include_archived", "true");
  params.set("page", String(applied.page));
  params.set("page_size", String(PAGE_SIZE));

  const fragments = useQuery({
    queryKey: ["fragments", applied, includeArchived],
    queryFn: () => api<FragmentPage>(`/api/v1/admin/fragments?${params.toString()}`),
  });

  const archive = useMutation({
    mutationFn: (id: string) => api<void>(`/api/v1/admin/fragments/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      setArchiveTarget(null);
      qc.invalidateQueries({ queryKey: ["fragments"] });
    },
  });

  const groupsById = new Map(groups.map((group) => [group.id, group.name]));

  function runSearch(event: React.FormEvent) {
    event.preventDefault();
    setApplied({ query: query.trim(), group: groupFilter, scope: scopeFilter, page: 1 });
  }

  function closeArchive() {
    archive.reset();
    setArchiveTarget(null);
  }

  return (
    <Section
      title={t("tabFragments")}
      actions={
        <Button icon="plus" onClick={() => setForm({ open: true, fragment: null })}>
          {t("newFragment")}
        </Button>
      }
    >
      {form.open && (
        <FragmentForm
          key={form.fragment?.id ?? "create-fragment"}
          groups={groups}
          fragment={form.fragment}
          onClose={() => setForm({ open: false, fragment: null })}
        />
      )}

      <form
        onSubmit={runSearch}
        role="search"
        aria-label={t("search")}
        className="space-y-3 pb-4"
      >
        <Field label={t("search")}>
          <input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t("searchPlaceholder")}
            className={inputClass}
          />
        </Field>
        <div className="grid grid-cols-2 gap-2 sm:flex sm:flex-wrap sm:items-end">
          <Field label={t("filterGroup")}>
            <select
              value={groupFilter}
              onChange={(e) => setGroupFilter(e.target.value)}
              className={`${selectClass} sm:w-44`}
            >
              <option value="">{t("allGroups")}</option>
              {groups.map((group) => (
                <option key={group.id} value={group.id}>
                  {group.name}
                </option>
              ))}
            </select>
          </Field>
          <Field label={t("filterScope")}>
            <select
              value={scopeFilter}
              onChange={(e) => setScopeFilter(e.target.value)}
              className={`${selectClass} sm:w-40`}
            >
              <option value="">{t("allScopes")}</option>
              {SCOPES.map((scope) => (
                <option key={scope} value={scope}>
                  {t(SCOPE_LABEL[scope])}
                </option>
              ))}
            </select>
          </Field>
          <label className="col-span-2 inline-flex min-h-11 cursor-pointer items-center gap-2.5 text-sm text-ink sm:col-span-1 sm:min-h-9">
            <input
              type="checkbox"
              checked={includeArchived}
              onChange={(e) => setIncludeArchived(e.target.checked)}
              className={checkClass}
            />
            {t("includeArchived")}
          </label>
          <Button type="submit" icon="search" className="col-span-2 sm:col-span-1">
            {t("search")}
          </Button>
        </div>
      </form>

      {fragments.isLoading ? (
        <SkeletonRows rows={6} />
      ) : fragments.isError ? (
        <ErrorRetry message={errorText(fragments.error)} onRetry={() => fragments.refetch()} />
      ) : fragments.data && fragments.data.items.length > 0 ? (
        <>
          <ul aria-label={t("tabFragments")} className="divide-y divide-line border-y border-line">
            {fragments.data.items.map((fragment) => (
              <li key={fragment.id} className="flex min-w-0 flex-col gap-3 py-3.5 sm:flex-row sm:items-start sm:justify-between">
                <div className="min-w-0 flex-1 space-y-1">
                  <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
                    <h3 className="min-w-0 break-words text-sm font-semibold text-ink">{fragment.name}</h3>
                    {fragment.name_en && (
                      <span className="min-w-0 break-words text-sm text-ink-2">{fragment.name_en}</span>
                    )}
                    <Stamp>{t(SCOPE_LABEL[fragment.scope])}</Stamp>
                    {fragment.archived && <Stamp>{t("archived")}</Stamp>}
                  </div>
                  <p className="line-clamp-1 break-words text-xs text-ink-2">{fragment.prompt}</p>
                  <p className="flex min-w-0 flex-wrap items-center gap-x-2 text-xs text-ink-3">
                    <JobTag id={fragment.id} />
                    <span className="break-words">{groupsById.get(fragment.group_id) ?? t("none")}</span>
                    {fragment.source && <span className="break-words">· {fragment.source}</span>}
                    <span className="mono">{formatDate(fragment.updated_at)}</span>
                  </p>
                  {fragment.notes && <p className="break-words text-xs text-ink-3">{fragment.notes}</p>}
                  {fragment.colors && (
                    <ul aria-label={t("paletteToggle")} className="flex flex-wrap gap-x-3 gap-y-1 pt-0.5">
                      {ROLES.map(([role, label]) => (
                        <li key={role} className="inline-flex items-center gap-1.5 text-[11px] text-ink-2">
                          <span
                            aria-hidden="true"
                            className="size-3.5 rounded-[3px] border border-line-strong"
                            style={{ backgroundColor: fragment.colors?.[role] }}
                          />
                          <span>{t(label)}</span>
                          <span className="mono">{fragment.colors?.[role]}</span>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
                <div className="flex shrink-0 flex-wrap gap-2">
                  <Button variant="secondary" icon="edit" onClick={() => setForm({ open: true, fragment })}>
                    {t("edit")}
                  </Button>
                  {!fragment.archived && (
                    <Button
                      variant="danger"
                      onClick={() => {
                        archive.reset();
                        setArchiveTarget(fragment);
                      }}
                    >
                      {t("archive")}
                    </Button>
                  )}
                </div>
              </li>
            ))}
          </ul>
          <Pager
            page={fragments.data.page}
            pageSize={fragments.data.page_size}
            total={fragments.data.total}
            onChange={(next) => setApplied({ ...applied, page: next })}
          />
        </>
      ) : (
        <EmptyState title={t("noFragments")} icon="library">
          <Button icon="plus" onClick={() => setForm({ open: true, fragment: null })}>
            {t("newFragment")}
          </Button>
        </EmptyState>
      )}

      {archiveTarget && (
        <ConfirmDialog
          title={t("archive")}
          body={t("archiveConfirm")}
          confirmLabel={t("archive")}
          busy={archive.isPending}
          error={archive.isError ? errorText(archive.error) : null}
          onConfirm={() => archive.mutate(archiveTarget.id)}
          onClose={closeArchive}
        />
      )}
    </Section>
  );
}

/* ---------- Page ---------- */

function Library() {
  const { t } = useI18n();
  const [tab, setTab] = useState<"groups" | "fragments">("fragments");
  const groups = useQuery({
    queryKey: ["groups"],
    queryFn: () => api<GroupList>("/api/v1/admin/fragment-groups"),
  });

  return (
    <>
      <PageHeader title={t("libraryTitle")} subtitle={t("librarySubtitle")} />
      <div className="mb-2">
        <Tabs<"groups" | "fragments">
          variant="segmented"
          label={t("libraryTitle")}
          value={tab}
          onChange={setTab}
          items={[
            { id: "fragments", label: t("tabFragments") },
            { id: "groups", label: t("tabGroups") },
          ]}
        />
      </div>

      {groups.isLoading ? (
        <SkeletonRows rows={4} className="mt-5" />
      ) : groups.isError ? (
        <div className="mt-5">
          <ErrorRetry message={errorText(groups.error)} onRetry={() => groups.refetch()} />
        </div>
      ) : (
        groups.data &&
        (tab === "groups" ? (
          <GroupSection groups={groups.data.items} />
        ) : (
          <FragmentSection groups={groups.data.items} />
        ))
      )}
    </>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <AppShell>
        <Library />
      </AppShell>
    </RequireAuth>
  );
}
