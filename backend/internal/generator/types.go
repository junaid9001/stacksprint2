package generator

type GenerateRequest struct {
	Language             string            `json:"language"`
	Framework            string            `json:"framework"`
	Architecture         string            `json:"architecture"`
	Services             []ServiceConfig   `json:"services"`
	Database             string            `json:"db"`
	UseORM               bool              `json:"use_orm"`
	Infra                InfraOptions      `json:"infra"`
	Features             FeatureOptions    `json:"features"`
	FileToggles          FileToggleOptions `json:"file_toggles"`
	Custom               CustomOptions     `json:"custom"`
	Root                 RootOptions       `json:"root"`
	ServiceCommunication string            `json:"service_communication"`
}

type ServiceConfig struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}

type InfraOptions struct {
	Redis bool `json:"redis"`
	Kafka bool `json:"kafka"`
	NATS  bool `json:"nats"`
}

type FeatureOptions struct {
	JWTAuth       bool `json:"jwt_auth"`
	Swagger       bool `json:"swagger"`
	GitHubActions bool `json:"github_actions_ci"`
	Makefile      bool `json:"makefile"`
	Logger        bool `json:"logger"`
	GlobalError   bool `json:"global_error_handler"`
	Health        bool `json:"health_endpoint"`
	SampleTest    bool `json:"sample_test"`
}

type FileToggleOptions struct {
	Env         *bool `json:"env"`
	Gitignore   *bool `json:"gitignore"`
	Dockerfile  *bool `json:"dockerfile"`
	Compose     *bool `json:"docker_compose"`
	Readme      *bool `json:"readme"`
	Config      *bool `json:"config_loader"`
	Logger      *bool `json:"logger"`
	BaseRoute   *bool `json:"base_route"`
	ExampleCRUD *bool `json:"example_crud"`
	HealthCheck *bool `json:"health_check"`
}

type CustomOptions struct {
	AddFolders      []string     `json:"add_folders"`
	AddFiles        []CustomFile `json:"add_files"`
	AddServiceNames []string     `json:"add_service_names"`
	RemoveFolders   []string     `json:"remove_folders"`
	RemoveFiles     []string     `json:"remove_files"`
}

type CustomFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type RootOptions struct {
	Mode    string `json:"mode"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	GitInit bool   `json:"git_init"`
	Module  string `json:"module"`
}

type GenerateResponse struct {
	BashScript       string   `json:"bash_script"`
	PowerShellScript string   `json:"powershell_script"`
	FilePaths        []string `json:"file_paths"`
	Warnings         []string `json:"warnings"`
}

type FileTree struct {
	Files map[string]string
	Dirs  map[string]struct{}
}
