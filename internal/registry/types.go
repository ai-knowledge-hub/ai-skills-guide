package registry

type Manifest struct {
	ID          string
	Name        string
	Description string
	Version     string
	ReleasedAt  string
	Category    string
	Tags        []string
	Runtimes    []string
	Deprecated  bool
	ReplacedBy  string
}

type Index struct {
	RegistryVersion string       `json:"registry_version"`
	GeneratedAt     string       `json:"generated_at"`
	Skills          []SkillEntry `json:"skills"`
}

type SkillEntry struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Category    string         `json:"category"`
	Latest      string         `json:"latest"`
	Versions    []VersionEntry `json:"versions"`
	Runtimes    []string       `json:"runtimes"`
	Tags        []string       `json:"tags"`
	Deprecated  bool           `json:"deprecated"`
	ReplacedBy  string         `json:"replaced_by,omitempty"`
}

type VersionEntry struct {
	Version     string `json:"version"`
	ReleasedAt  string `json:"released_at"`
	ManifestURL string `json:"manifest_url"`
	ArtifactURL string `json:"artifact_url"`
	SHA256      string `json:"sha256"`
}
