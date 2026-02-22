package generator

import (
	"bytes"
	"strings"
	"text/template"
)

type modelTemplateData struct {
	Models []DataModel
}

func resolvedModels(in []DataModel) []DataModel {
	clean := make([]DataModel, 0, len(in))
	for _, m := range in {
		name := strings.TrimSpace(m.Name)
		if name == "" {
			continue
		}
		fields := make([]DataField, 0, len(m.Fields))
		for _, f := range m.Fields {
			fn := strings.TrimSpace(f.Name)
			if fn == "" {
				continue
			}
			ft := strings.TrimSpace(f.Type)
			if ft == "" {
				ft = "string"
			}
			fields = append(fields, DataField{Name: fn, Type: ft})
		}
		if len(fields) == 0 {
			fields = []DataField{{Name: "name", Type: "string"}}
		}
		clean = append(clean, DataModel{Name: toPascal(name), Fields: fields})
	}
	if len(clean) == 0 {
		return []DataModel{{
			Name:   "Item",
			Fields: []DataField{{Name: "id", Type: "int"}, {Name: "name", Type: "string"}},
		}}
	}
	return clean
}

func renderGoORMModels(models []DataModel) string {
	const tpl = `package models

{{ range .Models -}}
type {{ .Name }} struct {
{{- range .Fields }}
	{{ goFieldName .Name }} {{ goType .Type }} ` + "`json:\"{{ .Name }}\" gorm:\"column:{{ .Name }}\"`" + `
{{- end }}
}

{{ end -}}
`
	return renderModelTemplate(tpl, models, template.FuncMap{
		"goType":      goType,
		"goFieldName": func(v string) string { return toPascal(v) },
	})
}

func renderPrismaSchema(db string, models []DataModel) string {
	provider := "postgresql"
	if db == "mysql" {
		provider = "mysql"
	}
	const tpl = `generator client {
  provider = "prisma-client-js"
}

datasource db {
  provider = "{{ .Provider }}"
  url      = env("DATABASE_URL")
}

{{ range .Models -}}
model {{ .Name }} {
{{- range .Fields }}
  {{ prismaFieldName .Name }} {{ prismaType .Type }}
{{- end }}
}

{{ end -}}
`
	data := struct {
		Provider string
		Models   []DataModel
	}{Provider: provider, Models: resolvedModels(models)}
	t, err := template.New("prisma").Funcs(template.FuncMap{
		"prismaType":      prismaType,
		"prismaFieldName": prismaFieldName,
	}).Parse(tpl)
	if err != nil {
		return ""
	}
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return ""
	}
	return b.String()
}

func renderSQLAlchemyModels(models []DataModel) string {
	const tpl = `from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column
from sqlalchemy import Integer, String, Boolean, Float, DateTime

class Base(DeclarativeBase):
    pass

{{ range .Models -}}
class {{ .Name }}(Base):
    __tablename__ = "{{ tableName .Name }}"
{{- range .Fields }}
    {{ .Name }}: Mapped[{{ pyHint .Type }}] = mapped_column({{ sqlalchemyType .Type }})
{{- end }}

{{ end -}}
`
	return renderModelTemplate(tpl, models, template.FuncMap{
		"sqlalchemyType": sqlalchemyType,
		"tableName":      func(v string) string { return strings.ToLower(v) + "s" },
		"pyHint":         pythonHint,
	})
}

func renderModelTemplate(tpl string, models []DataModel, funcs template.FuncMap) string {
	t, err := template.New("models").Funcs(funcs).Parse(tpl)
	if err != nil {
		return ""
	}
	var b bytes.Buffer
	if err := t.Execute(&b, modelTemplateData{Models: resolvedModels(models)}); err != nil {
		return ""
	}
	return b.String()
}

func goType(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "int", "integer":
		return "int"
	case "float", "float64", "double":
		return "float64"
	case "bool", "boolean":
		return "bool"
	case "datetime", "timestamp", "time":
		return "string"
	default:
		return "string"
	}
}

func prismaType(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "int", "integer":
		return "Int"
	case "float", "float64", "double":
		return "Float"
	case "bool", "boolean":
		return "Boolean"
	case "datetime", "timestamp", "time":
		return "DateTime"
	default:
		return "String"
	}
}

func prismaFieldName(v string) string {
	name := strings.TrimSpace(v)
	if strings.EqualFold(name, "id") {
		return "id Int @id @default(autoincrement())"
	}
	return name
}

func sqlalchemyType(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "int", "integer":
		return "Integer"
	case "float", "float64", "double":
		return "Float"
	case "bool", "boolean":
		return "Boolean"
	case "datetime", "timestamp", "time":
		return "DateTime"
	default:
		return "String"
	}
}

func pythonHint(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "int", "integer":
		return "int"
	case "float", "float64", "double":
		return "float"
	case "bool", "boolean":
		return "bool"
	default:
		return "str"
	}
}

func toPascal(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "Field"
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}
	return strings.Join(parts, "")
}
