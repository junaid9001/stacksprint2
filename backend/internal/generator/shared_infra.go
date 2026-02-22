package generator

// shared_infra.go — Language-agnostic infrastructure generation utilities.
//
// PURITY CONTRACT: This file must NEVER reference:
//   - req.Language or any language enum value ("go", "node", "python", etc.)
//   - req.Framework or any framework name
//   - Any language-specific helper (Django, Gin, Express, FastAPI, etc.)
//
// If you need language-specific logic here, it belongs in the language generator
// implementation (go_generator.go, node_generator.go, python_generator.go).

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// =========================================================================
// ComposeSpec — structured Docker Compose specification model.
//
// All compose output is produced by populating this model and marshaling via
// yaml.v3. No raw YAML string concatenation is allowed in this file.
// =========================================================================

// ComposeSpec is the top-level docker-compose.yaml specification.
type ComposeSpec struct {
	Services map[string]ComposeService `yaml:"services,omitempty"`
	Networks map[string]ComposeNetwork `yaml:"networks,omitempty"`
	Volumes  map[string]ComposeVolume  `yaml:"volumes,omitempty"`
}

// ComposeService describes a single docker-compose service.
type ComposeService struct {
	Image         string                `yaml:"image,omitempty"`
	Build         *ComposeBuild         `yaml:"build,omitempty"`
	ContainerName string                `yaml:"container_name,omitempty"`
	Restart       string                `yaml:"restart,omitempty"`
	Ports         []string              `yaml:"ports,omitempty"`
	Environment   *ComposeEnvironment   `yaml:"environment,omitempty"`
	EnvFile       []string              `yaml:"env_file,omitempty"`
	DependsOn     map[string]ComposeDep `yaml:"depends_on,omitempty"`
	Volumes       []string              `yaml:"volumes,omitempty"`
	Command       string                `yaml:"command,omitempty"`
	Healthcheck   *ComposeHealthcheck   `yaml:"healthcheck,omitempty"`
	Networks      []string              `yaml:"networks,omitempty"`
}

// ComposeBuild is the build context for a service.
type ComposeBuild struct {
	Context    string `yaml:"context"`
	Dockerfile string `yaml:"dockerfile,omitempty"`
}

// ComposeDep represents a depends_on entry with an optional condition.
type ComposeDep struct {
	Condition string `yaml:"condition,omitempty"`
}

// ComposeHealthcheck defines a service health check.
type ComposeHealthcheck struct {
	Test     []string `yaml:"test,omitempty"`
	Interval string   `yaml:"interval,omitempty"`
	Timeout  string   `yaml:"timeout,omitempty"`
	Retries  int      `yaml:"retries,omitempty"`
}

// ComposeNetwork is a named network definition.
type ComposeNetwork struct {
	Driver string `yaml:"driver,omitempty"`
}

// ComposeVolume is a named volume definition.
type ComposeVolume struct {
	Driver string `yaml:"driver,omitempty"`
}

// ComposeEnvironment wraps a map so that yaml.v3 renders it as a key: value
// block in insertion-sorted order (sorted by key for determinism).
type ComposeEnvironment struct {
	pairs []envPair
}

type envPair struct{ k, v string }

// Set adds or replaces an environment variable.
func (e *ComposeEnvironment) Set(key, value string) {
	for i, p := range e.pairs {
		if p.k == key {
			e.pairs[i].v = value
			return
		}
	}
	e.pairs = append(e.pairs, envPair{key, value})
}

// MarshalYAML implements yaml.Marshaler to emit sorted key: value pairs.
func (e ComposeEnvironment) MarshalYAML() (interface{}, error) {
	sorted := make([]envPair, len(e.pairs))
	copy(sorted, e.pairs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].k < sorted[j].k })

	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, p := range sorted {
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: p.k},
			&yaml.Node{Kind: yaml.ScalarNode, Value: p.v},
		)
	}
	return node, nil
}

// =========================================================================
// ComposeSpec builder
// =========================================================================

// buildCompose constructs a docker-compose.yaml from the request configuration.
// It populates a ComposeSpec and marshals it with gopkg.in/yaml.v3.
func buildCompose(req GenerateRequest) string {
	spec := newComposeSpec(req)
	return renderComposeSpec(spec)
}

