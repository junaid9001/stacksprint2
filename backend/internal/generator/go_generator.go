package generator

import (
	"fmt"
	"path"
	"strings"
)

type GoGenerator struct{}

func (g *GoGenerator) GenerateArchitecture(req *GenerateRequest, ctx *GenerationContext) error {
	if req.Architecture == "microservices" {
		for _, svc := range req.Services {
			svcRoot := path.Join("services", svc.Name)
			if err := g.generateServiceArch(req, ctx, svcRoot, svc); err != nil {
				return err
			}
			if isEnabled(req.FileToggles.BaseRoute) {
				addFile(ctx.FileTree, path.Join(svcRoot, "internal/routes/base.go"), "package routes\n\nconst BasePath = \"/api/v1\"\n")
			}
			if isEnabled(req.FileToggles.ExampleCRUD) {
				if req.Framework == "gin" {
					addFile(ctx.FileTree, path.Join(svcRoot, "internal/handlers/items.go"), "package handlers\n\nimport \"github.com/gin-gonic/gin\"\n\nfunc ListItems(c *gin.Context) {\n\tc.JSON(200, []gin.H{{\"id\": 1, \"name\": \"sample\"}})\n}\n")
				} else {
					addFile(ctx.FileTree, path.Join(svcRoot, "internal/handlers/items.go"), "package handlers\n\nimport \"github.com/gofiber/fiber/v2\"\n\nfunc ListItems(c *fiber.Ctx) error {\n\treturn c.JSON([]map[string]any{{\"id\": 1, \"name\": \"sample\"}})\n}\n")
				}
			}
			if isEnabled(req.FileToggles.HealthCheck) || req.Features.Health {
				addFile(ctx.FileTree, path.Join(svcRoot, "internal/health/handler.go"), "package health\n\nfunc Message() string { return \"ok\" }\n")
			}
			if req.Features.JWTAuth {
				addFile(ctx.FileTree, path.Join(svcRoot, "internal/auth/jwt.go"), "package auth\n\nimport \"os\"\n\nfunc Secret() string { return os.Getenv(\"JWT_SECRET\") }\n")
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
			addFile(ctx.FileTree, "internal/routes/base.go", "package routes\n\nconst BasePath = \"/api/v1\"\n")
		}
		if isEnabled(req.FileToggles.ExampleCRUD) {
			if req.Framework == "gin" {
				addFile(ctx.FileTree, "internal/handlers/items.go", "package handlers\n\nimport \"github.com/gin-gonic/gin\"\n\nfunc ListItems(c *gin.Context) {\n\tc.JSON(200, []gin.H{{\"id\": 1, \"name\": \"sample\"}})\n}\n")
			} else {
				addFile(ctx.FileTree, "internal/handlers/items.go", "package handlers\n\nimport \"github.com/gofiber/fiber/v2\"\n\nfunc ListItems(c *fiber.Ctx) error {\n\treturn c.JSON([]map[string]any{{\"id\": 1, \"name\": \"sample\"}})\n}\n")
			}
		}
		if isEnabled(req.FileToggles.HealthCheck) || req.Features.Health {
			addFile(ctx.FileTree, "internal/health/handler.go", "package health\n\nfunc Message() string { return \"ok\" }\n")
		}
		if req.Features.JWTAuth {
			addFile(ctx.FileTree, "internal/auth/jwt.go", "package auth\n\nimport \"os\"\n\nfunc Secret() string { return os.Getenv(\"JWT_SECRET\") }\n")
		}
		if strings.EqualFold(req.ServiceCommunication, "grpc") {
			addFile(ctx.FileTree, "proto/README.md", "# Shared proto definitions\n\nPlace your protobuf contracts here.\n")
			addFile(ctx.FileTree, "proto/common.proto", "syntax = \"proto3\";\npackage stacksprint;\n\nservice InternalService {\n  rpc Ping(PingRequest) returns (PingReply);\n}\n\nmessage PingRequest {\n  string source = 1;\n}\n\nmessage PingReply {\n  string message = 1;\n}\n")
			g.addGRPCBoilerplate(ctx.FileTree, req, "")
		}
	}
	return nil
}

func (g *GoGenerator) GenerateModels(req *GenerateRequest, ctx *GenerationContext) error {
	if req.Architecture == "microservices" {
		for _, svc := range req.Services {
			svcRoot := path.Join("services", svc.Name)
			if req.Database != "none" {
				g.addDatabaseBoilerplate(ctx.FileTree, req, svcRoot)
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
					g.renderGoOtherDynamicModel(ctx, data, model, req.Architecture, svcRoot)
				}
			}
		}
	} else {
		if req.Database != "none" {
			g.addDatabaseBoilerplate(ctx.FileTree, req, "")
			module := resolveGoModule(req.Root, "stacksprint/generated")
			addFile(ctx.FileTree, "cmd/seeder/main.go", renderGoSeederScript(module, req.Custom.Models, req.UseORM))
			addFile(ctx.FileTree, "migrations/001_initial.sql", sampleMigration(req.Database, req.Custom.Models))
			addFile(ctx.FileTree, "db/init/001_init.sql", sampleDBInit(req.Database, req.Custom.Models))
		}
		if isEnabled(req.FileToggles.ExampleCRUD) {
			module := resolveGoModule(req.Root, "stacksprint/generated")
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
			for _, model := range resolvedModels(req.Custom.Models) {
				if req.Architecture == "clean" {
					if err := g.renderGoCleanDynamicModel(ctx, data, model, ""); err != nil {
						return err
					}
				} else {
					if err := g.renderGoOtherDynamicModel(ctx, data, model, req.Architecture, ""); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (g *GoGenerator) GenerateInfra(req *GenerateRequest, ctx *GenerationContext) error {
	handleInfra := func(root string, port int) {
		if req.Infra.Redis {
			addFile(ctx.FileTree, path.Join(root, "internal/cache/redis.go"), "package cache\n\nimport \"os\"\n\ntype RedisCache struct {\n\tAddr string\n}\n\nfunc NewRedisCache() *RedisCache {\n\taddr := os.Getenv(\"REDIS_ADDR\")\n\tif addr == \"\" {\n\t\taddr = \"redis:6379\"\n\t}\n\treturn &RedisCache{Addr: addr}\n}\n\nfunc (r *RedisCache) Ping() string {\n\treturn \"redis configured at \" + r.Addr\n}\n")
		}
		if req.Infra.Kafka {
			addFile(ctx.FileTree, path.Join(root, "internal/messaging/kafka_producer.go"), "package messaging\n\nimport \"os\"\n\ntype KafkaProducer struct {\n\tBrokers string\n}\n\nfunc NewKafkaProducer() *KafkaProducer {\n\tb := os.Getenv(\"KAFKA_BROKERS\")\n\tif b == \"\" {\n\t\tb = \"kafka:9092\"\n\t}\n\treturn &KafkaProducer{Brokers: b}\n}\n\nfunc (p *KafkaProducer) Publish(topic, payload string) string {\n\treturn \"publish stub to \" + topic + \" via \" + p.Brokers + \" payload=\" + payload\n}\n")
			addFile(ctx.FileTree, path.Join(root, "internal/messaging/kafka_consumer.go"), "package messaging\n\nimport \"os\"\n\ntype KafkaConsumer struct {\n\tBrokers string\n}\n\nfunc NewKafkaConsumer() *KafkaConsumer {\n\tb := os.Getenv(\"KAFKA_BROKERS\")\n\tif b == \"\" {\n\t\tb = \"kafka:9092\"\n\t}\n\treturn &KafkaConsumer{Brokers: b}\n}\n\nfunc (c *KafkaConsumer) Subscribe(topic string) string {\n\treturn \"consumer stub subscribed to \" + topic + \" via \" + c.Brokers\n}\n")
		}
		if isEnabled(req.FileToggles.Env) {
			svcName := ""
			if root != "" {
				svcName = path.Base(root)
			}
			addFile(ctx.FileTree, path.Join(root, ".env"), buildEnv(*req, svcName, port))
		}
		if isEnabled(req.FileToggles.Dockerfile) {
			addFile(ctx.FileTree, path.Join(root, "Dockerfile"), "FROM golang:1.23-alpine AS build\nWORKDIR /app\nCOPY . .\nRUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app ./cmd/server\n\nFROM scratch\nWORKDIR /app\nCOPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/\nCOPY --from=build /app/app .\nEXPOSE 8080\nCMD [\"./app\"]\n")
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

func (g *GoGenerator) GenerateDevTools(req *GenerateRequest, ctx *GenerationContext) error {
	if isEnabled(req.FileToggles.Gitignore) {
		addFile(ctx.FileTree, ".gitignore", "bin/\nobj/\n.env\n.DS_Store\nnode_modules/\nvendor/\n__pycache__/\n*.sqlite3\n")
	}
	if isEnabled(req.FileToggles.Readme) {
		addFile(ctx.FileTree, "README.md", fmt.Sprintf("# StackSprint Generated Project\n\nLanguage: %s\nFramework: %s\nArchitecture: %s\nDatabase: %s\n\n## Run\n\n```bash\ndocker compose up --build\n```\n", req.Language, req.Framework, req.Architecture, req.Database))
	}
	if req.Features.GitHubActions {
		addFile(ctx.FileTree, ".github/workflows/ci.yaml", "name: CI\n\non:\n  push:\n  pull_request:\n\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v4\n      - uses: actions/setup-go@v5\n        with:\n          go-version: '1.23'\n      - run: go test ./...\n")
	}
	if req.Features.Makefile {
		var b strings.Builder
		b.WriteString("up:\n\tdocker compose up --build\n\ndown:\n\tdocker compose down -v\n\ntest:\n\t@echo \"Run language-specific tests\"\n")
		if req.Database != "none" {
			b.WriteString("\nmigrate-up:\n\t@echo \"Running migrations up\"\n\t# migrate -path db/migrations -database \"$$DATABASE_URL\" up\n")
			b.WriteString("\nmigrate-down:\n\t@echo \"Running migrations down\"\n\t# migrate -path db/migrations -database \"$$DATABASE_URL\" down\n")
			b.WriteString("\nseed:\n\t@echo \"Running seeder\"\n\tgo run cmd/seeder/main.go\n")
		}
		addFile(ctx.FileTree, "Makefile", b.String())
	}
	if req.Features.Swagger {
		addFile(ctx.FileTree, "docs/openapi.yaml", "openapi: 3.0.3\ninfo:\n  title: StackSprint API\n  version: 1.0.0\npaths:\n  /health:\n    get:\n      responses:\n        '200':\n          description: OK\n")
	}
	return nil
}

// GetInitCommand returns the bash init command for Go projects.
func (g *GoGenerator) GetInitCommand(req *GenerateRequest) string {
	mod := req.Root.Module
	if mod == "" {
		mod = path.Base(req.Root.Name)
	}
	return fmt.Sprintf("go mod init %q\n", mod)
}

// -------------------------------------------------------------------------
// Helper Functions (Internal go_generator)
// -------------------------------------------------------------------------

func (g *GoGenerator) generateMonolithArch(req *GenerateRequest, ctx *GenerationContext, root string) error {
	module := resolveGoModule(req.Root, "stacksprint/generated")
	specs := goMonolithTemplateSpecs(*req)
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
	if err := g.renderSpecs(ctx, specs, data, root); err != nil {
		return err
	}
	addFile(ctx.FileTree, "go.mod", goModV2(req.Framework, req.Root, req.Database, req.UseORM, strings.EqualFold(req.ServiceCommunication, "grpc")))

	if isEnabled(req.FileToggles.Config) {
		addFile(ctx.FileTree, "internal/config/config.go", "package config\n\nimport (\n\t\"log\"\n\n\t\"github.com/kelseyhightower/envconfig\"\n)\n\ntype Config struct {\n\tPort        int    `envconfig:\"PORT\" default:\"8080\"`\n\tDatabaseURL string `envconfig:\"DATABASE_URL\"`\n\tJWTSecret   string `envconfig:\"JWT_SECRET\" default:\"default_dev_secret_replace_in_prod\"`\n}\n\nvar AppConfig Config\n\nfunc Init() {\n\terr := envconfig.Process(\"\", &AppConfig)\n\tif err != nil {\n\t\tlog.Fatalf(\"❌ Environment variable validation failed: %v\", err)\n\t}\n}\n\nfunc Port() string {\n\treturn \"%d\" // we will return fmt.Sprint(AppConfig.Port) implicitly by changing the references in main later. This is just a stub for backwards compat if needed, but the main template should use config.AppConfig.Port now.\n\t// We'll update main.go to call config.Init()\n}\n")
	}
	if req.Features.Logger || isEnabled(req.FileToggles.Logger) {
		addFile(ctx.FileTree, "internal/logger/logger.go", "package logger\n\nimport \"log\"\n\nfunc Info(msg string) { log.Println(\"INFO:\", msg) }\nfunc Error(msg string) { log.Println(\"ERROR:\", msg) }\n")
	}
	if req.Features.GlobalError {
		if req.Framework == "fiber" {
			addFile(ctx.FileTree, "internal/middleware/error.go", "package middleware\n\nimport \"github.com/gofiber/fiber/v2\"\n\nfunc ErrorHandler(c *fiber.Ctx, err error) error {\n\treturn c.Status(500).JSON(fiber.Map{\"error\": err.Error()})\n}\n")
		} else {
			addFile(ctx.FileTree, "internal/middleware/error.go", "package middleware\n\nimport \"github.com/gin-gonic/gin\"\n\nfunc ErrorHandler(c *gin.Context) {\n\tc.Next()\n\tif len(c.Errors) > 0 {\n\t\tc.JSON(500, gin.H{\"error\": c.Errors.String()})\n\t}\n}\n")
		}
	}
	if req.Features.SampleTest {
		addFile(ctx.FileTree, "internal/handlers/item_handler_test.go", "package handlers\n\nimport \"testing\"\n\nfunc TestPlaceholder(t *testing.T) {\n\tif false {\n\t\tt.Fatal(\"expected true\")\n\t}\n}\n")
	}
	g.addAutopilotBoilerplate(ctx.FileTree, req, root)
	g.addDBRetry(ctx.FileTree, req, root)
	g.injectGoRoutes(ctx, req, root, module)
	return nil
}

func (g *GoGenerator) generateServiceArch(req *GenerateRequest, ctx *GenerationContext, svcRoot string, svc ServiceConfig) error {
	module := fmt.Sprintf("stacksprint/%s", svc.Name)
	specs := goMicroserviceTemplateSpecs(*req)
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
	if err := g.renderSpecs(ctx, specs, data, svcRoot); err != nil {
		return err
	}
	addFile(ctx.FileTree, path.Join(svcRoot, "go.mod"), goModV2(req.Framework, RootOptions{Module: module}, req.Database, req.UseORM, strings.EqualFold(req.ServiceCommunication, "grpc")))

	g.addAutopilotBoilerplate(ctx.FileTree, req, svcRoot)
	g.addDBRetry(ctx.FileTree, req, svcRoot)
	g.injectGoRoutes(ctx, req, svcRoot, module)
	return nil
}

func (g *GoGenerator) renderSpecs(ctx *GenerationContext, specs []templateSpec, data map[string]any, root string) error {
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

func (g *GoGenerator) addGRPCBoilerplate(tree *FileTree, req *GenerateRequest, root string) {
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	addFile(tree, prefix+"internal/grpc/server/server.go", "package server\n\nimport \"context\"\n\ntype PingRequest struct{ Source string }\ntype PingReply struct{ Message string }\n\ntype Service struct{}\n\nfunc (s *Service) Ping(_ context.Context, req *PingRequest) (*PingReply, error) {\n\treturn &PingReply{Message: \"pong from \" + req.Source}, nil\n}\n")
	addFile(tree, prefix+"internal/grpc/client/client.go", "package client\n\nimport \"fmt\"\n\ntype Client struct{ Address string }\n\nfunc New(address string) *Client {\n\tif address == \"\" {\n\t\taddress = \"127.0.0.1:9090\"\n\t}\n\treturn &Client{Address: address}\n}\n\nfunc (c *Client) Ping() string {\n\treturn fmt.Sprintf(\"ping stub to %s\", c.Address)\n}\n")
}

func (g *GoGenerator) addDatabaseBoilerplate(tree *FileTree, req *GenerateRequest, root string) {
	p := func(parts ...string) string {
		if root == "" {
			return strings.Join(parts, "/")
		}
		return root + "/" + strings.Join(parts, "/")
	}
	if isSQLDB(req.Database) && req.UseORM {
		driverImport := "\"gorm.io/driver/postgres\""
		driverOpen := "postgres.Open(dsn)"
		if req.Database == "mysql" {
			driverImport = "\"gorm.io/driver/mysql\""
			driverOpen := "mysql.Open(dsn)"
			_ = driverOpen
		}
		addFile(tree, p("internal", "db", "connection.go"), "package db\n\nimport (\n\t\"os\"\n\n\t"+driverImport+"\n\t\"gorm.io/gorm\"\n)\n\nfunc Connect() (*gorm.DB, error) {\n\tdsn := os.Getenv(\"DATABASE_URL\")\n\treturn gorm.Open("+driverOpen+", &gorm.Config{})\n}\n")
		addFile(tree, p("internal", "models", "models.go"), renderGoORMModels(req.Custom.Models))
	} else {
		stdImport := "\"database/sql\"\n\t_ \"github.com/jackc/pgx/v5/stdlib\""
		driver := "\"pgx\""
		if req.Database == "mysql" {
			stdImport = "\"database/sql\"\n\t_ \"github.com/go-sql-driver/mysql\""
			driver = "\"mysql\""
		}
		addFile(tree, p("internal", "db", "connection.go"), "package db\n\nimport (\n\t"+stdImport+"\n\t\"os\"\n)\n\nfunc Connect() (*sql.DB, error) {\n\treturn sql.Open("+driver+", os.Getenv(\"DATABASE_URL\"))\n}\n")
	}
	if !req.UseORM || !isSQLDB(req.Database) {
		addFile(tree, p("internal", "models", "item.go"), "package models\n\ntype Item struct {\n\tID int `json:\"id\"`\n\tName string `json:\"name\"`\n}\n")
	}
}

func (g *GoGenerator) renderGoCleanDynamicModel(ctx *GenerationContext, baseData map[string]any, model DataModel, root string) error {
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

		body, err := ctx.Registry.Render(spec.Template, data)
		if err != nil {
			return err
		}
		addFile(ctx.FileTree, prefix+spec.Output, body)
	}
	return nil
}

func (g *GoGenerator) renderGoOtherDynamicModel(ctx *GenerationContext, baseData map[string]any, model DataModel, arch, root string) error {
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
	default:
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

		body, err := ctx.Registry.Render(spec.Template, data)
		if err == nil {
			addFile(ctx.FileTree, prefix+spec.Output, body)
		}
	}
	return nil
}

func (g *GoGenerator) addAutopilotBoilerplate(tree *FileTree, req *GenerateRequest, root string) {
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	if req.Framework == "fiber" {
		addFile(tree, prefix+"internal/middleware/requestid.go", "package middleware\n\nimport (\n\t\"github.com/gofiber/fiber/v2\"\n\t\"github.com/google/uuid\"\n)\n\nfunc RequestID() fiber.Handler {\n\treturn func(c *fiber.Ctx) error {\n\t\tid := c.Get(\"X-Request-ID\")\n\t\tif id == \"\" {\n\t\t\tid = uuid.NewString()\n\t\t}\n\t\tc.Set(\"X-Request-ID\", id)\n\t\tc.Locals(\"requestID\", id)\n\t\treturn c.Next()\n\t}\n}\n")
		addFile(tree, prefix+"internal/middleware/requestlogger.go", "package middleware\n\nimport (\n\t\"fmt\"\n\t\"time\"\n\n\t\"github.com/gofiber/fiber/v2\"\n)\n\nfunc RequestLogger() fiber.Handler {\n\treturn func(c *fiber.Ctx) error {\n\t\tstart := time.Now()\n\t\terr := c.Next()\n\t\trid, _ := c.Locals(\"requestID\").(string)\n\t\tfmt.Printf(\"[%s] %s %s → %d (%s) rid=%s\\n\",\n\t\t\ttime.Now().Format(time.RFC3339),\n\t\t\tc.Method(), c.Path(),\n\t\t\tc.Response().StatusCode(),\n\t\t\ttime.Since(start),\n\t\t\trid,\n\t\t)\n\t\treturn err\n\t}\n}\n")
	} else {
		addFile(tree, prefix+"internal/middleware/requestid.go", "package middleware\n\nimport (\n\t\"github.com/gin-gonic/gin\"\n\t\"github.com/google/uuid\"\n)\n\nfunc RequestID() gin.HandlerFunc {\n\treturn func(c *gin.Context) {\n\t\tid := c.GetHeader(\"X-Request-ID\")\n\t\tif id == \"\" {\n\t\t\tid = uuid.NewString()\n\t\t}\n\t\tc.Header(\"X-Request-ID\", id)\n\t\tc.Set(\"requestID\", id)\n\t\tc.Next()\n\t}\n}\n")
		addFile(tree, prefix+"internal/middleware/requestlogger.go", "package middleware\n\nimport (\n\t\"fmt\"\n\t\"time\"\n\n\t\"github.com/gin-gonic/gin\"\n)\n\nfunc RequestLogger() gin.HandlerFunc {\n\treturn func(c *gin.Context) {\n\t\tstart := time.Now()\n\t\tc.Next()\n\t\trid, _ := c.Get(\"requestID\")\n\t\tfmt.Printf(\"[%s] %s %s → %d (%s) rid=%v\\n\",\n\t\t\ttime.Now().Format(time.RFC3339),\n\t\t\tc.Request.Method, c.FullPath(),\n\t\t\tc.Writer.Status(),\n\t\t\ttime.Since(start),\n\t\t\trid,\n\t\t)\n\t}\n}\n")
	}

	addFile(tree, prefix+"internal/pagination/pagination.go", "package pagination\n\ntype Page struct {\n\tLimit  int\n\tOffset int\n}\n\nfunc Parse(limit, offset int) Page {\n\tif limit <= 0 {\n\t\tlimit = 20\n\t}\n\tif limit > 100 {\n\t\tlimit = 100\n\t}\n\tif offset < 0 {\n\t\toffset = 0\n\t}\n\treturn Page{Limit: limit, Offset: offset}\n}\n")
}

func (g *GoGenerator) addDBRetry(tree *FileTree, req *GenerateRequest, root string) {
	if req.Database == "none" {
		return
	}
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	addFile(tree, prefix+"internal/db/retry.go", "package db\n\nimport (\n\t\"database/sql\"\n\t\"fmt\"\n\t\"time\"\n)\n\nfunc ConnectWithRetry(driver, dsn string, maxRetries int) (*sql.DB, error) {\n\tvar db *sql.DB\n\tvar err error\n\twait := time.Second\n\tfor i := 0; i < maxRetries; i++ {\n\t\tdb, err = sql.Open(driver, dsn)\n\t\tif err == nil {\n\t\t\tif err = db.Ping(); err == nil {\n\t\t\t\treturn db, nil\n\t\t\t}\n\t\t}\n\t\tfmt.Printf(\"DB not ready (attempt %d/%d): %v — retrying in %s\\n\", i+1, maxRetries, err, wait)\n\t\ttime.Sleep(wait)\n\t\twait *= 2\n\t}\n\treturn nil, fmt.Errorf(\"database unavailable after %d retries: %w\", maxRetries, err)\n}\n")
}

// GetConfigWarnings returns Go-specific configuration warnings.
// Go has no framework-specific config warnings at this time.
func (g *GoGenerator) GetConfigWarnings(_ *GenerateRequest) []Warning {
	return nil
}

// -------------------------------------------------------------------------
// Go Template Spec Builders
// -------------------------------------------------------------------------

func goMonolithTemplateSpecs(req GenerateRequest) []templateSpec {
	arch := archTemplateName(req.Architecture)
	withCRUD := isEnabled(req.FileToggles.ExampleCRUD)

	switch req.Architecture {
	case "clean":
		base := []templateSpec{
			{Template: "go/clean/cmd/server/main.tmpl", Output: "cmd/server/main.go"},
		}
		if withCRUD {
			return base
		}
		return append(base,
			templateSpec{Template: "go/clean/internal/domain/item.tmpl", Output: "internal/domain/item.go"},
			templateSpec{Template: "go/clean/internal/usecase/item.tmpl", Output: "internal/usecase/item_usecase.go"},
			templateSpec{Template: "go/clean/internal/repository/item.tmpl", Output: "internal/repository/item_repository.go"},
			templateSpec{Template: "go/clean/internal/delivery/http/item.tmpl", Output: "internal/delivery/http/item_handler.go"},
		)
	case "hexagonal":
		base := []templateSpec{
			{Template: "go/hexagonal/cmd/server/main.tmpl", Output: "cmd/server/main.go"},
		}
		if withCRUD {
			return base
		}
		return append(base,
			templateSpec{Template: "go/hexagonal/internal/core/ports/item_port.tmpl", Output: "internal/core/ports/item_port.go"},
			templateSpec{Template: "go/hexagonal/internal/core/services/item_service.tmpl", Output: "internal/core/services/item_service.go"},
			templateSpec{Template: "go/hexagonal/internal/adapters/primary/http/item_handler.tmpl", Output: "internal/adapters/primary/http/item_handler.go"},
			templateSpec{Template: "go/hexagonal/internal/adapters/secondary/database/item_adapter.tmpl", Output: "internal/adapters/secondary/database/item_adapter.go"},
		)
	case "modular-monolith":
		base := []templateSpec{
			{Template: "go/modular/cmd/server/main.tmpl", Output: "cmd/server/main.go"},
		}
		if withCRUD {
			return base
		}
		return append(base,
			templateSpec{Template: "go/modular/internal/modules/item/http.tmpl", Output: "internal/modules/item/http.go"},
			templateSpec{Template: "go/modular/internal/modules/item/service.tmpl", Output: "internal/modules/item/service.go"},
			templateSpec{Template: "go/modular/internal/modules/item/repository.tmpl", Output: "internal/modules/item/repository.go"},
		)
	default: // mvp
		base := []templateSpec{
			{Template: fmt.Sprintf("go/%s/cmd/server/main.tmpl", arch), Output: "cmd/server/main.go"},
		}
		if withCRUD {
			return base
		}
		return append(base,
			templateSpec{Template: fmt.Sprintf("go/%s/internal/handlers/item_handler.tmpl", arch), Output: "internal/handlers/item_handler.go"},
		)
	}
}

func goMicroserviceTemplateSpecs(req GenerateRequest) []templateSpec {
	return []templateSpec{
		{Template: "go/microservice/cmd/server/main.tmpl", Output: "cmd/server/main.go"},
	}
}

func renderGoSeederScript(module string, models []DataModel, useORM bool) string {
	resolved := resolvedModels(models)
	var b strings.Builder
	b.WriteString("package main\n\nimport (\n\t\"fmt\"\n")
	if useORM {
		b.WriteString(fmt.Sprintf("\t\"%s/internal/db\"\n\t\"%s/internal/models\"\n", module, module))
	} else {
		b.WriteString(fmt.Sprintf("\t\"%s/internal/db\"\n", module))
	}
	b.WriteString(")\n\nfunc main() {\n\tfmt.Println(\"Running seeder...\")\n")
	if useORM {
		b.WriteString("\tconn, err := db.Connect()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n")
		for _, m := range resolved {
			b.WriteString(fmt.Sprintf("\tconn.AutoMigrate(&models.%s{})\n", m.Name))
		}
		b.WriteString("\tfmt.Println(\"Seeding complete.\")\n}\n")
	} else {
		b.WriteString("\t_, _ = db.Connect()\n\t// TODO: implement raw SQL inserts\n\tfmt.Println(\"Seeding complete.\")\n}\n")
	}
	return b.String()
}

func (g *GoGenerator) injectGoRoutes(ctx *GenerationContext, req *GenerateRequest, root, module string) {
	if !isEnabled(req.FileToggles.ExampleCRUD) {
		return
	}
	mainPath := "cmd/server/main.go"
	if root != "" {
		mainPath = path.Join(root, mainPath)
	}
	main, ok := ctx.FileTree.Files[mainPath]
	if !ok {
		return
	}

	var imports, routes strings.Builder
	for _, model := range resolvedModels(req.Custom.Models) {
		nameLow := strings.ToLower(model.Name)
		if req.Architecture == "clean" {
			imports.WriteString(fmt.Sprintf("\n\t\"%s/internal/delivery/http\"\n\t\"%s/internal/repository\"\n\t\"%s/internal/usecase\"", module, module, module))
			routes.WriteString(fmt.Sprintf("\n\t_ = http.New%sHandler(usecase.New%sUsecase(repository.New%sRepository()))", model.Name, model.Name, model.Name))
		} else if req.Architecture == "hexagonal" {
			imports.WriteString(fmt.Sprintf("\n\thttpPrimary \"%s/internal/adapters/primary/http\"\n\t\"%s/internal/adapters/secondary/database\"\n\t\"%s/internal/core/services\"", module, module, module))
			routes.WriteString(fmt.Sprintf("\n\t_ = httpPrimary.New%sHandler(services.New%sService(database.New%sAdapter()))", model.Name, model.Name, model.Name))
		} else if req.Architecture == "modular-monolith" {
			imports.WriteString(fmt.Sprintf("\n\t\"%s/internal/modules/%s\"", module, nameLow))
			routes.WriteString(fmt.Sprintf("\n\t_ = %s.NewService(%s.NewRepository())", nameLow, nameLow))
		} else {
			imports.WriteString(fmt.Sprintf("\n\t\"%s/internal/handlers\"", module))
			if req.Framework == "gin" {
				routes.WriteString(fmt.Sprintf("\n\tr.GET(\"/%ss\", handlers.List%ss)", nameLow, model.Name))
				routes.WriteString(fmt.Sprintf("\n\tr.POST(\"/%ss\", handlers.Create%s)", nameLow, model.Name))
			} else {
				routes.WriteString(fmt.Sprintf("\n\tapp.Get(\"/%ss\", handlers.List%ss)", nameLow, model.Name))
				routes.WriteString(fmt.Sprintf("\n\tapp.Post(\"/%ss\", handlers.Create%s)", nameLow, model.Name))
			}
		}
	}

	var err error
	main, err = InjectByMarker(main, "imports", imports.String())
	if err != nil {
		ctx.Warnings = append(ctx.Warnings, Warning{
			Code:     "INJECTION_MARKER_MISSING",
			Severity: "error", // Use error severity to highlight injection breakage to the user
			Message:  "Failed to inject dynamic imports",
			Reason:   err.Error(),
		})
	}

	main, err = InjectByMarker(main, "routes", routes.String())
	if err != nil {
		ctx.Warnings = append(ctx.Warnings, Warning{
			Code:     "INJECTION_MARKER_MISSING",
			Severity: "error",
			Message:  "Failed to inject dynamic routes",
			Reason:   err.Error(),
		})
	}

	ctx.FileTree.Files[mainPath] = main
}
