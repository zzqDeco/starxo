package service

import (
	"strings"

	"starxo/internal/agent"
	"starxo/internal/config"
)

func newRuntimeSubagentRegistry(defs []config.SubagentDefinitionConfig) *agent.SubagentRegistry {
	if len(defs) == 0 {
		return agent.DefaultSubagentRegistry()
	}
	converted := make([]agent.SubagentDefinition, 0, len(defs))
	for _, def := range defs {
		name := strings.TrimSpace(def.Name)
		if name == "" {
			continue
		}
		backgroundAllowed := true
		if def.BackgroundAllowed != nil {
			backgroundAllowed = *def.BackgroundAllowed
		}
		converted = append(converted, agent.SubagentDefinition{
			Name:              name,
			Description:       strings.TrimSpace(def.Description),
			Instruction:       strings.TrimSpace(def.Instruction),
			AllowedTools:      append([]string(nil), def.AllowedTools...),
			DefaultIsolation:  strings.TrimSpace(def.DefaultIsolation),
			BackgroundAllowed: backgroundAllowed,
		})
	}
	if len(converted) == 0 {
		return agent.DefaultSubagentRegistry()
	}
	return agent.NewSubagentRegistry(converted)
}
