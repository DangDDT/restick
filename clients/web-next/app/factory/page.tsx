"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import {
  Button,
  CopyButton,
  Dialog,
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
import { useI18n } from "@/lib/i18n";
import type {
  Cell,
  CharacterList,
  CompiledDesign,
  Design,
  DesignPage,
  Draft,
  Fragment,
  FragmentPage,
  GroupList,
} from "@/lib/types";

const DEFAULT_OUTLINE_WIDTH = 4;
const DEFAULT_OUTLINE_COLOR = "#FFFFFF";

function outlineWidthFromInput(value: string): number {
  if (value === "") return DEFAULT_OUTLINE_WIDTH;
  const parsed = Math.round(Number(value));
  return Number.isNaN(parsed) ? DEFAULT_OUTLINE_WIDTH : Math.min(16, Math.max(0, parsed));
}

// Local cell state always carries an array; the wire omits an empty
// `fragment_ids`, so loading normalises it here once.
interface CellState {
  action: string;
  expression: string;
  free_text: string;
  fragment_ids: string[];
}

function emptyCell(): CellState {
  return { action: "", expression: "", free_text: "", fragment_ids: [] };
}

function resizeCells(cells: Cell[], count: number): CellState[] {
  return Array.from({ length: count }, (_, index) => {
    const cell = cells[index];
    return cell
      ? {
          action: cell.action,
          expression: cell.expression,
          free_text: cell.free_text,
          fragment_ids: cell.fragment_ids ?? [],
        }
      : emptyCell();
  });
}

interface BuilderState {
  id: string | null;
  name: string;
  characterId: string;
  paletteId: string;
  groupIds: string[];
  selections: Record<string, string>;
  pinned: string[];
  freeText: string;
  rows: number;
  columns: number;
  cells: CellState[];
  outlineWidth: string;
  outlineColor: string;
}

function blankState(): BuilderState {
  return {
    id: null,
    name: "",
    characterId: "",
    paletteId: "",
    groupIds: [],
    selections: {},
    pinned: [],
    freeText: "",
    rows: 1,
    columns: 1,
    cells: [emptyCell()],
    outlineWidth: String(DEFAULT_OUTLINE_WIDTH),
    outlineColor: DEFAULT_OUTLINE_COLOR,
  };
}

async function fetchAllFragments(): Promise<Fragment[]> {
  const items: Fragment[] = [];
  for (let page = 1; ; page++) {
    const res = await api<FragmentPage>(
      `/api/v1/admin/fragments?include_archived=true&page=${page}&page_size=100`,
    );
    items.push(...res.items);
    if (res.items.length === 0 || items.length >= res.total) return items;
  }
}

async function fetchAllDesigns(): Promise<Design[]> {
  const items: Design[] = [];
  for (let page = 1; ; page++) {
    const res = await api<DesignPage>(`/api/v1/admin/designs?page=${page}&page_size=100`);
    items.push(...res.items);
    if (res.items.length === 0 || items.length >= res.total) return items;
  }
}

function stateFromDesign(design: Design, fragmentsById: Map<string, Fragment>): BuilderState {
  const draft = design.draft;
  const selections: Record<string, string> = {};
  const groupIds = [...(draft.group_ids ?? [])];
  for (const fragmentId of draft.fragment_ids ?? []) {
    const fragment = fragmentsById.get(fragmentId);
    if (!fragment) continue;
    selections[fragment.group_id] = fragmentId;
    if (!groupIds.includes(fragment.group_id)) groupIds.push(fragment.group_id);
  }
  const count = Math.min(64, Math.max(1, draft.rows * draft.columns));
  return {
    id: design.id,
    name: design.name,
    characterId: design.character_id || "",
    paletteId: draft.palette_id ?? "",
    groupIds,
    selections,
    pinned: (draft.pinned_fragment_ids ?? []).filter((id) => Object.values(selections).includes(id)),
    freeText: draft.free_text ?? "",
    rows: draft.rows,
    columns: draft.columns,
    cells: resizeCells(draft.cells ?? [], count),
    outlineWidth: String(design.outline_width),
    outlineColor: design.outline_color || DEFAULT_OUTLINE_COLOR,
  };
}

