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
}

type ToolBinding struct {
	Endpoint string `json:"endpoint"`
	Mode     string `json:"mode"` // read_only | read_write
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
	WorkflowSteps    []string               `json:"workflow_steps"`
}
