package generator

type GenerationContext struct {
	FileTree  *FileTree
	Warnings  []Warning
	Decisions []Decision
	Registry  *TemplateRegistry
}

func (ctx *GenerationContext) AddWarning(w Warning) {
	ctx.Warnings = append(ctx.Warnings, w)
}

func (ctx *GenerationContext) AddDecision(d Decision) {
	ctx.Decisions = append(ctx.Decisions, d)
}

type templateSpec struct {
	Template string
	Output   string
}

// Generator defines the contract for language-specific generation strategies.
type Generator interface {
	GenerateArchitecture(req *GenerateRequest, ctx *GenerationContext) error
	GenerateModels(req *GenerateRequest, ctx *GenerationContext) error
	GenerateInfra(req *GenerateRequest, ctx *GenerationContext) error
	GenerateDevTools(req *GenerateRequest, ctx *GenerationContext) error
	// GetInitCommand returns the bash init command for this language.
	GetInitCommand(req *GenerateRequest) string
	// GetConfigWarnings returns language/framework-specific configuration warnings.
	// This keeps req.Language and req.Framework checks OUT of shared pipeline code.
	GetConfigWarnings(req *GenerateRequest) []Warning
}

func isSQLDB(db string) bool {
	return db == "postgresql" || db == "mysql"
}

func GetGenerator(lang string) Generator {
	switch lang {
	case "go":
		return &GoGenerator{}
	case "node":
		return &NodeGenerator{}
	case "python":
		return &PythonGenerator{}
	default:
		// Fallback to Go as a default, though Validator should prevent this.
		return &GoGenerator{}
	}
}
