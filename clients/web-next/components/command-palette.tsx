"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { Dialog, Kbd, cx } from "@/components/ui";
import { Icon, type IconName } from "@/components/icons";
import { useI18n, type MessageKey } from "@/lib/i18n";

export interface Command {
  id: string;
  label: MessageKey;
  group: "go" | "cmd";
  icon: IconName;
  run: () => void;
}

/** Raycast-style palette: type to filter, arrows to move, Enter to run. */
export function CommandPalette({ commands, onClose }: { commands: Command[]; onClose: () => void }) {
  const { t } = useI18n();
  const [query, setQuery] = useState("");
  const [index, setIndex] = useState(0);
  const listRef = useRef<HTMLUListElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // The native dialog opens in a parent effect, so focus after it is open.
  useEffect(() => {
    const id = window.setTimeout(() => inputRef.current?.focus(), 0);
    return () => window.clearTimeout(id);
  }, []);

  const results = useMemo(() => {
    const q = query.trim().toLowerCase();
    return commands.filter((c) => !q || t(c.label).toLowerCase().includes(q));
  }, [commands, query, t]);

  useEffect(() => {
    listRef.current?.children[index]?.scrollIntoView({ block: "nearest" });
  }, [index]);

  function run(command: Command | undefined) {
    if (!command) return;
    onClose();
    command.run();
  }

  function onKey(event: React.KeyboardEvent) {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setIndex((i) => Math.min(i + 1, results.length - 1));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setIndex((i) => Math.max(i - 1, 0));
    } else if (event.key === "Enter") {
      event.preventDefault();
      run(results[index]);
    }
  }

  return (
    <Dialog title={t("cmdTitle")} onClose={onClose} size="sm">
      <div onKeyDown={onKey} className="-mx-1">
        <div className="relative">
          <Icon name="search" size={16} className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-ink-3" />
          <input
            ref={inputRef}
            role="combobox"
            aria-expanded="true"
            aria-controls="cmd-list"
            aria-activedescendant={results[index] ? `cmd-${results[index].id}` : undefined}
            value={query}
            onChange={(event) => {
              setQuery(event.target.value);
              setIndex(0);
            }}
            placeholder={t("cmdPlaceholder")}
            className="block min-h-11 w-full rounded-ctl border border-line-strong bg-panel pl-9 pr-3 text-sm text-ink placeholder:text-ink-3 focus:border-cut focus:outline-none focus:ring-2 focus:ring-cut/25"
          />
        </div>
        <ul id="cmd-list" ref={listRef} role="listbox" className="mt-2 max-h-72 overflow-y-auto">
          {results.length === 0 && <li className="px-3 py-6 text-center text-sm text-ink-2">{t("cmdEmpty")}</li>}
          {results.map((command, i) => (
            <li
              key={command.id}
              id={`cmd-${command.id}`}
              role="option"
              aria-selected={i === index}
              onMouseMove={() => setIndex(i)}
              onClick={() => run(command)}
              className={cx(
                "flex min-h-10 cursor-pointer items-center gap-3 rounded-ctl px-3 text-sm",
                i === index ? "bg-cut-soft text-cut-ink" : "text-ink",
              )}
            >
              <Icon name={command.icon} size={16} />
              <span className="flex-1">{t(command.label)}</span>
              <span className="text-xs text-ink-3">{command.group === "go" ? t("cmdGoTo") : t("cmdCommands")}</span>
            </li>
          ))}
        </ul>
        <p className="mt-3 flex items-center gap-1.5 border-t border-line pt-3 text-xs text-ink-3">
          <Kbd>↑</Kbd>
          <Kbd>↓</Kbd>
          <Kbd>Enter</Kbd>
        </p>
      </div>
    </Dialog>
  );
}
