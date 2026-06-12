package registry

import "time"

type AgentManifest struct {
	ID           string                `json:"id"`
	DID          string                `json:"did"`
	Name         string                `json:"name"`
	Version      string                `json:"version"`
	Description  string                `json:"description,omitempty"`
	Category     string                `json:"category,omitempty"`
	Capabilities []CapabilityManifest  `json:"capabilities,omitempty"`
	Tasks        []TaskManifest        `json:"tasks,omitempty"`
	Endpoints    []EndpointManifest    `json:"endpoints,omitempty"`
	Requirements []RequirementManifest `json:"requirements,omitempty"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

type CapabilityManifest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Inputs      []string `json:"inputs,omitempty"`
}

type TaskManifest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Inputs      []string `json:"inputs,omitempty"`
}

type EndpointManifest struct {
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
}

type RequirementManifest struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

func (m AgentManifest) Normalize() AgentManifest {
	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
	if m.Capabilities == nil {
		m.Capabilities = []CapabilityManifest{}
	}
	if m.Tasks == nil {
		m.Tasks = []TaskManifest{}
	}
	if m.Endpoints == nil {
		m.Endpoints = []EndpointManifest{}
	}
	if m.Requirements == nil {
		m.Requirements = []RequirementManifest{}
	}
	return m
}
