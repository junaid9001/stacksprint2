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
	req = normalize(req)
	if err := Validate(req); err != nil {
		return GenerateResponse{}, err
	}

	tree := FileTree{Files: map[string]string{}, Dirs: map[string]struct{}{}}
	tree.Dirs["."] = struct{}{}

	if err := e.generateCore(&tree, req); err != nil {
		return GenerateResponse{}, err
	}
	applyCustomizations(&tree, req.Custom)

	return BuildScripts(req, tree)
}

func normalize(req GenerateRequest) GenerateRequest {
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
	if req.Root.Mode == "new" && !req.Root.GitInit {
		req.Root.GitInit = true
	}
	if req.Architecture == "microservices" && len(req.Services) == 0 {
		req.Services = []ServiceConfig{{Name: "users", Port: 8081}, {Name: "orders", Port: 8082}}
	}
	if req.Architecture == "microservices" && len(req.Custom.AddServiceNames) > 0 {
		basePort := 8081
		req.Services = req.Services[:0]
		for i, name := range req.Custom.AddServiceNames {
			req.Services = append(req.Services, ServiceConfig{Name: name, Port: basePort + i})
		}
	}
	return req
}

func (e *Engine) generateCore(tree *FileTree, req GenerateRequest) error {
	if req.Architecture == "microservices" {
		if err := e.generateMicroservices(tree, req); err != nil {
			return err
		}
	} else {
		if err := e.generateMonolith(tree, req); err != nil {
			return err
		}
	}

	if isEnabled(req.FileToggles.Compose) {
		addFile(tree, "docker-compose.yaml", buildCompose(req))
	}
	if isEnabled(req.FileToggles.Env) && req.Architecture != "microservices" {
		addFile(tree, ".env", buildEnv(req, "", 8080))
	}
	if isEnabled(req.FileToggles.Gitignore) {
		addFile(tree, ".gitignore", baseGitignore(req.Language))
	}
	if isEnabled(req.FileToggles.Readme) {
		addFile(tree, "README.md", buildREADME(req))
	}

	if req.Features.GitHubActions {
		addFile(tree, ".github/workflows/ci.yaml", buildCIPipeline(req))
	}
	if req.Features.Makefile {
		addFile(tree, "Makefile", buildMakefile(req))
	}
	if req.Features.Swagger {
		addFile(tree, "docs/openapi.yaml", buildOpenAPI(req))
	}
	if req.Database != "none" {
		addFile(tree, "migrations/001_initial.sql", sampleMigration(req.Database))
		addFile(tree, "db/init/001_init.sql", sampleDBInit(req.Database))
	}
	if strings.EqualFold(req.ServiceCommunication, "grpc") {
		addFile(tree, "proto/README.md", "# Shared proto definitions\n\nPlace your protobuf contracts here.\n")
		addFile(tree, "proto/common.proto", "syntax = \"proto3\";\npackage stacksprint;\n\nservice InternalService {\n  rpc Ping(PingRequest) returns (PingReply);\n}\n\nmessage PingRequest {\n  string source = 1;\n}\n\nmessage PingReply {\n  string message = 1;\n}\n")
		addGRPCBoilerplate(tree, req, "")
	}
	return nil
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
		case "node":
			if err := e.generateNodeService(tree, req, svcRoot, svc); err != nil {
				return err
			}
		case "python":
			if err := e.generatePythonService(tree, req, svcRoot, svc); err != nil {
				return err
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

func applyCustomizations(tree *FileTree, c CustomOptions) {
	for _, d := range c.AddFolders {
		tree.Dirs[filepath.ToSlash(d)] = struct{}{}
	}
	for _, f := range c.AddFiles {
		addFile(tree, f.Path, f.Content)
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
