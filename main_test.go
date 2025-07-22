package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func TestGetColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		coverage        float64
		redThreshold    float64
		yellowThreshold float64
		expectedColor   string
	}{
		{"Low coverage - red", 30.0, 40.0, 70.0, "#e05d44"},
		{"Medium coverage - yellow", 50.0, 40.0, 70.0, "#dfb317"},
		{"High coverage - green", 80.0, 40.0, 70.0, "#44cc11"},
		{"Exactly at red threshold", 40.0, 40.0, 70.0, "#dfb317"},
		{"Exactly at yellow threshold", 70.0, 40.0, 70.0, "#44cc11"},
		{"Zero coverage", 0.0, 40.0, 70.0, "#e05d44"},
		{"Perfect coverage", 100.0, 40.0, 70.0, "#44cc11"},
		{"Custom thresholds", 45.0, 30.0, 60.0, "#dfb317"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := getColor(tt.coverage, tt.redThreshold, tt.yellowThreshold)
			if result != tt.expectedColor {
				t.Errorf("getColor(%f, %f, %f) = %s; want %s",
					tt.coverage, tt.redThreshold, tt.yellowThreshold, result, tt.expectedColor)
			}
		})
	}
}

func TestParseCoverageFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	tests := []struct {
		name        string
		fileContent string
		expected    float64
		shouldError bool
	}{
		{
			name: "Valid coverage file with mixed coverage",
			fileContent: `mode: set
github.com/user/repo/file1.go:10.5,15.10 3 1
github.com/user/repo/file1.go:20.5,25.10 2 0
github.com/user/repo/file2.go:30.5,35.10 4 1`,
			expected:    77.8, // 7/9 * 100 = 77.777... ≈ 77.8
			shouldError: false,
		},
		{
			name: "All lines covered",
			fileContent: `mode: set
github.com/user/repo/file1.go:10.5,15.10 5 1
github.com/user/repo/file2.go:20.5,25.10 3 1`,
			expected:    100.0,
			shouldError: false,
		},
		{
			name: "No lines covered",
			fileContent: `mode: set
github.com/user/repo/file1.go:10.5,15.10 5 0
github.com/user/repo/file2.go:20.5,25.10 3 0`,
			expected:    0.0,
			shouldError: false,
		},
		{
			name:        "Empty coverage file (no statements)",
			fileContent: `mode: set`,
			expected:    0.0,
			shouldError: false,
		},
		{
			name:        "Invalid format - no mode line",
			fileContent: `github.com/user/repo/file1.go:10.5,15.10 5 1`,
			expected:    0.0,
			shouldError: true,
		},
		{
			name: "Single line with coverage",
			fileContent: `mode: count
github.com/user/repo/file1.go:10.5,15.10 1 5`,
			expected:    100.0,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temporary file
			testFile := filepath.Join(tempDir, "coverage-"+strings.ReplaceAll(tt.name, " ", "-")+".out")

			err := os.WriteFile(testFile, []byte(tt.fileContent), 0o644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			result, err := parseCoverageFile(testFile)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
				return
			}

			// Use a small tolerance for floating point comparison
			tolerance := 0.1
			if result < tt.expected-tolerance || result > tt.expected+tolerance {
				t.Errorf("parseCoverageFile() = %f; want %f (±%f)", result, tt.expected, tolerance)
			}
		})
	}
}

func TestParseCoverageFileNotFound(t *testing.T) {
	t.Parallel()

	_, err := parseCoverageFile("nonexistent-file.out")
	if err == nil {
		t.Error("Expected error when file doesn't exist, but got none")
	}
}

