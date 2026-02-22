package generator

import (
	"fmt"
	"path"
	"strings"
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

	if isEnabled(req.FileToggles.ExampleCRUD) {
		var imports, routes strings.Builder
		for _, model := range resolvedModels(req.Custom.Models) {
			renderNodeDynamicModel(tree, req, model, req.Architecture, "")
			nameLow := strings.ToLower(model.Name)
			if req.Architecture == "clean" {
				imports.WriteString(fmt.Sprintf("import * as %sController from './controllers/%sController.js';\n", nameLow, nameLow))
				routes.WriteString(fmt.Sprintf("app.get('/%ss', %sController.list%ssHandler);\napp.post('/%ss', %sController.create%sHandler);\n", nameLow, nameLow, model.Name, nameLow, nameLow, model.Name))
			} else if req.Architecture == "hexagonal" {
				imports.WriteString(fmt.Sprintf("import { list%ss, get%s, create%s } from './adapters/primary/http/%sController.js';\n", model.Name, model.Name, model.Name, nameLow))
				routes.WriteString(fmt.Sprintf("app.get('/%ss', list%ss);\napp.get('/%ss/:id', get%s);\napp.post('/%ss', create%s);\n", nameLow, model.Name, nameLow, model.Name, nameLow, model.Name))
			} else {
				if req.Framework == "express" {
					imports.WriteString(fmt.Sprintf("import %sRoutes from './routes/%ss.js';\n", nameLow, nameLow))
					routes.WriteString(fmt.Sprintf("app.use('/%ss', %sRoutes);\n", nameLow, nameLow))
				} else {
					imports.WriteString(fmt.Sprintf("import %sRoutes from './routes/%ss.js';\n", nameLow, nameLow))
					routes.WriteString(fmt.Sprintf("app.register(%sRoutes, { prefix: '/%ss' });\n", nameLow, nameLow))
				}
			}
		}
		if main, ok := tree.Files["src/index.js"]; ok {
			if req.Framework == "express" {
				main = strings.Replace(main, "const server = app.listen", imports.String()+"\n"+routes.String()+"\nconst server = app.listen", 1)
			} else {
				main = strings.Replace(main, "await app.listen", imports.String()+"\n"+routes.String()+"\nawait app.listen", 1)
			}
			tree.Files["src/index.js"] = main
		}
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
			return base
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
			return base
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
		addFile(tree, prefix+"prisma/seed.js", renderNodeSeedScript(req.Custom.Models, true))
		return
	}
	if req.Database == "postgresql" {
		addFile(tree, prefix+"src/db/sqlClient.js", "import pg from 'pg';\n\nconst { Pool } = pg;\nexport const db = new Pool({ connectionString: process.env.DATABASE_URL });\n")
		addFile(tree, prefix+"scripts/seed.js", renderNodeSeedScript(req.Custom.Models, false))
		return
	}
	addFile(tree, prefix+"src/db/sqlClient.js", "import mysql from 'mysql2/promise';\n\nexport const db = await mysql.createConnection(process.env.DATABASE_URL || 'mysql://app:app@mysql:3306/app');\n")
	addFile(tree, prefix+"scripts/seed.js", renderNodeSeedScript(req.Custom.Models, false))
}

func renderNodeSeedScript(models []DataModel, useORM bool) string {
	var b strings.Builder
	if useORM {
		b.WriteString("import { PrismaClient } from '@prisma/client';\n")
		b.WriteString("const prisma = new PrismaClient();\n\n")
		b.WriteString("async function main() {\n")
		for _, m := range resolvedModels(models) {
			low := strings.ToLower(m.Name)
			sample := buildNodeSampleObject(m)
			b.WriteString(fmt.Sprintf("  await prisma.%s.create({ data: %s });\n", low, sample))
		}
		b.WriteString("}\n\nmain().catch(console.error).finally(() => prisma.$disconnect());\n")
		return b.String()
	}

	b.WriteString("import { db } from '../src/db/sqlClient.js';\n\nasync function main() {\n")
	for _, m := range resolvedModels(models) {
		table := strings.ToLower(m.Name) + "s"
		b.WriteString(fmt.Sprintf("  // Raw SQL seeding for %s (Implementation depends on the exact driver args)\n", table))
		b.WriteString(fmt.Sprintf("  console.log('Seeding %s');\n", table))
	}
	b.WriteString("  console.log('Done');\n  process.exit(0);\n}\n\nmain().catch(console.error);\n")
	return b.String()
}

