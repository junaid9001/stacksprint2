package generator

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

func goMod(framework string, root RootOptions) string {
	module := strings.TrimSpace(root.Module)
	if module == "" {
		module = "stacksprint/generated"
	}
	fwDep := "github.com/gin-gonic/gin v1.10.0\n\tgithub.com/kelseyhightower/envconfig v1.4.0"
	if framework == "fiber" {
		fwDep = "github.com/gofiber/fiber/v2 v2.52.6\n\tgithub.com/kelseyhightower/envconfig v1.4.0"
	}
	return fmt.Sprintf("module %s\n\ngo 1.23\n\nrequire (\n\t%s\n)\n", module, fwDep)
}

func nodePackageJSON(framework string, db string, useORM bool) string {
	dep := "express"
	if framework == "fastify" {
		dep = "fastify"
	}
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

func pythonRequirements(framework string, db string, useORM bool) string {
	if framework == "django" {
		base := "Django==5.1.5\ndjangorestframework==3.15.2\n"
		if db == "postgresql" {
			return base + "psycopg[binary]==3.2.3\n"
		}
		if db == "mysql" {
			return base + "mysqlclient==2.2.7\n"
		}
		return base
	}
	base := "fastapi==0.116.0\nuvicorn==0.34.0\npydantic-settings==2.6.1\n"
	if db == "postgresql" {
		if useORM {
			return base + "SQLAlchemy==2.0.36\npsycopg[binary]==3.2.3\n"
		}
		return base + "psycopg[binary]==3.2.3\n"
	}
	if db == "mysql" {
		if useORM {
			return base + "SQLAlchemy==2.0.36\nPyMySQL==1.1.1\n"
		}
		return base + "PyMySQL==1.1.1\n"
	}
	if db != "none" && useORM {
		return base + "SQLAlchemy==2.0.36\npsycopg[binary]==3.2.3\n"
	}
	return base
}

func goConfigLoader() string {
	return `package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Port        int    ` + "`envconfig:\"PORT\" default:\"8080\"`" + `
	DatabaseURL string ` + "`envconfig:\"DATABASE_URL\"`" + `
	JWTSecret   string ` + "`envconfig:\"JWT_SECRET\" default:\"default_dev_secret_replace_in_prod\"`" + `
}

var AppConfig Config

func Init() {
	err := envconfig.Process("", &AppConfig)
	if err != nil {
		log.Fatalf("❌ Environment variable validation failed: %v", err)
	}
}

func Port() string {
	return "%d" // we will return fmt.Sprint(AppConfig.Port) implicitly by changing the references in main later. This is just a stub for backwards compat if needed, but the main template should use config.AppConfig.Port now.
	// We'll update main.go to call config.Init()
}
`
}

func nodeConfigLoader() string {
	return `import dotenv from 'dotenv';
import { z } from 'zod';

dotenv.config();

const envSchema = z.object({
  PORT: z.string().transform(Number).default('8080'),
  DATABASE_URL: z.string().url().optional(),
  JWT_SECRET: z.string().min(8).default('default_dev_secret_replace_in_prod'),
});

const parsed = envSchema.safeParse(process.env);
if (!parsed.success) {
  console.error('❌ Invalid environment variables:', parsed.error.format());
  process.exit(1);
}

export const config = {
  port: parsed.data.PORT,
  dbUrl: parsed.data.DATABASE_URL || '',
  jwtSecret: parsed.data.JWT_SECRET,
};
`
}

func pythonConfigLoader() string {
	return `from pydantic_settings import BaseSettings, SettingsConfigDict

class Settings(BaseSettings):
    port: int = 8080
    database_url: str = ""
    jwt_secret: str = "default_dev_secret_replace_in_prod"

    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8", extra="ignore")

try:
    settings = Settings()
except Exception as e:
    print(f"❌ Configuration error: {e}")
    exit(1)

PORT = settings.port
DATABASE_URL = settings.database_url
`
}

func goLogger() string {
	return "package logger\n\nimport \"log\"\n\nfunc Info(msg string) { log.Println(\"INFO:\", msg) }\nfunc Error(msg string) { log.Println(\"ERROR:\", msg) }\n"
}

func nodeLogger() string {
	return "export const logger = { info: (...a) => console.log('[INFO]', ...a), error: (...a) => console.error('[ERROR]', ...a) };\n"
}

func pythonLogger() string {
	return "import logging\n\nlogging.basicConfig(level=logging.INFO)\nlogger = logging.getLogger(\"stacksprint\")\n"
}

func goGlobalErrorMiddleware(framework string) string {
	if framework == "fiber" {
		return "package middleware\n\nimport \"github.com/gofiber/fiber/v2\"\n\nfunc ErrorHandler(c *fiber.Ctx, err error) error {\n\treturn c.Status(500).JSON(fiber.Map{\"error\": err.Error()})\n}\n"
	}
	return "package middleware\n\nimport \"github.com/gin-gonic/gin\"\n\nfunc ErrorHandler(c *gin.Context) {\n\tc.Next()\n\tif len(c.Errors) > 0 {\n\t\tc.JSON(500, gin.H{\"error\": c.Errors.String()})\n\t}\n}\n"
}

func nodeGlobalError() string {
	return "export function globalError(err, req, res, next) {\n  res.status(500).json({ error: err.message || 'internal error' });\n}\n"
}

func pythonErrorHandler() string {
	return "from fastapi import Request\nfrom fastapi.responses import JSONResponse\n\nasync def global_exception_handler(request: Request, exc: Exception):\n    return JSONResponse(status_code=500, content={\"error\": str(exc)})\n"
}

func goSampleTest() string {
	return "package handlers\n\nimport \"testing\"\n\nfunc TestPlaceholder(t *testing.T) {\n\tif false {\n\t\tt.Fatal(\"expected true\")\n\t}\n}\n"
}

func nodeSampleTest(framework string) string {
	_ = framework
	return "import test from 'node:test';\nimport assert from 'node:assert/strict';\n\ntest('sample', () => {\n  assert.equal(1 + 1, 2);\n});\n"
}

func pythonSampleTest() string {
	return "def test_sample():\n    assert 1 + 1 == 2\n"
}

func addDatabaseBoilerplate(tree *FileTree, req GenerateRequest, root string) {
	p := func(parts ...string) string {
		if root == "" {
			return strings.Join(parts, "/")
		}
		return root + "/" + strings.Join(parts, "/")
	}
	switch req.Language {
	case "go":
		if isSQLDB(req.Database) && req.UseORM {
			driverImport := "\"gorm.io/driver/postgres\""
			driverOpen := "postgres.Open(dsn)"
			if req.Database == "mysql" {
				driverImport = "\"gorm.io/driver/mysql\""
				driverOpen = "mysql.Open(dsn)"
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
	case "node":
		if isSQLDB(req.Database) && req.UseORM {
			addFile(tree, p("src", "db", "connection.js"), "import { PrismaClient } from '@prisma/client';\n\nexport const db = new PrismaClient();\n")
			addFile(tree, p("prisma", "schema.prisma"), renderPrismaSchema(req.Database, req.Custom.Models))
		} else {
			addFile(tree, p("src", "db", "connection.js"), "export const databaseUrl = process.env.DATABASE_URL || '';\n")
		}
		addFile(tree, p("src", "models", "item.js"), "export class Item {\n  constructor(id, name) { this.id = id; this.name = name; }\n}\n")
	case "python":
		if req.Framework == "django" {
			addFile(tree, p("api", "models.py"), "from django.db import models\n\nclass Item(models.Model):\n    name = models.CharField(max_length=255)\n")
		} else {
			if isSQLDB(req.Database) && req.UseORM {
				addFile(tree, p("app", "db.py"), "import os\nfrom sqlalchemy import create_engine\nfrom sqlalchemy.orm import sessionmaker\n\nDATABASE_URL = os.getenv('DATABASE_URL', '')\nengine = create_engine(DATABASE_URL, pool_pre_ping=True)\nSessionLocal = sessionmaker(bind=engine)\n")
				addFile(tree, p("app", "models_orm.py"), renderSQLAlchemyModels(req.Custom.Models))
			} else {
				addFile(tree, p("app", "db.py"), "import os\n\nDATABASE_URL = os.getenv('DATABASE_URL', '')\n")
			}
			addFile(tree, p("app", "models.py"), "from pydantic import BaseModel\n\nclass Item(BaseModel):\n    id: int\n    name: str\n")
		}
	}
}

func addAuthBoilerplate(tree *FileTree, req GenerateRequest, root string) {
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	switch req.Language {
	case "go":
		addFile(tree, prefix+"internal/auth/jwt.go", "package auth\n\nimport \"os\"\n\nfunc Secret() string { return os.Getenv(\"JWT_SECRET\") }\n")
	case "node":
		addFile(tree, prefix+"src/auth/jwt.js", "export const jwtSecret = process.env.JWT_SECRET || 'changeme';\n")
	case "python":
		addFile(tree, prefix+"app/auth.py", "import os\nJWT_SECRET = os.getenv('JWT_SECRET', 'changeme')\n")
	}
}

func addHealthBoilerplate(tree *FileTree, req GenerateRequest, root string) {
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	switch req.Language {
	case "go":
		addFile(tree, prefix+"internal/health/handler.go", "package health\n\nfunc Message() string { return \"ok\" }\n")
	case "node":
		addFile(tree, prefix+"src/routes/health.js", "export default function health(req, res) { res.send({ status: 'ok' }); }\n")
	case "python":
		addFile(tree, prefix+"app/health.py", "def health():\n    return {\"status\": \"ok\"}\n")
	}
}

func addBaseRoute(tree *FileTree, req GenerateRequest, root string) {
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	switch req.Language {
	case "go":
		addFile(tree, prefix+"internal/routes/base.go", "package routes\n\nconst BasePath = \"/api/v1\"\n")
	case "node":
		addFile(tree, prefix+"src/routes/base.js", "export const basePath = '/api/v1';\n")
	case "python":
		addFile(tree, prefix+"app/routes.py", "BASE_PATH = '/api/v1'\n")
	}
}

func addCRUDRoute(tree *FileTree, req GenerateRequest, root string) {
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	switch req.Language {
	case "go":
		if req.Framework == "gin" {
			addFile(tree, prefix+"internal/handlers/items.go", "package handlers\n\nimport \"github.com/gin-gonic/gin\"\n\nfunc ListItems(c *gin.Context) {\n\tc.JSON(200, []gin.H{{\"id\": 1, \"name\": \"sample\"}})\n}\n")
		} else {
			addFile(tree, prefix+"internal/handlers/items.go", "package handlers\n\nimport \"github.com/gofiber/fiber/v2\"\n\nfunc ListItems(c *fiber.Ctx) error {\n\treturn c.JSON([]map[string]any{{\"id\": 1, \"name\": \"sample\"}})\n}\n")
		}
	case "node":
		addFile(tree, prefix+"src/routes/items.js", "export function listItems(req, res) {\n  res.json([{ id: 1, name: 'sample' }]);\n}\n")
	case "python":
		if req.Framework == "django" {
			addFile(tree, prefix+"api/views.py", "from rest_framework.decorators import api_view\nfrom rest_framework.response import Response\n\n@api_view(['GET'])\ndef items(request):\n    return Response([{\"id\": 1, \"name\": \"sample\"}])\n")
		} else {
			addFile(tree, prefix+"app/items.py", "from fastapi import APIRouter\n\nrouter = APIRouter(prefix='/items')\n\n@router.get('')\ndef list_items():\n    return [{\"id\": 1, \"name\": \"sample\"}]\n")
		}
	}
}

func dockerfile(req GenerateRequest, service string) string {
	_ = service
	switch req.Language {
	case "go":
		return "FROM golang:1.23-alpine AS build\nWORKDIR /app\nCOPY . .\nRUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app ./cmd/server\n\nFROM scratch\nWORKDIR /app\nCOPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/\nCOPY --from=build /app/app .\nEXPOSE 8080\nCMD [\"./app\"]\n"
	case "node":
		return "FROM node:22-alpine AS deps\nWORKDIR /app\nCOPY package*.json ./\nRUN npm ci\n\nFROM node:22-alpine AS runner\nWORKDIR /app\nENV NODE_ENV production\nCOPY --from=deps /app/node_modules ./node_modules\nCOPY . .\nEXPOSE 8080\nCMD [\"npm\", \"start\"]\n"
	default:
		if req.Framework == "django" {
			return "FROM python:3.12-slim AS builder\nWORKDIR /app\nRUN python -m venv /opt/venv\nENV PATH=\"/opt/venv/bin:$PATH\"\nCOPY requirements.txt .\nRUN pip install --no-cache-dir -r requirements.txt\n\nFROM python:3.12-slim\nWORKDIR /app\nCOPY --from=builder /opt/venv /opt/venv\nENV PATH=\"/opt/venv/bin:$PATH\"\nCOPY . .\nEXPOSE 8080\nCMD [\"python\", \"manage.py\", \"runserver\", \"0.0.0.0:8080\"]\n"
		}
		return "FROM python:3.12-slim AS builder\nWORKDIR /app\nRUN python -m venv /opt/venv\nENV PATH=\"/opt/venv/bin:$PATH\"\nCOPY requirements.txt .\nRUN pip install --no-cache-dir -r requirements.txt\n\nFROM python:3.12-slim\nWORKDIR /app\nCOPY --from=builder /opt/venv /opt/venv\nENV PATH=\"/opt/venv/bin:$PATH\"\nCOPY . .\nEXPOSE 8080\nCMD [\"uvicorn\", \"app.main:app\", \"--host\", \"0.0.0.0\", \"--port\", \"8080\"]\n"
	}
}

func buildEnv(req GenerateRequest, service string, port int) string {
	prefix := ""
	if service != "" {
		prefix = strings.ToUpper(strings.ReplaceAll(service, "-", "_")) + "_"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("PORT=%d\n", port))
	if req.Database != "none" {
		switch req.Database {
		case "postgresql":
			b.WriteString("DATABASE_URL=postgres://app:app@postgres:5432/app?sslmode=disable\n")
		case "mysql":
			b.WriteString("DATABASE_URL=mysql://app:app@mysql:3306/app\n")
		case "mongodb":
			b.WriteString("DATABASE_URL=mongodb://mongo:27017/app\n")
		}
	}
	if req.Features.JWTAuth {
		b.WriteString("JWT_SECRET=replace-me\n")
	}
	if req.Infra.Redis {
		b.WriteString(prefix + "REDIS_ADDR=redis:6379\n")
	}
	if req.Infra.Kafka {
		b.WriteString(prefix + "KAFKA_BROKERS=kafka:9092\n")
	}
	if req.Infra.NATS {
		b.WriteString(prefix + "NATS_URL=nats://nats:4222\n")
	}
	return b.String()
}

func buildCompose(req GenerateRequest) string {
	var b strings.Builder
	b.WriteString("services:\n")
	if req.Architecture == "microservices" {
		for _, svc := range req.Services {
			b.WriteString(fmt.Sprintf("  %s:\n", svc.Name))
			b.WriteString(fmt.Sprintf("    build: ./services/%s\n", svc.Name))
			b.WriteString(fmt.Sprintf("    ports:\n      - \"%d:%d\"\n", svc.Port, svc.Port))
			b.WriteString(fmt.Sprintf("    env_file:\n      - ./services/%s/.env\n", svc.Name))
			if req.Database != "none" {
				b.WriteString("    depends_on:\n")
				b.WriteString(fmt.Sprintf("      %s:\n        condition: service_healthy\n", composeDBServiceName(req.Database)))
			}
		}
	} else {
		b.WriteString("  app:\n")
		b.WriteString("    build: .\n")
		b.WriteString("    ports:\n      - \"8080:8080\"\n")
		if isEnabled(req.FileToggles.Env) {
			b.WriteString("    env_file:\n      - ./.env\n")
		}
		if req.Database != "none" {
			b.WriteString("    depends_on:\n")
			b.WriteString(fmt.Sprintf("      %s:\n        condition: service_healthy\n", composeDBServiceName(req.Database)))
		}
	}

	appendDBCompose(&b, req.Database)
	if req.Infra.Redis {
		b.WriteString("  redis:\n    image: redis:7-alpine\n    ports:\n      - \"6379:6379\"\n    healthcheck:\n      test: [\"CMD\", \"redis-cli\", \"ping\"]\n      interval: 5s\n      timeout: 3s\n      retries: 10\n")
	}
	if req.Infra.Kafka {
		b.WriteString("  kafka:\n    image: bitnami/kafka:3.9\n    ports:\n      - \"9092:9092\"\n    environment:\n      - KAFKA_CFG_NODE_ID=1\n      - KAFKA_CFG_PROCESS_ROLES=broker,controller\n      - KAFKA_CFG_CONTROLLER_LISTENER_NAMES=CONTROLLER\n      - KAFKA_CFG_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093\n      - KAFKA_CFG_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092\n      - KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT\n      - KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=1@kafka:9093\n      - ALLOW_PLAINTEXT_LISTENER=yes\n    healthcheck:\n      test: [\"CMD-SHELL\", \"kafka-broker-api-versions.sh --bootstrap-server localhost:9092\"]\n      interval: 10s\n      timeout: 5s\n      retries: 10\n")
	}
	if req.Infra.NATS {
		b.WriteString("  nats:\n    image: nats:2.10-alpine\n    ports:\n      - \"4222:4222\"\n")
	}
	return b.String()
}

func appendDBCompose(b *strings.Builder, db string) {
	switch db {
	case "postgresql":
		b.WriteString("  postgres:\n    image: postgres:16-alpine\n    environment:\n      POSTGRES_DB: app\n      POSTGRES_USER: app\n      POSTGRES_PASSWORD: app\n    volumes:\n      - ./db/init:/docker-entrypoint-initdb.d\n    ports:\n      - \"5432:5432\"\n    healthcheck:\n      test: [\"CMD-SHELL\", \"pg_isready -U app -d app\"]\n      interval: 5s\n      timeout: 5s\n      retries: 12\n")
	case "mysql":
		b.WriteString("  mysql:\n    image: mysql:8.4\n    environment:\n      MYSQL_DATABASE: app\n      MYSQL_USER: app\n      MYSQL_PASSWORD: app\n      MYSQL_ROOT_PASSWORD: root\n    volumes:\n      - ./db/init:/docker-entrypoint-initdb.d\n    ports:\n      - \"3306:3306\"\n    healthcheck:\n      test: [\"CMD-SHELL\", \"mysqladmin ping -h localhost -uapp -papp\"]\n      interval: 5s\n      timeout: 5s\n      retries: 12\n")
	case "mongodb":
		b.WriteString("  mongo:\n    image: mongo:8\n    ports:\n      - \"27017:27017\"\n    healthcheck:\n      test: [\"CMD-SHELL\", \"mongosh --quiet --eval 'db.adminCommand({ ping: 1 })'\"]\n      interval: 5s\n      timeout: 5s\n      retries: 12\n")
	}
}

func composeDBServiceName(db string) string {
	if db == "postgresql" {
		return "postgres"
	}
	if db == "mysql" {
		return "mysql"
	}
	if db == "mongodb" {
		return "mongo"
	}
	return "db"
}

func sampleMigration(db string, models []DataModel) string {
	if db == "mongodb" {
		return "// MongoDB migrations are usually handled by migration tools at runtime.\n"
	}
	return renderSQLTablesTemplate(models, false)
}

func sampleDBInit(db string, models []DataModel) string {
	if db == "mongodb" {
		return "db = db.getSiblingDB('app');\ndb.createCollection('items');\n"
	}
	return renderSQLTablesTemplate(models, true)
}

func renderSQLTablesTemplate(models []DataModel, withSeed bool) string {
	const tpl = `{{ range .Models -}}
CREATE TABLE IF NOT EXISTS {{ .TableName }} (
  id SERIAL PRIMARY KEY{{ range .Columns }},
  {{ .Name }} {{ .SQLType }}{{ end }}
);
{{ if $.WithSeed }}INSERT INTO {{ .TableName }} DEFAULT VALUES;
{{ end }}

{{ end -}}`

	type sqlColumn struct {
		Name    string
		SQLType string
	}
	type sqlTable struct {
		TableName string
		Columns   []sqlColumn
	}
	type sqlPayload struct {
		WithSeed bool
		Models   []sqlTable
	}

	resolved := resolvedModels(models)
	tables := make([]sqlTable, 0, len(resolved))
	for _, model := range resolved {
		table := sqlTable{
			TableName: strings.ToLower(model.Name) + "s",
			Columns:   make([]sqlColumn, 0, len(model.Fields)),
		}
		for _, field := range model.Fields {
			if strings.EqualFold(field.Name, "id") {
				continue
			}
			table.Columns = append(table.Columns, sqlColumn{
				Name:    strings.ToLower(field.Name),
				SQLType: sqlTypeFromField(field.Type),
			})
		}
		tables = append(tables, table)
	}

	t, err := template.New("sql-migrations").Parse(tpl)
	if err != nil {
		return ""
	}
	var b bytes.Buffer
	if err := t.Execute(&b, sqlPayload{WithSeed: withSeed, Models: tables}); err != nil {
		return ""
	}
	return b.String()
}

func sqlTypeFromField(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "int", "integer":
		return "INT"
	case "float", "float64", "double", "decimal":
		return "DECIMAL(10,2)"
	case "bool", "boolean":
		return "BOOLEAN"
	case "datetime", "timestamp", "time":
		return "TIMESTAMP"
	default:
		return "VARCHAR(255)"
	}
}

func buildREADME(req GenerateRequest) string {
	return fmt.Sprintf("# StackSprint Generated Project\n\nLanguage: %s\nFramework: %s\nArchitecture: %s\nDatabase: %s\n\n## Run\n\n```bash\ndocker compose up --build\n```\n", req.Language, req.Framework, req.Architecture, req.Database)
}

func buildCIPipeline(req GenerateRequest) string {
	_ = req
	return "name: CI\n\non:\n  push:\n  pull_request:\n\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v4\n      - uses: actions/setup-node@v4\n        if: hashFiles('package.json') != ''\n        with:\n          node-version: '22'\n      - uses: actions/setup-go@v5\n        if: hashFiles('go.mod') != ''\n        with:\n          go-version: '1.23'\n      - uses: actions/setup-python@v5\n        if: hashFiles('requirements.txt') != ''\n        with:\n          python-version: '3.12'\n      - run: go test ./...\n        if: hashFiles('go.mod') != ''\n      - run: npm test\n        if: hashFiles('package.json') != ''\n      - run: pytest\n        if: hashFiles('requirements.txt') != ''\n"
}

func buildMakefile(req GenerateRequest) string {
	var b strings.Builder
	b.WriteString("up:\n\tdocker compose up --build\n\ndown:\n\tdocker compose down -v\n\ntest:\n\t@echo \"Run language-specific tests\"\n")
	if req.Database != "none" {
		if req.Language == "go" {
			b.WriteString("\nmigrate-up:\n\t@echo \"Running migrations up\"\n\t# migrate -path db/migrations -database \"$$DATABASE_URL\" up\n")
			b.WriteString("\nmigrate-down:\n\t@echo \"Running migrations down\"\n\t# migrate -path db/migrations -database \"$$DATABASE_URL\" down\n")
			b.WriteString("\nseed:\n\t@echo \"Running seeder\"\n\tgo run cmd/seeder/main.go\n")
		} else if req.Language == "node" {
			if req.UseORM {
				b.WriteString("\nseed:\n\t@echo \"Running Prisma Seeder\"\n\tnext prisma db seed\n")
			} else {
				b.WriteString("\nseed:\n\t@echo \"Running raw SQL seed\"\n\tnode scripts/seed.js\n")
			}
		} else if req.Language == "python" {
			b.WriteString("\nseed:\n\t@echo \"Running Python Seeder\"\n\tpython scripts/seed.py\n")
		}
	}
	return b.String()
}

func buildOpenAPI(req GenerateRequest) string {
	_ = req
	return "openapi: 3.0.3\ninfo:\n  title: StackSprint API\n  version: 1.0.0\npaths:\n  /health:\n    get:\n      responses:\n        '200':\n          description: OK\n"
}

func addDjangoFiles(tree *FileTree, req GenerateRequest, main string) {
	_ = main
	addFile(tree, "manage.py", "#!/usr/bin/env python\nimport os\nimport sys\n\nif __name__ == '__main__':\n    os.environ.setdefault('DJANGO_SETTINGS_MODULE', 'config.settings')\n    from django.core.management import execute_from_command_line\n    execute_from_command_line(sys.argv)\n")
	addFile(tree, "config/__init__.py", "")
	addFile(tree, "config/settings.py", djangoSettings(req.Database != "none"))
	addFile(tree, "config/urls.py", "from django.urls import include, path\n\nurlpatterns = [path('api/', include('api.urls')),]\n")
	addFile(tree, "config/wsgi.py", "import os\nfrom django.core.wsgi import get_wsgi_application\nos.environ.setdefault('DJANGO_SETTINGS_MODULE', 'config.settings')\napplication = get_wsgi_application()\n")
	addFile(tree, "api/__init__.py", "")
	addFile(tree, "api/apps.py", "from django.apps import AppConfig\n\nclass ApiConfig(AppConfig):\n    default_auto_field = 'django.db.models.BigAutoField'\n    name = 'api'\n")
	addFile(tree, "api/urls.py", "from django.urls import path\nfrom .views import health, items\n\nurlpatterns = [\n    path('health', health),\n    path('items', items),\n]\n")
	addFile(tree, "api/views.py", "from rest_framework.decorators import api_view\nfrom rest_framework.response import Response\n\n@api_view(['GET'])\ndef health(request):\n    return Response({\"status\": \"ok\"})\n\n@api_view(['GET'])\ndef items(request):\n    return Response([{\"id\": 1, \"name\": \"sample\"}])\n")
	addFile(tree, "requirements.txt", pythonRequirements("django", req.Database, req.UseORM))
}

func addDjangoFilesAtRoot(tree *FileTree, req GenerateRequest, main string, root string) {
	_ = main
	addFile(tree, root+"/manage.py", "#!/usr/bin/env python\nimport os\nimport sys\n\nif __name__ == '__main__':\n    os.environ.setdefault('DJANGO_SETTINGS_MODULE', 'config.settings')\n    from django.core.management import execute_from_command_line\n    execute_from_command_line(sys.argv)\n")
	addFile(tree, root+"/config/__init__.py", "")
	addFile(tree, root+"/config/settings.py", djangoSettings(req.Database != "none"))
	addFile(tree, root+"/config/urls.py", "from django.urls import include, path\nurlpatterns = [path('api/', include('api.urls')),]\n")
	addFile(tree, root+"/config/wsgi.py", "import os\nfrom django.core.wsgi import get_wsgi_application\nos.environ.setdefault('DJANGO_SETTINGS_MODULE', 'config.settings')\napplication = get_wsgi_application()\n")
	addFile(tree, root+"/api/__init__.py", "")
	addFile(tree, root+"/api/apps.py", "from django.apps import AppConfig\n\nclass ApiConfig(AppConfig):\n    default_auto_field = 'django.db.models.BigAutoField'\n    name = 'api'\n")
	addFile(tree, root+"/api/urls.py", "from django.urls import path\nfrom .views import health, items\nurlpatterns = [path('health', health), path('items', items)]\n")
	addFile(tree, root+"/api/views.py", "from rest_framework.decorators import api_view\nfrom rest_framework.response import Response\n\n@api_view(['GET'])\ndef health(request):\n    return Response({\"status\": \"ok\"})\n\n@api_view(['GET'])\ndef items(request):\n    return Response([{\"id\":1,\"name\":\"sample\"}])\n")
	addFile(tree, root+"/requirements.txt", pythonRequirements("django", req.Database, req.UseORM))
}

func djangoSettings(withDB bool) string {
	db := "\"ENGINE\": \"django.db.backends.sqlite3\", \"NAME\": BASE_DIR / \"db.sqlite3\""
	if withDB {
		db = "\"ENGINE\": \"django.db.backends.postgresql\", \"NAME\": \"app\", \"USER\": \"app\", \"PASSWORD\": \"app\", \"HOST\": \"postgres\", \"PORT\": \"5432\""
	}
	return fmt.Sprintf("from pathlib import Path\n\nBASE_DIR = Path(__file__).resolve().parent.parent\nSECRET_KEY = 'dev'\nDEBUG = True\nALLOWED_HOSTS = ['*']\nINSTALLED_APPS = ['django.contrib.contenttypes', 'django.contrib.auth', 'rest_framework', 'api']\nMIDDLEWARE = []\nROOT_URLCONF = 'config.urls'\nTEMPLATES = []\nWSGI_APPLICATION = 'config.wsgi.application'\nDATABASES = {'default': {%s}}\nLANGUAGE_CODE = 'en-us'\nTIME_ZONE = 'UTC'\nUSE_I18N = True\nUSE_TZ = True\n", db)
}
