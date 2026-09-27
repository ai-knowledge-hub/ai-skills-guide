package agents

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

type manifestDocument struct {
	ID            string                 `yaml:"id"`
	Name          string                 `yaml:"name"`
	Description   string                 `yaml:"description"`
	Version       string                 `yaml:"version"`
	ReleasedAt    string                 `yaml:"released_at"`
	Category      string                 `yaml:"category"`
	Tags          []string               `yaml:"tags"`
	Runtimes      []string               `yaml:"runtimes"`
	Dependencies  manifestDependencies   `yaml:"dependencies"`
	Orchestration *OrchestrationContract `yaml:"orchestration"`
}

type manifestDependencies struct {
	Agents []string `yaml:"agents"`
	Skills []string `yaml:"skills"`
	Tools  []string `yaml:"tools"`
}

func ParseAgentManifest(path string) (Manifest, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("open manifest %s: %w", path, err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(payload))
	var document manifestDocument
	if err := decoder.Decode(&document); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest %s: %w", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err != nil {
			return Manifest{}, fmt.Errorf("decode trailing YAML in manifest %s: %w", path, err)
		}
		return Manifest{}, fmt.Errorf("manifest %s contains multiple YAML documents", path)
	}
	if document.ID == "" || document.Name == "" || document.Version == "" {
		return Manifest{}, fmt.Errorf("manifest %s missing required id/name/version", path)
	}
	return Manifest{
		ID: document.ID, Name: document.Name, Description: document.Description,
		Version: document.Version, ReleasedAt: document.ReleasedAt, Category: document.Category,
		Tags: document.Tags, Runtimes: document.Runtimes,
		DependencyAgents: document.Dependencies.Agents,
		DependencySkills: document.Dependencies.Skills,
		DependencyTools:  document.Dependencies.Tools,
		Orchestration:    document.Orchestration,
	}, nil
}
