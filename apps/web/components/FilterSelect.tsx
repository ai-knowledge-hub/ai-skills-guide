"use client";

import { useEffect, useRef, useState } from "react";

export type FilterOption = {
  value: string;
  label: string;
  count?: number;
  group?: string;
};

type FilterSelectProps = {
  label: string;
  values: string[];
  options: FilterOption[];
  placeholder: string;
  onChange: (values: string[]) => void;
};

export default function FilterSelect({
  label,
  values,
  options,
  placeholder,
  onChange
}: FilterSelectProps) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    function onPointerDown(event: MouseEvent) {
      if (!rootRef.current) return;
      if (!rootRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    }

    document.addEventListener("mousedown", onPointerDown);
    return () => document.removeEventListener("mousedown", onPointerDown);
  }, []);

  const summary =
    values.length === 0
      ? placeholder
      : values.length === 1
        ? options.find((option) => option.value === values[0])?.label ?? values[0]
        : `${values.length} selected`;
  const groupedOptions = options.reduce<Record<string, FilterOption[]>>((acc, option) => {
    const key = option.group ?? "";
    if (!acc[key]) {
      acc[key] = [];
    }
    acc[key].push(option);
    return acc;
  }, {});
  const orderedGroups = Object.entries(groupedOptions);

  return (
    <div className="filter-select" ref={rootRef}>
      <button
        type="button"
        className="filter-trigger"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={label}
        onClick={() => setOpen((prev) => !prev)}
      >
        <span>{summary}</span>
        <span className="caret" aria-hidden>
          {open ? "▴" : "▾"}
        </span>
      </button>

      {open ? (
        <ul
          className="filter-menu"
          role="listbox"
          aria-label={label}
          aria-multiselectable="true"
        >
          <li>
            <button
              type="button"
              className={`filter-option ${values.length === 0 ? "is-selected" : ""}`}
              onClick={() => {
                onChange([]);
              }}
            >
              <span className="filter-option-text">{placeholder}</span>
              <span className="filter-option-mark" aria-hidden>
                {values.length === 0 ? "✓" : ""}
              </span>
            </button>
          </li>
          {orderedGroups.map(([group, groupOptions]) => (
            <li key={group || "ungrouped"} className="filter-group">
              {group ? <p className="filter-group-label">{group}</p> : null}
              <ul className="filter-group-list">
                {groupOptions.map((option) => (
                  <li key={option.value}>
                    <button
                      type="button"
                      className={`filter-option ${values.includes(option.value) ? "is-selected" : ""}`}
                      onClick={() => {
                        if (values.includes(option.value)) {
                          onChange(values.filter((entry) => entry !== option.value));
                          return;
                        }
                        onChange([...values, option.value]);
                      }}
                    >
                      <span className="filter-option-text">{option.label}</span>
                      <span className="filter-option-side">
                        {typeof option.count === "number" ? (
                          <span className="filter-option-count">{option.count}</span>
                        ) : null}
                        <span className="filter-option-mark" aria-hidden>
                          {values.includes(option.value) ? "✓" : ""}
                        </span>
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
