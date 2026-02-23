#!/usr/bin/env ruby
# frozen_string_literal: true

require "json"
require "yaml"
require "digest"

ROOT = File.expand_path("..", __dir__)
SKILLS_ROOT = File.join(ROOT, "skills")
OUTPUT_PATH = File.join(ROOT, "registry", "index.json")
BASE_URL = "https://hub.ai-knowledge-hub.org"

def normalized_rel_path(path)
  path.delete_prefix("#{ROOT}/")
end

def collect_skill_files(skill_dir)
  Dir.glob(File.join(skill_dir, "**", "*"), File::FNM_DOTMATCH)
     .select { |p| File.file?(p) }
     .reject { |p| p.include?("/.") }
     .sort
end

def digest_skill(skill_dir)
  hasher = Digest::SHA256.new
  collect_skill_files(skill_dir).each do |path|
    rel = path.delete_prefix("#{skill_dir}/")
    hasher << rel
    hasher << "\0"
    hasher << File.binread(path)
    hasher << "\0"
  end
  hasher.hexdigest
end

def load_manifest(path)
  data = YAML.safe_load(File.read(path), aliases: false)
  raise "invalid manifest format in #{path}" unless data.is_a?(Hash)

  data
end

def build_version_entry(manifest, manifest_rel_path, skill_dir)
  version = manifest.fetch("version")
  skill_id = manifest.fetch("id")
  released_at = manifest.fetch("released_at")
  {
    "version" => version,
    "released_at" => released_at,
    "manifest_url" => "#{BASE_URL}/#{manifest_rel_path}",
    "artifact_url" => "#{BASE_URL}/artifacts/#{skill_id}/#{version}.tar.gz",
    "sha256" => digest_skill(skill_dir)
  }
end

def build_skill_entry(manifest_path)
  manifest = load_manifest(manifest_path)
  skill_dir = File.dirname(manifest_path)
  manifest_rel_path = normalized_rel_path(manifest_path)

  version_entry = build_version_entry(manifest, manifest_rel_path, skill_dir)

  entry = {
    "id" => manifest.fetch("id"),
    "name" => manifest.fetch("name"),
    "description" => manifest.fetch("description"),
    "latest" => manifest.fetch("version"),
    "versions" => [version_entry],
    "runtimes" => manifest.fetch("runtimes"),
    "tags" => manifest.fetch("tags"),
    "deprecated" => manifest.fetch("deprecated", false)
  }
  replaced_by = manifest["replaced_by"]
  entry["replaced_by"] = replaced_by if replaced_by
  entry
end

manifest_paths = Dir.glob(File.join(SKILLS_ROOT, "*", "*", "skill.yaml")).sort
skill_entries = manifest_paths.map { |path| build_skill_entry(path) }
skill_entries.sort_by! { |entry| entry["id"] }
latest_release = skill_entries
                 .flat_map { |entry| entry["versions"].map { |version| version["released_at"] } }
                 .max || "1970-01-01T00:00:00Z"

registry = {
  "registry_version" => "1.0",
  "generated_at" => latest_release,
  "skills" => skill_entries
}

File.write(OUTPUT_PATH, "#{JSON.pretty_generate(registry)}\n")
puts "Wrote #{normalized_rel_path(OUTPUT_PATH)} with #{skill_entries.length} skill(s)."
