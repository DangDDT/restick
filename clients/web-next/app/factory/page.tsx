"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { RequireAuth } from "@/components/guards";
import { AiProviderLinks } from "@/components/ai-provider-links";
import { Icon } from "@/components/icons";
import {
  Button,
  ConfirmDialog,
  CopyButton,
  Dialog,
  ErrorBanner,
  Field,
  PageHeader,
  Panel,
  Section,
  Skeleton,
  SuccessBanner,
  checkClass,
  cx,
  inputClass,
  selectClass,
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
    <Dialog
      title={t("pickerTitle")}
      onClose={onClose}
      footer={
        <Button variant="secondary" onClick={onClose}>
          {t("pickerDone")}
        </Button>
      }
    >
      <Field label={t("pickerSearch")}>
        <input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          className={inputClass}
          data-autofocus
        />
      </Field>
      <div className="mt-3 max-h-72 space-y-0.5 overflow-y-auto">
        {visible.length === 0 && <p className="text-sm text-ink-2">{t("pickerEmpty")}</p>}
        {/* shortcut: first 200 matches; the search box narrows further */}
        {visible.slice(0, 200).map((fragment) => {
          const checked = selected.includes(fragment.id);
          return (
            <label
              key={fragment.id}
              className="flex min-h-11 items-start gap-2.5 rounded-ctl px-2 py-1.5 text-sm hover:bg-sunk sm:min-h-0"
            >
              <input
                type="checkbox"
                checked={checked}
                onChange={() => onToggle(fragment.id)}
                className={cx(checkClass, "mt-1")}
              />
              <span className="min-w-0">
                <span className="font-medium text-ink">{fragment.name}</span>
                {fragment.name_en && <span className="ml-2 text-xs text-ink-3">{fragment.name_en}</span>}
                <span className="block truncate text-xs text-ink-3">{fragment.prompt}</span>
              </span>
            </label>
          );
        })}
      </div>
    </Dialog>
  );
}

