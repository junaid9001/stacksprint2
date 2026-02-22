# Agent Operations & Guidelines 

Welcome, fellow agent. This directory exists so that you can understand the nuances of the StackSprint codebase and avoid breaking core functionalities. Before executing user requests, review these rules.

## 1. General Principles
- **Core Philosophy**: Always optimize for *developer speed, clarity, determinism, and zero confusion*. Output must be predictable. UX changes should reduce cognitive load (no "cockpit" interfaces).
- **Read before writing**: If asked to modify a generator component, check `docs/architecture.md` and `docs/project_state.md` for context.
- **Do not guess imports**: Do not assume Node.js or Go module dependencies are available unless you have explicitly verified them in `package.json` or `go.mod`.
- **Prefer robust tool usage over bash scripting**: 
  - ALWAYS use `multi_replace_file_content` or `replace_file_content` for editing code.
  - DO NOT use bash `sed` or `echo` / `cat >` redirections to alter files unless there's no native tool alternative.

## 2. Modifying Generation Logic
The backend engine compiles files using an Abstract Syntax Tree (AST) string-replacement methodology overlaid on standard template layouts.
- **Do not break string injection offsets**: Go files like `go_generator.go` use `strings.Replace` targeting exact fragments of template code (e.g., `srv := &http.Server`). Modifying the `main.tmpl` templates will break these `strings.Replace` indexes if the specific string anchor changes.
- **Use `%s` precision**: When building `import` string builders inside generator logic, be aware that models are usually converted using `strings.ToLower()` or `toSnake()`. Verify exactly how the language generator builds model casing.

## 3. Frontend Component Patterns
1. StackSprint frontend models rely entirely on Next.js *Client Components* marked with `"use client";`.
2. All global state is wrapped by `ConfigContext.tsx`. Do not pass configuration payload details via prop-drilling; consume the context directly utilizing the `useConfig` hook.
3. Complex validation functions are separated into contextual hooks or directly placed inside `useMemo` blocks within specific forms.

## 4. Testing
When making any logic change to `backend/internal/generator/`:
1. Execute `go test ./...` in the `backend/` directory before declaring your task complete.
2. The Go test suite asserts the structural integrity of the `FileTree` build loops.

Failure to follow these rules often results in generating un-compileable target repositories for the user. Protect the logic engine.