// newComposeSpec populates a ComposeSpec from the generate request.
func newComposeSpec(req GenerateRequest) ComposeSpec {
	spec := ComposeSpec{
		Services: map[string]ComposeService{},
	}

	// ── Application service(s) ────────────────────────────────────────────
	if req.Architecture == "microservices" {
		for _, svc := range req.Services {
			s := ComposeService{
				Build:   &ComposeBuild{Context: fmt.Sprintf("./services/%s", svc.Name)},
				Ports:   []string{fmt.Sprintf("%d:%d", svc.Port, svc.Port)},
				EnvFile: []string{fmt.Sprintf("./services/%s/.env", svc.Name)},
			}
			if req.Database != "none" {
				s.DependsOn = map[string]ComposeDep{
					composeDBServiceName(req.Database): {Condition: "service_healthy"},
				}
			}
			spec.Services[svc.Name] = s
		}
	} else {
		s := ComposeService{
			Build: &ComposeBuild{Context: "."},
			Ports: []string{"8080:8080"},
		}
		if isEnabled(req.FileToggles.Env) {
			s.EnvFile = []string{"./.env"}
		}
		if req.Database != "none" {
			s.DependsOn = map[string]ComposeDep{
				composeDBServiceName(req.Database): {Condition: "service_healthy"},
			}
		}
		spec.Services["app"] = s
	}

	// ── Database service ──────────────────────────────────────────────────
	addDBService(&spec, req.Database)

	// ── Infrastructure services ───────────────────────────────────────────
	if req.Infra.Redis {
		spec.Services["redis"] = ComposeService{
			Image: "redis:7-alpine",
			Ports: []string{"6379:6379"},
			Healthcheck: &ComposeHealthcheck{
				Test:     []string{"CMD", "redis-cli", "ping"},
				Interval: "5s",
				Timeout:  "3s",
				Retries:  10,
			},
		}
	}

	if req.Infra.Kafka {
		env := &ComposeEnvironment{}
		env.Set("ALLOW_PLAINTEXT_LISTENER", "yes")
		env.Set("KAFKA_CFG_ADVERTISED_LISTENERS", "PLAINTEXT://kafka:9092")
		env.Set("KAFKA_CFG_CONTROLLER_LISTENER_NAMES", "CONTROLLER")
		env.Set("KAFKA_CFG_CONTROLLER_QUORUM_VOTERS", "1@kafka:9093")
		env.Set("KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP", "CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT")
		env.Set("KAFKA_CFG_LISTENERS", "PLAINTEXT://:9092,CONTROLLER://:9093")
		env.Set("KAFKA_CFG_NODE_ID", "1")
		env.Set("KAFKA_CFG_PROCESS_ROLES", "broker,controller")
		spec.Services["kafka"] = ComposeService{
			Image:       "bitnami/kafka:3.9",
			Ports:       []string{"9092:9092"},
			Environment: env,
			Healthcheck: &ComposeHealthcheck{
				Test:     []string{"CMD-SHELL", "kafka-broker-api-versions.sh --bootstrap-server localhost:9092"},
				Interval: "10s",
				Timeout:  "5s",
				Retries:  10,
			},
		}
	}

	if req.Infra.NATS {
		spec.Services["nats"] = ComposeService{
			Image: "nats:2.10-alpine",
			Ports: []string{"4222:4222"},
		}
	}

	return spec
}

// addDBService adds the database service to the ComposeSpec.
func addDBService(spec *ComposeSpec, db string) {
	switch db {
	case "postgresql":
		env := &ComposeEnvironment{}
		env.Set("POSTGRES_DB", "app")
		env.Set("POSTGRES_PASSWORD", "app")
		env.Set("POSTGRES_USER", "app")
		spec.Services["postgres"] = ComposeService{
			Image:       "postgres:16-alpine",
			Environment: env,
			Volumes:     []string{"./db/init:/docker-entrypoint-initdb.d"},
			Ports:       []string{"5432:5432"},
			Healthcheck: &ComposeHealthcheck{
				Test:     []string{"CMD-SHELL", "pg_isready -U app -d app"},
				Interval: "5s",
				Timeout:  "5s",
				Retries:  12,
			},
		}
	case "mysql":
		env := &ComposeEnvironment{}
		env.Set("MYSQL_DATABASE", "app")
		env.Set("MYSQL_PASSWORD", "app")
		env.Set("MYSQL_ROOT_PASSWORD", "root")
		env.Set("MYSQL_USER", "app")
		spec.Services["mysql"] = ComposeService{
			Image:       "mysql:8.4",
			Environment: env,
			Volumes:     []string{"./db/init:/docker-entrypoint-initdb.d"},
			Ports:       []string{"3306:3306"},
			Healthcheck: &ComposeHealthcheck{
				Test:     []string{"CMD-SHELL", "mysqladmin ping -h localhost -uapp -papp"},
				Interval: "5s",
				Timeout:  "5s",
				Retries:  12,
			},
		}
	case "mongodb":
		spec.Services["mongo"] = ComposeService{
			Image: "mongo:8",
			Ports: []string{"27017:27017"},
			Healthcheck: &ComposeHealthcheck{
				Test:     []string{"CMD-SHELL", "mongosh --quiet --eval 'db.adminCommand({ ping: 1 })'"},
				Interval: "5s",
				Timeout:  "5s",
				Retries:  12,
			},
		}
	}
}