function Factory() {
  const { t } = useI18n();
  const qc = useQueryClient();
  const promptRef = useRef<HTMLTextAreaElement>(null);
  const sheetPromptRef = useRef<HTMLTextAreaElement>(null);
  const [state, setState] = useState<BuilderState>(blankState);
  const [compiledPrompt, setCompiledPrompt] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [suggesting, setSuggesting] = useState(false);
  const [compiling, setCompiling] = useState(false);
  const [pickerCell, setPickerCell] = useState<number | null>(null);
  const [addGroupId, setAddGroupId] = useState("");
  const [promptSheetOpen, setPromptSheetOpen] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

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
  const loadError = groups.error ?? fragments.error ?? characters.error;

  function retryLoads() {
    void groups.refetch();
    void fragments.refetch();
    void characters.refetch();
  }

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

  // Dropping the id makes the POST path create a new design, leaving the template intact.
  function saveCopy() {
    setNotice(null);
    saveDesign.mutate({ ...draftFromState(state), id: undefined, name: `${state.name.trim() || t("newDesign")} (${t("copySuffix")})` });
  }

  const availableGroups = (groups.data?.items ?? []).filter(
    (group) => !state.groupIds.includes(group.id),
  );
  const cellCount = state.rows * state.columns;
  const pickerFragments = (fragments.data ?? []).filter((f) => !f.archived);
  const activeCell = pickerCell === null ? null : state.cells[pickerCell];
  // The name check is shown beside its field; the banner only carries other errors.
  const nameError = error === t("designNameRequired") && !state.name.trim() ? error : undefined;

  return (
    <>
      <PageHeader
        title={t("factoryTitle")}
        subtitle={t("factorySubtitle")}
        actions={
          <Button
            variant="secondary"
            icon="plus"
            disabled={editorBusy}
            onClick={() => {
              setState(blankState());
              setCompiledPrompt("");
              setError(null);
              setNotice(null);
            }}
          >
            {t("newDesign")}
          </Button>
        }
      />

      <div className="mb-5 flex items-start gap-2.5 rounded-panel border border-line bg-sunk px-3.5 py-2.5 text-sm text-ink-2">
        <Icon name="info" size={16} className="mt-0.5 shrink-0" />
        <p className="min-w-0">{t("externalAiHint")}</p>
      </div>

      {notice && (
        <div className="mb-4">
          <SuccessBanner message={notice} />
        </div>
      )}
      {error && !nameError && (
        <div className="mb-4">
          <ErrorBanner message={error} />
        </div>
      )}

      <Section title={t("savedDesigns")}>
        {designs.isLoading ? (
          <div className="flex gap-2 overflow-hidden" role="status" aria-label={t("loading")}>
            {[0, 1, 2].map((i) => (
              <Skeleton key={i} className="h-16 w-56 shrink-0" />
            ))}
          </div>
        ) : designs.isError ? (
          <div className="flex flex-wrap items-center gap-3">
            <ErrorBanner message={errorText(designs.error)} />
            <Button variant="secondary" size="sm" icon="refresh" onClick={() => void designs.refetch()}>
              {t("retry")}
            </Button>
          </div>
        ) : designs.data && designs.data.length > 0 ? (
          // Horizontal strip: the padding keeps the selection outline inside the scroll box.
          <ul className="-mx-1 flex gap-2 overflow-x-auto px-1 py-1.5">
            {designs.data.map((design) => {
              const current = state.id === design.id;
              return (
                <li key={design.id} className="shrink-0">
                  <button
                    type="button"
                    onClick={() => loadDesign(design)}
                    disabled={!fragments.data || editorBusy}
                    aria-current={current || undefined}
                    className={cx(
                      "block w-56 min-h-11 rounded-panel border bg-panel px-3 py-2 text-left transition-colors md:min-h-0",
                      "disabled:cursor-not-allowed disabled:opacity-45",
                      current ? "cutline border-cut/60" : "border-line hover:border-line-strong hover:bg-sunk",
                    )}
                  >
                    <span className="block truncate text-sm font-medium text-ink">{design.name}</span>
                    <span className="nums block truncate text-xs text-ink-3">
                      {design.rows}×{design.columns} · {formatDate(design.updated_at)}
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>
        ) : (
          <p className="text-sm text-ink-2">{t("noDesigns")}</p>
        )}
      </Section>

      {fragments.isLoading || groups.isLoading ? (
        <div className="mt-5 space-y-4" role="status" aria-label={t("loading")}>
          <Skeleton className="h-32 w-full" />
          <Skeleton className="h-56 w-full" />
        </div>
      ) : (
        <div className="grid gap-x-10 lg:grid-cols-[minmax(0,1fr)_minmax(340px,420px)]">
          <div className="min-w-0">
            {loadError && (
              <div className="mt-5 flex flex-wrap items-center gap-3">
                <ErrorBanner message={errorText(loadError)} />
                <Button variant="secondary" size="sm" icon="refresh" onClick={retryLoads}>
                  {t("retry")}
                </Button>
              </div>
            )}

            <Section title={t("facBasics")}>
              <div className="grid gap-4 sm:grid-cols-3">
                <Field label={t("designName")} error={nameError}>
                  <input
                    value={state.name}
                    onChange={(e) => update({ name: e.target.value })}
                    aria-invalid={nameError ? true : undefined}
                    className={inputClass}
                  />
                </Field>
                <Field label={t("character")}>
                  <select
                    value={state.characterId}
                    onChange={(e) => update({ characterId: e.target.value })}
                    className={selectClass}
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
                    className={selectClass}
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
                    <p className="mb-1.5 text-[13px] font-medium text-ink">{t("paletteRoles")}</p>
                    <ul className="flex flex-wrap gap-x-4 gap-y-2 text-xs text-ink-2">
                      {COLOR_ROLES.map(([role, label]) => (
                        <li key={role} className="inline-flex items-center gap-1.5">
                          <span
                            aria-hidden="true"
                            className="size-4 rounded-[3px] border border-line-strong"
                            style={{ backgroundColor: paletteColors[role] }}
                          />
                          {t(label)} <span className="mono text-ink-3">{paletteColors[role]}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
                <div className="sm:col-span-3">
                  <Field label={t("designFreeText")} hint={t("freeTextHint")}>
                    <textarea
                      value={state.freeText}
                      onChange={(e) => update({ freeText: e.target.value })}
                      rows={2}
                      className={inputClass}
                    />
                  </Field>
                </div>
              </div>
            </Section>

            <Section title={t("groupsTitle")} hint={`${t("groupsHint")} ${t("pinnedHint")}`}>
              <div className="flex flex-wrap gap-2">
                <select
                  value={addGroupId}
                  onChange={(e) => setAddGroupId(e.target.value)}
                  aria-label={t("groupsTitle")}
                  className={cx(selectClass, "sm:max-w-xs")}
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
                <p className="mt-4 text-sm text-ink-2">{t("noSelection")}</p>
              ) : (
                <ul className="mt-4 divide-y divide-line border-y border-line">
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
                        className="grid items-end gap-3 py-3 sm:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)_auto_auto]"
                      >
                        <div className="min-w-0 text-sm">
                          <p className="font-medium text-ink">{group?.name ?? groupId}</p>
                          <p className="text-xs text-ink-2">{group?.description || t("none")}</p>
                        </div>
                        <Field label={t("fragment")}>
                          <select
                            value={selectedId}
                            onChange={(e) => selectFragment(groupId, e.target.value)}
                            className={selectClass}
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
                          variant="secondary"
                          disabled={!selectedId}
                          onClick={() => selectedId && togglePin(selectedId)}
                          className={pinned ? "bg-cut-soft text-cut-ink" : undefined}
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
            </Section>

            <Section title={t("gridTitle")}>
              <div className="grid gap-3 sm:grid-cols-4">
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
                  <input value={cellCount} readOnly className={cx(inputClass, "mono")} />
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
                      className="h-9 w-12 cursor-pointer rounded-ctl border border-line-strong bg-panel p-0.5 max-md:h-11 max-md:w-14"
                    />
                    <code className="mono text-xs text-ink-2">{state.outlineColor}</code>
                  </div>
                </Field>
              </div>

              <div className="mt-5 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                {resizeCells(state.cells, cellCount).map((cell, index) => {
                  const row = Math.floor(index / state.columns) + 1;
                  const column = (index % state.columns) + 1;
                  return (
                    <div key={index} className="rounded-panel border border-line p-3">
                      <p className="mono mb-3 text-xs text-ink-2">
                        {t("cell")} {row}.{column}
                      </p>
                      <div className="space-y-3">
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
                          <p className="mb-1.5 text-[13px] font-medium text-ink">{t("cellFragments")}</p>
                          <div className="mb-2 flex flex-wrap gap-1.5">
                            {cell.fragment_ids.map((fragmentId) => {
                              const name = fragmentsById.get(fragmentId)?.name ?? fragmentId;
                              return (
                                <span
                                  key={fragmentId}
                                  className="inline-flex items-center gap-1 rounded-[3px] border border-line-strong bg-sunk py-0.5 pl-2 pr-0.5 text-xs text-ink"
                                >
                                  {name}
                                  <button
                                    type="button"
                                    aria-label={`${t("removeFragment")}: ${name}`}
                                    onClick={() => {
                                      const cells = [...state.cells];
                                      cells[index] = {
                                        ...cell,
                                        fragment_ids: cell.fragment_ids.filter((id) => id !== fragmentId),
                                      };
                                      update({ cells });
                                    }}
                                    className="inline-flex size-5 items-center justify-center rounded-[3px] text-ink-3 hover:bg-panel hover:text-danger max-md:size-11 max-md:-my-3 max-md:-mr-1"
                                  >
                                    <Icon name="close" size={12} />
                                  </button>
                                </span>
                              );
                            })}
                          </div>
                          <Button variant="secondary" size="sm" onClick={() => setPickerCell(index)}>
                            {t("pickFragments")}
                          </Button>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </Section>

            <Section title={t("actions")}>
              <div className="flex flex-wrap gap-2">
                <Button variant="secondary" loading={suggesting} onClick={() => void suggest()}>
                  {suggesting ? t("suggesting") : t("suggest")}
                </Button>
                <Button variant="secondary" loading={compiling} onClick={() => void compile()}>
                  {compiling ? t("compiling") : t("compile")}
                </Button>
                <Button
                  variant="secondary"
                  loading={saveDesign.isPending}
                  disabled={editorBusy}
                  onClick={save}
                >
                  {saveDesign.isPending ? t("saving") : state.id ? t("updateDesign") : t("saveDesign")}
                </Button>
                {state.id && (
                  <Button variant="secondary" icon="copy" disabled={editorBusy} onClick={saveCopy}>
                    {t("saveCopy")}
                  </Button>
                )}
                {state.id && (
                  <Button
                    variant="danger"
                    icon="trash"
                    onClick={() => setConfirmDelete(true)}
                    disabled={editorBusy}
                  >
                    {t("deleteDesign")}
                  </Button>
                )}
              </div>
            </Section>
          </div>

          {/* Desktop: prompt output pinned beside the composition. */}
          <aside className="hidden py-5 lg:sticky lg:top-6 lg:block lg:self-start">
            <Panel
              title={t("compiledPrompt")}
              actions={<CopyButton text={compiledPrompt} selectRef={promptRef} variant="primary" />}
            >
              <p className="mb-3 text-xs text-ink-2">{t("compiledPromptHint")}</p>
              <textarea
                ref={promptRef}
                readOnly
                value={compiledPrompt}
                placeholder={t("noPrompt")}
                rows={12}
                aria-label={t("compiledPrompt")}
                className={cx(inputClass, "mono text-xs leading-5")}
              />
              <AiProviderLinks prompt={compiledPrompt} />
            </Panel>
          </aside>
        </div>
      )}

      {/* Mobile and tablet: prompt output in a bar above the bottom tabs. */}
      <div className="fixed inset-x-0 bottom-[calc(3.5rem_+_env(safe-area-inset-bottom))] z-20 flex items-center justify-between gap-3 border-t border-line bg-panel/95 px-4 py-2 backdrop-blur md:bottom-0 lg:hidden">
        <Button variant="secondary" icon="docs" onClick={() => setPromptSheetOpen(true)}>
          {t("facViewPrompt")}
        </Button>
        <CopyButton text={compiledPrompt} variant="primary" />
      </div>
      <div aria-hidden="true" className="h-24 lg:hidden" />

      {promptSheetOpen && (
        <Dialog
          title={t("compiledPrompt")}
          onClose={() => setPromptSheetOpen(false)}
          footer={<CopyButton text={compiledPrompt} selectRef={sheetPromptRef} variant="primary" />}
        >
          <p className="mb-3 text-xs text-ink-2">{t("compiledPromptHint")}</p>
          <textarea
            ref={sheetPromptRef}
            readOnly
            value={compiledPrompt}
            placeholder={t("noPrompt")}
            rows={12}
            aria-label={t("compiledPrompt")}
            className={cx(inputClass, "mono text-xs leading-5")}
          />
          <AiProviderLinks prompt={compiledPrompt} />
        </Dialog>
      )}

      {confirmDelete && state.id && (
        <ConfirmDialog
          title={t("deleteDesign")}
          body={t("deleteDesignConfirm")}
          confirmLabel={t("deleteDesign")}
          onClose={() => setConfirmDelete(false)}
          onConfirm={() => {
            const id = state.id;
            setConfirmDelete(false);
            if (id) deleteDesign.mutate(id);
          }}
        />
      )}

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