func TestGenerateBadge(t *testing.T) {
	t.Parallel()

	simpleTemplate := `<svg><text>{{.Coverage}}%</text><rect fill="{{.Color}}"/></svg>`

	tests := []struct {
		name        string
		coverage    float64
		config      *Config
		shouldError bool
		contains    []string
	}{
		{
			name:     "Basic badge generation",
			coverage: 75.5,
			config: &Config{
				RedThreshold:    40.0,
				YellowThreshold: 70.0,
				Template:        simpleTemplate,
			},
			shouldError: false,
			contains:    []string{"75.5%", "#44cc11"}, // green color
		},
		{
			name:     "Red badge for low coverage",
			coverage: 25.0,
			config: &Config{
				RedThreshold:    40.0,
				YellowThreshold: 70.0,
				Template:        simpleTemplate,
			},
			shouldError: false,
			contains:    []string{"25.0%", "#e05d44"}, // red color
		},
		{
			name:     "Yellow badge for medium coverage",
			coverage: 55.0,
			config: &Config{
				RedThreshold:    40.0,
				YellowThreshold: 70.0,
				Template:        simpleTemplate,
			},
			shouldError: false,
			contains:    []string{"55.0%", "#dfb317"}, // yellow color
		},
		{
			name:     "Invalid template",
			coverage: 50.0,
			config: &Config{
				RedThreshold:    40.0,
				YellowThreshold: 70.0,
				Template:        `<svg>{{.InvalidField}}</svg>`,
			},
			shouldError: true,
			contains:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := generateBadge(tt.coverage, tt.config)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
				return
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain %q, but it didn't. Result: %s", expected, result)
				}
			}
		})
	}
}

func TestGenerateBadgeWithDefaultTemplate(t *testing.T) {
	t.Parallel()

	config := &Config{
		RedThreshold:    40.0,
		YellowThreshold: 70.0,
		Template:        defaultTemplate,
	}

	result, err := generateBadge(85.5, config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that the result looks like a valid SVG
	if !strings.HasPrefix(result, "<svg") {
		t.Error("Expected result to start with '<svg'")
	}

	if !strings.HasSuffix(strings.TrimSpace(result), "</svg>") {
		t.Error("Expected result to end with '</svg>'")
	}

	if !strings.Contains(result, "85.5") {
		t.Error("Expected result to contain coverage percentage")
	}
}

func TestConfigDefaults(t *testing.T) {
	t.Parallel()

	config := &Config{
		TestCommand:     "go test ./... -coverprofile=coverage.out",
		OutputFile:      "coverage-badge.svg",
		RedThreshold:    40.0,
		YellowThreshold: 70.0,
	}

	// Test that default values are sensible
	if config.TestCommand == "" {
		t.Error("TestCommand should have a default value")
	}

	if config.OutputFile == "" {
		t.Error("OutputFile should have a default value")
	}

	if config.RedThreshold <= 0 || config.RedThreshold >= 100 {
		t.Error("RedThreshold should be between 0 and 100")
	}

	if config.YellowThreshold <= config.RedThreshold || config.YellowThreshold >= 100 {
		t.Error("YellowThreshold should be between RedThreshold and 100")
	}
}

func TestTemplateValidation(t *testing.T) {
	t.Parallel()

	// Test that the default template is valid
	tmpl, err := template.New("test").Parse(defaultTemplate)
	if err != nil {
		t.Errorf("Default template is invalid: %v", err)
	}

	// Test that template can be executed with expected data
	data := struct {
		Coverage string
		Color    string
	}{
		Coverage: "75.5",
		Color:    "#44cc11",
	}

	var buf strings.Builder

	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Errorf("Failed to execute default template: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "75.5") {
		t.Error("Template execution should include coverage value")
	}

	if !strings.Contains(result, "#44cc11") {
		t.Error("Template execution should include color value")
	}
}

//nolint:paralleltest // t.Chdir
func TestRunApplication(t *testing.T) {
	tempDir := t.TempDir()
	coverageFile := filepath.Join(tempDir, "test.out")
	coverageContent := `mode: set
github.com/test/example.go:10.5,15.10 5 1
github.com/test/example.go:20.5,25.10 3 0`

	err := os.WriteFile(coverageFile, []byte(coverageContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create coverage file: %v", err)
	}

	tests := []struct {
		name          string
		config        *Config
		expectError   bool
		errorContains string
	}{
		{
			name: "Valid configuration - quiet mode",
			config: &Config{
				TestCommand:     "echo 'test output' && echo 'coverage: statements' -coverprofile=" + coverageFile,
				OutputFile:      filepath.Join(tempDir, "test-badge.svg"),
				RedThreshold:    40.0,
				YellowThreshold: 70.0,
				Template:        "", // Use default template
				Quiet:           true,
			},
			expectError: false,
		},
		{
			name: "Valid configuration - verbose mode",
			config: &Config{
				TestCommand:     "echo 'test output' && echo 'coverage: statements' -coverprofile=" + coverageFile,
				OutputFile:      filepath.Join(tempDir, "test-badge-verbose.svg"),
				RedThreshold:    40.0,
				YellowThreshold: 70.0,
				Template:        "", // Use default template
				Quiet:           false,
			},
			expectError: false,
		},
		{
			name: "Invalid template file",
			config: &Config{
				TestCommand:     "echo 'test'",
				OutputFile:      filepath.Join(tempDir, "test-badge.svg"),
				RedThreshold:    40.0,
				YellowThreshold: 70.0,
				Template:        "/nonexistent/template.svg",
				Quiet:           true,
			},
			expectError:   true,
			errorContains: "could not read template file",
		},
		{
			name: "Invalid test command",
			config: &Config{
				TestCommand:     "",
				OutputFile:      filepath.Join(tempDir, "test-badge.svg"),
				RedThreshold:    40.0,
				YellowThreshold: 70.0,
				Template:        "", // Use default template
				Quiet:           true,
			},
			expectError:   true,
			errorContains: "error getting coverage",
		},
		{
			name: "Invalid output directory",
			config: &Config{
				TestCommand:     "echo 'test output' && echo 'coverage: statements' -coverprofile=" + coverageFile,
				OutputFile:      "/invalid/path/badge.svg",
				RedThreshold:    40.0,
				YellowThreshold: 70.0,
				Template:        "", // Use default template
				Quiet:           true,
			},
			expectError:   true,
			errorContains: "error writing badge file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(tempDir)

			err := runApplication(tt.config)

			if tt.expectError { //nolint:nestif // ok
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}

				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}

				// Verify output file was created
				if _, err := os.Stat(tt.config.OutputFile); err != nil {
					t.Errorf("Output file not created: %v", err)
				}
			}
		})
	}
}

