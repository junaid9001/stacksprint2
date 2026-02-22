package generator

import (
	"fmt"
	"path"
	"strings"
)

type NodeGenerator struct{}

func (g *NodeGenerator) GenerateArchitecture(req *GenerateRequest, ctx *GenerationContext) error {
	if req.Architecture == "microservices" {
		for _, svc := range req.Services {
			svcRoot := path.Join("services", svc.Name)
			if err := g.generateServiceArch(req, ctx, svcRoot, svc); err != nil {
				return err
			}
			if isEnabled(req.FileToggles.BaseRoute) {
				addFile(ctx.FileTree, path.Join(svcRoot, "src/routes/base.js"), "export const basePath = '/api/v1';\n")
			}
			if isEnabled(req.FileToggles.ExampleCRUD) {
				addFile(ctx.FileTree, path.Join(svcRoot, "src/routes/items.js"), "export function listItems(req, res) {\n  res.json([{ id: 1, name: 'sample' }]);\n}\n")
			}
			if isEnabled(req.FileToggles.HealthCheck) || req.Features.Health {
				addFile(ctx.FileTree, path.Join(svcRoot, "src/routes/health.js"), "export default function health(req, res) { res.send({ status: 'ok' }); }\n")
			}
			if req.Features.JWTAuth {
				addFile(ctx.FileTree, path.Join(svcRoot, "src/auth/jwt.js"), "export const jwtSecret = process.env.JWT_SECRET || 'changeme';\n")
			}
			if strings.EqualFold(req.ServiceCommunication, "grpc") {
				g.addGRPCBoilerplate(ctx.FileTree, req, svcRoot)
			}
		}
	} else {
		if err := g.generateMonolithArch(req, ctx, ""); err != nil {
			return err
		}
		if isEnabled(req.FileToggles.BaseRoute) {
			addFile(ctx.FileTree, "src/routes/base.js", "export const basePath = '/api/v1';\n")
		}
		if isEnabled(req.FileToggles.ExampleCRUD) {
			addFile(ctx.FileTree, "src/routes/items.js", "export function listItems(req, res) {\n  res.json([{ id: 1, name: 'sample' }]);\n}\n")
		}
		if isEnabled(req.FileToggles.HealthCheck) || req.Features.Health {
			addFile(ctx.FileTree, "src/routes/health.js", "export default function health(req, res) { res.send({ status: 'ok' }); }\n")
		}
		if req.Features.JWTAuth {
			addFile(ctx.FileTree, "src/auth/jwt.js", "export const jwtSecret = process.env.JWT_SECRET || 'changeme';\n")
		}
		if strings.EqualFold(req.ServiceCommunication, "grpc") {
			addFile(ctx.FileTree, "proto/README.md", "# Shared proto definitions\n\nPlace your protobuf contracts here.\n")
			addFile(ctx.FileTree, "proto/common.proto", "syntax = \"proto3\";\npackage stacksprint;\n\nservice InternalService {\n  rpc Ping(PingRequest) returns (PingReply);\n}\n\nmessage PingRequest {\n  string source = 1;\n}\n\nmessage PingReply {\n  string message = 1;\n}\n")
			g.addGRPCBoilerplate(ctx.FileTree, req, "")
		}
	}
	return nil
}

func (g *NodeGenerator) GenerateModels(req *GenerateRequest, ctx *GenerationContext) error {
	if req.Architecture == "microservices" {
		for _, svc := range req.Services {
			svcRoot := path.Join("services", svc.Name)
			if req.Database != "none" {
				g.addNodeDBBoilerplate(ctx.FileTree, req, svcRoot)
			}
			if isEnabled(req.FileToggles.ExampleCRUD) {
				for _, model := range resolvedModels(req.Custom.Models) {
					g.renderNodeDynamicModel(ctx.FileTree, req, model, req.Architecture, svcRoot)
				}
			}
		}
	} else {
		if req.Database != "none" {
			g.addNodeDBBoilerplate(ctx.FileTree, req, "")
		}
		if isEnabled(req.FileToggles.ExampleCRUD) {
			for _, model := range resolvedModels(req.Custom.Models) {
				g.renderNodeDynamicModel(ctx.FileTree, req, model, req.Architecture, "")
			}
		}
	}
	return nil
}

