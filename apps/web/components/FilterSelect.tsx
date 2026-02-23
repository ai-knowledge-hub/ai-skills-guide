"use client";

import { useEffect, useRef, useState } from "react";

type FilterSelectProps = {
  label: string;
  value: string;
  options: string[];
  placeholder: string;
  onChange: (value: string) => void;
};

export default function FilterSelect({ label, value, options, placeholder, onChange }: FilterSelectProps) {
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
        <span>{value || placeholder}</span>
        <span className="caret" aria-hidden>
          {open ? "▴" : "▾"}
        </span>
      </button>

      {open ? (
        <ul className="filter-menu" role="listbox" aria-label={label}>
          <li>
            <button
              type="button"
              className={`filter-option ${value === "" ? "is-selected" : ""}`}
              onClick={() => {
                onChange("");
                setOpen(false);
              }}
            >
              {placeholder}
            </button>
          </li>
          {options.map((option) => (
            <li key={option}>
              <button
                type="button"
                className={`filter-option ${value === option ? "is-selected" : ""}`}
                onClick={() => {
                  onChange(option);
                  setOpen(false);
                }}
              >
                {option}
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
