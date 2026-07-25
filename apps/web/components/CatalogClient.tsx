"use client";

import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useDeferredValue, useEffect, useMemo, useState } from "react";
import type { RegistryEntry } from "@/lib/registry";
import FilterSelect, { type FilterOption } from "@/components/FilterSelect";
import type { CatalogInitialFilters } from "@/lib/catalogFilters";
import UsabilityBadge from "@/components/UsabilityBadge";
import {
  formatDomainLabel,
  formatCategoryFamily,
  getDomainKey,
  normalizeDomainFilterValues
} from "@/lib/categoryLabels";

type CatalogClientProps = {
  entries: RegistryEntry[];
  categories: string[];
  tags: string[];
  basePath: "/skills" | "/agents" | "/tools-mcp" | "/plugins";
  initial: CatalogInitialFilters;
};

export default function CatalogClient({ entries, categories, tags, basePath, initial }: CatalogClientProps) {
  const isPluginCatalog = basePath === "/plugins";
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [q, setQ] = useState(initial.q);
  const [selectedTags, setSelectedTags] = useState(initial.tags);
  const [selectedCategories, setSelectedCategories] = useState(
    normalizeDomainFilterValues(initial.categories, basePath)
  );
  const [selectedRuntimes, setSelectedRuntimes] = useState(initial.runtimes);
  const deferredQ = useDeferredValue(q);

  const filtered = useMemo(() => {
    const qLower = deferredQ.toLowerCase();
    return entries.filter((entry) => {
      if (qLower) {
        const haystack = `${entry.id} ${entry.name} ${entry.description}`.toLowerCase();
        if (!haystack.includes(qLower)) {
          return false;
        }
      }
      if (selectedTags.length > 0 && !selectedTags.some((tag) => entry.tags.includes(tag))) {
        return false;
      }
      if (selectedCategories.length > 0 && !selectedCategories.includes(getDomainKey(entry.category, basePath))) {
        return false;
      }
      if (selectedRuntimes.length > 0 && !selectedRuntimes.some((runtime) => entry.runtimes.includes(runtime))) {
        return false;
      }
      return true;
    });
  }, [basePath, deferredQ, entries, selectedCategories, selectedRuntimes, selectedTags]);

  const categoryOptions = useMemo<FilterOption[]>(
    () =>
      categories.map((category) => ({
        value: category,
        label: formatDomainLabel(category, basePath),
        count: entries.filter((entry) => getDomainKey(entry.category, basePath) === category).length
      })),
    [basePath, categories, entries]
  );

  const tagOptions = useMemo<FilterOption[]>(
    () =>
      tags.map((tag) => ({
        value: tag,
        label: tag,
        count: entries.filter((entry) => entry.tags.includes(tag)).length
      })),
    [entries, tags]
  );

  const runtimeOptions = useMemo<FilterOption[]>(
    () =>
      ["codex", "claude", "generic"].map((runtime) => ({
        value: runtime,
        label: runtime,
        count: entries.filter((entry) => entry.runtimes.includes(runtime)).length
      })),
    [entries]
  );

  useEffect(() => {
    const handle = window.setTimeout(() => {
      const nextParams = new URLSearchParams();

      if (q.trim()) {
        nextParams.set("q", q.trim());
      }
      for (const tag of selectedTags) {
        nextParams.append("tag", tag);
      }
      for (const category of selectedCategories) {
        nextParams.append("category", category);
      }
      for (const runtime of selectedRuntimes) {
        nextParams.append("runtime", runtime);
      }

      const nextQuery = nextParams.toString();
      const currentQuery = searchParams.toString();
      if (nextQuery === currentQuery) {
        return;
      }

      router.replace(nextQuery ? `${pathname}?${nextQuery}` : pathname);
    }, 180);

    return () => window.clearTimeout(handle);
  }, [pathname, q, router, searchParams, selectedCategories, selectedRuntimes, selectedTags]);

  function resetFilters() {
    setQ("");
    setSelectedTags([]);
    setSelectedCategories([]);
    setSelectedRuntimes([]);
  }

  const activeFilters = [
    ...selectedCategories.map((value) => ({ kind: "category", value })),
    ...selectedTags.map((value) => ({ kind: "tag", value })),
    ...selectedRuntimes.map((value) => ({ kind: "runtime", value }))
  ];

  function removeFilter(kind: "category" | "tag" | "runtime", value: string) {
    if (kind === "category") {
      setSelectedCategories((current) => current.filter((entry) => entry !== value));
    } else if (kind === "tag") {
      setSelectedTags((current) => current.filter((entry) => entry !== value));
    } else {
      setSelectedRuntimes((current) => current.filter((entry) => entry !== value));
    }
  }

  return (
    <>
      <div className="filters">
        <input
          className="input"
          value={q}
          onChange={(event) => setQ(event.target.value)}
          placeholder="Search name, id, description"
        />

        <FilterSelect
          label="Domain"
          values={selectedCategories}
          options={categoryOptions}
          placeholder="All domains"
          onChange={setSelectedCategories}
        />

        <FilterSelect
          label="Tag"
          values={selectedTags}
          options={tagOptions}
          placeholder="All tags"
          onChange={setSelectedTags}
        />

        <FilterSelect
          label="Runtime"
          values={selectedRuntimes}
          options={runtimeOptions}
          placeholder="All runtimes"
          onChange={setSelectedRuntimes}
        />

        <button className="button button--secondary" type="button" onClick={resetFilters}>
          Reset Filters
        </button>
      </div>

      {activeFilters.length > 0 ? (
        <div className="active-filters">
          {activeFilters.map((filter) => (
            <button
              key={`${filter.kind}:${filter.value}`}
              type="button"
              className="active-filter-chip"
              onClick={() => removeFilter(filter.kind as "category" | "tag" | "runtime", filter.value)}
            >
              <span className="meta">{filter.kind}</span>
              <span>{filter.kind === "category" ? formatDomainLabel(filter.value, basePath) : filter.value}</span>
              <span aria-hidden>×</span>
            </button>
          ))}
        </div>
      ) : null}

      <p className="meta">{filtered.length} result(s)</p>

      <section className="grid">
        {filtered.map((entry) => (
          <article key={entry.id} className="card catalog-card">
            <Link href={`${basePath}/${entry.id}`} className="catalog-card-link">
              <div className="catalog-card-head">
                <p className="meta catalog-card-domain">{formatCategoryFamily(entry.category)}</p>
                <UsabilityBadge availability={entry.usability.availability} />
              </div>
              <h2>{entry.name}</h2>
              <p>{entry.description}</p>
            </Link>
            {isPluginCatalog ? (
              <div className="catalog-card-actions">
                <a
                  href={`https://github.com/ai-knowledge-hub/ai-skills-guide/blob/main/plugins/${entry.id}/plugin.yaml`}
                  className="catalog-card-secondary-link"
                  target="_blank"
                  rel="noreferrer"
                >
                  View package file
                </a>
              </div>
            ) : null}
          </article>
        ))}
      </section>
    </>
  );
}
