package generator

import (
	"fmt"
	"path"
	"strings"
)

type templateSpec struct {
	Template string
	Output   string
}

func (e *Engine) generateGoMonolith(tree *FileTree, req GenerateRequest) error {
	module := resolveGoModule(req.Root, "stacksprint/generated")
	specs := goMonolithTemplateSpecs(req)
	data := map[string]any{
		"Framework":    req.Framework,
		"Architecture": req.Architecture,
		"Port":         8080,
		"UseDB":        req.Database != "none",
		"UseSQL":       isSQLDB(req.Database),
		"UseORM":       req.UseORM,
		"DBKind":       req.Database,
		"Module":       module,
		"Service":      "app",
	}
	if err := e.renderSpecs(tree, specs, data, ""); err != nil {
		return err
	}
	addFile(tree, "go.mod", goModV2(req.Framework, req.Root, req.Database, req.UseORM, strings.EqualFold(req.ServiceCommunication, "grpc")))
	if isEnabled(req.FileToggles.Config) {
		addFile(tree, "internal/config/config.go", goConfigLoader())
	}
	if req.Features.Logger || isEnabled(req.FileToggles.Logger) {
		addFile(tree, "internal/logger/logger.go", goLogger())
	}
	if req.Features.GlobalError {
		addFile(tree, "internal/middleware/error.go", goGlobalErrorMiddleware(req.Framework))
	}
	if req.Features.SampleTest {
		addFile(tree, "internal/handlers/item_handler_test.go", goSampleTest())
	}
	if strings.EqualFold(req.ServiceCommunication, "grpc") {
		addGRPCBoilerplate(tree, req, "")
	}
	return nil
}

func (e *Engine) generateGoService(tree *FileTree, req GenerateRequest, svcRoot string, svc ServiceConfig) error {
	module := fmt.Sprintf("stacksprint/%s", svc.Name)
	specs := goMicroserviceTemplateSpecs(req)
	data := map[string]any{
		"Framework":    req.Framework,
		"Architecture": req.Architecture,
		"Port":         svc.Port,
		"UseDB":        req.Database != "none",
		"UseSQL":       isSQLDB(req.Database),
		"UseORM":       req.UseORM,
		"DBKind":       req.Database,
		"Module":       module,
		"Service":      svc.Name,
	}
	if err := e.renderSpecs(tree, specs, data, svcRoot); err != nil {
		return err
	}
	addFile(tree, path.Join(svcRoot, "go.mod"), goModV2(req.Framework, RootOptions{Module: module}, req.Database, req.UseORM, strings.EqualFold(req.ServiceCommunication, "grpc")))
	return nil
}

func (e *Engine) renderSpecs(tree *FileTree, specs []templateSpec, data map[string]any, root string) error {
	for _, spec := range specs {
		body, err := e.registry.Render(spec.Template, data)
		if err != nil {
			return err
		}
		out := spec.Output
		if root != "" {
			out = path.Join(root, out)
		}
		addFile(tree, out, body)
	}
	return nil
}

func goMonolithTemplateSpecs(req GenerateRequest) []templateSpec {
	withCRUD := isEnabled(req.FileToggles.ExampleCRUD)
	switch req.Architecture {
	case "clean":
		base := []templateSpec{
			{Template: "go/clean/cmd/server/main.tmpl", Output: "cmd/server/main.go"},
		}
		if withCRUD {
			return append(base, []templateSpec{
				{Template: "go/clean/internal/domain/item.tmpl", Output: "internal/domain/item.go"},
				{Template: "go/clean/internal/usecase/item_usecase.tmpl", Output: "internal/usecase/item_usecase.go"},
				{Template: "go/clean/internal/delivery/http/item_handler.tmpl", Output: "internal/delivery/http/item_handler.go"},
				{Template: "go/clean/internal/repository/item_repository.tmpl", Output: "internal/repository/item_repository.go"},
			}...)
		}
		return append(base, []templateSpec{
			{Template: "go/clean/internal/domain/ping.tmpl", Output: "internal/domain/ping.go"},
			{Template: "go/clean/internal/usecase/ping_usecase.tmpl", Output: "internal/usecase/ping_usecase.go"},
			{Template: "go/clean/internal/delivery/http/ping_handler.tmpl", Output: "internal/delivery/http/ping_handler.go"},
			{Template: "go/clean/internal/repository/ping_repository.tmpl", Output: "internal/repository/ping_repository.go"},
		}...)
	case "hexagonal":
		base := []templateSpec{
			{Template: "go/hexagonal/cmd/server/main.tmpl", Output: "cmd/server/main.go"},
		}
		if withCRUD {
			return append(base, []templateSpec{
				{Template: "go/hexagonal/core/ports/item_repository.tmpl", Output: "core/ports/item_repository.go"},
				{Template: "go/hexagonal/core/services/item_service.tmpl", Output: "core/services/item_service.go"},
				{Template: "go/hexagonal/adapters/primary/http/item_handler.tmpl", Output: "adapters/primary/http/item_handler.go"},
				{Template: "go/hexagonal/adapters/secondary/database/item_repository.tmpl", Output: "adapters/secondary/database/item_repository.go"},
			}...)
		}
		return append(base, []templateSpec{
			{Template: "go/hexagonal/core/ports/ping_port.tmpl", Output: "core/ports/ping_port.go"},
			{Template: "go/hexagonal/core/services/ping_service.tmpl", Output: "core/services/ping_service.go"},
			{Template: "go/hexagonal/adapters/primary/http/ping_handler.tmpl", Output: "adapters/primary/http/ping_handler.go"},
			{Template: "go/hexagonal/adapters/secondary/database/ping_repository.tmpl", Output: "adapters/secondary/database/ping_repository.go"},
		}...)
	case "modular-monolith":
		return []templateSpec{
			{Template: "go/modular/cmd/server/main.tmpl", Output: "cmd/server/main.go"},
			{Template: "go/modular/internal/modules/catalog/module.tmpl", Output: "internal/modules/catalog/module.go"},
			{Template: "go/modular/internal/modules/catalog/http.tmpl", Output: "internal/modules/catalog/http.go"},
		}
	default:
		base := []templateSpec{
			{Template: "go/mvp/cmd/server/main.tmpl", Output: "cmd/server/main.go"},
		}
		if withCRUD {
			return append(base, []templateSpec{
				{Template: "go/mvp/internal/handlers/item_handler.tmpl", Output: "internal/handlers/item_handler.go"},
			}...)
		}
		return append(base, templateSpec{Template: "go/mvp/internal/handlers/ping_handler.tmpl", Output: "internal/handlers/ping_handler.go"})
	}
}

func goMicroserviceTemplateSpecs(req GenerateRequest) []templateSpec {
	base := []templateSpec{
		{Template: "go/microservice/cmd/server/main.tmpl", Output: "cmd/server/main.go"},
	}
	if isEnabled(req.FileToggles.ExampleCRUD) {
		return append(base, []templateSpec{
			{Template: "go/microservice/internal/handlers/item_handler.tmpl", Output: "internal/handlers/item_handler.go"},
		}...)
	}
	return append(base, templateSpec{Template: "go/microservice/internal/handlers/ping_handler.tmpl", Output: "internal/handlers/ping_handler.go"})
}

func isSQLDB(db string) bool {
	return db == "postgresql" || db == "mysql"
}
