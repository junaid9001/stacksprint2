package generator

import (
	"context"
	"path"
	"path/filepath"
	"strings"
)

type Engine struct {
	registry *TemplateRegistry
}

func NewEngine(registry *TemplateRegistry) *Engine {
	return &Engine{registry: registry}
}

func (e *Engine) Generate(_ context.Context, req GenerateRequest) (GenerateResponse, error) {
	req = NormalizeConfig(req)
	req, decisions, ruleWarnings := ApplyRuleEngine(req)
	if err := ValidateConfig(req); err != nil {
		return GenerateResponse{}, err
	}

	// Complexity analysis is advisory-only — runs after validation, before generation.
	// It does NOT modify req and does NOT block generation.
	complexityReport := AnalyzeComplexity(req)

	tree, err := GenerateFileTree(req, e)
	if err != nil {
		return GenerateResponse{}, err
	}

	mutWarnings := ApplyMutations(&tree, req.Custom)
	resp, err := BuildScripts(req, tree)
	if err != nil {
		return GenerateResponse{}, err
	}

	allWarnings := append(ruleWarnings, mutWarnings...)
	result := BuildMetadata(&resp, allWarnings, decisions)
	result.ComplexityReport = complexityReport
	return result, nil
}

func NormalizeConfig(req GenerateRequest) GenerateRequest {
	req.Language = strings.ToLower(strings.TrimSpace(req.Language))
	req.Framework = strings.ToLower(strings.TrimSpace(req.Framework))
	req.Architecture = strings.ToLower(strings.TrimSpace(req.Architecture))
	req.Database = strings.ToLower(strings.TrimSpace(req.Database))
	req.Root.Mode = strings.ToLower(strings.TrimSpace(req.Root.Mode))
	if req.Database == "" {
		req.Database = "none"
	}
	if req.Root.Mode == "" {
		req.Root.Mode = "new"
	}
	if req.Root.Mode == "new" && req.Root.Name == "" {
		req.Root.Name = "stacksprint-generated"
	}
	return req
}

func ApplyRuleEngine(req GenerateRequest) (GenerateRequest, []Decision, []Warning) {
	var decisions []Decision
	var warnings []Warning

	if req.Architecture == "microservices" && len(req.Services) == 0 {
		req.Services = []ServiceConfig{{Name: "users", Port: 8081}, {Name: "orders", Port: 8082}}
		decisions = append(decisions, Decision{
			Code:        "DEFAULT_MICROSERVICES_INJECTED",
			Description: "Injected default services for microservice architecture.",
			TriggeredBy: "ApplyRuleEngine",
		})
	}
	if req.Architecture == "microservices" && len(req.Custom.AddServiceNames) > 0 {
		basePort := 8081
		req.Services = req.Services[:0]
		for i, name := range req.Custom.AddServiceNames {
			req.Services = append(req.Services, ServiceConfig{Name: name, Port: basePort + i})
		}
		decisions = append(decisions, Decision{
			Code:        "DYNAMIC_SERVICES_MAPPED",
			Description: "Mapped dynamic custom services to microservice array.",
			TriggeredBy: "ApplyRuleEngine",
		})
	}
	if req.Database == "none" && req.UseORM {
		req.UseORM = false
		decisions = append(decisions, Decision{
			Code:        "ORM_DISABLED_NO_DB",
			Description: "Disabled ORM generation since database was set to none.",
			TriggeredBy: "ApplyRuleEngine",
		})
	}

	if req.Architecture == "mvp" && req.Infra.Kafka {
		warnings = append(warnings, Warning{
			Code:     "MVP_WITH_KAFKA",
			Severity: "warn",
			Message:  "Using a heavy event broker like Kafka within an MVP architecture boundaries is generally anti-pattern.",
			Reason:   "Kafka is optimized for distributed boundaries. An MVP monolith will suffer operational latency.",
		})
	}

	return req, decisions, warnings
}

func ValidateConfig(req GenerateRequest) error {
	return Validate(req)
}