func (g *NodeGenerator) GenerateInfra(req *GenerateRequest, ctx *GenerationContext) error {
	handleInfra := func(root string, port int) {
		if req.Infra.Redis {
			addFile(ctx.FileTree, path.Join(root, "src/cache/redis.js"), "export class RedisCache {\n  constructor(addr = process.env.REDIS_ADDR || 'redis:6379') {\n    this.addr = addr;\n  }\n\n  ping() {\n    return `redis configured at ${this.addr}`;\n  }\n}\n")
		}
		if req.Infra.Kafka {
			addFile(ctx.FileTree, path.Join(root, "src/messaging/kafkaProducer.js"), "export class KafkaProducer {\n  constructor(brokers = process.env.KAFKA_BROKERS || 'kafka:9092') {\n    this.brokers = brokers;\n  }\n\n  publish(topic, payload) {\n    return `publish stub to ${topic} via ${this.brokers}: ${payload}`;\n  }\n}\n")
			addFile(ctx.FileTree, path.Join(root, "src/messaging/kafkaConsumer.js"), "export class KafkaConsumer {\n  constructor(brokers = process.env.KAFKA_BROKERS || 'kafka:9092') {\n    this.brokers = brokers;\n  }\n\n  subscribe(topic) {\n    return `consumer stub subscribed to ${topic} via ${this.brokers}`;\n  }\n}\n")
		}
		if isEnabled(req.FileToggles.Env) {
			svcName := ""
			if root != "" {
				svcName = path.Base(root)
			}
			addFile(ctx.FileTree, path.Join(root, ".env"), buildEnv(*req, svcName, port))
		}
		if isEnabled(req.FileToggles.Dockerfile) {
			addFile(ctx.FileTree, path.Join(root, "Dockerfile"), "FROM node:22-alpine AS deps\nWORKDIR /app\nCOPY package*.json ./\nRUN npm ci\n\nFROM node:22-alpine AS runner\nWORKDIR /app\nENV NODE_ENV production\nCOPY --from=deps /app/node_modules ./node_modules\nCOPY . .\nEXPOSE 8080\nCMD [\"npm\", \"start\"]\n")
		}
	}

	if req.Architecture == "microservices" {
		for _, svc := range req.Services {
			svcRoot := path.Join("services", svc.Name)
			handleInfra(svcRoot, svc.Port)
		}
	} else {
		handleInfra("", 8080)
	}

	if isEnabled(req.FileToggles.Compose) {
		addFile(ctx.FileTree, "docker-compose.yaml", buildCompose(*req))
	}
	return nil
}

func (g *NodeGenerator) GenerateDevTools(req *GenerateRequest, ctx *GenerationContext) error {
	if isEnabled(req.FileToggles.Gitignore) {
		addFile(ctx.FileTree, ".gitignore", "bin/\nobj/\n.env\n.DS_Store\nnode_modules/\nvendor/\n__pycache__/\n*.sqlite3\n")
	}
	if isEnabled(req.FileToggles.Readme) {
		addFile(ctx.FileTree, "README.md", fmt.Sprintf("# StackSprint Generated Project\n\nLanguage: %s\nFramework: %s\nArchitecture: %s\nDatabase: %s\n\n## Run\n\n```bash\ndocker compose up --build\n```\n", req.Language, req.Framework, req.Architecture, req.Database))
	}
	if req.Features.GitHubActions {
		addFile(ctx.FileTree, ".github/workflows/ci.yaml", "name: CI\n\non:\n  push:\n  pull_request:\n\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v4\n      - uses: actions/setup-node@v4\n        with:\n          node-version: '22'\n      - run: npm test\n")
	}
	if req.Features.Makefile {
		var b strings.Builder
		b.WriteString("up:\n\tdocker compose up --build\n\ndown:\n\tdocker compose down -v\n\ntest:\n\t@echo \"Run language-specific tests\"\n")
		if req.Database != "none" {
			if req.UseORM {
				b.WriteString("\nseed:\n\t@echo \"Running Prisma Seeder\"\n\tnpx prisma db seed\n")
			} else {
				b.WriteString("\nseed:\n\t@echo \"Running raw SQL seed\"\n\tnode scripts/seed.js\n")
			}
		}
		addFile(ctx.FileTree, "Makefile", b.String())
	}
	if req.Features.Swagger {
		addFile(ctx.FileTree, "docs/openapi.yaml", "openapi: 3.0.3\ninfo:\n  title: StackSprint API\n  version: 1.0.0\npaths:\n  /health:\n    get:\n      responses:\n        '200':\n          description: OK\n")
	}
	return nil
}

