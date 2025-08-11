package parser

import (
	"testing"
)

func TestParseGlobalCacheAndImage(t *testing.T) {
	tests := []struct {
		name          string
		yaml          string
		expectedImage string
		expectedCache *Cache
	}{
		{
			name: "simple global cache and image",
			yaml: `image: golang:latest
cache:
  key: test-key
  paths:
    - .cache/
stages:
  - test
test_job:
  stage: test
  script:
    - echo "test"`,
			expectedImage: "golang:latest",
			expectedCache: &Cache{
				Key:   "test-key",
				Paths: []string{".cache/"},
			},
		},
		{
			name: "complex cache key with files",
			yaml: `image: golang:1.21-alpine
cache:
  key:
    files:
      - go.mod
      - go.sum
  paths:
    - .go/pkg/mod/
  policy: pull-push
stages:
  - build
build_job:
  stage: build
  script:
    - go build`,
			expectedImage: "golang:1.21-alpine",
			expectedCache: &Cache{
				Key: map[string]interface{}{
					"files": []interface{}{"go.mod", "go.sum"},
				},
				Paths:  []string{".go/pkg/mod/"},
				Policy: "pull-push",
			},
		},
		{
			name: "no global cache or image",
			yaml: `stages:
  - test
test_job:
  stage: test
  script:
    - echo "test"`,
			expectedImage: "",
			expectedCache: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := Parse([]byte(tt.yaml))
			if err != nil {
				t.Fatalf("Failed to parse YAML: %v", err)
			}

			// Check image
			if config.Image != tt.expectedImage {
				t.Errorf("Expected image '%s', got '%s'", tt.expectedImage, config.Image)
			}

			// Check cache
			if tt.expectedCache == nil {
				if config.Cache != nil {
					t.Errorf("Expected no cache, but got: %+v", config.Cache)
				}
			} else {
				if config.Cache == nil {
					t.Errorf("Expected cache, but got nil")
					return
				}

				if len(tt.expectedCache.Paths) > 0 {
					if len(config.Cache.Paths) != len(tt.expectedCache.Paths) {
						t.Errorf("Expected %d cache paths, got %d", len(tt.expectedCache.Paths), len(config.Cache.Paths))
					} else {
						for i, path := range tt.expectedCache.Paths {
							if config.Cache.Paths[i] != path {
								t.Errorf("Expected cache path '%s', got '%s'", path, config.Cache.Paths[i])
							}
						}
					}
				}

				if tt.expectedCache.Policy != "" && config.Cache.Policy != tt.expectedCache.Policy {
					t.Errorf("Expected cache policy '%s', got '%s'", tt.expectedCache.Policy, config.Cache.Policy)
				}

				// Check cache key (can be string or complex object)
				if stringKey, ok := tt.expectedCache.Key.(string); ok {
					if configKey, ok := config.Cache.Key.(string); ok {
						if configKey != stringKey {
							t.Errorf("Expected cache key '%s', got '%s'", stringKey, configKey)
						}
					} else {
						t.Errorf("Expected string cache key '%s', got %T: %v", stringKey, config.Cache.Key, config.Cache.Key)
					}
				}
			}
		})
	}
}

func TestParseJobWithCache(t *testing.T) {
	yaml := `stages:
  - test
test_job:
  stage: test
  script:
    - echo "test"
  cache:
    key: job-cache
    paths:
      - node_modules/`

	config, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	job, exists := config.Jobs["test_job"]
	if !exists {
		t.Fatal("Expected test_job to exist")
	}

	if job.Cache == nil {
		t.Fatal("Expected job to have cache configuration")
	}

	expectedKey := "job-cache"
	if keyStr, ok := job.Cache.Key.(string); !ok || keyStr != expectedKey {
		t.Errorf("Expected job cache key '%s', got %v", expectedKey, job.Cache.Key)
	}

	expectedPaths := []string{"node_modules/"}
	if len(job.Cache.Paths) != len(expectedPaths) {
		t.Errorf("Expected %d cache paths, got %d", len(expectedPaths), len(job.Cache.Paths))
	} else if job.Cache.Paths[0] != expectedPaths[0] {
		t.Errorf("Expected cache path '%s', got '%s'", expectedPaths[0], job.Cache.Paths[0])
	}
}
