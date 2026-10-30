"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useDeferredValue, useEffect, useMemo, useRef, useState } from "react";
import type { RegistryEntry } from "@/lib/registry";
import FilterSelect, { type FilterOption } from "@/components/FilterSelect";
import { parseMultiValue, type CatalogInitialFilters } from "@/lib/catalogFilters";
import UsabilityBadge from "@/components/UsabilityBadge";
import { formatDomainLabel, formatCategoryFamily, getDomainKey, normalizeDomainFilterValues } from "@/lib/categoryLabels";
import { brand } from "@/lib/brand";

type Props = { entries: RegistryEntry[]; categories: string[]; tags: string[]; basePath: "/skills" | "/agents" | "/tools-mcp" | "/plugins"; initial: CatalogInitialFilters };
type FilterKind = "category" | "tag" | "runtime";

export default function CatalogClient({ entries, categories, tags, basePath, initial }: Props) {
  const params = useSearchParams();
  const query = params.get("q") ?? "";
  const [q, setQ] = useState(initial.q);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const deferredQ = useDeferredValue(q);
  const selectedTags = parseMultiValue(params.getAll("tag"));
  const selectedCategories = normalizeDomainFilterValues(parseMultiValue(params.getAll("category")), basePath);
  const selectedRuntimes = parseMultiValue(params.getAll("runtime"));

  // URL state is authoritative for filters and browser back/forward. Native history
  // updates avoid a server round-trip for data already present in the client.
  useEffect(() => {
    if (timer.current) clearTimeout(timer.current);
    setQ(query);
  }, [query]);
  useEffect(() => () => { if (timer.current) clearTimeout(timer.current); }, []);

  function updateUrl(updates: Record<string, string[]>) {
    const next = new URLSearchParams(window.location.search);
    for (const [key, values] of Object.entries(updates)) {
      next.delete(key);
      values.forEach(value => next.append(key, value));
    }
    const suffix = next.toString();
    window.history.replaceState(null, "", `${window.location.pathname}${suffix ? `?${suffix}` : ""}${window.location.hash}`);
  }
  function search(value: string) {
    setQ(value);
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => updateUrl({ q: value.trim() ? [value.trim()] : [] }), 180);
  }
  function select(kind: FilterKind, values: string[]) {
    if (timer.current) clearTimeout(timer.current);
    updateUrl({ [kind]: values, q: q.trim() ? [q.trim()] : [] });
  }
  function resetFilters() {
    if (timer.current) clearTimeout(timer.current);
    setQ("");
    updateUrl({ q: [], tag: [], category: [], runtime: [] });
  }

  const filtered = entries.filter(entry => {
    const needle = deferredQ.trim().toLowerCase();
    return (!needle || `${entry.id} ${entry.name} ${entry.description}`.toLowerCase().includes(needle))
      && (!selectedTags.length || selectedTags.some(tag => entry.tags.includes(tag)))
      && (!selectedCategories.length || selectedCategories.includes(getDomainKey(entry.category, basePath)))
      && (!selectedRuntimes.length || selectedRuntimes.some(runtime => entry.runtimes.includes(runtime)));
  });
  const categoryOptions = useMemo<FilterOption[]>(() => categories.map(category => ({ value: category, label: formatDomainLabel(category, basePath), count: entries.filter(entry => getDomainKey(entry.category, basePath) === category).length })), [categories, entries, basePath]);
  const tagOptions = useMemo<FilterOption[]>(() => tags.map(tag => ({ value: tag, label: tag, count: entries.filter(entry => entry.tags.includes(tag)).length })), [tags, entries]);
  const runtimeOptions = useMemo<FilterOption[]>(() => ["codex", "claude", "generic"].map(runtime => ({ value: runtime, label: runtime, count: entries.filter(entry => entry.runtimes.includes(runtime)).length })), [entries]);
  const activeFilters: { kind: FilterKind; value: string; values: string[] }[] = [
    ...selectedCategories.map(value => ({ kind: "category" as const, value, values: selectedCategories })),
    ...selectedTags.map(value => ({ kind: "tag" as const, value, values: selectedTags })),
    ...selectedRuntimes.map(value => ({ kind: "runtime" as const, value, values: selectedRuntimes }))
  ];

  return <>
    <div className="filters" role="search" aria-label="Filter catalogue">
      <label className="search-label">Search the catalogue<input type="search" className="input" value={q} onChange={event => search(event.target.value)} placeholder="Name, ID or description" /></label>
      <FilterSelect label="Domain" values={selectedCategories} options={categoryOptions} placeholder="All domains" onChange={values => select("category", values)} />
      <FilterSelect label="Tag" values={selectedTags} options={tagOptions} placeholder="All tags" onChange={values => select("tag", values)} />
      <FilterSelect label="Runtime" values={selectedRuntimes} options={runtimeOptions} placeholder="All runtimes" onChange={values => select("runtime", values)} />
      <button className="button button--secondary" type="button" onClick={resetFilters}>Reset Filters</button>
    </div>
    {activeFilters.length ? <div className="active-filters" aria-label="Active filters">
      {activeFilters.map(filter => <button key={`${filter.kind}:${filter.value}`} type="button" className="active-filter-chip" aria-label={`Remove ${filter.kind} ${filter.value}`} onClick={() => select(filter.kind, filter.values.filter(value => value !== filter.value))}>
        <span className="meta">{filter.kind}</span><span>{filter.kind === "category" ? formatDomainLabel(filter.value, basePath) : filter.value}</span><span aria-hidden="true">×</span>
      </button>)}
    </div> : null}
    <p className="meta results-count" role="status">{filtered.length} result(s)</p>
    {filtered.length ? <section className="grid" aria-label="Catalogue results">
      {filtered.map(entry => <article key={entry.id} className="card catalog-card">
        <Link href={`${basePath}/${entry.id}`} className="catalog-card-link">
          <div className="catalog-card-head"><p className="meta catalog-card-domain">{formatCategoryFamily(entry.category)}</p><UsabilityBadge availability={entry.usability.availability} /></div>
          <h2>{entry.name}</h2><p>{entry.description}</p>
          <p className="meta">Execution: {entry.usability.execution}</p>
          {entry.usability.limitations?.[0] ? <p className="meta">Limit: {entry.usability.limitations[0]}</p> : null}
        </Link>
        {basePath === "/plugins" ? <div className="catalog-card-actions"><a href={`${brand.repository}/blob/main/plugins/${entry.id}/plugin.yaml`} className="catalog-card-secondary-link" target="_blank" rel="noreferrer">View package file</a></div> : null}
      </article>)}
    </section> : <section className="empty-results"><h2>No matching entries</h2><p>Try a different search or clear the filters to see the full catalogue.</p><button className="button button--secondary" onClick={resetFilters} type="button">Clear search and filters</button></section>}
  </>;
}
