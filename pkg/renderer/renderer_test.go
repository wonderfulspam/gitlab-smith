package renderer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderer_GitLabClientIntegration(t *testing.T) {
	// Mock GitLab API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pipelines/123") && !strings.Contains(r.URL.Path, "/jobs") {
			// Pipeline endpoint
			pipeline := PipelineExecution{
				ID:       123,
				Status:   "success",
				Ref:      "main",
				SHA:      "abc123",
				Duration: 300,
			}
			json.NewEncoder(w).Encode(pipeline)
		} else if strings.Contains(r.URL.Path, "/pipelines/123/jobs") {
			// Jobs endpoint
			jobs := []JobExecution{
				{
					ID:       1,
					Name:     "build",
					Stage:    "build",
					Status:   "success",
					Duration: 60.0,
				},
				{
					ID:       2,
					Name:     "test",
					Stage:    "test",
					Status:   "success",
					Duration: 120.0,
				},
			}
			json.NewEncoder(w).Encode(jobs)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewGitLabClient(server.URL, "test-token", "123")
	renderer := New(client)

	ctx := context.Background()
	pipeline, err := renderer.RenderPipeline(ctx, 123)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if pipeline.ID != 123 {
		t.Errorf("Expected pipeline ID 123, got %d", pipeline.ID)
	}

	if len(pipeline.Jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(pipeline.Jobs))
	}

	if pipeline.Jobs[0].Name != "build" {
		t.Errorf("Expected first job to be 'build', got %s", pipeline.Jobs[0].Name)
	}
}
