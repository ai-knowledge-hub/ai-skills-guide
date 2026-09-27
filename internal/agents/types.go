package agents

type Manifest struct {
	ID               string
	Name             string
	Description      string
	Version          string
	ReleasedAt       string
	Category         string
	Tags             []string
	Runtimes         []string
	DependencyAgents []string
	DependencySkills []string
	DependencyTools  []string
	Orchestration    *OrchestrationContract
}

type OrchestrationContract struct {
	Model      ModelRequirement   `json:"model" yaml:"model"`
	Memory     ProfileRequirement `json:"memory" yaml:"memory"`
	Governance ProfileRequirement `json:"governance" yaml:"governance"`
	Bindings   []PackageBinding   `json:"bindings" yaml:"bindings"`
}

type ModelRequirement struct {
	Selection            string   `json:"selection" yaml:"selection"`
	RequiredCapabilities []string `json:"required_capabilities" yaml:"required_capabilities"`
}

type ProfileRequirement struct {
	Mode     string `json:"mode" yaml:"mode"`
	Path     string `json:"path,omitempty" yaml:"path,omitempty"`
	Required bool   `json:"required" yaml:"required"`
}

type PackageBinding struct {
	Name         string `json:"name" yaml:"name"`
	Kind         string `json:"kind" yaml:"kind"`
	Package      string `json:"package,omitempty" yaml:"package,omitempty"`
	Version      string `json:"version,omitempty" yaml:"version,omitempty"`
	Requirement  string `json:"requirement" yaml:"requirement"`
	Access       string `json:"access,omitempty" yaml:"access,omitempty"`
	Availability string `json:"availability" yaml:"availability"`
	Reason       string `json:"reason,omitempty" yaml:"reason,omitempty"`
}

type ToolBinding struct {
	Package       string `json:"package,omitempty"`
	Version       string `json:"version,omitempty"`
	Requirement   string `json:"requirement,omitempty"`
	Endpoint      string `json:"endpoint"`
	Mode          string `json:"mode"` // read_only | read_write
	Status        string `json:"status,omitempty"`
	Reason        string `json:"reason,omitempty"`
	ConfigureHint string `json:"configure_hint,omitempty"`
}

type ModelReadiness struct {
	Selection            string   `json:"selection,omitempty"`
	Status               string   `json:"status"` // not-applicable | unverified | ready
	Identity             string   `json:"identity,omitempty"`
	Version              string   `json:"version,omitempty"`
	EvidenceReference    string   `json:"evidence_reference,omitempty"`
	RequiredCapabilities []string `json:"required_capabilities,omitempty"`
	AttestedCapabilities []string `json:"attested_capabilities,omitempty"`
}

type ToolBindingsFile struct {
	Tools map[string]ToolBinding `json:"tools"`
}

type MemoryProfile struct {
	ContextFiles  []string `json:"context_files"`
	StateStore    string   `json:"state_store"`
	RetentionDays int      `json:"retention_days"`
}

type GovernanceProfile struct {
	RequireApprovalForLive bool `json:"require_approval_for_live"`
	AllowWriteTools        bool `json:"allow_write_tools"`
	MaxToolCalls           int  `json:"max_tool_calls"`
}

type RunReport struct {
	AgentID          string                 `json:"agent_id"`
	Status           string                 `json:"status"` // ready | blocked | failed
	Checks           []string               `json:"checks"`
	Warnings         []string               `json:"warnings"`
	BlockingReasons  []string               `json:"blocking_reasons"`
	DependencyAgents []string               `json:"dependency_agents"`
	DependencySkills []string               `json:"dependency_skills"`
	DependencyTools  []string               `json:"dependency_tools"`
	ResolvedTools    map[string]ToolBinding `json:"resolved_tools"`
	Model            ModelReadiness         `json:"model"`
	WorkflowSteps    []string               `json:"workflow_steps"`
}
