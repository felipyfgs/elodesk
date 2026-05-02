package service

import "backend/internal/dto"

// tkPtr is a tiny helper used for terminal_kind pointers in template literals.
func tkPtr(s string) *string { return &s }

var pipelineTemplates = []dto.PipelineTemplate{
	{
		Key:         "sales-crm",
		Name:        "Vendas / CRM",
		Description: "Acompanhe leads do primeiro contato ao fechamento.",
		Icon:        "i-lucide-target",
		Color:       "#6366f1",
		CardKind:    "contact",
		Stages: []dto.PipelineTemplateStage{
			{Name: "Lead", Color: "#94a3b8"},
			{Name: "Qualificado", Color: "#3b82f6"},
			{Name: "Proposta", Color: "#f59e0b"},
			{Name: "Negociação", Color: "#f97316"},
			{Name: "Ganho", Color: "#16a34a", IsTerminal: true, TerminalKind: tkPtr("won")},
			{Name: "Perdido", Color: "#ef4444", IsTerminal: true, TerminalKind: tkPtr("lost")},
		},
	},
	{
		Key:         "support",
		Name:        "Suporte",
		Description: "Workflow de tickets do recebimento à resolução.",
		Icon:        "i-lucide-life-buoy",
		Color:       "#0ea5e9",
		CardKind:    "conversation",
		Stages: []dto.PipelineTemplateStage{
			{Name: "Novo", Color: "#94a3b8"},
			{Name: "Triagem", Color: "#3b82f6"},
			{Name: "Em atendimento", Color: "#f59e0b"},
			{Name: "Aguardando cliente", Color: "#a855f7"},
			{Name: "Resolvido", Color: "#16a34a", IsTerminal: true, TerminalKind: tkPtr("resolved")},
		},
	},
	{
		Key:         "tasks",
		Name:        "Tarefas",
		Description: "Quadro de tarefas genérico estilo Trello.",
		Icon:        "i-lucide-list-checks",
		Color:       "#10b981",
		CardKind:    "free",
		Stages: []dto.PipelineTemplateStage{
			{Name: "Backlog", Color: "#94a3b8"},
			{Name: "A fazer", Color: "#3b82f6"},
			{Name: "Fazendo", Color: "#f59e0b"},
			{Name: "Revisão", Color: "#a855f7"},
			{Name: "Concluído", Color: "#16a34a"},
		},
	},
	{
		Key:         "blank",
		Name:        "Em branco",
		Description: "Comece do zero e desenhe seu próprio fluxo.",
		Icon:        "i-lucide-plus",
		Color:       "#94a3b8",
		CardKind:    "free",
		Stages: []dto.PipelineTemplateStage{
			{Name: "Coluna 1", Color: "#94a3b8"},
		},
	},
}

// ListTemplates returns the catalogue of pre-configured pipeline templates.
func ListTemplates() []dto.PipelineTemplate {
	out := make([]dto.PipelineTemplate, len(pipelineTemplates))
	copy(out, pipelineTemplates)
	return out
}

// GetTemplate looks up a template by key. Returns (template, ok=true) when found.
func GetTemplate(key string) (dto.PipelineTemplate, bool) {
	for _, t := range pipelineTemplates {
		if t.Key == key {
			return t, true
		}
	}
	return dto.PipelineTemplate{}, false
}
