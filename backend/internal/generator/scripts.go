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
	bash := buildBash(req, tree)
	pwsh := buildPowerShell(req, tree)
	return GenerateResponse{BashScript: bash, PowerShellScript: pwsh}, nil
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

func buildPowerShell(req GenerateRequest, tree FileTree) string {
	var b strings.Builder
	b.WriteString("$ErrorActionPreference = 'Stop'\n\n")
	b.WriteString(fmt.Sprintf("$RootDir = %s\n", rootExpressionPS(req)))
	b.WriteString("New-Item -ItemType Directory -Path $RootDir -Force | Out-Null\n")
	b.WriteString("Set-Location $RootDir\n\n")

	if strings.ToLower(req.Root.Mode) == "new" {
		if req.Root.GitInit {
			b.WriteString("git init\n")
		}
		b.WriteString(languageInitPS(req))
	}

	dirs := dirsSorted(tree.Dirs)
	for _, d := range dirs {
		if d == "." || d == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("New-Item -ItemType Directory -Path '%s' -Force | Out-Null\n", strings.ReplaceAll(d, "'", "''")))
	}
	if len(dirs) > 0 {
		b.WriteString("\n")
	}

	files := fileNamesSorted(tree.Files)
	for _, f := range files {
		escaped := strings.ReplaceAll(f, "'", "''")
		b.WriteString(fmt.Sprintf("@'\n%s\n'@ | Set-Content -NoNewline '%s'\n\n", trimFinalNewline(tree.Files[f]), escaped))
	}

	b.WriteString("Write-Host 'StackSprint project generated successfully.'\n")
	b.WriteString("Write-Host 'Run: docker compose up --build'\n")
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

func rootExpressionPS(req GenerateRequest) string {
	if strings.ToLower(req.Root.Mode) == "existing" {
		return fmt.Sprintf("'%s'", strings.ReplaceAll(req.Root.Path, "'", "''"))
	}
	return fmt.Sprintf("'%s'", strings.ReplaceAll(req.Root.Name, "'", "''"))
}

func languageInitBash(req GenerateRequest) string {
	lang := strings.ToLower(req.Language)
	mod := req.Root.Module
	if mod == "" {
		mod = path.Base(req.Root.Name)
	}
	switch lang {
	case "go":
		return fmt.Sprintf("go mod init %q\n", mod)
	case "node":
		return "npm init -y\n"
	default:
		return ""
	}
}

func languageInitPS(req GenerateRequest) string {
	lang := strings.ToLower(req.Language)
	mod := req.Root.Module
	if mod == "" {
		mod = path.Base(req.Root.Name)
	}
	switch lang {
	case "go":
		return fmt.Sprintf("go mod init '%s'\n", strings.ReplaceAll(mod, "'", "''"))
	case "node":
		return "npm init -y\n"
	default:
		return ""
	}
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

func trimFinalNewline(v string) string {
	return strings.TrimSuffix(v, "\n")
}
