"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import {
  Button,
  EmptyState,
  ErrorBanner,
  Field,
  PageHeader,
  Pager,
  Spinner,
  inputClass,
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
const EMPTY_COLORS: Colors = {
  primary: "#FFFFFF",
  secondary: "#FFFFFF",
  accent: "#FFFFFF",
  line: "#000000",
};

const SCOPE_LABEL: Record<Scope, "scopeIdentity" | "scopePack" | "scopeSticker"> = {
  identity: "scopeIdentity",
  pack: "scopePack",
  sticker: "scopeSticker",
};

function GroupSection() {
  const { t } = useI18n();
  const qc = useQueryClient();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [sortOrder, setSortOrder] = useState("0");
  const [error, setError] = useState<string | null>(null);

  const groups = useQuery({
    queryKey: ["groups"],
    queryFn: () => api<GroupList>("/api/v1/admin/fragment-groups"),
  });

  const saveGroup = useMutation({
    mutationFn: (body: GroupDraft) =>
      editingId
        ? api<Group>(`/api/v1/admin/fragment-groups/${editingId}`, { method: "PUT", json: body })
        : api<Group>("/api/v1/admin/fragment-groups", { method: "POST", json: body }),
    onSuccess: () => {
      setError(null);
      setFormOpen(false);
      setEditingId(null);
      qc.invalidateQueries({ queryKey: ["groups"] });
      qc.invalidateQueries({ queryKey: ["fragments"] });
    },
    onError: (err) => setError(errorText(err)),
  });

  const deleteGroup = useMutation({
    mutationFn: (id: string) =>
      api<void>(`/api/v1/admin/fragment-groups/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      setError(null);
      qc.invalidateQueries({ queryKey: ["groups"] });
    },
    onError: (err) => setError(errorText(err)),
  });

  function startCreate() {
    setEditingId(null);
    setName("");
    setDescription("");
    setSortOrder("0");
    setFormOpen(true);
  }

  function startEdit(group: Group) {
    setEditingId(group.id);
    setName(group.name);
    setDescription(group.description);
    setSortOrder(String(group.sort_order));
    setFormOpen(true);
  }

  return (
    <section className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-sm font-semibold text-zinc-700">{t("tabGroups")}</h2>
        <Button onClick={startCreate}>{t("newGroup")}</Button>
      </div>

      {error && <ErrorBanner message={error} />}

      {formOpen && (
        <form
          onSubmit={(event) => {
            event.preventDefault();
            if (!name.trim()) {
              setError(t("nameRequired"));
              return;
            }
            saveGroup.mutate({
              name: name.trim(),
              description: description.trim(),
              sort_order: Number(sortOrder) || 0,
            });
          }}
          className="space-y-3 rounded-xl border border-zinc-200 bg-white p-4"
        >
          <div className="grid gap-3 sm:grid-cols-[2fr_1fr]">
            <Field label={t("groupName")}>
              <input value={name} onChange={(e) => setName(e.target.value)} className={inputClass} />
            </Field>
            <Field label={t("groupSortOrder")}>
              <input
                type="number"
                value={sortOrder}
                onChange={(e) => setSortOrder(e.target.value)}
                className={inputClass}
              />
            </Field>
          </div>
          <Field label={t("groupDescription")}>
            <input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className={inputClass}
            />
          </Field>
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setFormOpen(false)}>
              {t("cancel")}
            </Button>
            <Button type="submit" disabled={saveGroup.isPending}>
              {saveGroup.isPending ? t("saving") : t("save")}
            </Button>
          </div>
        </form>
      )}

      {groups.isLoading ? (
        <Spinner />
      ) : groups.isError ? (
        <ErrorBanner message={errorText(groups.error)} />
      ) : groups.data && groups.data.items.length > 0 ? (
        <ul className="divide-y divide-zinc-100 overflow-hidden rounded-xl border border-zinc-200 bg-white">
          {groups.data.items.map((group) => (
            <li key={group.id} className="flex flex-wrap items-center justify-between gap-2 px-4 py-3">
              <div className="min-w-0">
                <p className="truncate text-sm font-medium text-zinc-900">{group.name}</p>
                <p className="truncate text-xs text-zinc-500">
                  {group.description || t("none")} · {t("groupSortOrder")} {group.sort_order}
                </p>
              </div>
              <div className="flex gap-2">
                <Button variant="secondary" onClick={() => startEdit(group)}>
                  {t("edit")}
                </Button>
                <Button
                  variant="danger"
                  onClick={() => {
                    if (window.confirm(t("groupDeleteConfirm"))) deleteGroup.mutate(group.id);
                  }}
                  disabled={deleteGroup.isPending}
                >
                  {t("delete")}
                </Button>
              </div>
            </li>
          ))}
        </ul>
      ) : (
        <EmptyState title={t("noGroups")} />
      )}
    </section>
  );
}

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
  const [groupId, setGroupId] = useState(fragment?.group_id ?? groups[0]?.id ?? "");
  const [name, setName] = useState(fragment?.name ?? "");
  const [nameEn, setNameEn] = useState(fragment?.name_en ?? "");
  const [prompt, setPrompt] = useState(fragment?.prompt ?? "");
  const [notes, setNotes] = useState(fragment?.notes ?? "");
  const [source, setSource] = useState(fragment?.source ?? "");
  const [scope, setScope] = useState<Scope>(fragment?.scope ?? "pack");
  const [isPalette, setIsPalette] = useState(Boolean(fragment?.colors));
  const [colors, setColors] = useState<Colors>(fragment?.colors ?? EMPTY_COLORS);
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
    if (!groupId) return setError(t("groupRequired"));
    if (!name.trim()) return setError(t("nameRequired"));
    if (!prompt.trim()) return setError(t("promptRequired"));
    if (isPalette && !Object.values(colors).every((value) => /^#[0-9A-Fa-f]{6}$/.test(value))) {
      return setError(t("colorsRequired"));
    }
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
    <form onSubmit={submit} className="space-y-3 rounded-xl border border-zinc-200 bg-white p-4">
      <h3 className="text-sm font-semibold text-zinc-700">
        {fragment ? t("edit") : t("newFragment")}
      </h3>
      {error && <ErrorBanner message={error} />}
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label={t("filterGroup")}>
          <select value={groupId} onChange={(e) => setGroupId(e.target.value)} className={inputClass}>
            <option value="">{t("chooseFragment")}</option>
            {groups.map((group) => (
              <option key={group.id} value={group.id}>
                {group.name}
              </option>
            ))}
          </select>
        </Field>
        <Field label={t("fragmentScope")}>
          <select
            value={scope}
            onChange={(e) => setScope(e.target.value as Scope)}
            className={inputClass}
          >
            {SCOPES.map((value) => (
              <option key={value} value={value}>
                {t(SCOPE_LABEL[value])}
              </option>
            ))}
          </select>
        </Field>
        <Field label={t("fragmentName")}>
          <input value={name} onChange={(e) => setName(e.target.value)} className={inputClass} />
        </Field>
        <Field label={t("fragmentNameEn")}>
          <input value={nameEn} onChange={(e) => setNameEn(e.target.value)} className={inputClass} />
        </Field>
      </div>
      <Field label={t("fragmentPrompt")}>
        <textarea
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          rows={3}
          className={inputClass}
        />
      </Field>
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label={t("fragmentNotes")}>
          <input value={notes} onChange={(e) => setNotes(e.target.value)} className={inputClass} />
        </Field>
        <Field label={t("fragmentSource")}>
          <input value={source} onChange={(e) => setSource(e.target.value)} className={inputClass} />
        </Field>
      </div>

      <label className="flex items-center gap-2 text-sm text-zinc-700">
        <input
          type="checkbox"
          checked={isPalette}
          onChange={(e) => setIsPalette(e.target.checked)}
          className="h-4 w-4 rounded border-zinc-300"
        />
        {t("paletteToggle")}
      </label>
      <p className="text-xs text-zinc-500">{t("paletteToggleHint")}</p>
      {isPalette && (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          {(
            [
              ["primary", "rolePrimary"],
              ["secondary", "roleSecondary"],
              ["accent", "roleAccent"],
              ["line", "roleLine"],
            ] as const
          ).map(([role, label]) => (
            <Field key={role} label={t(label)}>
              <div className="flex items-center gap-2">
                <span
                  aria-hidden="true"
                  className="h-6 w-6 shrink-0 rounded border border-zinc-200"
                  style={{ backgroundColor: colors[role] }}
                />
                <input
                  type="color"
                  value={colors[role]}
                  onChange={(e) => setColors({ ...colors, [role]: e.target.value.toUpperCase() })}
                  aria-label={t(label)}
                  className="h-9 w-12 rounded border border-zinc-300 bg-white"
                />
                <code className="text-xs text-zinc-600">{colors[role]}</code>
              </div>
            </Field>
          ))}
        </div>
      )}

      <div className="flex justify-end gap-2">
        <Button variant="secondary" onClick={onClose}>
          {t("cancel")}
        </Button>
        <Button type="submit" disabled={save.isPending}>
          {save.isPending ? t("saving") : t("save")}
        </Button>
      </div>
    </form>
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
  const [error, setError] = useState<string | null>(null);

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
      setError(null);
      qc.invalidateQueries({ queryKey: ["fragments"] });
    },
    onError: (err) => setError(errorText(err)),
  });

  const groupsById = new Map(groups.map((group) => [group.id, group.name]));

  function runSearch(event: React.FormEvent) {
    event.preventDefault();
    setApplied({ query: query.trim(), group: groupFilter, scope: scopeFilter, page: 1 });
  }

  return (
    <section className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-sm font-semibold text-zinc-700">{t("tabFragments")}</h2>
        <Button onClick={() => setForm({ open: true, fragment: null })}>{t("newFragment")}</Button>
      </div>

      {error && <ErrorBanner message={error} />}

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
        className="grid gap-3 rounded-xl border border-zinc-200 bg-white p-4 sm:grid-cols-2 lg:grid-cols-5"
      >
        <Field label={t("search")}>
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t("searchPlaceholder")}
            className={inputClass}
          />
        </Field>
        <Field label={t("filterGroup")}>
          <select
            value={groupFilter}
            onChange={(e) => setGroupFilter(e.target.value)}
            className={inputClass}
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
            className={inputClass}
          >
            <option value="">{t("allScopes")}</option>
            {SCOPES.map((scope) => (
              <option key={scope} value={scope}>
                {t(SCOPE_LABEL[scope])}
              </option>
            ))}
          </select>
        </Field>
        <label className="flex items-end gap-2 pb-2 text-sm text-zinc-700">
          <input
            type="checkbox"
            checked={includeArchived}
            onChange={(e) => setIncludeArchived(e.target.checked)}
            className="h-4 w-4 rounded border-zinc-300"
          />
          {t("includeArchived")}
        </label>
        <div className="flex items-end">
          <Button type="submit">{t("search")}</Button>
        </div>
      </form>

      {fragments.isLoading ? (
        <Spinner />
      ) : fragments.isError ? (
        <ErrorBanner message={errorText(fragments.error)} />
      ) : fragments.data && fragments.data.items.length > 0 ? (
        <>
          <ul className="divide-y divide-zinc-100 overflow-hidden rounded-xl border border-zinc-200 bg-white">
            {fragments.data.items.map((fragment) => (
              <li key={fragment.id} className="px-4 py-3">
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-zinc-900">
                      {fragment.name}
                      {fragment.name_en && (
                        <span className="ml-2 text-xs font-normal text-zinc-500">
                          {fragment.name_en}
                        </span>
                      )}
                      {fragment.archived && (
                        <span className="ml-2 rounded bg-zinc-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase text-zinc-500">
                          {t("archived")}
                        </span>
                      )}
                    </p>
                    <p className="mt-0.5 truncate text-xs text-zinc-500">
                      {groupsById.get(fragment.group_id) ?? t("none")} · {t(SCOPE_LABEL[fragment.scope])}
                      {fragment.source ? ` · ${fragment.source}` : ""} · {formatDate(fragment.updated_at)}
                    </p>
                  </div>
                  <div className="flex shrink-0 gap-2">
                    <Button
                      variant="secondary"
                      onClick={() => setForm({ open: true, fragment })}
                    >
                      {t("edit")}
                    </Button>
                    {!fragment.archived && (
                      <Button
                        variant="danger"
                        onClick={() => {
                          if (window.confirm(t("archiveConfirm"))) archive.mutate(fragment.id);
                        }}
                        disabled={archive.isPending}
                      >
                        {t("archive")}
                      </Button>
                    )}
                  </div>
                </div>
                <p className="mt-1 line-clamp-2 text-xs text-zinc-600">{fragment.prompt}</p>
                {fragment.notes && (
                  <p className="mt-1 text-xs text-zinc-400">{fragment.notes}</p>
                )}
                {fragment.colors && (
                  <div className="mt-2 flex flex-wrap items-center gap-3 text-[11px] text-zinc-600">
                    {(
                      [
                        ["primary", "rolePrimary"],
                        ["secondary", "roleSecondary"],
                        ["accent", "roleAccent"],
                        ["line", "roleLine"],
                      ] as const
                    ).map(([role, label]) => (
                      <span key={role} className="inline-flex items-center gap-1">
                        <span
                          aria-hidden="true"
                          className="h-4 w-4 rounded border border-zinc-200"
                          style={{ backgroundColor: fragment.colors?.[role] }}
                        />
                        {t(label)} {fragment.colors?.[role]}
                      </span>
                    ))}
                  </div>
                )}
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
        <EmptyState title={t("noFragments")} />
      )}
    </section>
  );
}

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
      <div className="mb-5 flex gap-2" role="tablist" aria-label={t("libraryTitle")}>
        {(
          [
            ["fragments", "tabFragments"],
            ["groups", "tabGroups"],
          ] as const
        ).map(([value, label]) => (
          <button
            key={value}
            role="tab"
            aria-selected={tab === value}
            onClick={() => setTab(value)}
            className={`rounded-lg px-4 py-2 text-sm font-medium ${
              tab === value
                ? "bg-indigo-600 text-white"
                : "border border-zinc-300 bg-white text-zinc-700 hover:bg-zinc-50"
            }`}
          >
            {t(label)}
          </button>
        ))}
      </div>

      {groups.isError && <ErrorBanner message={errorText(groups.error)} />}
      {groups.data &&
        (tab === "groups" ? (
          <GroupSection />
        ) : (
          <FragmentSection groups={groups.data.items} />
        ))}
      {groups.isLoading && <Spinner />}
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
