package registry

type Manifest struct {
	ID               string
	Name             string
	Description      string
	Version          string
	ReleasedAt       string
	Category         string
	Tags             []string
	Runtimes         []string
	SecurityReviewed bool
	Deprecated       bool
	ReplacedBy       string
	Includes         IncludeSet
	Requires         RequirementSet
}

type IncludeSet struct {
	Skills []string `json:"skills,omitempty"`
	Agents []string `json:"agents,omitempty"`
	Tools  []string `json:"tools,omitempty"`
	Hooks  []string `json:"hooks,omitempty"`
}

type RequirementSet struct {
	Secrets   []string `json:"secrets,omitempty"`
	Approvals []string `json:"approvals,omitempty"`
}

type Index struct {
	RegistryVersion string       `json:"registry_version"`
	GeneratedAt     string       `json:"generated_at"`
	Skills          []SkillEntry `json:"skills"`
}

type SkillEntry struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	Category         string          `json:"category"`
	Latest           string          `json:"latest"`
	Versions         []VersionEntry  `json:"versions"`
	Runtimes         []string        `json:"runtimes"`
	Tags             []string        `json:"tags"`
	Readiness        string          `json:"readiness"`
	SecurityReviewed bool            `json:"security_reviewed"`
	Deprecated       bool            `json:"deprecated"`
	ReplacedBy       string          `json:"replaced_by,omitempty"`
	Includes         *IncludeSet     `json:"includes,omitempty"`
	Requires         *RequirementSet `json:"requires,omitempty"`
}

type VersionEntry struct {
	Version     string `json:"version"`
	ReleasedAt  string `json:"released_at"`
	ManifestURL string `json:"manifest_url"`
	ArtifactURL string `json:"artifact_url"`
	SHA256      string `json:"sha256"`
}
