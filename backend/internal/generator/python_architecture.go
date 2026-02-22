package generator

import (
	"fmt"
	"path"
	"strings"
)

func (e *Engine) generatePythonMonolith(tree *FileTree, req GenerateRequest) error {
	specs := pythonMonolithTemplateSpecs(req)
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
		main, err := e.registry.Render("python/"+archTemplateName(req.Architecture)+"/main.tmpl", data)
		if err != nil {
			return err
		}
		addDjangoFiles(tree, req, main)
	} else {
		if err := e.renderSpecs(tree, specs, data, ""); err != nil {
			return err
		}
		// Model-driven CRUD for FastAPI architectures
		if isEnabled(req.FileToggles.ExampleCRUD) {
			var imports, routes strings.Builder
			for _, model := range resolvedModels(req.Custom.Models) {
				renderPythonDynamicModel(tree, req, model, req.Architecture, "")
				nameLow := toSnake(model.Name)
				if req.Architecture == "clean" {
					imports.WriteString(fmt.Sprintf("from app.delivery.http.%s_controller import router as %s_router\n", nameLow, nameLow))
					routes.WriteString(fmt.Sprintf("app.include_router(%s_router)\n", nameLow))
				} else if req.Architecture == "hexagonal" {
					imports.WriteString(fmt.Sprintf("from app.adapters.primary.http.%s_controller import item_router as %s_router\n", nameLow, nameLow))
					routes.WriteString(fmt.Sprintf("app.include_router(%s_router)\n", nameLow))
				} else {
					imports.WriteString(fmt.Sprintf("from app.routes.%ss import router as %ss_router\n", nameLow, nameLow))
					routes.WriteString(fmt.Sprintf("app.include_router(%ss_router)\n", nameLow))
				}
			}

			if main, ok := tree.Files["app/main.py"]; ok {
				// Remove the statically injected item_router from base template if it exists
				main = strings.Replace(main, "from app.delivery.http.item_controller import router as items_router\napp.include_router(items_router)\n", "", 1)
				main = strings.Replace(main, "from app.adapters.primary.http.item_controller import item_router\napp.include_router(item_router)\n", "", 1)

				// Inject our dynamically generated imports and routes before standard boilerplate
				injectionTarget := "if __name__ == "
				if idx := strings.Index(main, injectionTarget); idx != -1 {
					injected := imports.String() + "\n" + routes.String() + "\n"
					main = main[:idx] + injected + main[idx:]
				} else {
					// Fallback to appending
					main += "\n" + imports.String() + "\n" + routes.String()
				}
				tree.Files["app/main.py"] = main
			}
		}
	}

	if req.Framework != "django" {
		addFile(tree, "requirements.txt", pythonRequirements(req.Framework, req.Database, req.UseORM))
	}
	addPythonDBBoilerplate(tree, req, "")
	return nil
}

func (e *Engine) generatePythonService(tree *FileTree, req GenerateRequest, svcRoot string, svc ServiceConfig) error {
	specs := pythonMicroserviceTemplateSpecs(req)
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
		main, err := e.registry.Render("python/microservice/main.tmpl", data)
		if err != nil {
			return err
		}
		addDjangoFilesAtRoot(tree, req, main, svcRoot)
	} else {
		if err := e.renderSpecs(tree, specs, data, svcRoot); err != nil {
			return err
		}
		addFile(tree, path.Join(svcRoot, "requirements.txt"), pythonRequirements(req.Framework, req.Database, req.UseORM))
	}
	addPythonDBBoilerplate(tree, req, svcRoot)
	return nil
}