// renderNodeDynamicModel generates per-model JS files for Clean, Hexagonal, MVP, Modular, and Microservice architectures.
func renderNodeDynamicModel(tree *FileTree, req GenerateRequest, model DataModel, arch, root string) {
	_ = req
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	name := model.Name
	nameLow := strings.ToLower(name)
	sample := buildNodeSampleObject(model)

	switch arch {
	case "clean":
		addFile(tree, prefix+"src/domain/"+nameLow+".js", buildNodeDomainClass(name, model))
		addFile(tree, prefix+"src/usecases/list"+name+"s.js",
			"import { "+name+"Repository } from '../repositories/"+nameLow+"Repository.js';\n\n"+
				"export async function list"+name+"s() {\n  return new "+name+"Repository().findAll();\n}\n")
		addFile(tree, prefix+"src/controllers/"+nameLow+"Controller.js",
			"import { list"+name+"s } from '../usecases/list"+name+"s.js';\n\n"+
				"export async function list"+name+"sHandler(req, res) { res.json(await list"+name+"s()); }\n"+
				"export async function create"+name+"Handler(req, res) { res.status(201).json({ ...req.body, id: Date.now() }); }\n")
		addFile(tree, prefix+"src/repositories/"+nameLow+"Repository.js",
			"export class "+name+"Repository {\n"+
				"  async findAll() { return ["+sample+"]; }\n"+
				"  async findById(id) { return { id: Number(id) }; }\n"+
				"  async create(data) { return { id: Date.now(), ...data }; }\n}\n")

	case "hexagonal":
		addFile(tree, prefix+"src/core/ports/"+nameLow+"RepositoryPort.js",
			"/** @interface "+name+"RepositoryPort\n"+
				" *  findAll():Promise<"+name+"[]>\n"+
				" *  findById(id:number):Promise<"+name+"|null>\n"+
				" *  create(data):Promise<"+name+">\n */\n")
		addFile(tree, prefix+"src/core/services/"+nameLow+"Service.js",
			"export class "+name+"Service {\n"+
				"  constructor(repo) { this.repo = repo; }\n"+
				"  listAll() { return this.repo.findAll(); }\n"+
				"  getById(id) { return this.repo.findById(id); }\n"+
				"  create(data) { return this.repo.create(data); }\n}\n")
		addFile(tree, prefix+"src/adapters/primary/http/"+nameLow+"Controller.js",
			"import { "+name+"Service } from '../../../core/services/"+nameLow+"Service.js';\n"+
				"import { "+name+"RepositoryAdapter } from '../../secondary/database/"+nameLow+"RepositoryAdapter.js';\n\n"+
				"const svc = new "+name+"Service(new "+name+"RepositoryAdapter());\n\n"+
				"export const list"+name+"s = async (req, res) => res.json(await svc.listAll());\n"+
				"export const get"+name+" = async (req, res) => res.json(await svc.getById(req.params.id));\n"+
				"export const create"+name+" = async (req, res) => res.status(201).json(await svc.create(req.body));\n")
		addFile(tree, prefix+"src/adapters/secondary/database/"+nameLow+"RepositoryAdapter.js",
			"export class "+name+"RepositoryAdapter {\n"+
				"  async findAll() { return ["+sample+"]; }\n"+
				"  async findById(id) { return { id: Number(id) }; }\n"+
				"  async create(data) { return { id: Date.now(), ...data }; }\n}\n")

	default: // mvp, modular-monolith, microservices — flat route per model
		if req.Framework == "fastify" {
			addFile(tree, prefix+"src/routes/"+nameLow+"s.js",
				"export default async function (fastify, opts) {\n"+
					"  fastify.get('/', async (request, reply) => {\n"+
					"    const limit = Math.min(Number(request.query.limit) || 20, 100);\n"+
					"    const offset = Number(request.query.offset) || 0;\n"+
					"    return { limit, offset, data: ["+sample+"] };\n  });\n\n"+
					"  fastify.get('/:id', async (request, reply) => ({ id: request.params.id }));\n"+
					"  fastify.post('/', async (request, reply) => {\n"+
					"    reply.code(201);\n"+
					"    return { id: Date.now(), ...request.body };\n  });\n"+
					"  fastify.put('/:id', async (request, reply) => ({ id: request.params.id, ...request.body }));\n"+
					"  fastify.delete('/:id', async (request, reply) => ({ deleted: request.params.id }));\n"+
					"}\n")
		} else {
			addFile(tree, prefix+"src/routes/"+nameLow+"s.js",
				"import { Router } from 'express';\nconst router = Router();\n\n"+
					"router.get('/', (req, res) => {\n"+
					"  const limit = Math.min(Number(req.query.limit) || 20, 100);\n"+
					"  const offset = Number(req.query.offset) || 0;\n"+
					"  res.json({ limit, offset, data: ["+sample+"] });\n});\n\n"+
					"router.get('/:id', (req, res) => res.json({ id: req.params.id }));\n"+
					"router.post('/', (req, res) => res.status(201).json({ id: Date.now(), ...req.body }));\n"+
					"router.put('/:id', (req, res) => res.json({ id: req.params.id, ...req.body }));\n"+
					"router.delete('/:id', (req, res) => res.json({ deleted: req.params.id }));\n\n"+
					"export default router;\n")
		}
	}
}

func buildNodeDomainClass(name string, model DataModel) string {
	var b strings.Builder
	params := make([]string, 0, len(model.Fields))
	for _, f := range model.Fields {
		params = append(params, strings.ToLower(f.Name))
	}
	b.WriteString("export class " + name + " {\n  constructor(" + strings.Join(params, ", ") + ") {\n")
	for _, f := range model.Fields {
		fn := strings.ToLower(f.Name)
		b.WriteString("    this." + fn + " = " + fn + ";\n")
	}
	b.WriteString("  }\n}\n")
	return b.String()
}

func buildNodeSampleObject(model DataModel) string {
	var b strings.Builder
	b.WriteString("{")
	for i, f := range model.Fields {
		if i > 0 {
			b.WriteString(", ")
		}
		fn := strings.ToLower(f.Name)
		switch strings.ToLower(f.Type) {
		case "int", "integer":
			b.WriteString(fn + ": 1")
		case "float", "float64", "double":
			b.WriteString(fn + ": 1.0")
		case "bool", "boolean":
			b.WriteString(fn + ": true")
		default:
			b.WriteString(fn + ": 'sample'")
		}
	}
	b.WriteString("}")
	return b.String()
}
