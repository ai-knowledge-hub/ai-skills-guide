"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import type { RegistryEntry } from "@/lib/registry";
import FilterSelect from "@/components/FilterSelect";

type CatalogClientProps = {
  entries: RegistryEntry[];
  categories: string[];
  tags: string[];
  basePath: "/skills" | "/agents" | "/tools-mcp";
  initial: {
    q: string;
    tag: string;
    category: string;
    runtime: string;
  };
};

export default function CatalogClient({ entries, categories, tags, basePath, initial }: CatalogClientProps) {
  const [draftQ, setDraftQ] = useState(initial.q);
  const [draftTag, setDraftTag] = useState(initial.tag);
  const [draftCategory, setDraftCategory] = useState(initial.category);
  const [draftRuntime, setDraftRuntime] = useState(initial.runtime);

  const [q, setQ] = useState(initial.q);
  const [tag, setTag] = useState(initial.tag);
  const [category, setCategory] = useState(initial.category);
  const [runtime, setRuntime] = useState(initial.runtime);

  const filtered = useMemo(() => {
    const qLower = q.toLowerCase();
    return entries.filter((entry) => {
      if (qLower) {
        const haystack = `${entry.id} ${entry.name} ${entry.description}`.toLowerCase();
        if (!haystack.includes(qLower)) {
          return false;
        }
      }
      if (tag && !entry.tags.includes(tag)) {
        return false;
      }
      if (category && entry.category !== category) {
        return false;
      }
      if (runtime && !entry.runtimes.includes(runtime)) {
        return false;
      }
      return true;
    });
  }, [entries, q, tag, category, runtime]);

  function applyFilters() {
    setQ(draftQ);
    setTag(draftTag);
    setCategory(draftCategory);
    setRuntime(draftRuntime);
  }

  return (
    <>
      <div className="filters">
        <input
          className="input"
          value={draftQ}
          onChange={(event) => setDraftQ(event.target.value)}
          placeholder="Search name, id, description"
        />

        <FilterSelect
          label="Category"
          value={draftCategory}
          options={categories}
          placeholder="All categories"
          onChange={setDraftCategory}
        />

        <FilterSelect
          label="Tag"
          value={draftTag}
          options={tags}
          placeholder="All tags"
          onChange={setDraftTag}
        />

        <FilterSelect
          label="Runtime"
          value={draftRuntime}
          options={["codex", "claude", "generic"]}
          placeholder="All runtimes"
          onChange={setDraftRuntime}
        />

        <button className="button button--accent" type="button" onClick={applyFilters}>
          Apply Filters
        </button>
      </div>

      <p className="meta">{filtered.length} result(s)</p>

      <section className="grid">
        {filtered.map((entry) => (
          <Link key={entry.id} href={`${basePath}/${entry.id}`} className="card">
            <p className="meta">{entry.id}</p>
            <h2>{entry.name}</h2>
            <p>{entry.description}</p>
            <div className="tags">
              {entry.tags.map((tagEntry) => (
                <span key={tagEntry} className="tag">{tagEntry}</span>
              ))}
            </div>
          </Link>
        ))}
      </section>
    </>
  );
}