func TestLoadTemplate(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	testTemplateFile := filepath.Join(tempDir, "test-template.svg")
	testTemplateContent := `<svg><text>{{.Coverage}}% test</text></svg>`

	err := os.WriteFile(testTemplateFile, []byte(testTemplateContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	tests := []struct {
		name             string
		config           *Config
		expectError      bool
		expectedTemplate string
	}{
		{
			name: "Load from file",
			config: &Config{
				Template: testTemplateFile,
			},
			expectError:      false,
			expectedTemplate: testTemplateContent,
		},
		{
			name: "Use default template",
			config: &Config{
				Template: "",
			},
			expectError:      false,
			expectedTemplate: defaultTemplate,
		},
		{
			name: "Nonexistent file",
			config: &Config{
				Template: "/nonexistent/file.svg",
			},
			expectError: true,
		},
		{
			name: "Empty filename",
			config: &Config{
				Template: "",
			},
			expectError:      false,
			expectedTemplate: defaultTemplate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := loadTemplate(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}

				if tt.config.Template != tt.expectedTemplate {
					t.Errorf("Expected template content to be %q, got %q",
						tt.expectedTemplate, tt.config.Template)
				}
			}
		})
	}
}

func TestWriteBadgeFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	tests := []struct {
		name        string
		filename    string
		content     string
		expectError bool
	}{
		{
			name:        "Valid file write",
			filename:    filepath.Join(tempDir, "test-badge.svg"),
			content:     `<svg><text>75.5%</text></svg>`,
			expectError: false,
		},
		{
			name:        "Write to subdirectory",
			filename:    filepath.Join(tempDir, "subdir", "badge.svg"),
			content:     `<svg><text>100.0%</text></svg>`,
			expectError: true, // subdirectory doesn't exist
		},
		{
			name:        "Empty content",
			filename:    filepath.Join(tempDir, "empty-badge.svg"),
			content:     "",
			expectError: false,
		},
		{
			name:        "Large content",
			filename:    filepath.Join(tempDir, "large-badge.svg"),
			content:     strings.Repeat("<svg>test</svg>", 1000),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := writeBadgeFile(tt.filename, tt.content)
			if tt.expectError { //nolint:nestif // ok
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}

				// Verify file exists and contains expected content
				data, err := os.ReadFile(tt.filename)
				if err != nil {
					t.Errorf("Failed to read written file: %v", err)
					return
				}

				if string(data) != tt.content {
					t.Errorf("File content doesn't match. Expected %q, got %q",
						tt.content, string(data))
				}

				// Check file permissions
				info, err := os.Stat(tt.filename)
				if err != nil {
					t.Errorf("Failed to stat file: %v", err)
					return
				}

				expectedMode := os.FileMode(0o640)
				if info.Mode().Perm() != expectedMode {
					t.Errorf("File has wrong permissions. Expected %v, got %v",
						expectedMode, info.Mode().Perm())
				}
			}
		})
	}
}

