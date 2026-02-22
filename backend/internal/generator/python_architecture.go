package generator

import (
	"fmt"
	"path"
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
			return append(base, []templateSpec{
				{Template: "python/clean/app/domain/item.tmpl", Output: "app/domain/item.py"},
				{Template: "python/clean/app/usecases/list_items.tmpl", Output: "app/usecases/list_items.py"},
				{Template: "python/clean/app/delivery/http/item_controller.tmpl", Output: "app/delivery/http/item_controller.py"},
				{Template: "python/clean/app/repository/item_repository.tmpl", Output: "app/repository/item_repository.py"},
			}...)
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
			return append(base, []templateSpec{
				{Template: "python/hexagonal/app/core/ports/item_repository_port.tmpl", Output: "app/core/ports/item_repository_port.py"},
				{Template: "python/hexagonal/app/core/services/item_service.tmpl", Output: "app/core/services/item_service.py"},
				{Template: "python/hexagonal/app/adapters/primary/http/item_controller.tmpl", Output: "app/adapters/primary/http/item_controller.py"},
				{Template: "python/hexagonal/app/adapters/secondary/database/item_repository_adapter.tmpl", Output: "app/adapters/secondary/database/item_repository_adapter.py"},
			}...)
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
		return
	}

	if req.Database == "postgresql" {
		addFile(tree, prefix+"app/repository/sql_driver.py", "import os\nimport psycopg\n\nDATABASE_URL = os.getenv('DATABASE_URL', 'postgresql://app:app@postgres:5432/app')\n\ndef connect():\n    return psycopg.connect(DATABASE_URL)\n")
		return
	}
	addFile(tree, prefix+"app/repository/sql_driver.py", "import os\nimport pymysql\n\ndef connect():\n    return pymysql.connect(host='mysql', user='app', password='app', database='app')\n")
}
