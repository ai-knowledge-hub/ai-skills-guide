"use client";

import { useEffect, useId, useRef, useState } from "react";

export type FilterOption = { value: string; label: string; count?: number; group?: string };
type Props = { label: string; values: string[]; options: FilterOption[]; placeholder: string; onChange: (values: string[]) => void };

export default function FilterSelect({ label, values, options, placeholder, onChange }: Props) {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const id = useId();
  useEffect(() => {
    if (!open) return;
    function outside(event: PointerEvent) {
      if (!root.current?.contains(event.target as Node)) setOpen(false);
    }
    document.addEventListener("pointerdown", outside);
    return () => document.removeEventListener("pointerdown", outside);
  }, [open]);

  const summary = values.length === 0 ? placeholder : values.length === 1
    ? options.find(option => option.value === values[0])?.label ?? values[0] : `${values.length} selected`;
  const groups = options.reduce<Record<string, FilterOption[]>>((result, option) => {
    (result[option.group ?? ""] ??= []).push(option);
    return result;
  }, {});

  return (
    <div className="filter-select" ref={root} onKeyDown={event => {
      if (event.key === "Escape") { setOpen(false); trigger.current?.focus(); }
    }} onBlur={event => {
      if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setOpen(false);
    }}>
      <span className="filter-trigger-label" id={`${id}-label`}>{label}</span>
      <button ref={trigger} type="button" className="filter-trigger" aria-expanded={open} aria-controls={`${id}-options`} aria-label={label} onClick={() => setOpen(current => !current)}>
        <span>{summary}</span><span aria-hidden="true">{open ? "−" : "+"}</span>
      </button>
      {open ? <div className="filter-menu" id={`${id}-options`}>
        <button type="button" className="filter-clear" onClick={() => onChange([])}>Clear {label.toLowerCase()}</button>
        {Object.entries(groups).map(([group, groupOptions]) => (
          <fieldset key={group} aria-labelledby={`${id}-label`}>
            {group ? <legend>{group}</legend> : null}
            {groupOptions.map(option => <label className="filter-option" key={option.value}>
              <input type="checkbox" checked={values.includes(option.value)} onChange={() => onChange(values.includes(option.value) ? values.filter(value => value !== option.value) : [...values, option.value])} />
              <span>{option.label}</span>{typeof option.count === "number" ? <span className="filter-option-count" aria-hidden="true">{option.count}</span> : null}
            </label>)}
          </fieldset>
        ))}
        <button type="button" className="button button--secondary filter-close" onClick={() => { setOpen(false); trigger.current?.focus(); }}>Done</button>
      </div> : null}
    </div>
  );
}