func GenerateFileTree(req GenerateRequest, e *Engine) (FileTree, error) {
	tree := FileTree{Files: map[string]string{}, Dirs: map[string]struct{}{}}
	tree.Dirs["."] = struct{}{}

	ctx := &GenerationContext{
		FileTree:  &tree,
		Warnings:  []Warning{},
		Decisions: []Decision{},
		Registry:  e.registry,
	}

	gen := GetGenerator(req.Language)

	if err := gen.GenerateArchitecture(&req, ctx); err != nil {
		return tree, err
	}
	if err := gen.GenerateModels(&req, ctx); err != nil {
		return tree, err
	}
	if err := gen.GenerateInfra(&req, ctx); err != nil {
		return tree, err
	}
	if err := gen.GenerateDevTools(&req, ctx); err != nil {
		return tree, err
	}

	return tree, nil
}

func ApplyMutations(tree *FileTree, custom CustomOptions) []Warning {
	return applyCustomizations(tree, custom)
}

func BuildMetadata(resp *GenerateResponse, warnings []Warning, decisions []Decision) GenerateResponse {
	uniqueWarnings := make(map[string]Warning)
	for _, w := range warnings {
		uniqueWarnings[w.Code] = w
	}
	for _, w := range resp.Warnings {
		uniqueWarnings[w.Code] = w
	}

	uniqueDecisions := make(map[string]Decision)
	for _, d := range decisions {
		uniqueDecisions[d.Code] = d
	}
	for _, d := range resp.Decisions {
		uniqueDecisions[d.Code] = d
	}

	resp.Warnings = make([]Warning, 0, len(uniqueWarnings))
	for _, w := range uniqueWarnings {
		resp.Warnings = append(resp.Warnings, w)
	}

	resp.Decisions = make([]Decision, 0, len(uniqueDecisions))
	for _, d := range uniqueDecisions {
		resp.Decisions = append(resp.Decisions, d)
	}

	sortWarnings(resp.Warnings)
	sortDecisions(resp.Decisions)

	return *resp
}

func sortWarnings(w []Warning) {
	priority := map[string]int{"error": 3, "warn": 2, "info": 1}
	for i := 0; i < len(w); i++ {
		for j := i + 1; j < len(w); j++ {
			p1 := priority[w[i].Severity]
			p2 := priority[w[j].Severity]
			if p1 < p2 || (p1 == p2 && w[i].Code > w[j].Code) {
				w[i], w[j] = w[j], w[i]
			}
		}
	}
}

func sortDecisions(d []Decision) {
	for i := 0; i < len(d); i++ {
		for j := i + 1; j < len(d); j++ {
			if d[i].Code > d[j].Code {
				d[i], d[j] = d[j], d[i]
			}
		}
	}
}

func addFile(tree *FileTree, p, content string) {
	p = filepath.ToSlash(strings.TrimPrefix(p, "./"))
	tree.Files[p] = content
	d := path.Dir(p)
	for d != "." && d != "/" && d != "" {
		tree.Dirs[d] = struct{}{}
		d = path.Dir(d)
	}
}

func applyCustomizations(tree *FileTree, c CustomOptions) []Warning {
	var warnings []Warning
	for _, d := range c.AddFolders {
		tree.Dirs[filepath.ToSlash(d)] = struct{}{}
	}
	for _, f := range c.AddFiles {
		p := filepath.ToSlash(strings.TrimPrefix(f.Path, "./"))
		if _, exists := tree.Files[p]; exists {
			warnings = append(warnings, Warning{
				Code:     "DUPLICATE_CUSTOM_FILE",
				Severity: "warn",
				Message:  "Attempted to inject duplicate custom file payload.",
				Reason:   p,
			})
			continue
		}
		addFile(tree, p, f.Content)
	}
	for _, d := range c.RemoveFolders {
		d = filepath.ToSlash(d)
		delete(tree.Dirs, d)
		for file := range tree.Files {
			if file == d || strings.HasPrefix(file, d+"/") {
				delete(tree.Files, file)
			}
		}
	}
	for _, f := range c.RemoveFiles {
		delete(tree.Files, filepath.ToSlash(f))
	}
	return warnings
}
