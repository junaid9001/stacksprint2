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
	if isEnabled(req.FileToggles.ExampleCRUD) {
		for _, model := range resolvedModels(req.Custom.Models) {
			if req.Architecture == "clean" {
				if err := e.renderGoCleanDynamicModel(tree, data, model, ""); err != nil {
					return err
				}
			} else {
				if err := e.renderGoOtherDynamicModel(tree, data, model, req.Architecture, ""); err != nil {
					return err
				}
			}
		}
	}
	if req.Database != "none" {
		addFile(tree, "cmd/seeder/main.go", renderGoSeederScript(module, req.Custom.Models, req.UseORM))
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
			return base
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

func (e *Engine) renderGoCleanDynamicModel(tree *FileTree, baseData map[string]any, model DataModel, root string) error {
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	modelNameLower := strings.ToLower(model.Name)
	specs := []templateSpec{
		{Template: "go/clean/internal/domain/dynamic.tmpl", Output: "internal/domain/" + modelNameLower + ".go"},
		{Template: "go/clean/internal/usecase/dynamic.tmpl", Output: "internal/usecase/" + modelNameLower + "_usecase.go"},
		{Template: "go/clean/internal/repository/dynamic.tmpl", Output: "internal/repository/" + modelNameLower + "_repository.go"},
		{Template: "go/clean/internal/delivery/http/dynamic.tmpl", Output: "internal/delivery/http/" + modelNameLower + "_handler.go"},
	}

	type goTemplateField struct {
		Name     string
		Type     string
		JSONName string
	}
	type goTemplateModel struct {
		Name   string
		Fields []goTemplateField
	}
	templModel := goTemplateModel{Name: model.Name, Fields: make([]goTemplateField, 0, len(model.Fields))}
	for _, field := range model.Fields {
		if strings.EqualFold(field.Name, "id") {
			continue
		}
		templModel.Fields = append(templModel.Fields, goTemplateField{
			Name:     toPascal(field.Name),
			Type:     goType(field.Type),
			JSONName: strings.ToLower(field.Name),
		})
	}

	for _, spec := range specs {
		data := make(map[string]any, len(baseData)+1)
		for k, v := range baseData {
			data[k] = v
		}
		data["Model"] = templModel

		body, err := e.registry.Render(spec.Template, data)
		if err != nil {
			return err
		}
		addFile(tree, prefix+spec.Output, body)
	}
	return nil
}

func (e *Engine) renderGoOtherDynamicModel(tree *FileTree, baseData map[string]any, model DataModel, arch, root string) error {
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	modelNameLower := strings.ToLower(model.Name)
	var specs []templateSpec

	switch arch {
	case "hexagonal":
		specs = []templateSpec{
			{Template: "go/hexagonal/internal/core/ports/dynamic_port.tmpl", Output: "internal/core/ports/" + modelNameLower + "_port.go"},
			{Template: "go/hexagonal/internal/core/services/dynamic_service.tmpl", Output: "internal/core/services/" + modelNameLower + "_service.go"},
			{Template: "go/hexagonal/internal/adapters/primary/http/dynamic_handler.tmpl", Output: "internal/adapters/primary/http/" + modelNameLower + "_handler.go"},
			{Template: "go/hexagonal/internal/adapters/secondary/database/dynamic_adapter.tmpl", Output: "internal/adapters/secondary/database/" + modelNameLower + "_adapter.go"},
		}
	case "modular-monolith":
		specs = []templateSpec{
			{Template: "go/modular/internal/modules/dynamic/http.tmpl", Output: "internal/modules/" + modelNameLower + "/http.go"},
			{Template: "go/modular/internal/modules/dynamic/service.tmpl", Output: "internal/modules/" + modelNameLower + "/service.go"},
			{Template: "go/modular/internal/modules/dynamic/repository.tmpl", Output: "internal/modules/" + modelNameLower + "/repository.go"},
		}
	default: // mvp / microservice
		specs = []templateSpec{
			{Template: "go/mvp/internal/handlers/dynamic_handler.tmpl", Output: "internal/handlers/" + modelNameLower + "_handler.go"},
		}
	}

	type goTemplateField struct {
		Name     string
		Type     string
		JSONName string
	}
	type goTemplateModel struct {
		Name   string
		Fields []goTemplateField
	}
	templModel := goTemplateModel{Name: model.Name, Fields: make([]goTemplateField, 0, len(model.Fields))}
	for _, field := range model.Fields {
		if strings.EqualFold(field.Name, "id") {
			continue
		}
		templModel.Fields = append(templModel.Fields, goTemplateField{
			Name:     toPascal(field.Name),
			Type:     goType(field.Type),
			JSONName: strings.ToLower(field.Name),
		})
	}

	for _, spec := range specs {
		data := make(map[string]any, len(baseData)+1)
		for k, v := range baseData {
			data[k] = v
		}
		data["Model"] = templModel

		body, err := e.registry.Render(spec.Template, data)
		// We ignore missing template errors here because we haven't created these dynamic templates yet!
		// But if they exist, we render them.
		if err == nil {
			addFile(tree, prefix+spec.Output, body)
		}
	}
	return nil
}

func renderGoSeederScript(module string, models []DataModel, useORM bool) string {
	var b strings.Builder
	b.WriteString("package main\n\nimport (\n")
	b.WriteString(fmt.Sprintf("\t\"%s/internal/db\"\n", module))
	if useORM {
		b.WriteString(fmt.Sprintf("\t\"%s/internal/models\"\n", module))
	}
	b.WriteString("\t\"log\"\n)\n\nfunc main() {\n")
	b.WriteString("\tconn, err := db.Connect()\n\tif err != nil {\n\t\tlog.Fatalf(\"Failed to connect: %v\", err)\n\t}\n")

	if useORM {
		b.WriteString("\tlog.Println(\"Seeding database with GORM...\")\n")
		for _, m := range resolvedModels(models) {
			b.WriteString(fmt.Sprintf("\n\t// Seed %s\n\tconn.Create(&models.%s{})\n", m.Name, m.Name))
		}
	} else {
		b.WriteString("\tdefer conn.Close()\n\tlog.Println(\"Seeding database with raw SQL...\")\n")
		for _, m := range resolvedModels(models) {
			table := strings.ToLower(m.Name) + "s"
			b.WriteString(fmt.Sprintf("\n\t// Seed %s\n\t_, err = conn.Exec(\"INSERT INTO %s DEFAULT VALUES\")\n", table, table))
			b.WriteString(fmt.Sprintf("\tif err != nil { log.Printf(\"Error seeding %s: %%v\", err) }\n", table))
		}
	}
	b.WriteString("\tlog.Println(\"Seeding complete.\")\n}\n")
	return b.String()
}
