# StackSprint Project State

This document exists for human and AI developers to orient themselves on the most recently completed features and current development trajectory of StackSprint.

## Recent Updates

**Generator Route Injection Fixes**
- **Issue**: Previously, when the system generated custom backend code utilizing the `<Language>Generator` structs (like `PythonGenerator` or `GoGenerator`), adding "Custom Dynamic Models" via the UI would create the necessary folders (`handlers/`, `controllers/`, `repository/`, `domain/`), but fail to associate those newly generated routes with the main entrypoint `src/index.js`, `main.go`, or `main.py`. These routes were dead code.
- **Resolution**:
  - The Python template (`mvp/main.tmpl`) was stripped of inline hardcoded routes to prevent collision.
  - The Python generator now correctly formats the import string injected during Hexagonal mode string replacements (`%s_router` vs `item_router`).
  - The Node.js and Go generators both received massive logic overhauls for AST injection into their specific Framework (Gin/Fiber/Express/Fastify) for Monolithic and Microservices configurations.
  - `strings.Replace` was extensively used in `internal/generator/go_generator.go`, effectively fixing the previously fatal unassociated Go API handler bug across MVP, Hexagonal, Clean, and Modular-Monolith structures.

**Philosophy & Intelligent UI Upgrades**
- StackSprint V2 integrated multiple developer-friendly interface aspects aimed at lowering cognitive load, ensuring state determinism, and minimizing confusion during project modeling (The "Anti-Cockpit" redesign).
- **Architecture Recommender:** The frontend now provides dynamic, contextual hints to guide users toward the best architecture for their selected language (e.g., suggesting Clean Architecture for Go).
- **Actionable Complexity Feedback:** The backend `ComplexityReport` now provides specific, actionable point-reduction strategies (e.g., "Remove Kafka to drop 15 points") instead of passive warnings.
- **Config Diff Indicator:** The UI now highlights file count changes (`+2 files`, `-1 file`) immediately when generating, ensuring the user understands the exact impact of their configuration toggles.
- **Live Validation:** Inputs like Go Module Names are now validated live on the client to prevent backend build failures.

## Known Areas of Future Work

1. **Robust Type Mappings**: While `types.go` holds the main domain mapping models, the engine relies heavily on string mapping switches in `node_generator.go` or `python_generator.go` for specific database and class mapping conversions. It might benefit from abstracted Type registry factories in a later sprint.
2. **Expanding Preset Library**: The `QuickStart` component currently holds 4 hardcoded templates. Future work could allow loading community presets from an external Github repository list.
