package generator

import (
	"context"
	"fmt"
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
	req, decisions := ApplyRuleEngine(req)
	if err := ValidateConfig(req); err != nil {
		return GenerateResponse{}, err
	}

	tree, err := GenerateFileTree(req, e)
	if err != nil {
		return GenerateResponse{}, err
	}

	warnings := ApplyMutations(&tree, req.Custom)
	resp, err := BuildScripts(req, tree)
	if err != nil {
		return GenerateResponse{}, err
	}

	return BuildMetadata(&resp, warnings, decisions), nil
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

func ApplyRuleEngine(req GenerateRequest) (GenerateRequest, []string) {
	var decisions []string

	if req.Architecture == "microservices" && len(req.Services) == 0 {
		req.Services = []ServiceConfig{{Name: "users", Port: 8081}, {Name: "orders", Port: 8082}}
		decisions = append(decisions, "Injected default services for microservice architecture.")
	}
	if req.Architecture == "microservices" && len(req.Custom.AddServiceNames) > 0 {
		basePort := 8081
		req.Services = req.Services[:0]
		for i, name := range req.Custom.AddServiceNames {
			req.Services = append(req.Services, ServiceConfig{Name: name, Port: basePort + i})
		}
		decisions = append(decisions, "Mapped dynamic custom services to microservice array.")
	}
	if req.Database == "none" && req.UseORM {
		req.UseORM = false
		decisions = append(decisions, "Disabled ORM generation since database was set to none.")
	}

	return req, decisions
}

func ValidateConfig(req GenerateRequest) error {
	return Validate(req)
}

func GenerateFileTree(req GenerateRequest, e *Engine) (FileTree, error) {
	tree := FileTree{Files: map[string]string{}, Dirs: map[string]struct{}{}}
	tree.Dirs["."] = struct{}{}

	if req.Architecture == "microservices" {
		if err := e.generateMicroservices(&tree, req); err != nil {
			return tree, err
		}
	} else {
		if err := e.generateMonolith(&tree, req); err != nil {
			return tree, err
		}
	}

	if isEnabled(req.FileToggles.Compose) {
		addFile(&tree, "docker-compose.yaml", buildCompose(req))
	}
	if isEnabled(req.FileToggles.Env) && req.Architecture != "microservices" {
		addFile(&tree, ".env", buildEnv(req, "", 8080))
	}
	if isEnabled(req.FileToggles.Gitignore) {
		addFile(&tree, ".gitignore", baseGitignore(req.Language))
	}
	if isEnabled(req.FileToggles.Readme) {
		addFile(&tree, "README.md", buildREADME(req))
	}

	if req.Features.GitHubActions {
		addFile(&tree, ".github/workflows/ci.yaml", buildCIPipeline(req))
	}
	if req.Features.Makefile {
		addFile(&tree, "Makefile", buildMakefile(req))
	}
	if req.Features.Swagger {
		addFile(&tree, "docs/openapi.yaml", buildOpenAPI(req))
	}
	if req.Database != "none" {
		addFile(&tree, "migrations/001_initial.sql", sampleMigration(req.Database, req.Custom.Models))
		addFile(&tree, "db/init/001_init.sql", sampleDBInit(req.Database, req.Custom.Models))
	}
	if strings.EqualFold(req.ServiceCommunication, "grpc") {
		addFile(&tree, "proto/README.md", "# Shared proto definitions\n\nPlace your protobuf contracts here.\n")
		addFile(&tree, "proto/common.proto", "syntax = \"proto3\";\npackage stacksprint;\n\nservice InternalService {\n  rpc Ping(PingRequest) returns (PingReply);\n}\n\nmessage PingRequest {\n  string source = 1;\n}\n\nmessage PingReply {\n  string message = 1;\n}\n")
		addGRPCBoilerplate(&tree, req, "")
	}

	return tree, nil
}

func ApplyMutations(tree *FileTree, custom CustomOptions) []string {
	return applyCustomizations(tree, custom)
}

func BuildMetadata(resp *GenerateResponse, warnings, decisions []string) GenerateResponse {
	if len(warnings) > 0 {
		resp.Warnings = append(resp.Warnings, warnings...)
	}
	// Note: Engine doesn't have a `.Decisions` struct exposed to HTTP natively.
	// For now, mapping into Warnings pipeline or logging. (Assuming Warnings slice handles string arrays)
	if len(decisions) > 0 {
		resp.Warnings = append(resp.Warnings, decisions...)
	}
	return *resp
}

func (e *Engine) generateMonolith(tree *FileTree, req GenerateRequest) error {
	switch req.Language {
	case "go":
		if err := e.generateGoMonolith(tree, req); err != nil {
			return err
		}
	case "node":
		if err := e.generateNodeMonolith(tree, req); err != nil {
			return err
		}
	case "python":
		if err := e.generatePythonMonolith(tree, req); err != nil {
			return err
		}
	}

	if req.Database != "none" {
		addDatabaseBoilerplate(tree, req, "")
	}
	if req.Features.JWTAuth {
		addAuthBoilerplate(tree, req, "")
	}
	if isEnabled(req.FileToggles.HealthCheck) || req.Features.Health {
		addHealthBoilerplate(tree, req, "")
	}
	if isEnabled(req.FileToggles.BaseRoute) {
		addBaseRoute(tree, req, "")
	}
	if isEnabled(req.FileToggles.ExampleCRUD) {
		addCRUDRoute(tree, req, "")
	}
	addInfraBoilerplate(tree, req, "")
	addAutopilotBoilerplate(tree, req, "")
	addDBRetry(tree, req, "")
	if isEnabled(req.FileToggles.Dockerfile) {
		addFile(tree, "Dockerfile", dockerfile(req, ""))
	}
	return nil
}

func (e *Engine) generateMicroservices(tree *FileTree, req GenerateRequest) error {
	for _, svc := range req.Services {
		svcRoot := path.Join("services", svc.Name)

		switch req.Language {
		case "go":
			if err := e.generateGoService(tree, req, svcRoot, svc); err != nil {
				return err
			}
			if isEnabled(req.FileToggles.ExampleCRUD) {
				data := map[string]any{
					"Framework":    req.Framework,
					"Architecture": req.Architecture,
					"Port":         svc.Port,
					"UseDB":        req.Database != "none",
					"UseSQL":       isSQLDB(req.Database),
					"UseORM":       req.UseORM,
					"DBKind":       req.Database,
					"Module":       fmt.Sprintf("stacksprint/%s", svc.Name),
					"Service":      svc.Name,
				}
				for _, model := range resolvedModels(req.Custom.Models) {
					e.renderGoOtherDynamicModel(tree, data, model, req.Architecture, svcRoot)
				}
			}
		case "node":
			if err := e.generateNodeService(tree, req, svcRoot, svc); err != nil {
				return err
			}
			if isEnabled(req.FileToggles.ExampleCRUD) {
				for _, model := range resolvedModels(req.Custom.Models) {
					renderNodeDynamicModel(tree, req, model, req.Architecture, svcRoot)
				}
			}
		case "python":
			if err := e.generatePythonService(tree, req, svcRoot, svc); err != nil {
				return err
			}
			if isEnabled(req.FileToggles.ExampleCRUD) {
				for _, model := range resolvedModels(req.Custom.Models) {
					renderPythonDynamicModel(tree, req, model, req.Architecture, svcRoot)
				}
			}
		}

		if isEnabled(req.FileToggles.Env) {
			addFile(tree, path.Join(svcRoot, ".env"), buildEnv(req, svc.Name, svc.Port))
		}
		if isEnabled(req.FileToggles.Dockerfile) {
			addFile(tree, path.Join(svcRoot, "Dockerfile"), dockerfile(req, svc.Name))
		}
		if req.Database != "none" {
			addDatabaseBoilerplate(tree, req, svcRoot)
		}
		if req.Features.JWTAuth {
			addAuthBoilerplate(tree, req, svcRoot)
		}
		if isEnabled(req.FileToggles.HealthCheck) || req.Features.Health {
			addHealthBoilerplate(tree, req, svcRoot)
		}
		if isEnabled(req.FileToggles.BaseRoute) {
			addBaseRoute(tree, req, svcRoot)
		}
		if isEnabled(req.FileToggles.ExampleCRUD) {
			addCRUDRoute(tree, req, svcRoot)
		}
		addInfraBoilerplate(tree, req, svcRoot)
		if strings.EqualFold(req.ServiceCommunication, "grpc") {
			addGRPCBoilerplate(tree, req, svcRoot)
		}
		addAutopilotBoilerplate(tree, req, svcRoot)
		addDBRetry(tree, req, svcRoot)
	}

	if isEnabled(req.FileToggles.Readme) {
		addFile(tree, "README.md", buildREADME(req))
	}
	if isEnabled(req.FileToggles.Gitignore) {
		addFile(tree, ".gitignore", baseGitignore(req.Language))
	}
	return nil
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

func applyCustomizations(tree *FileTree, c CustomOptions) []string {
	var warnings []string
	for _, d := range c.AddFolders {
		tree.Dirs[filepath.ToSlash(d)] = struct{}{}
	}
	for _, f := range c.AddFiles {
		p := filepath.ToSlash(strings.TrimPrefix(f.Path, "./"))
		if _, exists := tree.Files[p]; exists {
			warnings = append(warnings, "Duplicate custom file path detected and ignored: "+p)
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

func baseGitignore(lang string) string {
	base := "# StackSprint\n.env\n*.log\n.DS_Store\n"
	switch lang {
	case "go":
		return base + "bin/\ncoverage.out\n"
	case "node":
		return base + "node_modules/\n"
	default:
		return base + "__pycache__/\n.venv/\n"
	}
}