// GetInitCommand returns the bash init command for Node.js projects.
func (g *NodeGenerator) GetInitCommand(_ *GenerateRequest) string {
	return "npm init -y\n"
}

// GetConfigWarnings returns Node.js-specific configuration warnings.
// Node has no framework-specific config warnings at this time.
func (g *NodeGenerator) GetConfigWarnings(_ *GenerateRequest) []Warning {
	return nil
}

// -------------------------------------------------------------------------
// Helper Functions (Internal node_generator)
// -------------------------------------------------------------------------

func (g *NodeGenerator) generateMonolithArch(req *GenerateRequest, ctx *GenerationContext, root string) error {
	specs := nodeMonolithTemplateSpecs(*req)
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
	if err := g.renderSpecs(ctx, specs, data, root); err != nil {
		return err
	}

	if isEnabled(req.FileToggles.ExampleCRUD) {
		var imports, routes strings.Builder
		for _, model := range resolvedModels(req.Custom.Models) {
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
		if main, ok := ctx.FileTree.Files["src/index.js"]; ok {
			var err error
			main, err = InjectByMarker(main, "imports", imports.String())
			if err != nil {
				ctx.Warnings = append(ctx.Warnings, Warning{Code: "INJECTION_MARKER_MISSING", Severity: "error", Message: "Failed to inject dynamic imports", Reason: err.Error()})
			}
			main, err = InjectByMarker(main, "routes", routes.String())
			if err != nil {
				ctx.Warnings = append(ctx.Warnings, Warning{Code: "INJECTION_MARKER_MISSING", Severity: "error", Message: "Failed to inject dynamic routes", Reason: err.Error()})
			}
			ctx.FileTree.Files["src/index.js"] = main
		}
	}
	addFile(ctx.FileTree, "package.json", nodePackageJSON(req.Framework, req.Database, req.UseORM))
	if isEnabled(req.FileToggles.Config) {
		addFile(ctx.FileTree, "src/config/index.js", "import dotenv from 'dotenv';\nimport { z } from 'zod';\n\ndotenv.config();\n\nconst envSchema = z.object({\n  PORT: z.string().transform(Number).default('8080'),\n  DATABASE_URL: z.string().url().optional(),\n  JWT_SECRET: z.string().min(8).default('default_dev_secret_replace_in_prod'),\n});\n\nconst parsed = envSchema.safeParse(process.env);\nif (!parsed.success) {\n  console.error('❌ Invalid environment variables:', parsed.error.format());\n  process.exit(1);\n}\n\nexport const config = {\n  port: parsed.data.PORT,\n  dbUrl: parsed.data.DATABASE_URL || '',\n  jwtSecret: parsed.data.JWT_SECRET,\n};\n")
	}
	if req.Features.Logger || isEnabled(req.FileToggles.Logger) {
		addFile(ctx.FileTree, "src/logger/index.js", "export const logger = { info: (...a) => console.log('[INFO]', ...a), error: (...a) => console.error('[ERROR]', ...a) };\n")
	}
	if req.Features.GlobalError {
		addFile(ctx.FileTree, "src/middleware/error.js", "export function globalError(err, req, res, next) {\n  res.status(500).json({ error: err.message || 'internal error' });\n}\n")
	}
	if req.Features.SampleTest {
		addFile(ctx.FileTree, "tests/items.test.js", "import test from 'node:test';\nimport assert from 'node:assert/strict';\n\ntest('sample', () => {\n  assert.equal(1 + 1, 2);\n});\n")
	}
	g.addNodeAutopilot(ctx.FileTree, req, root)
	g.addNodeDBRetry(ctx.FileTree, req, root)
	return nil
}

func (g *NodeGenerator) generateServiceArch(req *GenerateRequest, ctx *GenerationContext, svcRoot string, svc ServiceConfig) error {
	specs := nodeMicroserviceTemplateSpecs(*req)
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
	if err := g.renderSpecs(ctx, specs, data, svcRoot); err != nil {
		return err
	}
	if isEnabled(req.FileToggles.ExampleCRUD) {
		var imports, routes strings.Builder
		for _, model := range resolvedModels(req.Custom.Models) {
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
		if main, ok := ctx.FileTree.Files[path.Join(svcRoot, "src/index.js")]; ok {
			var err error
			main, err = InjectByMarker(main, "imports", imports.String())
			if err != nil {
				ctx.Warnings = append(ctx.Warnings, Warning{Code: "INJECTION_MARKER_MISSING", Severity: "error", Message: "Failed to inject dynamic imports for service " + svc.Name, Reason: err.Error()})
			}
			main, err = InjectByMarker(main, "routes", routes.String())
			if err != nil {
				ctx.Warnings = append(ctx.Warnings, Warning{Code: "INJECTION_MARKER_MISSING", Severity: "error", Message: "Failed to inject dynamic routes for service " + svc.Name, Reason: err.Error()})
			}
			ctx.FileTree.Files[path.Join(svcRoot, "src/index.js")] = main
		}
	}
	addFile(ctx.FileTree, path.Join(svcRoot, "package.json"), nodePackageJSON(req.Framework, req.Database, req.UseORM))

	g.addNodeAutopilot(ctx.FileTree, req, svcRoot)
	g.addNodeDBRetry(ctx.FileTree, req, svcRoot)
	return nil
}

func (g *NodeGenerator) renderSpecs(ctx *GenerationContext, specs []templateSpec, data map[string]any, root string) error {
	for _, spec := range specs {
		body, err := ctx.Registry.Render(spec.Template, data)
		if err != nil {
			return err
		}
		out := spec.Output
		if root != "" {
			out = path.Join(root, out)
		}
		addFile(ctx.FileTree, out, body)
	}
	return nil
}

func (g *NodeGenerator) addGRPCBoilerplate(tree *FileTree, req *GenerateRequest, root string) {
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	addFile(tree, prefix+"src/grpc/server.js", "export function startGrpcServer() {\n  return 'gRPC server stub started';\n}\n")
	addFile(tree, prefix+"src/grpc/client.js", "export function pingGrpc(target = '127.0.0.1:9090') {\n  return `gRPC client stub pinging ${target}`;\n}\n")
}

func (g *NodeGenerator) addNodeDBBoilerplate(tree *FileTree, req *GenerateRequest, root string) {
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

func (g *NodeGenerator) buildNodeDomainClass(name string, model DataModel) string {
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

func (g *NodeGenerator) renderNodeDynamicModel(tree *FileTree, req *GenerateRequest, model DataModel, arch, root string) {
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	name := model.Name
	nameLow := strings.ToLower(name)
	sample := buildNodeSampleObject(model)

	switch arch {
	case "clean":
		addFile(tree, prefix+"src/domain/"+nameLow+".js", g.buildNodeDomainClass(name, model))
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

	default:
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

func (g *NodeGenerator) addNodeAutopilot(tree *FileTree, req *GenerateRequest, root string) {
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	addFile(tree, prefix+"src/middleware/requestId.js", "import { randomUUID } from 'node:crypto';\n\n/**\n * Attaches a unique X-Request-ID header to every request/response.\n * Works with Express and Fastify (via onRequest hook).\n */\nexport function requestId(req, res, next) {\n  const id = req.headers['x-request-id'] || randomUUID();\n  req.requestId = id;\n  res.setHeader('X-Request-ID', id);\n  if (next) next();\n}\n")
	addFile(tree, prefix+"src/middleware/requestLogger.js",
		"/**\n"+
			" * Logs method, path, status code, and latency for every request.\n"+
			" */\n"+
			"export function requestLogger(req, res, next) {\n"+
			"  const start = Date.now();\n"+
			"  res.on('finish', () => {\n"+
			"    const msg = '[' + new Date().toISOString() + '] ' + req.method + ' ' + req.url +\n"+
			"      ' -> ' + res.statusCode + ' (' + (Date.now() - start) + 'ms) rid=' + (req.requestId || '-');\n"+
			"    console.log(msg);\n"+
			"  });\n"+
			"  if (next) next();\n"+
			"}\n")
	addFile(tree, prefix+"src/utils/pagination.js", "/**\n * Parses limit/offset from a query object and returns safe defaults.\n * @param {{ limit?: string|number, offset?: string|number }} query\n * @returns {{ limit: number, offset: number }}\n */\nexport function parsePage(query = {}) {\n  const limit = Math.min(Math.max(Number(query.limit) || 20, 1), 100);\n  const offset = Math.max(Number(query.offset) || 0, 0);\n  return { limit, offset };\n}\n")
}

func (g *NodeGenerator) addNodeDBRetry(tree *FileTree, req *GenerateRequest, root string) {
	if req.Database == "none" {
		return
	}
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	addFile(tree, prefix+"src/db/retry.js",
		"/**\n"+
			" * Retries a DB connect function with exponential back-off.\n"+
			" * @param {() => Promise<any>} connectFn\n"+
			" * @param {number} maxRetries\n"+
			" * @returns {Promise<any>}\n"+
			" */\n"+
			"export async function connectWithRetry(connectFn, maxRetries = 10) {\n"+
			"  let wait = 1000;\n"+
			"  for (let i = 0; i < maxRetries; i++) {\n"+
			"    try {\n"+
			"      return await connectFn();\n"+
			"    } catch (err) {\n"+
			"      console.log('DB not ready (attempt ' + (i+1) + '/' + maxRetries + '): ' + err.message + ' - retrying in ' + wait + 'ms');\n"+
			"      await new Promise((r) => setTimeout(r, wait));\n"+
			"      wait = Math.min(wait * 2, 16000);\n"+
			"    }\n"+
			"  }\n"+
			"  throw new Error('Database unavailable after ' + maxRetries + ' retries');\n"+
			"}\n")
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

func nodePackageJSON(framework string, db string, useORM bool) string {
	dep := framework
	extra := ""
	if db == "postgresql" {
		if useORM {
			extra = ",\n    \"@prisma/client\": \"^6.2.1\""
		} else {
			extra = ",\n    \"pg\": \"^8.13.3\""
		}
	}
	if db == "mysql" {
		if useORM {
			extra = ",\n    \"@prisma/client\": \"^6.2.1\""
		} else {
			extra = ",\n    \"mysql2\": \"^3.12.0\""
		}
	}
	devExtra := ""
	seedCmd := "node scripts/seed.js"
	if useORM && (db == "postgresql" || db == "mysql") {
		devExtra = ",\n  \"devDependencies\": {\n    \"prisma\": \"^6.2.1\"\n  }"
		seedCmd = "node prisma/seed.js"
	}
	return fmt.Sprintf(`{
  "name": "stacksprint-generated",
  "version": "1.0.0",
  "private": true,
  "type": "module",
  "scripts": {
    "start": "node src/index.js",
    "dev": "node src/index.js",
    "test": "node --test",
    "seed": "%s"
  },
  "dependencies": {
    "%s": "^5.0.0",
    "dotenv": "^16.4.5",
    "zod": "^3.23.8"%s
  }%s
}
`, seedCmd, dep, extra, devExtra)
}
