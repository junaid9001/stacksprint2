package generator

import (
	"context"
	"strings"
	"testing"
)

func ptr(b bool) *bool { return &b }

func TestEngine_Generate(t *testing.T) {
	registry, err := NewTemplateRegistry("../../../templates")
	if err != nil {
		t.Fatalf("failed to init registry: %v", err)
	}
	engine := NewEngine(registry)

	tests := []struct {
		name          string
		req           GenerateRequest
		expectedFiles []string
	}{
		{
			name: "Go Clean with Dynamic Models and Autopilot",
			req: GenerateRequest{
				Language:     "go",
				Framework:    "fiber",
				Architecture: "clean",
				Database:     "postgresql",
				FileToggles:  FileToggleOptions{ExampleCRUD: ptr(true)},
				Custom: CustomOptions{
					Models: []DataModel{
						{Name: "User", Fields: []DataField{{Name: "Email", Type: "string"}}},
						{Name: "Product", Fields: []DataField{{Name: "Price", Type: "float64"}}},
					},
				},
			},
			expectedFiles: []string{
				"internal/domain/user.go",
				"internal/domain/product.go",
				"internal/usecase/user_usecase.go",
				"internal/usecase/product_usecase.go",
				"internal/middleware/requestid.go", // autopilot
				"internal/db/retry.go",             // db retry
			},
		},
		{
			name: "Node Hexagonal with Dynamic Models",
			req: GenerateRequest{
				Language:     "node",
				Framework:    "express",
				Architecture: "hexagonal",
				Database:     "mysql",
				FileToggles:  FileToggleOptions{ExampleCRUD: ptr(true)},
				Custom: CustomOptions{
					Models: []DataModel{
						{Name: "Order", Fields: []DataField{{Name: "Total", Type: "float"}}},
					},
				},
			},
			expectedFiles: []string{
				"src/core/ports/orderRepositoryPort.js",
				"src/core/services/orderService.js",
				"src/adapters/primary/http/orderController.js",
				"src/middleware/requestId.js", // autopilot
				"src/db/retry.js",             // db retry
			},
		},
		{
			name: "Python MVP with Dynamic Models",
			req: GenerateRequest{
				Language:     "python",
				Framework:    "fastapi",
				Architecture: "mvp",
				Database:     "postgresql",
				FileToggles:  FileToggleOptions{ExampleCRUD: ptr(true)},
				Custom: CustomOptions{
					Models: []DataModel{
						{Name: "Customer", Fields: []DataField{{Name: "Name", Type: "string"}}},
					},
				},
			},
			expectedFiles: []string{
				"app/routes/customers.py",
				"app/middleware/request_id.py", // autopilot
				"app/db/retry.py",              // db retry
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := engine.Generate(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("Generate() failed: %v", err)
			}

			// Convert tree into a flat map of paths for easy checking
			files := make(map[string]bool)
			for _, path := range resp.FilePaths {
				files[path] = true
			}

			for _, expected := range tt.expectedFiles {
				if !files[expected] {
					// print keys to debug if missing
					var available []string
					for k := range files {
						available = append(available, k)
					}
					t.Errorf("Expected file %q not found in generated tree.\nAvailable files:\n%s", expected, strings.Join(available, "\n"))
				}
			}
		})
	}
}
