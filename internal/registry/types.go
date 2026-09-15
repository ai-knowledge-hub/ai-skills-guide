package registry

type Manifest struct {
	SchemaVersion     string                 `json:"schema_version"`
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Version           string                 `json:"version"`
	ReleasedAt        string                 `json:"released_at"`
	Category          string                 `json:"category"`
	Tags              []string               `json:"tags"`
	Runtimes          []string               `json:"runtimes"`
	Entrypoints       map[string]string      `json:"entrypoints"`
	SecurityReviewed  bool                   `json:"-"`
	Deprecated        bool                   `json:"deprecated"`
	ReplacedBy        string                 `json:"replaced_by"`
	Operational       OperationalMetadata    `json:"operational"`
	Usability         UsabilityMetadata      `json:"usability"`
	Execution         ExecutionMetadata      `json:"execution"`
	Artifact          ArtifactMetadata       `json:"artifact"`
	Authentication    AuthenticationMetadata `json:"authentication"`
	Verification      VerificationMetadata   `json:"verification"`
	Dependencies      DependencySet          `json:"dependencies"`
	Includes          IncludeSet             `json:"includes"`
	Requires          RequirementSet         `json:"requires"`
	executionSet      bool
	artifactSet       bool
	authenticationSet bool
	verificationSet   bool
	usabilitySet      bool
}

type ExecutionMetadata struct {
	Kind               string   `json:"kind"`
	Command            []string `json:"command,omitempty"`
	Healthcheck        []string `json:"healthcheck,omitempty"`
	SmokeTest          []string `json:"smoke_test,omitempty"`
	SupportedPlatforms []string `json:"supported_platforms"`
	SupportedRuntimes  []string `json:"supported_runtimes"`
}

type ArtifactMetadata struct {
	SelfContained  bool    `json:"self_contained"`
	DependencyLock *string `json:"dependency_lock"`
	Checksums      *string `json:"checksums"`
	SBOM           *string `json:"sbom"`
}

type AuthenticationMetadata struct {
	Status             string   `json:"status"`
	Methods            []string `json:"methods"`
	CredentialBindings []string `json:"credential_bindings"`
	Scopes             []string `json:"scopes"`
	SetupURL           string   `json:"setup_url,omitempty"`
	CredentialStorage  string   `json:"credential_storage"`
	Validation         string   `json:"validation"`
	Revocation         string   `json:"revocation"`
}

type VerificationMetadata struct {
	Evidence       []string `json:"evidence"`
	LastVerifiedAt string   `json:"last_verified_at"`
}

type UsabilityMetadata struct {
	Availability      string                     `json:"availability"`
	Execution         string                     `json:"execution"`
	RequiresSetup     []string                   `json:"requires_setup,omitempty"`
	Limitations       []string                   `json:"limitations,omitempty"`
	Quickstart        string                     `json:"quickstart,omitempty"`
	ExecutableHelpers []ExecutableHelperMetadata `json:"executable_helpers,omitempty"`
	Source            string                     `json:"source"`
}

type ExecutableHelperMetadata struct {
	Entrypoint   string   `json:"entrypoint"`
	Availability string   `json:"availability"`
	Execution    string   `json:"execution"`
	Limitations  []string `json:"limitations,omitempty"`
	Quickstart   string   `json:"quickstart,omitempty"`
}

type OperationalMetadata struct {
	ConnectedSystem  string   `json:"connected_system,omitempty"`
	Capabilities     []string `json:"capabilities,omitempty"`
	AuthRequired     []string `json:"auth_required,omitempty"`
	AccessLevel      string   `json:"access_level,omitempty"`
	TrustBoundary    string   `json:"trust_boundary,omitempty"`
	ApprovalBoundary string   `json:"approval_boundary,omitempty"`
	Role             string   `json:"role,omitempty"`
	Coordinates      []string `json:"coordinates,omitempty"`
	AutonomyLevel    string   `json:"autonomy_level,omitempty"`
	Outputs          []string `json:"outputs,omitempty"`
	UseWhen          string   `json:"use_when,omitempty"`
	ExecutionMode    string   `json:"execution_mode,omitempty"`
}

type DependencySet struct {
	Agents     []string `json:"agents,omitempty"`
	Skills     []string `json:"skills,omitempty"`
	Tools      []string `json:"tools,omitempty"`
	APIs       []string `json:"apis,omitempty"`
	MCPServers []string `json:"mcp_servers,omitempty"`
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
	SchemaVersion    string                  `json:"schema_version,omitempty"`
	ID               string                  `json:"id"`
	Name             string                  `json:"name"`
	Description      string                  `json:"description"`
	Category         string                  `json:"category"`
	Latest           string                  `json:"latest"`
	Versions         []VersionEntry          `json:"versions"`
	Runtimes         []string                `json:"runtimes"`
	Tags             []string                `json:"tags"`
	Readiness        string                  `json:"readiness"`
	SecurityReviewed bool                    `json:"security_reviewed"`
	Deprecated       bool                    `json:"deprecated"`
	ReplacedBy       string                  `json:"replaced_by,omitempty"`
	Operational      *OperationalMetadata    `json:"operational,omitempty"`
	Usability        UsabilityMetadata       `json:"usability"`
	Execution        *ExecutionMetadata      `json:"execution,omitempty"`
	Artifact         *ArtifactMetadata       `json:"artifact,omitempty"`
	Authentication   *AuthenticationMetadata `json:"authentication,omitempty"`
	Verification     *VerificationMetadata   `json:"verification,omitempty"`
	Dependencies     *DependencySet          `json:"dependencies,omitempty"`
	Includes         *IncludeSet             `json:"includes,omitempty"`
	Requires         *RequirementSet         `json:"requires,omitempty"`
}

type VersionEntry struct {
	Version        string `json:"version"`
	ReleasedAt     string `json:"released_at"`
	ManifestURL    string `json:"manifest_url"`
	ManifestSHA256 string `json:"manifest_sha256,omitempty"`
	ArtifactURL    string `json:"artifact_url"`
	SHA256         string `json:"sha256"`
}
