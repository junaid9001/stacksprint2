package generator

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

const bashHeredocDelimiter = "EOF_STACKSPRINT_GEN_9942"

func BuildScripts(req GenerateRequest, tree FileTree) (GenerateResponse, error) {
	ensureGitKeepFiles(&tree)
	return GenerateResponse{
		BashScript: buildBash(req, tree),
		FilePaths:  collectFilePaths(tree),
		Warnings:   buildGenerationWarnings(req),
	}, nil
}

func buildBash(req GenerateRequest, tree FileTree) string {
	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("set -euo pipefail\n\n")
	rootExpr := rootExpressionBash(req)
	b.WriteString(fmt.Sprintf("ROOT_DIR=%s\n", rootExpr))
	b.WriteString("mkdir -p \"$ROOT_DIR\"\n")
	b.WriteString("cd \"$ROOT_DIR\"\n\n")

	if strings.ToLower(req.Root.Mode) == "new" {
		if req.Root.GitInit {
			b.WriteString("git init\n")
		}
		b.WriteString(languageInitBash(req))
	}

	dirs := dirsSorted(tree.Dirs)
	for _, d := range dirs {
		if d == "." || d == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("mkdir -p %q\n", d))
	}
	if len(dirs) > 0 {
		b.WriteString("\n")
	}

	files := fileNamesSorted(tree.Files)
	for _, f := range files {
		content := tree.Files[f]
		b.WriteString(fmt.Sprintf("cat > %q <<'%s'\n", f, bashHeredocDelimiter))
		b.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			b.WriteString("\n")
		}
		b.WriteString(bashHeredocDelimiter + "\n\n")
	}

	b.WriteString("echo \"StackSprint project generated successfully.\"\n")
	b.WriteString("echo \"Run: docker compose up --build\"\n")
	return b.String()
}

func ensureGitKeepFiles(tree *FileTree) {
	for _, d := range dirsSorted(tree.Dirs) {
		if d == "." || d == "" {
			continue
		}
		hasChildFile := false
		prefix := d + "/"
		for file := range tree.Files {
			if strings.HasPrefix(file, prefix) {
				hasChildFile = true
				break
			}
		}
		if !hasChildFile {
			keep := path.Join(d, ".gitkeep")
			tree.Files[keep] = ""
		}
	}
}

func rootExpressionBash(req GenerateRequest) string {
	if strings.ToLower(req.Root.Mode) == "existing" {
		return fmt.Sprintf("%q", req.Root.Path)
	}
	return fmt.Sprintf("%q", req.Root.Name)
}

func languageInitBash(req GenerateRequest) string {
	gen := GetGenerator(req.Language)
	return gen.GetInitCommand(&req)
}

func dirsSorted(in map[string]struct{}) []string {
	out := make([]string, 0, len(in))
	for d := range in {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

func fileNamesSorted(in map[string]string) []string {
	out := make([]string, 0, len(in))
	for f := range in {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

func collectFilePaths(tree FileTree) []string {
	paths := make([]string, 0, len(tree.Dirs)+len(tree.Files))
	for d := range tree.Dirs {
		if d == "." || d == "" {
			continue
		}
		paths = append(paths, d)
	}
	for f := range tree.Files {
		paths = append(paths, f)
	}
	sort.Strings(paths)
	return paths
}

func buildGenerationWarnings(req GenerateRequest) []Warning {
	warnings := make([]Warning, 0)

	if req.Architecture == "microservices" && req.ServiceCommunication == "none" {
		warnings = append(warnings, Warning{
			Code:     "MICROSERVICES_NO_COMMUNICATION",
			Severity: "warn",
			Message:  "Microservices selected without service-to-service communication (http/grpc).",
			Reason:   "Architecture implies distributed orchestration, but no internal endpoints were bound.",
		})
	}
	if req.UseORM && (req.Database == "none" || req.Database == "mongodb") {
		warnings = append(warnings, Warning{
			Code:     "ORM_NON_SQL_DATABASE",
			Severity: "info",
			Message:  "ORM toggle is enabled but current database is non-SQL; ORM setting is ignored.",
			Reason:   req.Database,
		})
	}
	if req.Infra.Kafka && req.ServiceCommunication == "none" && req.Architecture != "microservices" {
		warnings = append(warnings, Warning{
			Code:     "KAFKA_MONOLITH_NO_COMMUNICATION",
			Severity: "info",
			Message:  "Kafka enabled for a monolith without explicit service communication; verify topic usage in app flow.",
			Reason:   "Event brokers usually pair with distributed domains.",
		})
	}

	// Delegate language/framework-specific warnings to the generator — keeps
	// req.Language and req.Framework checks OUT of this shared pipeline file.
	gen := GetGenerator(req.Language)
	warnings = append(warnings, gen.GetConfigWarnings(&req)...)

	return warnings
}
