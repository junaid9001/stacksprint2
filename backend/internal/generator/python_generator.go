package generator

import (
	"fmt"
	"path"
	"strings"
)

type PythonGenerator struct{}

func (g *PythonGenerator) GenerateArchitecture(req *GenerateRequest, ctx *GenerationContext) error {
	if req.Architecture == "microservices" {
		for _, svc := range req.Services {
			svcRoot := path.Join("services", svc.Name)
			if err := g.generateServiceArch(req, ctx, svcRoot, svc); err != nil {
				return err
			}
			if isEnabled(req.FileToggles.BaseRoute) {
				if req.Framework != "django" {
					addFile(ctx.FileTree, path.Join(svcRoot, "app/routes/base.py"), "from fastapi import APIRouter\n\nrouter = APIRouter()\n")

					if main, ok := ctx.FileTree.Files[path.Join(svcRoot, "app/main.py")]; ok {
						if !strings.Contains(main, "from app.routes.base import router as base_router") {
							var err error
							main, err = InjectByMarker(main, "imports", "from app.routes.base import router as base_router\n")
							if err == nil {
								main, _ = InjectByMarker(main, "routes", "app.include_router(base_router, prefix=\"/api/v1\")\n")
								ctx.FileTree.Files[path.Join(svcRoot, "app/main.py")] = main
							}
						}
					}
				}
			}
			if isEnabled(req.FileToggles.ExampleCRUD) {
				if req.Framework != "django" {
					// example crud already injected via dynamic models mostly, but we add a generic item one if needed. (Skipped if clean/hexagonal since it generates models).
					// Actually we just generate the dynamic models in GenerateModels.
				}
			}
			if isEnabled(req.FileToggles.HealthCheck) || req.Features.Health {
				if req.Framework != "django" {
					addFile(ctx.FileTree, path.Join(svcRoot, "app/routes/health.py"), "from fastapi import APIRouter\n\nrouter = APIRouter()\n\n@router.get('/health')\ndef health():\n    return {'status': 'ok'}\n")
				}
			}
			if req.Features.JWTAuth {
				if req.Framework != "django" {
					addFile(ctx.FileTree, path.Join(svcRoot, "app/auth/jwt.py"), "import os\n\nJWT_SECRET = os.getenv('JWT_SECRET', 'changeme')\n")
				}
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
			if req.Framework != "django" {
				addFile(ctx.FileTree, "app/routes/base.py", "from fastapi import APIRouter\n\nrouter = APIRouter()\n")
			}
		}
		if isEnabled(req.FileToggles.HealthCheck) || req.Features.Health {
			if req.Framework != "django" {
				addFile(ctx.FileTree, "app/routes/health.py", "from fastapi import APIRouter\n\nrouter = APIRouter()\n\n@router.get('/health')\ndef health():\n    return {'status': 'ok'}\n")
			}
		}
		if req.Features.JWTAuth {
			if req.Framework != "django" {
				addFile(ctx.FileTree, "app/auth/jwt.py", "import os\n\nJWT_SECRET = os.getenv('JWT_SECRET', 'changeme')\n")
			}
		}
		if strings.EqualFold(req.ServiceCommunication, "grpc") {
			addFile(ctx.FileTree, "proto/README.md", "# Shared proto definitions\n\nPlace your protobuf contracts here.\n")
			addFile(ctx.FileTree, "proto/common.proto", "syntax = \"proto3\";\npackage stacksprint;\n\nservice InternalService {\n  rpc Ping(PingRequest) returns (PingReply);\n}\n\nmessage PingRequest {\n  string source = 1;\n}\n\nmessage PingReply {\n  string message = 1;\n}\n")
			g.addGRPCBoilerplate(ctx.FileTree, req, "")
		}
	}
	return nil
}

func (g *PythonGenerator) GenerateModels(req *GenerateRequest, ctx *GenerationContext) error {
	if req.Architecture == "microservices" {
		for _, svc := range req.Services {
			svcRoot := path.Join("services", svc.Name)
			if req.Database != "none" {
				g.addPythonDBBoilerplate(ctx.FileTree, req, svcRoot)
			}
			if isEnabled(req.FileToggles.ExampleCRUD) && req.Framework != "django" {
				for _, model := range resolvedModels(req.Custom.Models) {
					g.renderPythonDynamicModel(ctx.FileTree, req, model, req.Architecture, svcRoot)
				}
			}
		}
	} else {
		if req.Database != "none" {
			g.addPythonDBBoilerplate(ctx.FileTree, req, "")
		}
		if isEnabled(req.FileToggles.ExampleCRUD) && req.Framework != "django" {
			for _, model := range resolvedModels(req.Custom.Models) {
				g.renderPythonDynamicModel(ctx.FileTree, req, model, req.Architecture, "")
			}
		}
	}
	return nil
}

func (g *PythonGenerator) GenerateInfra(req *GenerateRequest, ctx *GenerationContext) error {
	handleInfra := func(root string, port int) {
		if req.Infra.Redis {
			addFile(ctx.FileTree, path.Join(root, "app/cache/redis_cache.py"), "import os\n\nclass RedisCache:\n    def __init__(self, addr: str | None = None):\n        self.addr = addr or os.getenv('REDIS_ADDR', 'redis:6379')\n\n    def ping(self) -> str:\n        return f'redis configured at {self.addr}'\n")
		}
		if req.Infra.Kafka {
			addFile(ctx.FileTree, path.Join(root, "app/messaging/kafka_producer.py"), "import os\n\nclass KafkaProducer:\n    def __init__(self, brokers: str | None = None):\n        self.brokers = brokers or os.getenv('KAFKA_BROKERS', 'kafka:9092')\n\n    def publish(self, topic: str, payload: str) -> str:\n        return f'publish stub to {topic} via {self.brokers}: {payload}'\n")
			addFile(ctx.FileTree, path.Join(root, "app/messaging/kafka_consumer.py"), "import os\n\nclass KafkaConsumer:\n    def __init__(self, brokers: str | None = None):\n        self.brokers = brokers or os.getenv('KAFKA_BROKERS', 'kafka:9092')\n\n    def subscribe(self, topic: str) -> str:\n        return f'consumer stub subscribed to {topic} via {self.brokers}'\n")
		}
		if isEnabled(req.FileToggles.Env) {
			svcName := ""
			if root != "" {
				svcName = path.Base(root)
			}
			addFile(ctx.FileTree, path.Join(root, ".env"), buildEnv(*req, svcName, port))
		}
		if isEnabled(req.FileToggles.Dockerfile) {
			addFile(ctx.FileTree, path.Join(root, "Dockerfile"), "FROM python:3.11-slim\nWORKDIR /app\nCOPY requirements.txt .\nRUN pip install --no-cache-dir -r requirements.txt\nCOPY . .\nEXPOSE 8080\nCMD [\"uvicorn\", \"app.main:app\", \"--host\", \"0.0.0.0\", \"--port\", \"8080\"]\n")
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

func (g *PythonGenerator) GenerateDevTools(req *GenerateRequest, ctx *GenerationContext) error {
	if isEnabled(req.FileToggles.Gitignore) {
		addFile(ctx.FileTree, ".gitignore", "venv/\n__pycache__/\n*.pyc\n.env\n.DS_Store\n*.sqlite3\n.coverage\n")
	}
	if isEnabled(req.FileToggles.Readme) {
		addFile(ctx.FileTree, "README.md", fmt.Sprintf("# StackSprint Generated Project\n\nLanguage: %s\nFramework: %s\nArchitecture: %s\nDatabase: %s\n\n## Run\n\n```bash\ndocker compose up --build\n```\n", req.Language, req.Framework, req.Architecture, req.Database))
	}
	if req.Features.GitHubActions {
		addFile(ctx.FileTree, ".github/workflows/ci.yaml", "name: CI\n\non:\n  push:\n  pull_request:\n\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v4\n      - uses: actions/setup-python@v5\n        with:\n          python-version: '3.11'\n      - run: pip install pytest fastapi && pytest\n")
	}
	if req.Features.Makefile {
		var b strings.Builder
		b.WriteString("up:\n\tdocker compose up --build\n\ndown:\n\tdocker compose down -v\n\ntest:\n\tpytest\n")
		if req.Database != "none" && req.Framework != "django" {
			if req.UseORM {
				b.WriteString("\nmigrate-up:\n\t@echo \"Running Alembic migrations\"\n\talembic upgrade head\n")
			}
			b.WriteString("\nseed:\n\t@echo \"Running Python Seeder\"\n\tpython scripts/seed.py\n")
		} else if req.Framework == "django" {
			b.WriteString("\nmigrate-up:\n\tpython manage.py migrate\n")
		}
		addFile(ctx.FileTree, "Makefile", b.String())
	}
	if req.Features.Swagger {
		// FastAPI has built-in swagger. We add openapi.yaml for completeness if requested.
		addFile(ctx.FileTree, "docs/openapi.yaml", "openapi: 3.0.3\ninfo:\n  title: StackSprint API\n  version: 1.0.0\npaths:\n  /health:\n    get:\n      responses:\n        '200':\n          description: OK\n")
	}
	return nil
}

// GetInitCommand returns the bash init command for Python projects.
func (g *PythonGenerator) GetInitCommand(_ *GenerateRequest) string {
	return "python -m venv venv\nsource venv/bin/activate\npip install -r requirements.txt\n"
}

// GetConfigWarnings returns Python/Django-specific configuration warnings.
// The Django ORM warning lives here — keeping req.Language/req.Framework checks OUT of scripts.go.
func (g *PythonGenerator) GetConfigWarnings(req *GenerateRequest) []Warning {
	if req.Framework == "django" && req.UseORM && req.Database != "none" {
		return []Warning{{
			Code:     "DJANGO_BUILTIN_ORM",
			Severity: "info",
			Message:  "Django uses built-in ORM; SQLAlchemy toggle is not applied for Django mode.",
			Reason:   "Framework boundary dictates internal ORM driver.",
		}}
	}
	return nil
}

// -------------------------------------------------------------------------
// Helper Functions (Internal python_generator)
// -------------------------------------------------------------------------

func (g *PythonGenerator) generateMonolithArch(req *GenerateRequest, ctx *GenerationContext, root string) error {
	specs := pythonMonolithTemplateSpecs(*req)
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

	if req.Framework == "django" {
		main, err := ctx.Registry.Render("python/"+archTemplateName(req.Architecture)+"/main.tmpl", data)
		if err != nil {
			return err
		}
		addDjangoFiles(ctx.FileTree, *req, main)
	} else {
		if err := g.renderSpecs(ctx, specs, data, root); err != nil {
			return err
		}
		if isEnabled(req.FileToggles.ExampleCRUD) {
			var imports, routes strings.Builder
			for _, model := range resolvedModels(req.Custom.Models) {
				nameLow := toSnake(model.Name)
				if req.Architecture == "clean" {
					imports.WriteString(fmt.Sprintf("from app.delivery.http.%s_controller import router as %s_router\n", nameLow, nameLow))
					routes.WriteString(fmt.Sprintf("app.include_router(%s_router)\n", nameLow))
				} else if req.Architecture == "hexagonal" {
					imports.WriteString(fmt.Sprintf("from app.adapters.primary.http.%s_controller import %s_router\n", nameLow, nameLow))
					routes.WriteString(fmt.Sprintf("app.include_router(%s_router)\n", nameLow))
				} else {
					imports.WriteString(fmt.Sprintf("from app.routes.%ss import router as %ss_router\n", nameLow, nameLow))
					routes.WriteString(fmt.Sprintf("app.include_router(%ss_router)\n", nameLow))
				}
			}

			if main, ok := ctx.FileTree.Files["app/main.py"]; ok {
				var err error
				main, err = InjectByMarker(main, "imports", imports.String())
				if err != nil {
					ctx.Warnings = append(ctx.Warnings, Warning{Code: "INJECTION_MARKER_MISSING", Severity: "error", Message: "Failed to inject dynamic imports", Reason: err.Error()})
				}
				main, err = InjectByMarker(main, "routes", routes.String())
				if err != nil {
					ctx.Warnings = append(ctx.Warnings, Warning{Code: "INJECTION_MARKER_MISSING", Severity: "error", Message: "Failed to inject dynamic routes", Reason: err.Error()})
				}
				ctx.FileTree.Files["app/main.py"] = main
			}
		}
	}

	if req.Framework != "django" {
		addFile(ctx.FileTree, "requirements.txt", pythonRequirements(req.Framework, req.Database, req.UseORM))
	}
	if isEnabled(req.FileToggles.Config) && req.Framework != "django" {
		addFile(ctx.FileTree, "app/config/settings.py", "from pydantic_settings import BaseSettings\n\nclass Settings(BaseSettings):\n    port: int = 8080\n    database_url: str = \"\"\n    jwt_secret: str = \"default_dev_secret_replace_in_prod\"\n\n    class Config:\n        env_file = \".env\"\n\nsettings = Settings()\n")
	}
	if (req.Features.Logger || isEnabled(req.FileToggles.Logger)) && req.Framework != "django" {
		addFile(ctx.FileTree, "app/logger/logger.py", "import logging\n\nlogger = logging.getLogger(\"stacksprint\")\nlogger.setLevel(logging.INFO)\nch = logging.StreamHandler()\nch.setFormatter(logging.Formatter(\"%(asctime)s - %(name)s - %(levelname)s - %(message)s\"))\nlogger.addHandler(ch)\n")
	}
	if req.Features.GlobalError && req.Framework != "django" {
		addFile(ctx.FileTree, "app/middleware/error_handler.py", "from fastapi import Request\nfrom fastapi.responses import JSONResponse\n\nasync def global_exception_handler(request: Request, exc: Exception):\n    return JSONResponse(status_code=500, content={\"error\": str(exc)})\n")
	}
	if req.Features.SampleTest && req.Framework != "django" {
		addFile(ctx.FileTree, "tests/test_items.py", "def test_sample():\n    assert 1 + 1 == 2\n")
	}

	g.addPythonAutopilot(ctx.FileTree, req, root)
	g.addPythonDBRetry(ctx.FileTree, req, root)
	return nil
}

func (g *PythonGenerator) generateServiceArch(req *GenerateRequest, ctx *GenerationContext, svcRoot string, svc ServiceConfig) error {
	specs := pythonMicroserviceTemplateSpecs(*req)
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

	if req.Framework == "django" {
		main, err := ctx.Registry.Render("python/microservice/main.tmpl", data)
		if err != nil {
			return err
		}
		addDjangoFilesAtRoot(ctx.FileTree, *req, main, svcRoot)
	} else {
		if err := g.renderSpecs(ctx, specs, data, svcRoot); err != nil {
			return err
		}
		if isEnabled(req.FileToggles.ExampleCRUD) {
			var imports, routes strings.Builder
			for _, model := range resolvedModels(req.Custom.Models) {
				nameLow := toSnake(model.Name)
				if req.Architecture == "clean" {
					imports.WriteString(fmt.Sprintf("from app.delivery.http.%s_controller import router as %s_router\n", nameLow, nameLow))
					routes.WriteString(fmt.Sprintf("app.include_router(%s_router)\n", nameLow))
				} else if req.Architecture == "hexagonal" {
					imports.WriteString(fmt.Sprintf("from app.adapters.primary.http.%s_controller import %s_router\n", nameLow, nameLow))
					routes.WriteString(fmt.Sprintf("app.include_router(%s_router)\n", nameLow))
				} else {
					imports.WriteString(fmt.Sprintf("from app.routes.%ss import router as %ss_router\n", nameLow, nameLow))
					routes.WriteString(fmt.Sprintf("app.include_router(%ss_router)\n", nameLow))
				}
			}

			if main, ok := ctx.FileTree.Files[path.Join(svcRoot, "app/main.py")]; ok {
				var err error
				main, err = InjectByMarker(main, "imports", imports.String())
				if err != nil {
					ctx.Warnings = append(ctx.Warnings, Warning{Code: "INJECTION_MARKER_MISSING", Severity: "error", Message: "Failed to inject dynamic imports for service " + svc.Name, Reason: err.Error()})
				}
				main, err = InjectByMarker(main, "routes", routes.String())
				if err != nil {
					ctx.Warnings = append(ctx.Warnings, Warning{Code: "INJECTION_MARKER_MISSING", Severity: "error", Message: "Failed to inject dynamic routes for service " + svc.Name, Reason: err.Error()})
				}
				ctx.FileTree.Files[path.Join(svcRoot, "app/main.py")] = main
			}
		}
		addFile(ctx.FileTree, path.Join(svcRoot, "requirements.txt"), pythonRequirements(req.Framework, req.Database, req.UseORM))
	}

	g.addPythonAutopilot(ctx.FileTree, req, svcRoot)
	g.addPythonDBRetry(ctx.FileTree, req, svcRoot)
	return nil
}

func (g *PythonGenerator) renderSpecs(ctx *GenerationContext, specs []templateSpec, data map[string]any, root string) error {
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

func (g *PythonGenerator) addGRPCBoilerplate(tree *FileTree, req *GenerateRequest, root string) {
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	addFile(tree, prefix+"app/grpc_server.py", "def start_grpc_server() -> str:\n    return 'gRPC server stub started'\n")
	addFile(tree, prefix+"app/grpc_client.py", "def ping_grpc(target: str = '127.0.0.1:9090') -> str:\n    return f'gRPC client stub pinging {target}'\n")
}

func (g *PythonGenerator) addPythonAutopilot(tree *FileTree, req *GenerateRequest, root string) {
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	if req.Framework == "django" {
		addFile(tree, prefix+"api/middleware.py", "import uuid\nfrom django.utils.deprecation import MiddlewareMixin\nimport time\nimport logging\n\nlogger = logging.getLogger(__name__)\n\nclass RequestIDMiddleware(MiddlewareMixin):\n    def process_request(self, request):\n        request_id = request.headers.get('X-Request-ID', str(uuid.uuid4()))\n        request.request_id = request_id\n\n    def process_response(self, request, response):\n        rid = getattr(request, 'request_id', '-')\n        response['X-Request-ID'] = rid\n        return response\n\nclass RequestLoggerMiddleware(MiddlewareMixin):\n    def process_request(self, request):\n        request._start_time = time.monotonic()\n\n    def process_response(self, request, response):\n        duration_ms = (time.monotonic() - getattr(request, '_start_time', time.monotonic())) * 1000\n        logger.info(\"%s %s → %d (%.1fms) rid=%s\", request.method, request.path, response.status_code, duration_ms, getattr(request, 'request_id', '-'))\n        return response\n")
		addFile(tree, prefix+"api/pagination.py", "def parse_page(limit: int = 20, offset: int = 0) -> dict:\n    return {\"limit\": min(max(limit, 1), 100), \"offset\": max(offset, 0)}\n")
		return
	}

	addFile(tree, prefix+"app/middleware/request_id.py", "import uuid\nfrom starlette.middleware.base import BaseHTTPMiddleware\nfrom starlette.requests import Request\n\nclass RequestIDMiddleware(BaseHTTPMiddleware):\n    async def dispatch(self, request: Request, call_next):\n        request_id = request.headers.get(\"X-Request-ID\", str(uuid.uuid4()))\n        request.state.request_id = request_id\n        response = await call_next(request)\n        response.headers[\"X-Request-ID\"] = request_id\n        return response\n")
	addFile(tree, prefix+"app/middleware/request_logger.py", "import time\nimport logging\nfrom starlette.middleware.base import BaseHTTPMiddleware\nfrom starlette.requests import Request\n\nlogger = logging.getLogger(\"stacksprint\")\n\nclass RequestLoggerMiddleware(BaseHTTPMiddleware):\n    async def dispatch(self, request: Request, call_next):\n        start = time.monotonic()\n        response = await call_next(request)\n        duration_ms = (time.monotonic() - start) * 1000\n        rid = getattr(request.state, \"request_id\", \"-\")\n        logger.info(\"%s %s → %d (%.1fms) rid=%s\", request.method, request.url.path, response.status_code, duration_ms, rid)\n        return response\n")
	addFile(tree, prefix+"app/utils/pagination.py", "from dataclasses import dataclass\nfrom fastapi import Query\n\n@dataclass\nclass PageParams:\n    limit: int\n    offset: int\n\ndef page_params(limit: int = Query(default=20, ge=1, le=100), offset: int = Query(default=0, ge=0)) -> PageParams:\n    return PageParams(limit=limit, offset=offset)\n")
}

func (g *PythonGenerator) addPythonDBRetry(tree *FileTree, req *GenerateRequest, root string) {
	if req.Database == "none" {
		return
	}
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	if req.Framework != "django" {
		addFile(tree, prefix+"app/db/retry.py", "import time\nimport logging\n\nlogger = logging.getLogger(\"stacksprint\")\n\ndef connect_with_retry(connect_fn, max_retries: int = 10):\n    wait = 1.0\n    for attempt in range(1, max_retries + 1):\n        try:\n            return connect_fn()\n        except Exception as exc:\n            logger.warning(\"DB not ready (attempt %d/%d): %s — retrying in %.1fs\", attempt, max_retries, exc, wait)\n            time.sleep(wait)\n            wait = min(wait * 2, 16)\n    raise RuntimeError(f\"Database unavailable after {max_retries} retries\")\n")
	}
}

func (g *PythonGenerator) addPythonDBBoilerplate(tree *FileTree, req *GenerateRequest, root string) {
	if !isSQLDB(req.Database) {
		return
	}
	prefix := root
	if prefix != "" {
		prefix += "/"
	}

	if req.Framework == "django" {
		addFile(tree, prefix+"api/models.py", "from django.db import models\n\nclass Item(models.Model):\n    name = models.CharField(max_length=255)\n")
		return
	}

	if req.UseORM {
		driver := "postgresql+psycopg"
		if req.Database == "mysql" {
			driver = "mysql+pymysql"
		}
		addFile(tree, prefix+"app/repository/sqlalchemy_session.py", fmt.Sprintf("import os\nfrom sqlalchemy import create_engine\nfrom sqlalchemy.orm import sessionmaker\n\nDATABASE_URL = os.getenv('DATABASE_URL', '%s://app:app@localhost/app')\nengine = create_engine(DATABASE_URL, pool_pre_ping=True)\nSessionLocal = sessionmaker(bind=engine, autoflush=False, autocommit=False)\n", driver))
		addFile(tree, prefix+"app/repository/models.py", renderSQLAlchemyModels(req.Custom.Models))
		addFile(tree, prefix+"alembic.ini", renderAlembicIni())
		addFile(tree, prefix+"migrations/env.py", renderAlembicEnv())
		addFile(tree, prefix+"migrations/script.py.mako", renderAlembicMako())
		addFile(tree, prefix+"scripts/seed.py", renderPythonSeedScript(req.Custom.Models, true))
		return
	}

	if req.Database == "postgresql" {
		addFile(tree, prefix+"app/repository/sql_driver.py", "import os\nimport psycopg\n\nDATABASE_URL = os.getenv('DATABASE_URL', 'postgresql://app:app@postgres:5432/app')\n\ndef connect():\n    return psycopg.connect(DATABASE_URL)\n")
		addFile(tree, prefix+"scripts/seed.py", renderPythonSeedScript(req.Custom.Models, false))
		return
	}
	addFile(tree, prefix+"app/repository/sql_driver.py", "import os\nimport pymysql\n\ndef connect():\n    return pymysql.connect(host='mysql', user='app', password='app', database='app')\n")
	addFile(tree, prefix+"scripts/seed.py", renderPythonSeedScript(req.Custom.Models, false))
}

func pythonMonolithTemplateSpecs(req GenerateRequest) []templateSpec {
	withCRUD := isEnabled(req.FileToggles.ExampleCRUD)
	switch req.Architecture {
	case "clean":
		base := []templateSpec{
			{Template: "python/clean/main.tmpl", Output: "app/main.py"},
		}
		if withCRUD {
			return base
		}
		return append(base, []templateSpec{
			{Template: "python/clean/app/domain/ping.tmpl", Output: "app/domain/ping.py"},
			{Template: "python/clean/app/usecases/ping_usecase.tmpl", Output: "app/usecases/ping_usecase.py"},
			{Template: "python/clean/app/delivery/http/ping_controller.tmpl", Output: "app/delivery/http/ping_controller.py"},
			{Template: "python/clean/app/repository/ping_repository.tmpl", Output: "app/repository/ping_repository.py"},
		}...)
	case "hexagonal":
		base := []templateSpec{
			{Template: "python/hexagonal/main.tmpl", Output: "app/main.py"},
		}
		if withCRUD {
			return base
		}
		return append(base, []templateSpec{
			{Template: "python/hexagonal/app/core/ports/ping_port.tmpl", Output: "app/core/ports/ping_port.py"},
			{Template: "python/hexagonal/app/core/services/ping_service.tmpl", Output: "app/core/services/ping_service.py"},
			{Template: "python/hexagonal/app/adapters/primary/http/ping_controller.tmpl", Output: "app/adapters/primary/http/ping_controller.py"},
			{Template: "python/hexagonal/app/adapters/secondary/database/ping_adapter.tmpl", Output: "app/adapters/secondary/database/ping_adapter.py"},
		}...)
	default:
		return []templateSpec{{Template: fmt.Sprintf("python/%s/main.tmpl", archTemplateName(req.Architecture)), Output: "app/main.py"}}
	}
}

func pythonMicroserviceTemplateSpecs(_ GenerateRequest) []templateSpec {
	return []templateSpec{{Template: "python/microservice/main.tmpl", Output: "app/main.py"}}
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

func renderAlembicIni() string {
	return "[alembic]\nscript_location = migrations\nprepend_sys_path = .\nversion_path_separator = os\n\n[post_write_hooks]\n[loggers]\nkeys = root,sqlalchemy,alembic\n\n[handlers]\nkeys = console\n\n[formatters]\nkeys = generic\n\n[logger_root]\nlevel = WARN\nhandlers = console\nqualname =\n\n[logger_sqlalchemy]\nlevel = WARN\nhandlers =\nqualname = sqlalchemy.engine\n\n[logger_alembic]\nlevel = INFO\nhandlers =\nqualname = alembic\n\n[handler_console]\nclass = StreamHandler\nargs = (sys.stderr,)\nlevel = NOTSET\nformatter = generic\n\n[formatter_generic]\nformat = %(levelname)-5.5s [%(name)s] %(message)s\ndatefmt = %H:%M:%S\n"
}

func renderAlembicEnv() string {
	return "from logging.config import fileConfig\nfrom sqlalchemy import engine_from_config\nfrom sqlalchemy import pool\nfrom alembic import context\nimport os\nimport sys\n\nsys.path.append(os.path.dirname(os.path.dirname(__file__)))\nfrom app.repository.models import Base\nconfig = context.config\n\nif config.config_file_name is not None:\n    fileConfig(config.config_file_name)\n\ntarget_metadata = Base.metadata\n\ndef get_url():\n    return os.getenv(\"DATABASE_URL\", \"sqlite:///./app.db\")\n\ndef run_migrations_offline() -> None:\n    context.configure(url=get_url(), target_metadata=target_metadata, literal_binds=True, dialect_opts={\"paramstyle\": \"named\"})\n    with context.begin_transaction():\n        context.run_migrations()\n\ndef run_migrations_online() -> None:\n    configuration = config.get_section(config.config_ini_section)\n    configuration[\"sqlalchemy.url\"] = get_url()\n    connectable = engine_from_config(configuration, prefix=\"sqlalchemy.\", poolclass=pool.NullPool)\n    with connectable.connect() as connection:\n        context.configure(connection=connection, target_metadata=target_metadata)\n        with context.begin_transaction():\n            context.run_migrations()\n\nif context.is_offline_mode():\n    run_migrations_offline()\nelse:\n    run_migrations_online()\n"
}

func renderAlembicMako() string {
	return "\"\"\"${message}\n\nRevision ID: ${up_revision}\nRevises: ${down_revision | comma,n}\nCreate Date: ${create_date}\n\n\"\"\"\nfrom typing import Sequence, Union\nfrom alembic import op\nimport sqlalchemy as sa\n${imports if imports else \"\"}\n\nrevision: str = ${repr(up_revision)}\ndown_revision: Union[str, None] = ${repr(down_revision)}\nbranch_labels: Union[str, Sequence[str], None] = ${repr(branch_labels)}\ndepends_on: Union[str, Sequence[str], None] = ${repr(depends_on)}\n\ndef upgrade() -> None:\n    ${upgrades if upgrades else \"pass\"}\n\ndef downgrade() -> None:\n    ${downgrades if downgrades else \"pass\"}\n"
}

func renderPythonSeedScript(models []DataModel, useORM bool) string {
	var b strings.Builder
	if useORM {
		b.WriteString("from app.repository.sqlalchemy_session import SessionLocal\nfrom app.repository.models import ")
		var mNames []string
		for _, m := range resolvedModels(models) {
			mNames = append(mNames, m.Name)
		}
		b.WriteString(strings.Join(mNames, ", "))
		b.WriteString("\n\ndef seed():\n    print('Seeding database...')\n    with SessionLocal() as session:\n")
		for _, m := range resolvedModels(models) {
			sample := buildPythonSampleDict(m)
			b.WriteString(fmt.Sprintf("        obj_%s = %s(**%s)\n        session.add(obj_%s)\n", toSnake(m.Name), m.Name, sample, toSnake(m.Name)))
		}
		b.WriteString("        session.commit()\n    print('Done.')\n\nif __name__ == '__main__':\n    seed()\n")
		return b.String()
	}

	b.WriteString("from app.repository.sql_driver import connect\n\ndef seed():\n    print('Seeding database with raw SQL...')\n    conn = connect()\n")
	for _, m := range resolvedModels(models) {
		b.WriteString(fmt.Sprintf("    # TODO: Implement native SQL insert for %ss\n", toSnake(m.Name)))
	}
	b.WriteString("    conn.close()\n    print('Done.')\n\nif __name__ == '__main__':\n    seed()\n")
	return b.String()
}

func (g *PythonGenerator) renderPythonDynamicModel(tree *FileTree, req *GenerateRequest, model DataModel, arch, root string) {
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	name := model.Name
	nameLow := strings.ToLower(name)
	snakeName := toSnake(name)

	pydanticFields := buildPydanticFields(model)
	sampleDict := buildPythonSampleDict(model)

	switch arch {
	case "clean":
		addFile(tree, prefix+"app/domain/"+snakeName+".py", "from pydantic import BaseModel\n\n\nclass "+name+"(BaseModel):\n"+pydanticFields)
		addFile(tree, prefix+"app/usecases/list_"+snakeName+"s.py", "from app.repository."+snakeName+"_repository import "+name+"Repository\n\n\ndef list_"+snakeName+"s():\n    return "+name+"Repository().find_all()\n")
		addFile(tree, prefix+"app/delivery/http/"+snakeName+"_controller.py", "from fastapi import APIRouter\nfrom app.usecases.list_"+snakeName+"s import list_"+snakeName+"s\nfrom app.domain."+snakeName+" import "+name+"\n\nrouter = APIRouter(prefix='/"+nameLow+"s', tags=['"+name+"'])\n\n@router.get('')\ndef get_"+snakeName+"s():\n    return list_"+snakeName+"s()\n\n@router.post('', status_code=201)\ndef create_"+snakeName+"(data: "+name+"):\n    return data\n")
		addFile(tree, prefix+"app/repository/"+snakeName+"_repository.py", "from app.domain."+snakeName+" import "+name+"\n\nclass "+name+"Repository:\n    def find_all(self) -> list:\n        return ["+name+"(**"+sampleDict+")]\n    def find_by_id(self, id: int):\n        return "+name+"(id=id, **"+sampleDict+")\n    def create(self, data: "+name+"):\n        return data\n")
	case "hexagonal":
		addFile(tree, prefix+"app/domain/"+snakeName+".py", "from pydantic import BaseModel\n\n\nclass "+name+"(BaseModel):\n"+pydanticFields)
		addFile(tree, prefix+"app/core/ports/"+snakeName+"_repository_port.py", "from abc import ABC, abstractmethod\nfrom app.domain."+snakeName+" import "+name+"\n\n\nclass "+name+"RepositoryPort(ABC):\n    @abstractmethod\n    def find_all(self) -> list: ...\n    @abstractmethod\n    def find_by_id(self, id: int): ...\n    @abstractmethod\n    def create(self, data: "+name+"): ...\n")
		addFile(tree, prefix+"app/core/services/"+snakeName+"_service.py", "from app.core.ports."+snakeName+"_repository_port import "+name+"RepositoryPort\nfrom app.domain."+snakeName+" import "+name+"\n\n\nclass "+name+"Service:\n    def __init__(self, repo: "+name+"RepositoryPort):\n        self.repo = repo\n\n    def list_all(self): return self.repo.find_all()\n    def get_by_id(self, id: int): return self.repo.find_by_id(id)\n    def create(self, data: "+name+"): return self.repo.create(data)\n")
		addFile(tree, prefix+"app/adapters/primary/http/"+snakeName+"_controller.py", "from fastapi import APIRouter\nfrom app.core.services."+snakeName+"_service import "+name+"Service\nfrom app.adapters.secondary.database."+snakeName+"_repository_adapter import "+name+"RepositoryAdapter\nfrom app.domain."+snakeName+" import "+name+"\n\n"+snakeName+"_router = APIRouter(prefix='/"+nameLow+"s', tags=['"+name+"'])\n_svc = "+name+"Service("+name+"RepositoryAdapter())\n\n@"+snakeName+"_router.get('')\ndef list_"+snakeName+"s(): return _svc.list_all()\n\n@"+snakeName+"_router.get('/{id}')\ndef get_"+snakeName+"(id: int): return _svc.get_by_id(id)\n\n@"+snakeName+"_router.post('', status_code=201)\ndef create_"+snakeName+"(data: "+name+"): return _svc.create(data)\n")
		addFile(tree, prefix+"app/adapters/secondary/database/"+snakeName+"_repository_adapter.py", "from app.core.ports."+snakeName+"_repository_port import "+name+"RepositoryPort\nfrom app.domain."+snakeName+" import "+name+"\n\n\nclass "+name+"RepositoryAdapter("+name+"RepositoryPort):\n    def find_all(self): return ["+name+"(**"+sampleDict+")]\n    def find_by_id(self, id: int): return "+name+"(id=id, **"+sampleDict+")\n    def create(self, data: "+name+"): return data\n")
	default:
		addFile(tree, prefix+"app/schemas/"+snakeName+".py", "from pydantic import BaseModel\n\n\nclass "+name+"(BaseModel):\n"+pydanticFields)
		addFile(tree, prefix+"app/routes/"+snakeName+"s.py", "from fastapi import APIRouter, Query\nfrom app.schemas."+snakeName+" import "+name+"\n\nrouter = APIRouter(prefix='/"+nameLow+"s', tags=['"+name+"'])\n\n@router.get('')\ndef list_"+snakeName+"s(limit: int = Query(default=20, le=100), offset: int = Query(default=0)):\n    return {\"limit\": limit, \"offset\": offset, \"data\": ["+sampleDict+"]}\n\n@router.get('/{id}')\ndef get_"+snakeName+"(id: int):\n    return {\"id\": id}\n\n@router.post('', status_code=201)\ndef create_"+snakeName+"(data: "+name+"):\n    return {\"id\": 1, **data.dict()}\n\n@router.put('/{id}')\ndef update_"+snakeName+"(id: int, data: "+name+"):\n    return {\"id\": id, **data.dict()}\n\n@router.delete('/{id}')\ndef delete_"+snakeName+"(id: int):\n    return {\"deleted\": id}\n")
	}
}

func buildPydanticFields(model DataModel) string {
	var b strings.Builder
	for _, f := range model.Fields {
		fn := strings.ToLower(f.Name)
		pyType := "str"
		switch strings.ToLower(f.Type) {
		case "int", "integer":
			pyType = "int"
		case "float", "float64", "double":
			pyType = "float"
		case "bool", "boolean":
			pyType = "bool"
		case "datetime", "timestamp", "time":
			pyType = "str"
		}
		b.WriteString("    " + fn + ": " + pyType + "\n")
	}
	return b.String()
}

func buildPythonSampleDict(model DataModel) string {
	var b strings.Builder
	b.WriteString("{")
	for i, f := range model.Fields {
		if i > 0 {
			b.WriteString(", ")
		}
		fn := strings.ToLower(f.Name)
		switch strings.ToLower(f.Type) {
		case "int", "integer":
			b.WriteString("\"" + fn + "\": 1")
		case "float", "float64", "double":
			b.WriteString("\"" + fn + "\": 1.0")
		case "bool", "boolean":
			b.WriteString("\"" + fn + "\": True")
		default:
			b.WriteString("\"" + fn + "\": \"sample\"")
		}
	}
	b.WriteString("}")
	return b.String()
}

func toSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r + 32)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// -------------------------------------------------------------------------
// Django scaffold helpers — called only by PythonGenerator
// -------------------------------------------------------------------------

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
