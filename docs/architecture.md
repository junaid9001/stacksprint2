# StackSprint Architecture & Internals

This document is designed to quickly orient AI agents and human developers on the overarching architecture and flow of StackSprint, a backend project initialization engine.

## Overview
StackSprint is a full-stack application (Next.js frontend + Go Fiber backend). Its primary purpose is to generate starter code for backend projects across multiple languages, frameworks, and architectural patterns based on user configuration.

The output from StackSprint is an executable script (bash) and an interactive live preview of the generated FileTree.

---

## 1. The Frontend (`/frontend`)
The frontend is built with **Next.js** (App Router) using React and Vanilla CSS. It provides the UI for users to configure their project.

### Core Concepts
- **State Management**: The entire configuration state is managed within `src/context/ConfigContext.tsx`. Everything the user selects in the UI updates this central state object.
- **Generation Trigger**: When the user clicks "Preview & Generate", the frontend serializes the `ConfigContext` into a JSON payload and POSTs it to the Go backend (`/generate`).
- **Live Preview**: The frontend consumes the `file_paths` array from the generation response and can fetch individual file contents on-demand via the `/file?path=...` endpoint to power the code preview split-pane.

---

## 2. The Backend (`/backend`)
The backend is a stateless **Go** application using the **Fiber** web framework (`cmd/server/main.go`). It does not use a database to store its own state (it is purely a code compiler/generator).

### HTTP Handlers (`internal/api/handlers.go`)
- `Generate(c *fiber.Ctx)`: The core endpoint. Parses the JSON request into a `generator.GenerateRequest` struct.
- `GetGeneratedFile(c *fiber.Ctx)`: Returns the string contents of a previously generated file from the temporary workspace, used by the frontend preview.

### The Engine (`internal/generator`)
This is the heart of StackSprint. It takes the `GenerateRequest` and runs it through a language-specific generator to construct a virtual `FileTree`.

- `types.go`: Defines the core domain models (`GenerateRequest`, `FileTree`, `Warning`, `DataModel`).
- `scaffolds.go` / `integration_scaffolds.go`: Provide the high-level orchestration for compiling the file tree depending on the language selected.
- **Language Generators**: `go_generator.go`, `node_generator.go`, `python_generator.go`. These structs implement the generation interface. They format the code, resolve module names, and invoke the text templates to populate the `FileTree`.

---

## 3. The Templates (`/templates`)
The `templates/` directory contains standard Go `text/template` files organized by language and architecture.

### Template Structure
Templates are categorized strictly by language, then architecture:
- `templates/go/clean/...`
- `templates/node/hexagonal/...`
- `templates/python/microservice/...`

### How Templates Are Evaluated
The language generators (e.g. `node_generator.go`) load these `.tmpl` files and evaluate them, passing in a map of data usually containing strings like `Framework`, `Architecture`, `Port`, `UseDB`, and structured objects like `Model` (representing the user's custom database schema fields).

### The "Dynamic Injection" Problem
Because StackSprint allows users to specify an arbitrary number of custom data models (e.g. `User`, `Product`, `Order`), the generators must dynamically construct route attachments for each model into the main application entry point (like `cmd/server/main.go` or `src/index.js`).

**Important Warning for Agents:**
Do not simply hardcode a route in a `/templates` layer unless it is guaranteed to be static (like `/health`). Custom routes must be string-injected (using `strings.Replace`) natively within the generator `.go` file logic. This is how we associate multiple components cleanly without compilation errors.