// renderComposeSpec marshals a ComposeSpec into a deterministic YAML string.
// Service, network, and volume keys are sorted before emission.
func renderComposeSpec(spec ComposeSpec) string {
	// Produce a sorted yaml.Node manually so key order is deterministic
	// regardless of map iteration order.
	root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	if len(spec.Services) > 0 {
		keys := sortedKeys(spec.Services)
		servicesNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for _, k := range keys {
			svc := spec.Services[k]
			svcNode, err := toYAMLNode(svc)
			if err != nil {
				continue
			}
			servicesNode.Content = append(servicesNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: k},
				svcNode,
			)
		}
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "services"},
			servicesNode,
		)
	}

	if len(spec.Networks) > 0 {
		keys := sortedKeysNetwork(spec.Networks)
		networksNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for _, k := range keys {
			netNode, err := toYAMLNode(spec.Networks[k])
			if err != nil {
				continue
			}
			networksNode.Content = append(networksNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: k},
				netNode,
			)
		}
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "networks"},
			networksNode,
		)
	}

	if len(spec.Volumes) > 0 {
		keys := sortedKeysVolume(spec.Volumes)
		volumesNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for _, k := range keys {
			volNode, err := toYAMLNode(spec.Volumes[k])
			if err != nil {
				continue
			}
			volumesNode.Content = append(volumesNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: k},
				volNode,
			)
		}
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "volumes"},
			volumesNode,
		)
	}

	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return "# compose generation error: " + err.Error() + "\n"
	}
	_ = enc.Close()
	return buf.String()
}

// toYAMLNode marshals any value into a *yaml.Node via round-trip marshal/unmarshal.
func toYAMLNode(v interface{}) (*yaml.Node, error) {
	b, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	var node yaml.Node
	if err := yaml.Unmarshal(b, &node); err != nil {
		return nil, err
	}
	// Unmarshal wraps in a DocumentNode; extract the content node.
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0], nil
	}
	return &node, nil
}

func sortedKeys(m map[string]ComposeService) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysNetwork(m map[string]ComposeNetwork) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysVolume(m map[string]ComposeVolume) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// =========================================================================
// composeDBServiceName — shared DB service name lookup
// =========================================================================

func composeDBServiceName(db string) string {
	switch db {
	case "postgresql":
		return "postgres"
	case "mysql":
		return "mysql"
	case "mongodb":
		return "mongo"
	default:
		return "db"
	}
}

// =========================================================================
// Environment file builder
// =========================================================================

// buildEnv constructs a .env file from request configuration.
// service is the service name prefix (empty for monolith), port is the app port.
func buildEnv(req GenerateRequest, service string, port int) string {
	prefix := ""
	if service != "" {
		prefix = strings.ToUpper(strings.ReplaceAll(service, "-", "_")) + "_"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("PORT=%d\n", port))
	if req.Database != "none" {
		switch req.Database {
		case "postgresql":
			b.WriteString("DATABASE_URL=postgres://app:app@postgres:5432/app?sslmode=disable\n")
		case "mysql":
			b.WriteString("DATABASE_URL=mysql://app:app@mysql:3306/app\n")
		case "mongodb":
			b.WriteString("DATABASE_URL=mongodb://mongo:27017/app\n")
		}
	}
	if req.Features.JWTAuth {
		b.WriteString("JWT_SECRET=replace-me\n")
	}
	if req.Infra.Redis {
		b.WriteString(prefix + "REDIS_ADDR=redis:6379\n")
	}
	if req.Infra.Kafka {
		b.WriteString(prefix + "KAFKA_BROKERS=kafka:9092\n")
	}
	if req.Infra.NATS {
		b.WriteString(prefix + "NATS_URL=nats://nats:4222\n")
	}
	return b.String()
}

