package generator

import (
	"fmt"
	"path"
)

func (e *Engine) generateNodeMonolith(tree *FileTree, req GenerateRequest) error {
	specs := nodeMonolithTemplateSpecs(req)
	data := map[string]any{
		"Framework":    req.Framework,
		"Architecture": req.Architecture,
		"Port":         8080,
		"UseDB":        req.Database != "none",
		"UseSQL":       isSQLDB(req.Database),
		"UseORM":       req.UseORM,
		"DBKind":       req.Database,
		"Service":      "app",
	}
	if err := e.renderSpecs(tree, specs, data, ""); err != nil {
		return err
	}

	addFile(tree, "package.json", nodePackageJSON(req.Framework, req.Database, req.UseORM))
	if isEnabled(req.FileToggles.Config) {
		addFile(tree, "src/config/index.js", nodeConfigLoader())
	}
	if req.Features.Logger || isEnabled(req.FileToggles.Logger) {
		addFile(tree, "src/logger/index.js", nodeLogger())
	}
	if req.Features.GlobalError {
		addFile(tree, "src/middleware/error.js", nodeGlobalError())
	}
	if req.Features.SampleTest {
		addFile(tree, "tests/items.test.js", nodeSampleTest(req.Framework))
	}
	addNodeDBBoilerplate(tree, req, "")
	return nil
}

func (e *Engine) generateNodeService(tree *FileTree, req GenerateRequest, svcRoot string, svc ServiceConfig) error {
	specs := nodeMicroserviceTemplateSpecs(req)
	data := map[string]any{
		"Framework":    req.Framework,
		"Architecture": req.Architecture,
		"Port":         svc.Port,
		"UseDB":        req.Database != "none",
		"UseSQL":       isSQLDB(req.Database),
		"UseORM":       req.UseORM,
		"DBKind":       req.Database,
		"Service":      svc.Name,
	}
	if err := e.renderSpecs(tree, specs, data, svcRoot); err != nil {
		return err
	}
	addFile(tree, path.Join(svcRoot, "package.json"), nodePackageJSON(req.Framework, req.Database, req.UseORM))
	addNodeDBBoilerplate(tree, req, svcRoot)
	return nil
}

func nodeMonolithTemplateSpecs(req GenerateRequest) []templateSpec {
	withCRUD := isEnabled(req.FileToggles.ExampleCRUD)
	switch req.Architecture {
	case "clean":
		base := []templateSpec{
			{Template: "node/clean/src/index.tmpl", Output: "src/index.js"},
		}
		if withCRUD {
			return append(base, []templateSpec{
				{Template: "node/clean/src/domain/item.tmpl", Output: "src/domain/item.js"},
				{Template: "node/clean/src/usecases/listItems.tmpl", Output: "src/usecases/listItems.js"},
				{Template: "node/clean/src/controllers/itemController.tmpl", Output: "src/controllers/itemController.js"},
				{Template: "node/clean/src/repositories/itemRepository.tmpl", Output: "src/repositories/itemRepository.js"},
			}...)
		}
		return append(base, []templateSpec{
			{Template: "node/clean/src/domain/ping.tmpl", Output: "src/domain/ping.js"},
			{Template: "node/clean/src/usecases/pingUsecase.tmpl", Output: "src/usecases/pingUsecase.js"},
			{Template: "node/clean/src/controllers/pingController.tmpl", Output: "src/controllers/pingController.js"},
			{Template: "node/clean/src/repositories/pingRepository.tmpl", Output: "src/repositories/pingRepository.js"},
		}...)
	case "hexagonal":
		base := []templateSpec{
			{Template: "node/hexagonal/src/index.tmpl", Output: "src/index.js"},
		}
		if withCRUD {
			return append(base, []templateSpec{
				{Template: "node/hexagonal/src/core/ports/itemRepositoryPort.tmpl", Output: "src/core/ports/itemRepositoryPort.js"},
				{Template: "node/hexagonal/src/core/services/itemService.tmpl", Output: "src/core/services/itemService.js"},
				{Template: "node/hexagonal/src/adapters/primary/http/itemController.tmpl", Output: "src/adapters/primary/http/itemController.js"},
				{Template: "node/hexagonal/src/adapters/secondary/database/itemRepositoryAdapter.tmpl", Output: "src/adapters/secondary/database/itemRepositoryAdapter.js"},
			}...)
		}
		return append(base, []templateSpec{
			{Template: "node/hexagonal/src/core/ports/pingPort.tmpl", Output: "src/core/ports/pingPort.js"},
			{Template: "node/hexagonal/src/core/services/pingService.tmpl", Output: "src/core/services/pingService.js"},
			{Template: "node/hexagonal/src/adapters/primary/http/pingController.tmpl", Output: "src/adapters/primary/http/pingController.js"},
			{Template: "node/hexagonal/src/adapters/secondary/database/pingAdapter.tmpl", Output: "src/adapters/secondary/database/pingAdapter.js"},
		}...)
	default:
		return []templateSpec{{Template: fmt.Sprintf("node/%s/main.tmpl", archTemplateName(req.Architecture)), Output: "src/index.js"}}
	}
}

func nodeMicroserviceTemplateSpecs(_ GenerateRequest) []templateSpec {
	return []templateSpec{{Template: "node/microservice/main.tmpl", Output: "src/index.js"}}
}

func addNodeDBBoilerplate(tree *FileTree, req GenerateRequest, root string) {
	if !isSQLDB(req.Database) {
		return
	}
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	if req.UseORM {
		addFile(tree, prefix+"prisma/schema.prisma", renderPrismaSchema(req.Database, req.Custom.Models))
		addFile(tree, prefix+"src/db/prismaClient.js", "import { PrismaClient } from '@prisma/client';\n\nexport const prisma = new PrismaClient();\n")
		return
	}
	if req.Database == "postgresql" {
		addFile(tree, prefix+"src/db/sqlClient.js", "import pg from 'pg';\n\nconst { Pool } = pg;\nexport const db = new Pool({ connectionString: process.env.DATABASE_URL });\n")
		return
	}
	addFile(tree, prefix+"src/db/sqlClient.js", "import mysql from 'mysql2/promise';\n\nexport const db = await mysql.createConnection(process.env.DATABASE_URL || 'mysql://app:app@mysql:3306/app');\n")
}