func pythonMonolithTemplateSpecs(req GenerateRequest) []templateSpec {
	withCRUD := isEnabled(req.FileToggles.ExampleCRUD)
	switch req.Architecture {
	case "clean":
		base := []templateSpec{
			{Template: "python/clean/main.tmpl", Output: "app/main.py"},
		}
		if withCRUD {
			return base // dynamic models will be rendered separately
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
			return base // dynamic models will be rendered separately
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

func addPythonDBBoilerplate(tree *FileTree, req GenerateRequest, root string) {
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

func renderAlembicIni() string {
	return `[alembic]
script_location = migrations
prepend_sys_path = .
version_path_separator = os

[post_write_hooks]
[loggers]
keys = root,sqlalchemy,alembic

[handlers]
keys = console

[formatters]
keys = generic

[logger_root]
level = WARN
handlers = console
qualname =

[logger_sqlalchemy]
level = WARN
handlers =
qualname = sqlalchemy.engine

[logger_alembic]
level = INFO
handlers =
qualname = alembic

[handler_console]
class = StreamHandler
args = (sys.stderr,)
level = NOTSET
formatter = generic

[formatter_generic]
format = %(levelname)-5.5s [%(name)s] %(message)s
datefmt = %H:%M:%S
`
}

func renderAlembicEnv() string {
	return `from logging.config import fileConfig
from sqlalchemy import engine_from_config
from sqlalchemy import pool
from alembic import context
import os
import sys

# Add app directory to path
sys.path.append(os.path.dirname(os.path.dirname(__file__)))

from app.repository.models import Base
config = context.config

if config.config_file_name is not None:
    fileConfig(config.config_file_name)

target_metadata = Base.metadata

def get_url():
    return os.getenv("DATABASE_URL", "sqlite:///./app.db")

def run_migrations_offline() -> None:
    url = get_url()
    context.configure(url=url, target_metadata=target_metadata, literal_binds=True, dialect_opts={"paramstyle": "named"})
    with context.begin_transaction():
        context.run_migrations()

def run_migrations_online() -> None:
    configuration = config.get_section(config.config_ini_section)
    configuration["sqlalchemy.url"] = get_url()
    connectable = engine_from_config(configuration, prefix="sqlalchemy.", poolclass=pool.NullPool)
    with connectable.connect() as connection:
        context.configure(connection=connection, target_metadata=target_metadata)
        with context.begin_transaction():
            context.run_migrations()

if context.is_offline_mode():
    run_migrations_offline()
else:
    run_migrations_online()
`
}

func renderAlembicMako() string {
	return `"""${message}

Revision ID: ${up_revision}
Revises: ${down_revision | comma,n}
Create Date: ${create_date}

"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa
${imports if imports else ""}

# revision identifiers, used by Alembic.
revision: str = ${repr(up_revision)}
down_revision: Union[str, None] = ${repr(down_revision)}
branch_labels: Union[str, Sequence[str], None] = ${repr(branch_labels)}
depends_on: Union[str, Sequence[str], None] = ${repr(depends_on)}

def upgrade() -> None:
    ${upgrades if upgrades else "pass"}

def downgrade() -> None:
    ${downgrades if downgrades else "pass"}
`
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

// renderPythonDynamicModel generates per-model Python files for Clean, Hexagonal, and MVP architectures.
func renderPythonDynamicModel(tree *FileTree, req GenerateRequest, model DataModel, arch, root string) {
	_ = req
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	name := model.Name
	nameLow := strings.ToLower(name)
	snakeName := toSnake(name)

	// Build Pydantic field definitions
	pydanticFields := buildPydanticFields(model)
	sampleDict := buildPythonSampleDict(model)

	switch arch {
	case "clean":
		// Domain entity (Pydantic model)
		addFile(tree, prefix+"app/domain/"+snakeName+".py",
			"from pydantic import BaseModel\n\n\nclass "+name+"(BaseModel):\n"+pydanticFields)
		// Use case
		addFile(tree, prefix+"app/usecases/list_"+snakeName+"s.py",
			"from app.repository."+snakeName+"_repository import "+name+"Repository\n\n\ndef list_"+snakeName+"s():\n"+
				"    return "+name+"Repository().find_all()\n")
		// Delivery / HTTP controller
		addFile(tree, prefix+"app/delivery/http/"+snakeName+"_controller.py",
			"from fastapi import APIRouter\nfrom app.usecases.list_"+snakeName+"s import list_"+snakeName+"s\n"+
				"from app.domain."+snakeName+" import "+name+"\n\n"+
				"router = APIRouter(prefix='/"+nameLow+"s', tags=['"+name+"'])\n\n"+
				"@router.get('')\ndef get_"+snakeName+"s():\n    return list_"+snakeName+"s()\n\n"+
				"@router.post('', status_code=201)\ndef create_"+snakeName+"(data: "+name+"):\n    return data\n")
		// Repository
		addFile(tree, prefix+"app/repository/"+snakeName+"_repository.py",
			"from app.domain."+snakeName+" import "+name+"\n\n"+
				"class "+name+"Repository:\n"+
				"    def find_all(self) -> list:\n        return ["+name+"(**"+sampleDict+")]\n"+
				"    def find_by_id(self, id: int):\n        return "+name+"(id=id, **"+sampleDict+")\n"+
				"    def create(self, data: "+name+"):\n        return data\n")

	case "hexagonal":
		addFile(tree, prefix+"app/domain/"+snakeName+".py",
			"from pydantic import BaseModel\n\n\nclass "+name+"(BaseModel):\n"+pydanticFields)
		// Port (abstract base)
		addFile(tree, prefix+"app/core/ports/"+snakeName+"_repository_port.py",
			"from abc import ABC, abstractmethod\nfrom app.domain."+snakeName+" import "+name+"\n\n\nclass "+name+"RepositoryPort(ABC):\n"+
				"    @abstractmethod\n    def find_all(self) -> list: ...\n"+
				"    @abstractmethod\n    def find_by_id(self, id: int): ...\n"+
				"    @abstractmethod\n    def create(self, data: "+name+"): ...\n")
		// Service
		addFile(tree, prefix+"app/core/services/"+snakeName+"_service.py",
			"from app.core.ports."+snakeName+"_repository_port import "+name+"RepositoryPort\nfrom app.domain."+snakeName+" import "+name+"\n\n\nclass "+name+"Service:\n"+
				"    def __init__(self, repo: "+name+"RepositoryPort):\n        self.repo = repo\n\n"+
				"    def list_all(self): return self.repo.find_all()\n"+
				"    def get_by_id(self, id: int): return self.repo.find_by_id(id)\n"+
				"    def create(self, data: "+name+"): return self.repo.create(data)\n")
		// Primary adapter (HTTP)
		addFile(tree, prefix+"app/adapters/primary/http/"+snakeName+"_controller.py",
			"from fastapi import APIRouter\nfrom app.core.services."+snakeName+"_service import "+name+"Service\n"+
				"from app.adapters.secondary.database."+snakeName+"_repository_adapter import "+name+"RepositoryAdapter\n"+
				"from app.domain."+snakeName+" import "+name+"\n\n"+
				snakeName+"_router = APIRouter(prefix='/"+nameLow+"s', tags=['"+name+"'])\n"+
				"_svc = "+name+"Service("+name+"RepositoryAdapter())\n\n"+
				"@"+snakeName+"_router.get('')\ndef list_"+snakeName+"s(): return _svc.list_all()\n\n"+
				"@"+snakeName+"_router.get('/{id}')\ndef get_"+snakeName+"(id: int): return _svc.get_by_id(id)\n\n"+
				"@"+snakeName+"_router.post('', status_code=201)\ndef create_"+snakeName+"(data: "+name+"): return _svc.create(data)\n")
		// Secondary adapter (DB stub)
		addFile(tree, prefix+"app/adapters/secondary/database/"+snakeName+"_repository_adapter.py",
			"from app.core.ports."+snakeName+"_repository_port import "+name+"RepositoryPort\nfrom app.domain."+snakeName+" import "+name+"\n\n\nclass "+name+"RepositoryAdapter("+name+"RepositoryPort):\n"+
				"    def find_all(self): return ["+name+"(**"+sampleDict+")]\n"+
				"    def find_by_id(self, id: int): return "+name+"(id=id, **"+sampleDict+")\n"+
				"    def create(self, data: "+name+"): return data\n")

	default: // mvp, modular-monolith, microservices — flat FastAPI router
		addFile(tree, prefix+"app/schemas/"+snakeName+".py",
			"from pydantic import BaseModel\n\n\nclass "+name+"(BaseModel):\n"+pydanticFields)
		addFile(tree, prefix+"app/routes/"+snakeName+"s.py",
			"from fastapi import APIRouter, Query\nfrom app.schemas."+snakeName+" import "+name+"\n\n"+
				"router = APIRouter(prefix='/"+nameLow+"s', tags=['"+name+"'])\n\n"+
				"@router.get('')\ndef list_"+snakeName+"s(limit: int = Query(default=20, le=100), offset: int = Query(default=0)):\n"+
				"    return {\"limit\": limit, \"offset\": offset, \"data\": ["+sampleDict+"]}\n\n"+
				"@router.get('/{id}')\ndef get_"+snakeName+"(id: int):\n    return {\"id\": id}\n\n"+
				"@router.post('', status_code=201)\ndef create_"+snakeName+"(data: "+name+"):\n    return {\"id\": 1, **data.dict()}\n\n"+
				"@router.put('/{id}')\ndef update_"+snakeName+"(id: int, data: "+name+"):\n    return {\"id\": id, **data.dict()}\n\n"+
				"@router.delete('/{id}')\ndef delete_"+snakeName+"(id: int):\n    return {\"deleted\": id}\n")
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

// toSnake converts PascalCase to snake_case (e.g. UserProfile → user_profile)
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