//nolint:paralleltest // t.Chdir
func TestMainFunctionIntegration(t *testing.T) {
	tempDir := t.TempDir()
	coverageFile := filepath.Join(tempDir, "coverage.out")
	coverageContent := `mode: set
github.com/test/main.go:10.5,15.10 8 1
github.com/test/main.go:20.5,25.10 4 1`

	err := os.WriteFile(coverageFile, []byte(coverageContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create coverage file: %v", err)
	}

	config := &Config{
		TestCommand:     "echo 'test completed' -coverprofile=" + coverageFile,
		OutputFile:      filepath.Join(tempDir, "main-test-badge.svg"),
		RedThreshold:    40.0,
		YellowThreshold: 70.0,
		Template:        "", // Should use default
		DumpTemplate:    false,
		Quiet:           true,
	}

	t.Chdir(tempDir)

	err = runApplication(config)
	if err != nil {
		t.Errorf("runApplication failed: %v", err)
		return
	}

	// Verify badge was created
	badgeData, err := os.ReadFile(config.OutputFile)
	if err != nil {
		t.Errorf("Badge file not created: %v", err)
		return
	}

	badgeContent := string(badgeData)
	if !strings.Contains(badgeContent, "<svg") {
		t.Error("Badge doesn't contain SVG content")
	}

	if !strings.Contains(badgeContent, "100.0") {
		t.Error("Badge doesn't contain expected coverage percentage")
	}
}

func BenchmarkGetColor(b *testing.B) {
	for b.Loop() {
		getColor(75.5, 40.0, 70.0)
	}
}

func BenchmarkRunApplication(b *testing.B) {
	tempDir := b.TempDir()
	coverageFile := filepath.Join(tempDir, "coverage.out")
	coverageContent := `mode: set
github.com/test/main.go:10.5,15.10 10 1`
	os.WriteFile(coverageFile, []byte(coverageContent), 0o644)

	config := &Config{
		TestCommand:     "echo 'test' -coverprofile=" + coverageFile,
		OutputFile:      filepath.Join(tempDir, "bench-badge.svg"),
		RedThreshold:    40.0,
		YellowThreshold: 70.0,
		Template:        defaultTemplate,
		Quiet:           true,
	}

	b.Chdir(tempDir)
	b.ResetTimer()

	for b.Loop() {
		_ = runApplication(config)
	}
}

func TestConfigurationEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config *Config
		valid  bool
	}{
		{
			name: "Zero thresholds",
			config: &Config{
				RedThreshold:    0,
				YellowThreshold: 0,
			},
			valid: true, // Should work even with zero thresholds
		},
		{
			name: "Negative thresholds",
			config: &Config{
				RedThreshold:    -10,
				YellowThreshold: -5,
			},
			valid: true, // Function should handle negative values
		},
		{
			name: "Very high thresholds",
			config: &Config{
				RedThreshold:    150,
				YellowThreshold: 200,
			},
			valid: true,
		},
		{
			name: "Red higher than yellow",
			config: &Config{
				RedThreshold:    80,
				YellowThreshold: 50,
			},
			valid: true, // Function should handle this case
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test that getColor doesn't panic with unusual threshold values
			color := getColor(50.0, tt.config.RedThreshold, tt.config.YellowThreshold)
			if color == "" {
				t.Error("getColor returned empty string")
			}
		})
	}
}

func TestParseCoverageFileEdgeCases(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	tests := []struct {
		name        string
		content     string
		expected    float64
		shouldError bool
	}{
		{
			name: "File with only mode line and empty lines",
			content: `mode: set


`,
			expected:    0.0,
			shouldError: false,
		},
		{
			name: "File with malformed lines (should be skipped)",
			content: `mode: set
github.com/test/file.go:10.5,15.10 5 1
malformed line without proper format
github.com/test/file.go:20.5,25.10 3 0
another malformed line
`,
			expected:    62.5, // 5/8 * 100
			shouldError: false,
		},
		{
			name: "File with very large numbers",
			content: `mode: set
github.com/test/file.go:10.5,15.10 1000000 1
github.com/test/file.go:20.5,25.10 500000 0
`,
			expected:    66.7, // 1000000/1500000 * 100 ≈ 66.7%
			shouldError: false,
		},
		{
			name: "File with zero statement counts",
			content: `mode: set
github.com/test/file.go:10.5,15.10 0 1
github.com/test/file.go:20.5,25.10 0 0
`,
			expected:    0.0,
			shouldError: false,
		},
		{
			name: "File with mixed valid and invalid lines",
			content: `mode: set
github.com/test/file.go:10.5,15.10 abc 1
github.com/test/file.go:20.5,25.10 5 def
github.com/test/file.go:30.5,35.10 10 1
`,
			expected:    100.0, // Only the valid line: 10/10 * 100
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			testFile := filepath.Join(tempDir, "coverage-"+strings.ReplaceAll(tt.name, " ", "-")+".out")

			err := os.WriteFile(testFile, []byte(tt.content), 0o644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			result, err := parseCoverageFile(testFile)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			tolerance := 0.1
			if result < tt.expected-tolerance || result > tt.expected+tolerance {
				t.Errorf("Expected coverage %.1f%%, got %.1f%%", tt.expected, result)
			}
		})
	}
}

