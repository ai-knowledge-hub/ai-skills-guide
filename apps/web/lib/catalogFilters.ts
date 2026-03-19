export type SearchParamValue = string | string[] | undefined;

export type CatalogInitialFilters = {
  q: string;
  tags: string[];
  categories: string[];
  runtimes: string[];
};

export function parseMultiValue(value: SearchParamValue) {
  if (Array.isArray(value)) {
    return [...new Set(value.flatMap((entry) => splitValue(entry)))];
  }
  return [...new Set(splitValue(value ?? ""))];
}

function splitValue(value: string) {
  return value
    .split(",")
    .map((entry) => entry.trim())
    .filter(Boolean);
}