function draftFromState(state: BuilderState): Draft {
  const cells = resizeCells(state.cells, state.rows * state.columns);
  const selected = state.groupIds.map((id) => state.selections[id]).filter(Boolean);
  return {
    id: state.id ?? undefined,
    name: state.name.trim(),
    character_id: state.characterId,
    group_ids: state.groupIds,
    fragment_ids: Array.from(new Set(selected)),
    pinned_fragment_ids: state.pinned.filter((id) => selected.includes(id)),
    palette_id: state.paletteId,
    free_text: state.freeText,
    rows: state.rows,
    columns: state.columns,
    cells: cells.map((cell) => ({
      action: cell.action,
      expression: cell.expression,
      fragment_ids: cell.fragment_ids,
      free_text: cell.free_text,
    })),
    outline_width: outlineWidthFromInput(state.outlineWidth),
    outline_color: state.outlineColor,
  };
}

const COLOR_ROLES = [
  ["primary", "rolePrimary"],
  ["secondary", "roleSecondary"],
  ["accent", "roleAccent"],
  ["line", "roleLine"],
] as const;

function FragmentPicker({
  fragments,
  selected,
  onToggle,
  onClose,
}: {
  fragments: Fragment[];
  selected: string[];
  onToggle: (id: string) => void;
  onClose: () => void;
}) {
  const { t } = useI18n();
  const [query, setQuery] = useState("");
  const needle = query.trim().toLowerCase();
  const visible = fragments.filter(
    (fragment) =>
      !needle ||
      fragment.name.toLowerCase().includes(needle) ||
      fragment.name_en.toLowerCase().includes(needle) ||
      fragment.prompt.toLowerCase().includes(needle),
  );

  return (
    <Dialog title={t("pickerTitle")} onClose={onClose}>
      <Field label={t("pickerSearch")}>
        <input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          className={inputClass}
          autoFocus
        />
      </Field>
      <div className="mt-3 max-h-72 space-y-1 overflow-y-auto">
        {visible.length === 0 && <p className="text-sm text-zinc-500">{t("pickerEmpty")}</p>}
        {/* shortcut: first 200 matches; the search box narrows further */}
        {visible.slice(0, 200).map((fragment) => {
          const checked = selected.includes(fragment.id);
          return (
            <label
              key={fragment.id}
              className="flex items-start gap-2 rounded-lg px-2 py-1.5 text-sm hover:bg-zinc-50"
            >
              <input
                type="checkbox"
                checked={checked}
                onChange={() => onToggle(fragment.id)}
                className="mt-1 h-4 w-4 rounded border-zinc-300"
              />
              <span className="min-w-0">
                <span className="font-medium text-zinc-900">{fragment.name}</span>
                {fragment.name_en && (
                  <span className="ml-2 text-xs text-zinc-500">{fragment.name_en}</span>
                )}
                <span className="block truncate text-xs text-zinc-500">
                  {fragment.prompt}
                </span>
              </span>
            </label>
          );
        })}
      </div>
      <div className="mt-4 flex justify-end">
        <Button onClick={onClose}>{t("pickerDone")}</Button>
      </div>
    </Dialog>
  );
}

