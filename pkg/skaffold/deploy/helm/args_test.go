package helm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/gcs"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

// mockGsutil is a mock implementation of gcs.Gsutil for testing
type mockGsutil struct {
	copyFunc func(ctx context.Context, src, dst string, recursive bool) error
}

func (m *mockGsutil) Copy(ctx context.Context, src, dst string, recursive bool) error {
	if m.copyFunc != nil {
		return m.copyFunc(ctx, src, dst, recursive)
	}
	return nil
}

func TestProcessGCSFlags(t *testing.T) {
	tests := []struct {
		name        string
		flags       []string
		expected    []string
		shouldError bool
		setupMock   func() *mockGsutil
	}{
		{
			name:     "empty flags",
			flags:    []string{},
			expected: []string{},
		},
		{
			name:     "no GCS URLs",
			flags:    []string{"--atomic=true", "--wait"},
			expected: []string{"--atomic=true", "--wait"},
		},
		{
			name:     "single --values=gs:// flag",
			flags:    []string{"--values=gs://bucket/file.yaml"},
			expected: []string{"--values=/tmp/test-file.yaml"}, // We'll mock the temp file path
			setupMock: func() *mockGsutil {
				return &mockGsutil{
					copyFunc: func(ctx context.Context, src, dst string, recursive bool) error {
						// Mock successful copy
						if strings.HasPrefix(src, "gs://") {
							// Create the destination file for testing
							return os.WriteFile(dst, []byte("test: value"), 0644)
						}
						return nil
					},
				}
			},
		},
		{
			name:     "mixed flags with GCS URL",
			flags:    []string{"--atomic=true", "--values=gs://bucket/infra.yaml", "--wait"},
			expected: []string{"--atomic=true", "--values=/tmp/test-file.yaml", "--wait"},
			setupMock: func() *mockGsutil {
				return &mockGsutil{
					copyFunc: func(ctx context.Context, src, dst string, recursive bool) error {
						return os.WriteFile(dst, []byte("test: value"), 0644)
					},
				}
			},
		},
		{
			name:     "separate --values flag with gs:// URL",
			flags:    []string{"--values", "gs://bucket/file.yaml", "--atomic=true"},
			expected: []string{"--values", "/tmp/test-file.yaml", "--atomic=true"},
			setupMock: func() *mockGsutil {
				return &mockGsutil{
					copyFunc: func(ctx context.Context, src, dst string, recursive bool) error {
						return os.WriteFile(dst, []byte("test: value"), 0644)
					},
				}
			},
		},
		{
			name:     "-f flag with gs:// URL",
			flags:    []string{"-f", "gs://bucket/values.yaml"},
			expected: []string{"-f", "/tmp/test-file.yaml"},
			setupMock: func() *mockGsutil {
				return &mockGsutil{
					copyFunc: func(ctx context.Context, src, dst string, recursive bool) error {
						return os.WriteFile(dst, []byte("test: value"), 0644)
					},
				}
			},
		},
		{
			name:     "multiple GCS URLs",
			flags:    []string{"--values=gs://bucket1/file1.yaml", "--values=gs://bucket2/file2.yaml"},
			expected: []string{"--values=/tmp/test-file1.yaml", "--values=/tmp/test-file2.yaml"},
			setupMock: func() *mockGsutil {
				fileCounter := 0
				return &mockGsutil{
					copyFunc: func(ctx context.Context, src, dst string, recursive bool) error {
						fileCounter++
						return os.WriteFile(dst, []byte("test: value"), 0644)
					},
				}
			},
		},
		{
			name:     "non-GCS URLs unchanged",
			flags:    []string{"--values=local-file.yaml", "--values=https://example.com/file.yaml"},
			expected: []string{"--values=local-file.yaml", "--values=https://example.com/file.yaml"},
		},
		{
			name:        "GCS download failure",
			flags:       []string{"--values=gs://bucket/nonexistent.yaml"},
			shouldError: true,
			setupMock: func() *mockGsutil {
				return &mockGsutil{
					copyFunc: func(ctx context.Context, src, dst string, recursive bool) error {
						return fmt.Errorf("file not found")
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock extractValueFileFromGCSFunc if we have a setup function
			if tt.setupMock != nil {
				mockGCS := tt.setupMock()

				// Create a mock implementation of extractValueFileFromGCSFunc
				originalExtractFunc := extractValueFileFromGCSFunc
				fileCounter := 0
				extractValueFileFromGCSFunc = func(gcsPath, tempDir string, gcs gcs.Gsutil) (string, error) {
					fileCounter++
					tempFile := filepath.Join(tempDir, "test-file.yaml")
					if fileCounter > 1 {
						tempFile = filepath.Join(tempDir, fmt.Sprintf("test-file%d.yaml", fileCounter))
					}

					err := mockGCS.Copy(context.TODO(), gcsPath, tempFile, false)
					if err != nil {
						return "", err
					}
					return tempFile, nil
				}
				defer func() {
					extractValueFileFromGCSFunc = originalExtractFunc
				}()
			}

			result, err := processGCSFlags(tt.flags)

			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// For tests with mock setup, we need to check the structure rather than exact paths
			if tt.setupMock != nil {
				if len(result) != len(tt.expected) {
					t.Errorf("expected %d flags, got %d: %v", len(tt.expected), len(result), result)
					return
				}

				for i, flag := range result {
					expectedFlag := tt.expected[i]
					if strings.HasPrefix(expectedFlag, "--values=/tmp/") {
						// For GCS flags, check that it starts with --values= and points to a temp file
						if !strings.HasPrefix(flag, "--values=") {
							t.Errorf("expected flag to start with --values=, got: %s", flag)
						}
						value := strings.TrimPrefix(flag, "--values=")
						if !strings.HasPrefix(value, "/") || !strings.HasSuffix(value, ".yaml") {
							t.Errorf("expected temp file path, got: %s", value)
						}
					} else if expectedFlag == "/tmp/test-file.yaml" {
						// For separate flags, check that it's a temp file path
						if !strings.HasPrefix(flag, "/") || !strings.HasSuffix(flag, ".yaml") {
							t.Errorf("expected temp file path, got: %s", flag)
						}
					} else {
						// For non-GCS flags, exact match
						if flag != expectedFlag {
							t.Errorf("expected flag %s, got %s", expectedFlag, flag)
						}
					}
				}
			} else {
				// For tests without mock setup, exact comparison
				if !reflect.DeepEqual(result, tt.expected) {
					t.Errorf("expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestProcessGCSFlags_Integration(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		// Create a real temp file to simulate GCS content
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test-values.yaml")
		testContent := `
env: test
replicas: 3
image:
  tag: latest
`
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Test with a local file (non-GCS) to ensure it passes through unchanged
		flags := []string{
			"--values=" + testFile,
			"--atomic=true",
		}

		result, err := processGCSFlags(flags)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []string{
			"--values=" + testFile,
			"--atomic=true",
		}

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})
}

func TestProcessGCSFlags_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		flags    []string
		expected []string
	}{
		{
			name:     "only --values flag without argument",
			flags:    []string{"--values"},
			expected: []string{"--values"},
		},
		{
			name:     "only -f flag without argument",
			flags:    []string{"-f"},
			expected: []string{"-f"},
		},
		{
			name:     "--values= with empty value",
			flags:    []string{"--values="},
			expected: []string{"--values="},
		},
		{
			name:     "gs:// URL that's not a --values flag",
			flags:    []string{"--set", "url=gs://bucket/file.yaml"},
			expected: []string{"--set", "url=gs://bucket/file.yaml"},
		},
		{
			name:     "mixed case sensitivity",
			flags:    []string{"--VALUES=gs://bucket/file.yaml"},
			expected: []string{"--VALUES=gs://bucket/file.yaml"}, // Should not match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processGCSFlags(tt.flags)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