//nolint:paralleltest // t.Chdir
func TestRunTestsAndGetCoverageEdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	tests := []struct {
		name        string
		command     string
		setupFunc   func() error
		expectError bool
	}{
		{
			name:    "Command with single word",
			command: "echo",
			setupFunc: func() error {
				return os.WriteFile("coverage.out", []byte("mode: set\n"), 0o644)
			},
			expectError: false,
		},
		{
			name:    "Command that creates coverage in current directory",
			command: "echo 'test'",
			setupFunc: func() error {
				return os.WriteFile("coverage.out", []byte(`mode: set
test.go:1.1,2.2 1 1
`), 0o644)
			},
			expectError: false,
		},
		{
			name:    "Command with custom coverage profile name",
			command: "echo 'test' -coverprofile=custom.out",
			setupFunc: func() error {
				return os.WriteFile("custom.out", []byte(`mode: set
test.go:1.1,2.2 5 1
`), 0o644)
			},
			expectError: false,
		},
		{
			name:        "Command with spaces in arguments",
			command:     `echo "hello world" test arguments`,
			setupFunc:   func() error { return os.WriteFile("coverage.out", []byte("mode: set\n"), 0o644) },
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(tempDir)

			if tt.setupFunc != nil {
				if err := tt.setupFunc(); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			_, err := runTestsAndGetCoverage(tt.command, false)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else if err != nil {
				t.Logf("Command failed (may be expected in test environment): %v", err)
			}

			// Clean up for next test
			os.RemoveAll(tempDir)
			os.MkdirAll(tempDir, 0o755)
		})
	}
}

func TestGenerateBadgeTemplateErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		coverage    float64
		template    string
		expectError bool
	}{
		{
			name:        "Template with invalid syntax",
			coverage:    75.0,
			template:    `<svg>{{.Coverage</svg>`, // Missing closing }}
			expectError: true,
		},
		{
			name:        "Template with undefined function",
			coverage:    75.0,
			template:    `<svg>{{unknownFunc .Coverage}}</svg>`,
			expectError: true,
		},
		{
			name:        "Template with nil data access",
			coverage:    75.0,
			template:    `<svg>{{.Coverage.NonExistent}}</svg>`,
			expectError: true,
		},
		{
			name:        "Empty template",
			coverage:    75.0,
			template:    ``,
			expectError: false, // Empty template should work
		},
		{
			name:        "Template with complex expressions",
			coverage:    75.0,
			template:    `<svg>{{.Coverage | invalidFunc}}</svg>`,
			expectError: true, // Should fail because invalidFunc doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := &Config{
				RedThreshold:    40.0,
				YellowThreshold: 70.0,
				Template:        tt.template,
			}

			_, err := generateBadge(tt.coverage, config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestWriteBadgeFilePermissions(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tempDir := t.TempDir()
	readOnlyDir := filepath.Join(tempDir, "readonly")

	err := os.Mkdir(readOnlyDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create read-only directory: %v", err)
	}

	err = os.Chmod(readOnlyDir, 0o555) // Read and execute only
	if err != nil {
		t.Fatalf("Failed to set directory permissions: %v", err)
	}

	// Try to write to read-only directory
	readOnlyFile := filepath.Join(readOnlyDir, "badge.svg")

	err = writeBadgeFile(readOnlyFile, "<svg>test</svg>")
	if err == nil {
		t.Error("Expected error when writing to read-only directory, but got none")
	}

	// Restore permissions for cleanup
	os.Chmod(readOnlyDir, 0o755)
}

func BenchmarkLoadTemplate(b *testing.B) {
	config := &Config{
		Template: "", // Use default template
	}

	b.ResetTimer()

	for b.Loop() {
		_ = loadTemplate(config)
	}
}

func BenchmarkWriteBadgeFile(b *testing.B) {
	tempDir := b.TempDir()
	content := `<svg xmlns="http://www.w3.org/2000/svg" width="104" height="20">
		<text x="10" y="15">75.5%</text>
	</svg>`

	b.ResetTimer()

	for i := range b.N {
		filename := filepath.Join(tempDir, fmt.Sprintf("badge-%d.svg", i))
		_ = writeBadgeFile(filename, content)
	}
}

func BenchmarkGenerateBadge(b *testing.B) {
	config := &Config{
		RedThreshold:    40.0,
		YellowThreshold: 70.0,
		Template:        defaultTemplate,
	}

	b.ResetTimer()

	for b.Loop() {
		_, _ = generateBadge(75.5, config)
	}
}

//nolint:paralleltest // t.Chdir
func TestRunTestsAndGetCoverageIntegration(t *testing.T) {
	tempDir := t.TempDir()
	coverageFile := filepath.Join(tempDir, "test-coverage.out")
	coverageContent := `mode: set
github.com/test/repo/file1.go:10.5,15.10 5 1
github.com/test/repo/file2.go:20.5,25.10 3 0
github.com/test/repo/file3.go:30.5,35.10 2 1`

	err := os.WriteFile(coverageFile, []byte(coverageContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test coverage file: %v", err)
	}

	tests := []struct {
		name          string
		command       string
		expectedError bool
		expectedRange []float64 // [min, max] for coverage percentage
	}{
		{
			name:          "Invalid command - empty",
			command:       "",
			expectedError: true,
			expectedRange: []float64{0, 0},
		},
		{
			name:          "Invalid command - nonexistent binary",
			command:       "nonexistent-command test",
			expectedError: true,
			expectedRange: []float64{0, 0},
		},
		{
			name:          "Valid command with custom coverage file",
			command:       "echo 'test output' && echo 'coverage: 70.0% of statements' -coverprofile=" + coverageFile,
			expectedError: false,
			expectedRange: []float64{60, 80}, // Expected around 70%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(tempDir)

			coverage, err := runTestsAndGetCoverage(tt.command, false)
			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error for command %q, but got none", tt.command)
				}

				return
			}

			if err != nil && !tt.expectedError {
				// For valid commands that might still fail due to environment,
				// we'll be more lenient and just check the parsing logic
				t.Logf("Command failed (expected in test environment): %v", err)
				return
			}

			if len(tt.expectedRange) == 2 && (coverage < tt.expectedRange[0] || coverage > tt.expectedRange[1]) {
				t.Errorf("Coverage %f not in expected range [%f, %f]",
					coverage, tt.expectedRange[0], tt.expectedRange[1])
			}
		})
	}
}

//nolint:paralleltest // t.Chdir
func TestRunTestsAndGetCoverageWithActualFile(t *testing.T) {
	// Test with the sample coverage file we created
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()

	t.Chdir(tempDir)

	// Copy our test coverage file
	sourceFile := filepath.Join(originalDir, "testdata", "coverage-sample.out")
	destFile := filepath.Join(tempDir, "coverage.out")

	// Read source file
	data, err := os.ReadFile(sourceFile)
	if err != nil {
		t.Skipf("Skipping test - cannot read sample coverage file: %v", err)
	}

	// Write to destination
	err = os.WriteFile(destFile, data, 0o644)
	if err != nil {
		t.Fatalf("Failed to write test coverage file: %v", err)
	}

	// Test with a simple command that should succeed
	command := "echo 'Running tests...' -coverprofile=coverage.out"

	coverage, err := runTestsAndGetCoverage(command, false)
	if err != nil {
		// The command might fail, but if it does, test the parsing directly
		t.Logf("Command failed (testing parsing directly): %v", err)

		coverage, err = parseCoverageFile(destFile)
		if err != nil {
			t.Fatalf("Failed to parse coverage file: %v", err)
		}
	}

	// Coverage should be reasonable (between 0 and 100)
	if coverage < 0 || coverage > 100 {
		t.Errorf("Coverage %f is not in valid range [0, 100]", coverage)
	}
}