// =========================================================================
// SQL migration and seed helpers
// =========================================================================

// sampleMigration returns an SQL migration script seeded from the data models.
func sampleMigration(db string, models []DataModel) string {
	if db == "mongodb" {
		return "// MongoDB migrations are usually handled by migration tools at runtime.\n"
	}
	return renderSQLTablesTemplate(models, false)
}

// sampleDBInit returns a Docker entrypoint SQL init script with optional seed rows.
func sampleDBInit(db string, models []DataModel) string {
	if db == "mongodb" {
		return "db = db.getSiblingDB('app');\ndb.createCollection('items');\n"
	}
	return renderSQLTablesTemplate(models, true)
}

func renderSQLTablesTemplate(models []DataModel, withSeed bool) string {
	const tpl = `{{ range .Models -}}
CREATE TABLE IF NOT EXISTS {{ .TableName }} (
  id SERIAL PRIMARY KEY{{ range .Columns }},
  {{ .Name }} {{ .SQLType }}{{ end }}
);
{{ if $.WithSeed }}INSERT INTO {{ .TableName }} DEFAULT VALUES;
{{ end }}

{{ end -}}`

	type sqlColumn struct {
		Name    string
		SQLType string
	}
	type sqlTable struct {
		TableName string
		Columns   []sqlColumn
	}
	type sqlPayload struct {
		WithSeed bool
		Models   []sqlTable
	}

	resolved := resolvedModels(models)
	tables := make([]sqlTable, 0, len(resolved))
	for _, model := range resolved {
		table := sqlTable{
			TableName: strings.ToLower(model.Name) + "s",
			Columns:   make([]sqlColumn, 0, len(model.Fields)),
		}
		for _, field := range model.Fields {
			if strings.EqualFold(field.Name, "id") {
				continue
			}
			table.Columns = append(table.Columns, sqlColumn{
				Name:    strings.ToLower(field.Name),
				SQLType: sqlTypeFromField(field.Type),
			})
		}
		tables = append(tables, table)
	}

	t, err := template.New("sql-migrations").Parse(tpl)
	if err != nil {
		return ""
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, sqlPayload{WithSeed: withSeed, Models: tables}); err != nil {
		return ""
	}
	return buf.String()
}

func sqlTypeFromField(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "int", "integer":
		return "INT"
	case "float", "float64", "double", "decimal":
		return "DECIMAL(10,2)"
	case "bool", "boolean":
		return "BOOLEAN"
	case "datetime", "timestamp", "time":
		return "TIMESTAMP"
	default:
		return "VARCHAR(255)"
	}
}

// =========================================================================
// Marker-Based AST Injection
// =========================================================================

// InjectByMarker locates a specific '// stacksprint:<marker>' token in the content
// and injects the payload immediately below it. It preserves the indentation of the marker.
// If the marker is missing and the payload is not empty, it returns the original content
// and an error that should be converted into a deterministic Warning by the caller.
func InjectByMarker(content, marker, payload string) (string, error) {
	if payload == "" {
		return content, nil // Nothing to inject, safe.
	}

	target := "// stacksprint:" + marker
	idx := strings.Index(content, target)
	if idx == -1 {
		// Try Python/shell comment style
		target = "# stacksprint:" + marker
		idx = strings.Index(content, target)
		if idx == -1 {
			return content, fmt.Errorf("INJECTION_MARKER_MISSING: %s", target)
		}
	}

	// Find the end of the line containing the marker
	endOfLineIdx := strings.IndexByte(content[idx:], '\n')
	if endOfLineIdx == -1 {
		endOfLineIdx = len(content)
	} else {
		endOfLineIdx += idx + 1 // Include the newline character
	}

	// Extract indentation of the marker to apply to the payload if needed
	// (Though usually the payload is pre-formatted, we just insert it directly after the newline)

	before := content[:endOfLineIdx]
	after := content[endOfLineIdx:]

	// Ensure payload ends with exactly one newline if it doesn't already, so it doesn't mangle 'after'
	cleanPayload := strings.TrimRight(payload, "\r\n") + "\n"

	return before + cleanPayload + after, nil
}
