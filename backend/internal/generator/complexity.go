package generator

import (
	"fmt"
	"strings"
)

// AnalyzeComplexity produces a deterministic ComplexityReport for the given request.
// This is analysis-only — it does NOT modify the request or affect generation.
func AnalyzeComplexity(req GenerateRequest) ComplexityReport {
	var notes []string
	archWeight := architectureWeight(req.Architecture)
	infraWeight := infraWeightScore(req)
	serviceWeight := serviceWeightScore(req)
	modelWeight := modelWeightScore(req)

	score := archWeight + dbWeight(req.Database) + infraWeight + serviceWeight + modelWeight
	if score > 100 {
		score = 100
	}

	// ── Risk rules ──────────────────────────────────────────────────────────
	risk := "low"
	if req.Architecture == "microservices" && req.Infra.Kafka &&
		strings.EqualFold(req.ServiceCommunication, "grpc") {
		risk = "high"
	} else if score >= 55 {
		risk = "high"
	} else if score >= 30 {
		risk = "moderate"
	}

	// ── Advisory notes ──────────────────────────────────────────────────────
	if req.Architecture == "mvp" && req.Infra.Kafka {
		notes = append(notes, "Action: Uncheck Kafka to drop 15 complexity points (excessive for MVP).")
	}
	if risk == "high" && req.Infra.Kafka && strings.EqualFold(req.ServiceCommunication, "grpc") {
		notes = append(notes, "Action: Switch from gRPC to HTTP (-10 pts) or remove Kafka (-15 pts) to drop to Moderate risk.")
	}
	if req.Architecture == "microservices" && len(req.Services) > 3 {
		saving := (len(req.Services) - 3) * 5
		notes = append(notes, fmt.Sprintf("Action: Consolidate to 3 services to save %d complexity points.", saving))
	}
	if req.Framework == "django" && req.Architecture == "microservices" {
		notes = append(notes, "Action: Django is traditionally monolithic. Consider FastAPI for microservices.")
	}

	return ComplexityReport{
		Score:              score,
		ArchitectureWeight: archWeight,
		InfraWeight:        infraWeight,
		ServiceWeight:      serviceWeight,
		ModelWeight:        modelWeight,
		RiskLevel:          risk,
		Notes:              notes,
	}
}

func architectureWeight(arch string) int {
	switch arch {
	case "mvp":
		return 5
	case "modular-monolith":
		return 15
	case "clean":
		return 20
	case "hexagonal":
		return 25
	case "microservices":
		return 40
	default:
		return 5
	}
}

func dbWeight(db string) int {
	switch db {
	case "postgresql", "mysql":
		return 10
	case "mongodb":
		return 8
	default:
		return 0
	}
}

func infraWeightScore(req GenerateRequest) int {
	w := 0
	if req.Infra.Redis {
		w += 5
	}
	if req.Infra.Kafka {
		w += 15
	}
	if req.Infra.NATS {
		w += 10
	}
	if strings.EqualFold(req.ServiceCommunication, "grpc") {
		w += 10
	}
	return w
}

func serviceWeightScore(req GenerateRequest) int {
	if req.Architecture != "microservices" {
		return 0
	}
	extra := len(req.Services) - 2
	if extra < 0 {
		extra = 0
	}
	return extra * 5
}

func modelWeightScore(req GenerateRequest) int {
	models := resolvedModels(req.Custom.Models)
	extra := len(models) - 3
	if extra < 0 {
		extra = 0
	}
	return extra * 3
}
