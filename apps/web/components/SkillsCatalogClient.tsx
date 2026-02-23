"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import type { SkillEntry } from "@/lib/registry";
import FilterSelect from "@/components/FilterSelect";

type SkillsCatalogClientProps = {
  skills: SkillEntry[];
  categories: string[];
  tags: string[];
  initial: {
    q: string;
    tag: string;
    category: string;
    runtime: string;
  };
};

export default function SkillsCatalogClient({ skills, categories, tags, initial }: SkillsCatalogClientProps) {
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
    return skills.filter((skill) => {
      if (qLower) {
        const haystack = `${skill.id} ${skill.name} ${skill.description}`.toLowerCase();
        if (!haystack.includes(qLower)) {
          return false;
        }
      }
      if (tag && !skill.tags.includes(tag)) {
        return false;
      }
      if (category && skill.category !== category) {
        return false;
      }
      if (runtime && !skill.runtimes.includes(runtime)) {
        return false;
      }
      return true;
    });
  }, [skills, q, tag, category, runtime]);

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
        {filtered.map((skill) => (
          <Link key={skill.id} href={`/skills/${skill.id}`} className="card">
            <p className="meta">{skill.id}</p>
            <h2>{skill.name}</h2>
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