function Factory() {
  const { t } = useI18n();
  const qc = useQueryClient();
  const promptRef = useRef<HTMLTextAreaElement>(null);
  const [state, setState] = useState<BuilderState>(blankState);
  const [compiledPrompt, setCompiledPrompt] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [suggesting, setSuggesting] = useState(false);
  const [compiling, setCompiling] = useState(false);
  const [pickerCell, setPickerCell] = useState<number | null>(null);
  const [addGroupId, setAddGroupId] = useState("");

  const groups = useQuery({
    queryKey: ["groups"],
    queryFn: () => api<GroupList>("/api/v1/admin/fragment-groups"),
  });
  const fragments = useQuery({ queryKey: ["fragments-all"], queryFn: fetchAllFragments });
  const characters = useQuery({
    queryKey: ["characters"],
    queryFn: () => api<CharacterList>("/api/v1/admin/characters"),
  });
  const designs = useQuery({ queryKey: ["designs-all"], queryFn: fetchAllDesigns });

  const fragmentsById = new Map((fragments.data ?? []).map((f) => [f.id, f]));
  const groupsById = new Map((groups.data?.items ?? []).map((g) => [g.id, g]));
  const palettes = (fragments.data ?? []).filter(
    (f) => f.colors && (!f.archived || f.id === state.paletteId),
  );
  const charactersById = new Map((characters.data?.items ?? []).map((c) => [c.id, c]));
  const selectedPalette = state.paletteId ? fragmentsById.get(state.paletteId) : undefined;
  const paletteColors = selectedPalette?.colors;

  const saveDesign = useMutation({
    mutationFn: (draft: Draft) =>
      draft.id
        ? api<Design>(`/api/v1/admin/designs/${draft.id}`, { method: "PUT", json: draft })
        : api<Design>("/api/v1/admin/designs", { method: "POST", json: draft }),
    onSuccess: (saved) => {
      setError(null);
      setNotice(t("designSaved"));
      setState((current) => ({ ...current, id: saved.id, name: saved.name }));
      setCompiledPrompt(saved.compiled_prompt);
      qc.invalidateQueries({ queryKey: ["designs-all"] });
    },
    onError: (err) => setError(errorText(err)),
  });

  const deleteDesign = useMutation({
    mutationFn: (id: string) => api<void>(`/api/v1/admin/designs/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      setError(null);
      setNotice(t("designDeleted"));
      setCompiledPrompt("");
      setState(blankState());
      qc.invalidateQueries({ queryKey: ["designs-all"] });
    },
    onError: (err) => setError(errorText(err)),
  });

  const editorBusy = saveDesign.isPending || deleteDesign.isPending;

  function update(patch: Partial<BuilderState>) {
    setState((current) => ({ ...current, ...patch }));
  }

  function loadDesign(design: Design) {
    if (editorBusy) return;
    setError(null);
    setNotice(null);
    setCompiledPrompt(design.compiled_prompt);
    setState(stateFromDesign(design, fragmentsById));
  }

  function setGrid(nextRows: number, nextColumns: number) {
    const rows = Math.min(8, Math.max(1, nextRows || 1));
    const columns = Math.min(8, Math.max(1, nextColumns || 1));
    setState((current) => ({
      ...current,
      rows,
      columns,
      cells: resizeCells(current.cells, rows * columns),
    }));
  }

  function togglePin(fragmentId: string) {
    setState((current) => ({
      ...current,
      pinned: current.pinned.includes(fragmentId)
        ? current.pinned.filter((id) => id !== fragmentId)
        : [...current.pinned, fragmentId],
    }));
  }

  function selectFragment(groupId: string, fragmentId: string) {
    setState((current) => ({
      ...current,
      selections: { ...current.selections, [groupId]: fragmentId },
      pinned: current.pinned.filter(
        (id) => id !== current.selections[groupId] && fragmentsById.get(id)?.group_id !== groupId,
      ),
    }));
  }

  function removeGroup(groupId: string) {
    setState((current) => {
      const selections = { ...current.selections };
      const removed = selections[groupId];
      delete selections[groupId];
      return {
        ...current,
        groupIds: current.groupIds.filter((id) => id !== groupId),
        selections,
        pinned: current.pinned.filter((id) => id !== removed && fragmentsById.get(id)?.group_id !== groupId),
      };
    });
  }

  async function suggest() {
    setSuggesting(true);
    setError(null);
    setNotice(null);
    try {
      const suggested = await api<Draft>("/api/v1/admin/designs/suggest", {
        method: "POST",
        json: draftFromState(state),
      });
      const selections: Record<string, string> = {};
      for (const fragmentId of suggested.fragment_ids ?? []) {
        const fragment = fragmentsById.get(fragmentId);
        if (fragment) selections[fragment.group_id] = fragmentId;
      }
      setState((current) => ({
        ...current,
        groupIds: suggested.group_ids ?? [],
        selections,
        pinned: suggested.pinned_fragment_ids ?? [],
      }));
    } catch (err) {
      setError(errorText(err));
    } finally {
      setSuggesting(false);
    }
  }

  async function compile() {
    setCompiling(true);
    setError(null);
    setNotice(null);
    try {
      const result = await api<CompiledDesign>("/api/v1/admin/designs/compile", {
        method: "POST",
        json: draftFromState(state),
      });
      setCompiledPrompt(result.compiled_prompt);
    } catch (err) {
      setError(errorText(err));
    } finally {
      setCompiling(false);
    }
  }

  function save() {
    setNotice(null);
    if (!state.name.trim()) {
      setError(t("designNameRequired"));
      return;
    }
    saveDesign.mutate(draftFromState(state));
  }

  const availableGroups = (groups.data?.items ?? []).filter(
    (group) => !state.groupIds.includes(group.id),
  );
  const cellCount = state.rows * state.columns;
  const pickerFragments = (fragments.data ?? []).filter((f) => !f.archived);
  const activeCell = pickerCell === null ? null : state.cells[pickerCell];

  return (
    <>
      <PageHeader
        title={t("factoryTitle")}
        subtitle={t("factorySubtitle")}
        actions={<Button disabled={editorBusy} onClick={() => { setState(blankState()); setCompiledPrompt(""); setError(null); setNotice(null); }}>{t("newDesign")}</Button>}
      />

      <p className="mb-5 rounded-lg border border-indigo-100 bg-indigo-50 px-4 py-3 text-sm text-indigo-900">
        {t("externalAiHint")}
      </p>

      {notice && (
        <div className="mb-4">
          <SuccessBanner message={notice} />
        </div>
      )}
      {error && (
        <div className="mb-4">
          <ErrorBanner message={error} />
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-[260px_1fr]">
        <aside>
          <h2 className="mb-2 text-sm font-semibold text-zinc-700">{t("savedDesigns")}</h2>
          {designs.isLoading ? (
            <Spinner />
          ) : designs.isError ? (
            <ErrorBanner message={errorText(designs.error)} />
          ) : designs.data && designs.data.length > 0 ? (
            <ul className="space-y-2">
              {designs.data.map((design) => (
                <li key={design.id}>
                  <button
                    onClick={() => loadDesign(design)}
                    disabled={!fragments.data || editorBusy}
                    className={`w-full rounded-lg border px-3 py-2 text-left disabled:cursor-not-allowed disabled:opacity-50 ${
                      state.id === design.id
                        ? "border-indigo-500 bg-indigo-50"
                        : "border-zinc-200 bg-white hover:bg-zinc-50"
                    }`}
                  >
                    <span className="block truncate text-sm font-medium text-zinc-900">
                      {design.name}
                    </span>
                    <span className="block text-xs text-zinc-500">
                      {design.rows}×{design.columns} · {formatDate(design.updated_at)}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          ) : (
            <EmptyState title={t("noDesigns")} />
          )}
        </aside>

        {fragments.isLoading || groups.isLoading ? (
          <Spinner />
        ) : (
          <div className="space-y-5">
            <section className="grid gap-3 rounded-xl border border-zinc-200 bg-white p-4 sm:grid-cols-3">
              <Field label={t("designName")}>
                <input
                  value={state.name}
                  onChange={(e) => update({ name: e.target.value })}
                  className={inputClass}
                />
              </Field>
              <Field label={t("character")}>
                <select
                  value={state.characterId}
                  onChange={(e) => update({ characterId: e.target.value })}
                  className={inputClass}
                >
                  <option value="">{t("characterNone")}</option>
                  {characters.data?.items.map((character) => (
                    <option key={character.id} value={character.id}>
                      {character.name}
                    </option>
                  ))}
                  {state.characterId && !charactersById.has(state.characterId) && (
                    <option value={state.characterId}>{state.characterId}</option>
                  )}
                </select>
              </Field>
              <Field label={t("palette")}>
                <select
                  value={state.paletteId}
                  onChange={(e) => update({ paletteId: e.target.value })}
                  className={inputClass}
                >
                  <option value="">{t("paletteNone")}</option>
                  {palettes.map((palette) => (
                    <option key={palette.id} value={palette.id}>
                      {palette.name}
                    </option>
                  ))}
                </select>
              </Field>
              {paletteColors && (
                <div className="sm:col-span-3">
                  <p className="mb-1 text-xs font-medium text-zinc-600">{t("paletteRoles")}</p>
                  <div className="flex flex-wrap gap-3 text-xs text-zinc-600">
                    {COLOR_ROLES.map(([role, label]) => (
                      <span key={role} className="inline-flex items-center gap-1.5">
                        <span
                          aria-hidden="true"
                          className="h-4 w-4 rounded border border-zinc-200"
                          style={{ backgroundColor: paletteColors[role] }}
                        />
                        {t(label)} {paletteColors[role]}
                      </span>
                    ))}
                  </div>
                </div>
              )}
            </section>

            <section className="rounded-xl border border-zinc-200 bg-white p-4">
              <h2 className="text-sm font-semibold text-zinc-700">{t("groupsTitle")}</h2>
              <p className="mt-0.5 text-xs text-zinc-500">{t("groupsHint")} {t("pinnedHint")}</p>

              <div className="mt-3 flex flex-wrap gap-2">
                <select
                  value={addGroupId}
                  onChange={(e) => setAddGroupId(e.target.value)}
                  aria-label={t("groupsTitle")}
                  className={`${inputClass} sm:max-w-xs`}
                >
                  <option value="">{t("allGroups")}</option>
                  {availableGroups.map((group) => (
                    <option key={group.id} value={group.id}>
                      {group.name}
                    </option>
                  ))}
                </select>
                <Button
                  variant="secondary"
                  onClick={() => {
                    if (!addGroupId) return;
                    update({ groupIds: [...state.groupIds, addGroupId] });
                    setAddGroupId("");
                  }}
                >
                  {t("addGroup")}
                </Button>
              </div>

              {state.groupIds.length === 0 ? (
                <p className="mt-3 text-sm text-zinc-500">{t("noSelection")}</p>
              ) : (
                <ul className="mt-3 space-y-3">
                  {state.groupIds.map((groupId) => {
                    const group = groupsById.get(groupId);
                    const candidates = (fragments.data ?? []).filter(
                      (fragment) =>
                        fragment.group_id === groupId &&
                        (!fragment.archived || fragment.id === state.selections[groupId]),
                    );
                    const selectedId = state.selections[groupId] ?? "";
                    const pinned = selectedId ? state.pinned.includes(selectedId) : false;
                    return (
                      <li
                        key={groupId}
                        className="grid items-end gap-2 rounded-lg border border-zinc-100 p-3 sm:grid-cols-[1fr_1fr_auto_auto]"
                      >
                        <div className="text-sm">
                          <p className="font-medium text-zinc-900">{group?.name ?? groupId}</p>
                          <p className="text-xs text-zinc-500">
                            {group?.description || t("none")}
                          </p>
                        </div>
                        <Field label={t("fragment")}>
                          <select
                            value={selectedId}
                            onChange={(e) => selectFragment(groupId, e.target.value)}
                            className={inputClass}
                          >
                            <option value="">{t("chooseFragment")}</option>
                            {candidates.map((fragment) => (
                              <option key={fragment.id} value={fragment.id}>
                                {fragment.name}
                                {fragment.archived ? ` (${t("archived")})` : ""}
                              </option>
                            ))}
                          </select>
                        </Field>
                        <Button
                          variant={pinned ? "primary" : "secondary"}
                          disabled={!selectedId}
                          onClick={() => selectedId && togglePin(selectedId)}
                          aria-pressed={pinned}
                        >
                          {pinned ? t("unpin") : t("pin")}
                        </Button>
                        <Button variant="ghost" onClick={() => removeGroup(groupId)}>
                          {t("delete")}
                        </Button>
                      </li>
                    );
                  })}
                </ul>
              )}
            </section>

            <section className="rounded-xl border border-zinc-200 bg-white p-4">
              <Field label={t("designFreeText")} hint={t("freeTextHint")}>
                <textarea
                  value={state.freeText}
                  onChange={(e) => update({ freeText: e.target.value })}
                  rows={2}
                  className={inputClass}
                />
              </Field>
            </section>

            <section className="rounded-xl border border-zinc-200 bg-white p-4">
              <h2 className="text-sm font-semibold text-zinc-700">{t("gridTitle")}</h2>
              <div className="mt-3 grid gap-3 sm:grid-cols-4">
                <Field label={t("rowsLabel")}>
                  <input
                    type="number"
                    min={1}
                    max={8}
                    value={state.rows}
                    onChange={(e) => setGrid(Number(e.target.value), state.columns)}
                    className={inputClass}
                  />
                </Field>
                <Field label={t("columnsLabel")}>
                  <input
                    type="number"
                    min={1}
                    max={8}
                    value={state.columns}
                    onChange={(e) => setGrid(state.rows, Number(e.target.value))}
                    className={inputClass}
                  />
                </Field>
                <Field label={t("cellsCount")} hint={t("cellsHint")}>
                  <input value={cellCount} readOnly className={inputClass} />
                </Field>
                <Field label={t("outlineWidth")} hint={t("outlineWidthHint")}>
                  <input
                    type="number"
                    min={0}
                    max={16}
                    value={state.outlineWidth}
                    onChange={(e) => update({ outlineWidth: e.target.value })}
                    className={inputClass}
                  />
                </Field>
                <Field label={t("outlineColor")}>
                  <div className="flex items-center gap-2">
                    <input
                      type="color"
                      value={state.outlineColor}
                      onChange={(e) => update({ outlineColor: e.target.value.toUpperCase() })}
                      aria-label={t("outlineColor")}
                      className="h-9 w-12 rounded border border-zinc-300 bg-white"
                    />
                    <code className="text-xs text-zinc-600">{state.outlineColor}</code>
                  </div>
                </Field>
              </div>

              <div className="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                {resizeCells(state.cells, cellCount).map((cell, index) => {
                  const row = Math.floor(index / state.columns) + 1;
                  const column = (index % state.columns) + 1;
                  return (
                    <div key={index} className="rounded-lg border border-zinc-200 p-3">
                      <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-zinc-500">
                        {t("cell")} {row}.{column}
                      </p>
                      <div className="space-y-2">
                        <Field label={t("cellAction")}>
                          <input
                            value={cell.action}
                            onChange={(e) => {
                              const cells = [...state.cells];
                              cells[index] = { ...cell, action: e.target.value };
                              update({ cells });
                            }}
                            className={inputClass}
                          />
                        </Field>
                        <Field label={t("cellExpression")}>
                          <input
                            value={cell.expression}
                            onChange={(e) => {
                              const cells = [...state.cells];
                              cells[index] = { ...cell, expression: e.target.value };
                              update({ cells });
                            }}
                            className={inputClass}
                          />
                        </Field>
                        <Field label={t("cellFreeText")}>
                          <textarea
                            value={cell.free_text}
                            onChange={(e) => {
                              const cells = [...state.cells];
                              cells[index] = { ...cell, free_text: e.target.value };
                              update({ cells });
                            }}
                            rows={2}
                            className={inputClass}
                          />
                        </Field>
                        <div>
                          <p className="mb-1 text-sm font-medium text-zinc-700">
                            {t("cellFragments")}
                          </p>
                          <div className="flex flex-wrap gap-1.5">
                            {cell.fragment_ids.map((fragmentId) => (
                              <span
                                key={fragmentId}
                                className="inline-flex items-center gap-1 rounded-full bg-zinc-100 px-2 py-0.5 text-xs text-zinc-700"
                              >
                                {fragmentsById.get(fragmentId)?.name ?? fragmentId}
                                <button
                                  aria-label={`${t("removeFragment")}: ${fragmentsById.get(fragmentId)?.name ?? fragmentId}`}
                                  onClick={() => {
                                    const cells = [...state.cells];
                                    cells[index] = {
                                      ...cell,
                                      fragment_ids: cell.fragment_ids.filter(
                                        (id) => id !== fragmentId,
                                      ),
                                    };
                                    update({ cells });
                                  }}
                                  className="text-zinc-500 hover:text-red-600"
                                >
                                  ×
                                </button>
                              </span>
                            ))}
                          </div>
                          <Button variant="secondary" onClick={() => setPickerCell(index)}>
                            {t("pickFragments")}
                          </Button>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </section>

            <section className="rounded-xl border border-zinc-200 bg-white p-4">
              <div className="flex flex-wrap gap-2">
                <Button variant="secondary" onClick={() => void suggest()} disabled={suggesting}>
                  {suggesting ? t("suggesting") : t("suggest")}
                </Button>
                <Button variant="secondary" onClick={() => void compile()} disabled={compiling}>
                  {compiling ? t("compiling") : t("compile")}
                </Button>
                <Button onClick={save} disabled={editorBusy}>
                  {saveDesign.isPending
                    ? t("saving")
                    : state.id
                      ? t("updateDesign")
                      : t("saveDesign")}
                </Button>
                {state.id && (
                  <Button
                    variant="danger"
                    onClick={() => {
                      if (window.confirm(t("deleteDesignConfirm"))) deleteDesign.mutate(state.id!);
                    }}
                    disabled={editorBusy}
                  >
                    {t("deleteDesign")}
                  </Button>
                )}
              </div>

              <div className="mt-4">
                <div className="mb-2 flex flex-wrap items-end justify-between gap-2">
                  <div>
                    <p className="text-sm font-medium text-zinc-700">{t("compiledPrompt")}</p>
                    <p className="text-xs text-zinc-500">{t("compiledPromptHint")}</p>
                  </div>
                  <CopyButton text={compiledPrompt} selectRef={promptRef} />
                </div>
                <textarea
                  ref={promptRef}
                  readOnly
                  value={compiledPrompt}
                  placeholder={t("noPrompt")}
                  rows={10}
                  aria-label={t("compiledPrompt")}
                  className={`${inputClass} font-mono text-xs`}
                />
              </div>
            </section>
          </div>
        )}
      </div>

      {activeCell && pickerCell !== null && (
        <FragmentPicker
          fragments={pickerFragments}
          selected={activeCell.fragment_ids}
          onToggle={(fragmentId) => {
            const index = pickerCell;
            if (index === null) return;
            setState((current) => {
              const cells = [...current.cells];
              const cell = cells[index] ?? emptyCell();
              cells[index] = {
                ...cell,
                fragment_ids: cell.fragment_ids.includes(fragmentId)
                  ? cell.fragment_ids.filter((id) => id !== fragmentId)
                  : [...cell.fragment_ids, fragmentId],
              };
              return { ...current, cells };
            });
          }}
          onClose={() => setPickerCell(null)}
        />
      )}
    </>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <AppShell>
        <Factory />
      </AppShell>
    </RequireAuth>
  );
}
