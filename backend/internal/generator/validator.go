package generator

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	allowedLanguages     = map[string]struct{}{"go": {}, "node": {}, "python": {}}
	allowedArchitectures = map[string]struct{}{
		"mvp": {}, "clean": {}, "hexagonal": {}, "modular-monolith": {}, "microservices": {},
	}
	allowedDBs          = map[string]struct{}{"postgresql": {}, "mysql": {}, "mongodb": {}, "none": {}}
	frameworkByLanguage = map[string]map[string]struct{}{
		"go":     {"gin": {}, "fiber": {}},
		"node":   {"express": {}, "fastify": {}},
		"python": {"fastapi": {}, "django": {}},
	}
	serviceNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
)

func Validate(req GenerateRequest) error {
	lang := strings.ToLower(strings.TrimSpace(req.Language))
	if _, ok := allowedLanguages[lang]; !ok {
		return errors.New("language must be one of: go, node, python")
	}

	fw := strings.ToLower(strings.TrimSpace(req.Framework))
	if fw == "" {
		return fmt.Errorf("framework is required for %s. please specify a valid framework (e.g. express, fastify). defaults are explicitly not supported.", req.Language)
	}
	if _, ok := frameworkByLanguage[lang][fw]; !ok {
		return fmt.Errorf("framework %q is not valid for %s", req.Framework, req.Language)
	}

	arch := strings.ToLower(strings.TrimSpace(req.Architecture))
	if _, ok := allowedArchitectures[arch]; !ok {
		return errors.New("architecture must be one of: mvp, clean, hexagonal, modular-monolith, microservices")
	}

	db := strings.ToLower(strings.TrimSpace(req.Database))
	if _, ok := allowedDBs[db]; !ok {
		return errors.New("db must be one of: postgresql, mysql, mongodb, none")
	}

	if arch == "microservices" {
		if len(req.Services) < 2 || len(req.Services) > 5 {
			return errors.New("microservices mode requires 2 to 5 services")
		}
		seen := map[string]struct{}{}
		for i, svc := range req.Services {
			name := strings.TrimSpace(svc.Name)
			if !serviceNameRegex.MatchString(name) {
				return fmt.Errorf("services[%d].name is invalid", i)
			}
			if _, ok := seen[strings.ToLower(name)]; ok {
				return fmt.Errorf("duplicate service name %q", name)
			}
			seen[strings.ToLower(name)] = struct{}{}
			if svc.Port <= 0 {
				return fmt.Errorf("services[%d].port must be a positive number", i)
			}
		}
	}

	rootMode := strings.ToLower(strings.TrimSpace(req.Root.Mode))
	if rootMode != "new" && rootMode != "existing" {
		return errors.New("root.mode must be either 'new' or 'existing'")
	}
	if rootMode == "new" && strings.TrimSpace(req.Root.Name) == "" {
		return errors.New("root.name is required when root.mode is 'new'")
	}
	if rootMode == "existing" && strings.TrimSpace(req.Root.Path) == "" {
		return errors.New("root.path is required when root.mode is 'existing'")
	}

	for _, p := range req.Custom.AddFolders {
		if err := validateRelPath(p); err != nil {
			return fmt.Errorf("invalid custom folder %q: %w", p, err)
		}
	}
	for _, f := range req.Custom.AddFiles {
		if err := validateRelPath(f.Path); err != nil {
			return fmt.Errorf("invalid custom file path %q: %w", f.Path, err)
		}
	}
	for _, p := range req.Custom.RemoveFolders {
		if err := validateRelPath(p); err != nil {
			return fmt.Errorf("invalid remove folder %q: %w", p, err)
		}
	}
	for _, p := range req.Custom.RemoveFiles {
		if err := validateRelPath(p); err != nil {
			return fmt.Errorf("invalid remove file %q: %w", p, err)
		}
	}

	return nil
}

func validateRelPath(p string) error {
	p = filepath.ToSlash(strings.TrimSpace(p))
	if p == "" || strings.HasPrefix(p, "/") || strings.Contains(p, "..") {
		return errors.New("must be a relative path without '..'")
	}
	return nil
}

func archTemplateName(arch string) string {
	a := strings.ToLower(arch)
	if a == "modular-monolith" {
		return "modular"
	}
	if a == "microservices" {
		return "microservice"
	}
	return a
}

func isEnabled(flag *bool) bool {
	if flag == nil {
		return true
	}
	return *flag
}
