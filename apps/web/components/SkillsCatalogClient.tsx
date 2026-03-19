"use client";

import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useMemo, useState } from "react";
import type { SkillEntry } from "@/lib/registry";
import FilterSelect, { type FilterOption } from "@/components/FilterSelect";
import type { CatalogInitialFilters } from "@/lib/catalogFilters";
import { formatCategoryLabel } from "@/lib/categoryLabels";

type SkillsCatalogClientProps = {
  skills: SkillEntry[];
  categories: string[];
  tags: string[];
  initial: CatalogInitialFilters;
};

export default function SkillsCatalogClient({ skills, categories, tags, initial }: SkillsCatalogClientProps) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [draftQ, setDraftQ] = useState(initial.q);
  const [draftTags, setDraftTags] = useState(initial.tags);
  const [draftCategories, setDraftCategories] = useState(initial.categories);
  const [draftRuntimes, setDraftRuntimes] = useState(initial.runtimes);

  const [q, setQ] = useState(initial.q);
  const [selectedTags, setSelectedTags] = useState(initial.tags);
  const [selectedCategories, setSelectedCategories] = useState(initial.categories);
  const [selectedRuntimes, setSelectedRuntimes] = useState(initial.runtimes);

  const filtered = useMemo(() => {
    const qLower = q.toLowerCase();
    return skills.filter((skill) => {
      if (qLower) {
        const haystack = `${skill.id} ${skill.name} ${skill.description}`.toLowerCase();
        if (!haystack.includes(qLower)) {
          return false;
        }
      }
      if (selectedTags.length > 0 && !selectedTags.some((tag) => skill.tags.includes(tag))) {
        return false;
      }
      if (selectedCategories.length > 0 && !selectedCategories.includes(skill.category)) {
        return false;
      }
      if (selectedRuntimes.length > 0 && !selectedRuntimes.some((runtime) => skill.runtimes.includes(runtime))) {
        return false;
      }
      return true;
    });
  }, [skills, q, selectedCategories, selectedRuntimes, selectedTags]);

  const categoryOptions = useMemo<FilterOption[]>(
    () =>
      categories.map((category) => ({
        value: category,
        label: formatCategoryLabel(category),
        count: skills.filter((skill) => skill.category === category).length,
        group: groupCategory(category)
      })),
    [categories, skills]
  );

  const tagOptions = useMemo<FilterOption[]>(
    () =>
      tags.map((tag) => ({
        value: tag,
        label: tag,
        count: skills.filter((skill) => skill.tags.includes(tag)).length
      })),
    [skills, tags]
  );

  const runtimeOptions = useMemo<FilterOption[]>(
    () =>
      ["codex", "claude", "generic"].map((runtime) => ({
        value: runtime,
        label: runtime,
        count: skills.filter((skill) => skill.runtimes.includes(runtime)).length
      })),
    [skills]
  );

  function applyFilters() {
    setQ(draftQ);
    setSelectedTags(draftTags);
    setSelectedCategories(draftCategories);
    setSelectedRuntimes(draftRuntimes);

    const nextParams = new URLSearchParams(searchParams.toString());
    nextParams.delete("q");
    nextParams.delete("tag");
    nextParams.delete("category");
    nextParams.delete("runtime");

    if (draftQ.trim()) {
      nextParams.set("q", draftQ.trim());
    }
    for (const tag of draftTags) {
      nextParams.append("tag", tag);
    }
    for (const category of draftCategories) {
      nextParams.append("category", category);
    }
    for (const runtime of draftRuntimes) {
      nextParams.append("runtime", runtime);
    }

    const query = nextParams.toString();
    router.replace(query ? `${pathname}?${query}` : pathname);
  }

  function resetFilters() {
    setDraftQ("");
    setDraftTags([]);
    setDraftCategories([]);
    setDraftRuntimes([]);
    setQ("");
    setSelectedTags([]);
    setSelectedCategories([]);
    setSelectedRuntimes([]);
    router.replace(pathname);
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
          values={draftCategories}
          options={categoryOptions}
          placeholder="All categories"
          onChange={setDraftCategories}
        />

        <FilterSelect
          label="Tag"
          values={draftTags}
          options={tagOptions}
          placeholder="All tags"
          onChange={setDraftTags}
        />

        <FilterSelect
          label="Runtime"
          values={draftRuntimes}
          options={runtimeOptions}
          placeholder="All runtimes"
          onChange={setDraftRuntimes}
        />

        <button className="button button--accent" type="button" onClick={applyFilters}>
          Apply Filters
        </button>

        <button className="button button--secondary" type="button" onClick={resetFilters}>
          Reset Filters
        </button>
      </div>

      <p className="meta">{filtered.length} result(s)</p>

      <section className="grid">
        {filtered.map((skill) => (
          <Link key={skill.id} href={`/skills/${skill.id}`} className="card">
            <p className="meta">{skill.id}</p>
            <h2>{skill.name}</h2>
            <p className="meta">{formatCategoryLabel(skill.category)}</p>
            <p>{skill.description}</p>
            <div className="tags">
              {skill.tags.map((entry) => (
                <span key={entry} className="tag">{entry}</span>
              ))}
            </div>
          </Link>
        ))}
      </section>
    </>
  );
}

function groupCategory(category: string) {
  const [family] = category.split("/");
  if (family === "marketing-tools") return "Marketing Tools";
  if (family === "adtech") return "Adtech";
  if (family === "engineering") return "Engineering";
  if (family === "security") return "Security";
  if (family === "agentops") return "AgentOps";
  return "Other";
}