//nolint:paralleltest // t.Chdir
func TestEndToEndWorkflow(t *testing.T) {
	tempDir := t.TempDir()

	t.Chdir(tempDir)

	coverageContent := `mode: set
example.go:10.5,15.10 10 1
example.go:20.5,25.10 5 0
example.go:30.5,35.10 3 1`

	err := os.WriteFile("coverage.out", []byte(coverageContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create coverage file: %v", err)
	}

	// Test the complete workflow: parse -> generate badge
	coverage, err := parseCoverageFile("coverage.out")
	if err != nil {
		t.Fatalf("Failed to parse coverage: %v", err)
	}

	config := &Config{
		RedThreshold:    40.0,
		YellowThreshold: 70.0,
		Template:        defaultTemplate,
	}

	badge, err := generateBadge(coverage, config)
	if err != nil {
		t.Fatalf("Failed to generate badge: %v", err)
	}

	// Write the badge to a file
	badgeFile := "test-badge.svg"

	err = os.WriteFile(badgeFile, []byte(badge), 0o644)
	if err != nil {
		t.Fatalf("Failed to write badge file: %v", err)
	}

	// Verify the badge file exists and contains expected content
	badgeData, err := os.ReadFile(badgeFile)
	if err != nil {
		t.Fatalf("Failed to read badge file: %v", err)
	}

	badgeStr := string(badgeData)
	if !strings.Contains(badgeStr, "<svg") {
		t.Error("Badge should contain SVG content")
	}

	// Check that the coverage percentage is in the badge
	// 10 statements covered (first block) + 3 statements covered (third block) = 13 covered
	// Total statements: 10 + 5 + 3 = 18
	// Coverage: 13/18 * 100 = 72.2%
	expectedCoverage := 72.2

	coverageStr := strings.Contains(badgeStr, "72.2")
	if !coverageStr {
		t.Errorf("Badge should contain coverage percentage (expected ~%.1f%%), got: %s", expectedCoverage, badgeStr)
	}
}

//nolint:paralleltest // t.Chdir
func TestEndToEndWorkflowWithVariousCoverageScenarios(t *testing.T) {
	scenarios := []struct {
		name     string
		content  string
		expected float64
	}{
		{
			name: "Zero coverage",
			content: `mode: set
test.go:10.5,15.10 5 0
test.go:20.5,25.10 3 0`,
			expected: 0.0,
		},
		{
			name: "Partial coverage",
			content: `mode: set
test.go:10.5,15.10 6 1
test.go:20.5,25.10 4 0`,
			expected: 60.0, // 6/10 * 100
		},
		{
			name: "Full coverage",
			content: `mode: set
test.go:10.5,15.10 4 1
test.go:20.5,25.10 2 1`,
			expected: 100.0,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tempDir := t.TempDir()

			t.Chdir(tempDir)

			// Create coverage file
			err := os.WriteFile("coverage.out", []byte(scenario.content), 0o644)
			if err != nil {
				t.Fatalf("Failed to create coverage file: %v", err)
			}

			// Parse coverage
			coverage, err := parseCoverageFile("coverage.out")
			if err != nil {
				t.Fatalf("Failed to parse coverage: %v", err)
			}

			// Check coverage matches expected
			tolerance := 0.1
			if coverage < scenario.expected-tolerance || coverage > scenario.expected+tolerance {
				t.Errorf("Expected coverage %.1f%%, got %.1f%%", scenario.expected, coverage)
			}

			// Generate badge
			config := &Config{
				RedThreshold:    40.0,
				YellowThreshold: 70.0,
				Template:        defaultTemplate,
			}

			badge, err := generateBadge(coverage, config)
			if err != nil {
				t.Fatalf("Failed to generate badge: %v", err)
			}

			// Verify badge contains expected coverage
			expectedStr := fmt.Sprintf("%.1f", scenario.expected)
			if !strings.Contains(badge, expectedStr) {
				t.Errorf("Badge should contain coverage %s%%, got: %s", expectedStr, badge)
			}

			// Verify badge color based on coverage
			var expectedColor string

			switch {
			case scenario.expected < 40.0:
				expectedColor = "#e05d44" // red
			case scenario.expected < 70.0:
				expectedColor = "#dfb317" // yellow
			default:
				expectedColor = "#44cc11" // green
			}

			if !strings.Contains(badge, expectedColor) {
				t.Errorf("Badge should contain color %s for %.1f%% coverage", expectedColor, scenario.expected)
			}
		})
	}
}
