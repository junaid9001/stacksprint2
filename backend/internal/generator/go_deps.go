package generator

import (
	"fmt"
	"path"
	"strings"
)

func resolveGoModule(root RootOptions, fallback string) string {
	module := strings.TrimSpace(root.Module)
	if module != "" {
		return module
	}
	if strings.TrimSpace(root.Name) != "" {
		return path.Base(root.Name)
	}
	return fallback
}

func goModV2(framework string, root RootOptions, db string, useORM bool, useGRPC bool) string {
	module := resolveGoModule(root, "stacksprint/generated")

	deps := []string{}
	if framework == "fiber" {
		deps = append(deps, "github.com/gofiber/fiber/v2 v2.52.6")
	} else {
		deps = append(deps, "github.com/gin-gonic/gin v1.10.0")
	}
	if db == "postgresql" {
		if useORM {
			deps = append(deps,
				"gorm.io/gorm v1.25.12",
				"gorm.io/driver/postgres v1.5.11",
			)
		} else {
			deps = append(deps, "github.com/jackc/pgx/v5 v5.7.1")
		}
	}
	if db == "mysql" {
		if useORM {
			deps = append(deps,
				"gorm.io/gorm v1.25.12",
				"gorm.io/driver/mysql v1.5.7",
			)
		} else {
			deps = append(deps, "github.com/go-sql-driver/mysql v1.8.1")
		}
	}
	if useGRPC {
		deps = append(deps,
			"google.golang.org/grpc v1.69.2",
			"google.golang.org/protobuf v1.36.1",
		)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("module %s\n\ngo 1.23\n\nrequire (\n", module))
	for _, dep := range deps {
		b.WriteString("\t" + dep + "\n")
	}
	b.WriteString(")\n")
	return b.String()
}
