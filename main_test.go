package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigMerge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		base      *config
		other     *config
		expected  *config
		wantError bool
	}{
		{
			name:     "Empty configs",
			base:     &config{},
			other:    &config{},
			expected: &config{},
		},
		{
			name: "Merge non-empty into empty",
			base: &config{},
			other: &config{
				TestCommand: "go test",
				OutputFile:  "badge.svg",
				Quiet:       true,
			},
			expected: &config{
				TestCommand: "go test",
				OutputFile:  "badge.svg",
				Quiet:       true,
			},
		},
		{
			name: "Merge into existing config",
			base: &config{
				TestCommand:  "original",
				OutputFile:   "original.svg",
				Quiet:        false,
				DumpTemplate: false,
			},
			other: &config{
				TestCommand: "go test",
				Quiet:       true,
			},
			expected: &config{
				TestCommand:  "go test",
				OutputFile:   "", // JSON marshal/unmarshal overwrites with zero value
				Quiet:        true,
				DumpTemplate: false, // JSON marshal/unmarshal overwrites with zero value
			},
		},
		{
			name: "Merge levels",
			base: &config{
				Levels: Levels{90.0: "#00ff00"},
			},
			other: &config{
				Levels: Levels{70.0: "#ffff00", 0.0: "#ff0000"},
			},
			expected: &config{
				Levels: Levels{70.0: "#ffff00", 0.0: "#ff0000"},
			},
		},
		{
			name: "JSON roundtrip test",
			base: &config{
				TestCommand:  "go test",
				OutputFile:   "badge.svg",
				Quiet:        true,
				AutoClean:    false,
				DumpTemplate: true,
				Levels:       Levels{90.0: "#00ff00", 0.0: "#ff0000"},
			},
			other: &config{
				TestCommand: "new command",
				Quiet:       false,
				AutoClean:   true,
			},
			expected: &config{
				TestCommand:  "new command",
				OutputFile:   "", // JSON marshal/unmarshal overwrites with zero value
				Quiet:        false,
				AutoClean:    true,
				DumpTemplate: false,    // JSON marshal/unmarshal overwrites with zero value
				Levels:       Levels{}, // JSON marshal/unmarshal overwrites with zero value
			},
		},
		{
			name: "Merge with CoveragePC set",
			base: &config{
				TestCommand: "original",
				OutputFile:  "original.svg",
			},
			other: &config{
				TestCommand: "new command",
				CoveragePC:  func() *float64 { v := 85.5; return &v }(),
			},
			expected: &config{
				TestCommand: "new command",
				OutputFile:  "", // JSON marshal/unmarshal overwrites with zero value
				CoveragePC:  func() *float64 { v := 85.5; return &v }(),
			},
		},
		{
			name: "Merge with nil CoveragePC",
			base: &config{
				TestCommand: "original",
				CoveragePC:  func() *float64 { v := 90.0; return &v }(),
			},
			other: &config{
				TestCommand: "new command",
				CoveragePC:  nil,
			},
			expected: &config{
				TestCommand: "new command",
				OutputFile:  "",                                         // JSON marshal/unmarshal overwrites with zero value
				CoveragePC:  func() *float64 { v := 90.0; return &v }(), // Should keep base value since other is nil
			},
		},
		{
			name: "Merge overwriting CoveragePC",
			base: &config{
				TestCommand: "original",
				CoveragePC:  func() *float64 { v := 90.0; return &v }(),
			},
			other: &config{
				TestCommand: "new command",
				CoveragePC:  func() *float64 { v := 75.2; return &v }(),
			},
			expected: &config{
				TestCommand: "new command",
				OutputFile:  "", // JSON marshal/unmarshal overwrites with zero value
				CoveragePC:  func() *float64 { v := 75.2; return &v }(),
			},
		},
		{
			name: "Merge with invalid levels causing unmarshal error",
			base: &config{
				TestCommand: "original",
			},
			other: &config{
				TestCommand: "new command",
				Levels:      Levels{}, // This will be modified to cause unmarshal error
			},
			wantError: false, // Even invalid levels won't cause JSON marshal/unmarshal to fail
			expected: &config{
				TestCommand: "new command",
				OutputFile:  "",
				Levels:      Levels{},
			},
		},
		{
			name: "Merge with complex levels",
			base: &config{
				TestCommand: "original",
				Levels:      Levels{50.0: "#yellow"},
			},
			other: &config{
				TestCommand: "new command",
				Levels:      Levels{90.0: "#00ff00", 70.0: "#ffff00", 0.0: "#ff0000"},
			},
			expected: &config{
				TestCommand: "new command",
				OutputFile:  "",
				Levels:      Levels{90.0: "#00ff00", 70.0: "#ffff00", 0.0: "#ff0000"},
			},
		},
		{
			name: "Merge preserving base CoveragePC when other is nil",
			base: &config{
				TestCommand: "original",
				CoveragePC:  func() *float64 { v := 88.8; return &v }(),
				Quiet:       true,
			},
			other: &config{
				TestCommand: "updated command",
				Quiet:       false,
				CoveragePC:  nil, // Explicitly nil
			},
			expected: &config{
				TestCommand: "updated command",
				OutputFile:  "",
				Quiet:       false,
				CoveragePC:  func() *float64 { v := 88.8; return &v }(), // Should preserve base value
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.base.merge(tt.other)
			if tt.wantError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expected != nil { //nolint:nestif // ok
				if tt.base.TestCommand != tt.expected.TestCommand {
					t.Errorf("TestCommand = %q, want %q", tt.base.TestCommand, tt.expected.TestCommand)
				}

				if tt.base.OutputFile != tt.expected.OutputFile {
					t.Errorf("OutputFile = %q, want %q", tt.base.OutputFile, tt.expected.OutputFile)
				}

				if tt.base.Quiet != tt.expected.Quiet {
					t.Errorf("Quiet = %v, want %v", tt.base.Quiet, tt.expected.Quiet)
				}

				if tt.base.AutoClean != tt.expected.AutoClean {
					t.Errorf("AutoClean = %v, want %v", tt.base.AutoClean, tt.expected.AutoClean)
				}

				if tt.base.DumpTemplate != tt.expected.DumpTemplate {
					t.Errorf("DumpTemplate = %v, want %v", tt.base.DumpTemplate, tt.expected.DumpTemplate)
				}

				if len(tt.expected.Levels) > 0 && !tt.base.Levels.eq(tt.expected.Levels) {
					t.Errorf("Levels = %v, want %v", tt.base.Levels, tt.expected.Levels)
				}

				if tt.expected.CoveragePC != nil {
					if tt.base.CoveragePC == nil {
						t.Error("Expected CoveragePC to be set but it was nil")
					} else if *tt.base.CoveragePC != *tt.expected.CoveragePC {
						t.Errorf("CoveragePC = %v, want %v", *tt.base.CoveragePC, *tt.expected.CoveragePC)
					}
				}
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		setupConfig       string
		setupDefaultFile  string
		expectError       bool
		expectedCmd       string
		expectedOutput    string
		expectedLevelsLen int
	}{
		{
			name: "Load embedded defaults",
			setupConfig: `{
				"testCommand": "go test ./... -coverprofile=coverage.out",
				"outputFile": "coverage-badge.svg",
				"levels": "70=#44cc11,40=#dfb317,0=#ff0001"
			}`,
			setupDefaultFile:  "stampli.json",
			expectedCmd:       "go test ./... -coverprofile=coverage.out",
			expectedOutput:    "coverage-badge.svg",
			expectedLevelsLen: 4,
		},
		{
			name:             "Invalid JSON in embedded defaults",
			setupConfig:      `{invalid json}`,
			setupDefaultFile: "stampli.json",
			expectError:      true,
		},
		{
			name: "Load with command line flags",
			setupConfig: `{
				"testCommand": "go test ./... -coverprofile=coverage.out",
				"outputFile": "coverage-badge.svg",
				"levels": "70=#44cc11,40=#dfb317,0=#ff0001"
			}`,
			setupDefaultFile:  "stampli.json",
			expectedCmd:       "custom test command",
			expectedOutput:    "custom-badge.svg",
			expectedLevelsLen: 4,
		},
		{
			name: "Config file parsing error",
			setupConfig: `{
				"testCommand": "go test ./... -coverprofile=coverage.out",
				"outputFile": "coverage-badge.svg"
			}`,
			setupDefaultFile: "custom-config.json",
			expectError:      true,
		},
		{
			name: "Load with invalid levels flag",
			setupConfig: `{
				"testCommand": "go test ./... -coverprofile=coverage.out",
				"outputFile": "coverage-badge.svg",
				"levels": "70=#44cc11,40=#dfb317,0=#ff0001"
			}`,
			setupDefaultFile: "stampli.json",
			expectError:      true,
		},
		{
			name: "Load with coverage flag",
			setupConfig: `{
				"testCommand": "go test ./... -coverprofile=coverage.out",
				"outputFile": "coverage-badge.svg"
			}`,
			setupDefaultFile:  "stampli.json",
			expectedCmd:       "go test ./... -coverprofile=coverage.out",
			expectedOutput:    "coverage-badge.svg",
			expectedLevelsLen: 4, // Default levels from embedded config
		},
		{
			name: "Non-existent custom config file",
			setupConfig: `{
				"testCommand": "go test ./... -coverprofile=coverage.out",
				"outputFile": "coverage-badge.svg"
			}`,
			setupDefaultFile:  "missing-config.json",
			expectError:       false, // If default config file doesn't exist, it's ignored
			expectedCmd:       "go test ./... -coverprofile=coverage.out",
			expectedOutput:    "coverage-badge.svg",
			expectedLevelsLen: 0, // No levels when config file is missing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup test environment
			a := app{
				defaultConfig:     tt.setupConfig,
				defaultConfigFile: tt.setupDefaultFile,
			}

			var args []string

			tempDir := t.TempDir()

			// Setup specific test scenarios
			switch tt.name {
			case "Load with command line flags":
				args = []string{
					"-command", "custom test command",
					"-output", "custom-badge.svg",
				}
			case "Load with invalid levels flag":
				args = []string{"-levels", "invalid=format"}
			case "Load with coverage flag":
				args = []string{"-coverage", "85.5"}
			case "Config file parsing error":
				// Create an invalid JSON config file
				configPath := filepath.Join(tempDir, "custom-config.json")

				err := os.WriteFile(configPath, []byte(`{invalid json}`), 0o644)
				if err != nil {
					t.Fatalf("Failed to create test config file: %v", err)
				}

				a.ConfigFile = configPath
				a.defaultConfigFile = configPath
			case "Non-existent custom config file":
				a.ConfigFile = filepath.Join(tempDir, "missing-config.json")
				a.defaultConfigFile = filepath.Join(tempDir, "missing-config.json")
			}

			err := a.loadConfig(flag.NewFlagSet("test", flag.ContinueOnError), args)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !tt.expectError {
				cfg := &a.config

				if tt.expectedCmd != "" && cfg.TestCommand != tt.expectedCmd {
					t.Errorf("TestCommand = %q, want %q", cfg.TestCommand, tt.expectedCmd)
				}

				if tt.expectedOutput != "" && cfg.OutputFile != tt.expectedOutput {
					t.Errorf("OutputFile = %q, want %q", cfg.OutputFile, tt.expectedOutput)
				}

				if tt.expectedLevelsLen > 0 && len(cfg.Levels) != tt.expectedLevelsLen {
					t.Errorf("Levels length = %d, want %d", len(cfg.Levels), tt.expectedLevelsLen)
				}
			}
		})
	}
}

func TestRunApplication(t *testing.T) {
	t.Parallel()

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	tests := []struct {
		name          string
		setupFunc     func(t *testing.T, tempDir string) *config
		expectError   bool
		errorContains string
		validateFunc  func(t *testing.T, tempDir string, config *config)
	}{
		{
			name: "Successful run with valid coverage",
			setupFunc: func(t *testing.T, tempDir string) *config {
				t.Helper()

				coverageFile := filepath.Join(tempDir, "coverage-valid.out")

				sourceFile := filepath.Join(originalWD, "testdata", "coverage-sample.out")
				data, err := os.ReadFile(sourceFile)
				if err != nil {
					t.Fatalf("Failed to read testdata coverage file: %v", err)
				}

				err = os.WriteFile(coverageFile, data, 0o644)
				if err != nil {
					t.Fatalf("Failed to create coverage file: %v", err)
				}

				return &config{
					TestCommand:  "echo 'test completed' -coverprofile=" + coverageFile,
					OutputFile:   filepath.Join(tempDir, "badge.svg"),
					Levels:       Levels{70.0: "#44cc11", 40.0: "#dfb317", 0.0: "#ff0001"},
					Template:     "", // Should use default
					DumpTemplate: false,
					Quiet:        true,
				}
			},
			validateFunc: func(t *testing.T, tempDir string, config *config) {
				t.Helper()

				badgeData, err := os.ReadFile(config.OutputFile)
				if err != nil {
					t.Errorf("Badge file not created: %v", err)
					return
				}

				badgeContent := string(badgeData)
				if !strings.Contains(badgeContent, "<svg") {
					t.Error("Badge doesn't contain SVG content")
				}

				if !strings.Contains(badgeContent, "97.4") {
					t.Error("Badge doesn't contain expected coverage percentage")
				}
			},
		},
		{
			name: "Partial coverage scenario",
			setupFunc: func(t *testing.T, tempDir string) *config {
				t.Helper()

				coverageFile := filepath.Join(tempDir, "partial-test.out")

				// Copy existing testdata coverage file
				sourceFile := filepath.Join(originalWD, "testdata", "coverage-sample.out")
				data, err := os.ReadFile(sourceFile)
				if err != nil {
					t.Fatalf("Failed to read testdata coverage file: %v", err)
				}

				err = os.WriteFile(coverageFile, data, 0o644)
				if err != nil {
					t.Fatalf("Failed to create coverage file: %v", err)
				}

				return &config{
					TestCommand:  "echo 'test' -coverprofile=" + coverageFile,
					OutputFile:   filepath.Join(tempDir, "badge.svg"),
					Levels:       Levels{70.0: "#44cc11", 40.0: "#dfb317", 0.0: "#ff0001"},
					Template:     "", // Use default
					DumpTemplate: false,
					Quiet:        true,
					AutoClean:    false,
				}
			},
			validateFunc: func(t *testing.T, tempDir string, config *config) {
				t.Helper()

				badgeData, err := os.ReadFile(config.OutputFile)
				if err != nil {
					t.Errorf("Badge file not created: %v", err)
					return
				}

				badgeContent := string(badgeData)
				if !strings.Contains(badgeContent, "<svg") {
					t.Error("Badge doesn't contain SVG content")
				}
			},
		},
		{
			name: "Invalid command",
			setupFunc: func(t *testing.T, tempDir string) *config {
				t.Helper()

				return &config{
					TestCommand: "nonexistent-command-12345",
					OutputFile:  filepath.Join(tempDir, "badge.svg"),
					Levels:      Levels{70.0: "#44cc11", 40.0: "#dfb317", 0.0: "#ff0001"},
				}
			},
			expectError:   true,
			errorContains: "error getting coverage",
		},
		{
			name: "Empty command",
			setupFunc: func(t *testing.T, tempDir string) *config {
				t.Helper()

				return &config{
					TestCommand: "",
					OutputFile:  filepath.Join(tempDir, "badge.svg"),
					Levels:      Levels{70.0: "#44cc11"},
				}
			},
			expectError:   true,
			errorContains: "empty command",
		},
		{
			name: "Invalid output directory",
			setupFunc: func(t *testing.T, tempDir string) *config {
				t.Helper()

				coverageFile := filepath.Join(tempDir, "invalid-dir-test.out")

				// Copy existing testdata coverage file
				sourceFile := filepath.Join(originalWD, "testdata", "coverage-sample.out")
				data, err := os.ReadFile(sourceFile)
				if err != nil {
					t.Fatalf("Failed to read testdata coverage file: %v", err)
				}

				err = os.WriteFile(coverageFile, data, 0o644)
				if err != nil {
					t.Fatalf("Failed to create coverage file: %v", err)
				}

				return &config{
					TestCommand: "echo 'test' -coverprofile=" + coverageFile,
					OutputFile:  "/nonexistent/directory/badge.svg",
					Levels:      Levels{70.0: "#44cc11"},
					Quiet:       true,
				}
			},
			expectError:   true,
			errorContains: "error writing badge file",
		},

		{
			name: "Run with direct coverage percentage",
			setupFunc: func(t *testing.T, tempDir string) *config {
				t.Helper()

				coverage := 82.5

				return &config{
					CoveragePC: &coverage,
					OutputFile: filepath.Join(tempDir, "badge.svg"),
					Levels:     Levels{70.0: "#44cc11", 40.0: "#dfb317", 0.0: "#ff0001"},
					Template:   "", // Use default
				}
			},
			validateFunc: func(t *testing.T, tempDir string, config *config) {
				t.Helper()

				badgeData, err := os.ReadFile(config.OutputFile)
				if err != nil {
					t.Errorf("Badge file not created: %v", err)
					return
				}

				badgeContent := string(badgeData)
				if !strings.Contains(badgeContent, "<svg") {
					t.Error("Badge doesn't contain SVG content")
				}

				if !strings.Contains(badgeContent, "82.5") {
					t.Error("Badge doesn't contain expected coverage percentage")
				}
			},
		},
		{
			name: "Template loading error",
			setupFunc: func(t *testing.T, tempDir string) *config {
				t.Helper()

				coverage := 75.0

				return &config{
					CoveragePC: &coverage,
					OutputFile: filepath.Join(tempDir, "badge.svg"),
					Template:   "/nonexistent/template.svg",
					Levels:     Levels{70.0: "#44cc11"},
				}
			},
			expectError:   true,
			errorContains: "could not read template file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()

			a := app{config: *tt.setupFunc(t, tempDir)}
			err := a.run()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expectError && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Error should contain %q, got %q", tt.errorContains, err.Error())
				}
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, tempDir, &a.config)
			}
		})
	}
}

func TestLoadTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		setupFunc        func(t *testing.T, tempDir string) *config
		expectError      bool
		expectedTemplate string
	}{
		{
			name: "Load custom template file",
			setupFunc: func(t *testing.T, tempDir string) *config {
				t.Helper()

				testTemplateFile := filepath.Join(tempDir, "test-template.svg")
				testTemplateContent := `<svg><text>{{.Coverage}}% test</text></svg>`

				err := os.WriteFile(testTemplateFile, []byte(testTemplateContent), 0o644)
				if err != nil {
					t.Fatalf("Failed to create test template: %v", err)
				}

				return &config{Template: testTemplateFile}
			},
			expectedTemplate: `<svg><text>{{.Coverage}}% test</text></svg>`,
		},
		{
			name: "Use default template when no custom template",
			setupFunc: func(t *testing.T, tempDir string) *config {
				t.Helper()

				return &config{Template: ""}
			},
			expectedTemplate: defaultTemplate,
		},
		{
			name: "Error loading non-existent template file",
			setupFunc: func(t *testing.T, tempDir string) *config {
				t.Helper()

				return &config{Template: filepath.Join(tempDir, "nonexistent-template.svg")}
			},
			expectError: true,
		},
		{
			name: "Load template from directory without read permissions",
			setupFunc: func(t *testing.T, tempDir string) *config {
				t.Helper()

				if os.Geteuid() == 0 {
					t.Skip("Skipping permission test when running as root")
				}

				restrictedDir := filepath.Join(tempDir, "restricted")
				err := os.Mkdir(restrictedDir, 0o000)
				if err != nil {
					t.Fatalf("Failed to create restricted directory: %v", err)
				}

				return &config{Template: filepath.Join(restrictedDir, "template.svg")}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			a := app{config: *tt.setupFunc(t, tempDir)}

			err := a.loadTemplate()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !tt.expectError && tt.expectedTemplate != "" {
				if a.Template != tt.expectedTemplate {
					t.Errorf("Template content = %q, want %q", a.Template, tt.expectedTemplate)
				}
			}
		})
	}
}

func TestRunTestsAndGetCoverage(t *testing.T) {
	t.Parallel()

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	tests := []struct {
		name          string
		command       string
		autoClean     bool
		setupFunc     func(t *testing.T, tempDir string) error
		expectError   bool
		expectedRange []float64 // [min, max]
	}{
		{
			name:    "Command with single word",
			command: "echo",
			setupFunc: func(t *testing.T, tempDir string) error {
				t.Helper()

				return os.WriteFile("coverage.out", []byte("mode: set\n"), 0o644)
			},
		},
		{
			name:        "Empty command",
			command:     "",
			expectError: true,
		},
		{
			name:        "Command that fails",
			command:     "false",
			expectError: true,
		},
		{
			name:    "Command with coverage file extraction",
			command: func() string { return "" }(), // Will be set in setupFunc
			setupFunc: func(t *testing.T, tempDir string) error {
				t.Helper()

				coverageFile := filepath.Join(tempDir, "custom.out")

				// Copy existing testdata coverage file
				sourceFile := filepath.Join(originalWD, "testdata", "coverage-sample.out")
				data, err := os.ReadFile(sourceFile)
				if err != nil {
					return fmt.Errorf("failed to read testdata coverage file: %w", err)
				}

				return os.WriteFile(coverageFile, data, 0o644)
			},
			expectedRange: []float64{95.0, 99.0}, // Using testdata coverage which is ~97%
		},
		{
			name:      "Auto clean coverage file",
			command:   func() string { return "" }(), // Will be set in setupFunc
			autoClean: true,
			setupFunc: func(t *testing.T, tempDir string) error {
				t.Helper()

				coverageFile := filepath.Join(tempDir, "coverage.out")

				// Copy existing testdata coverage file
				sourceFile := filepath.Join(originalWD, "testdata", "coverage-sample.out")
				data, err := os.ReadFile(sourceFile)
				if err != nil {
					return fmt.Errorf("failed to read testdata coverage file: %w", err)
				}

				return os.WriteFile(coverageFile, data, 0o644)
			},
		},
		{
			name:    "Integration test with actual coverage parsing",
			command: func() string { return "" }(), // Will be set in setupFunc
			setupFunc: func(t *testing.T, tempDir string) error {
				t.Helper()

				coverageFile := filepath.Join(tempDir, "integration-coverage.out")

				// Copy existing testdata coverage file
				sourceFile := filepath.Join(originalWD, "testdata", "coverage-sample.out")
				data, err := os.ReadFile(sourceFile)
				if err != nil {
					return fmt.Errorf("failed to read testdata coverage file: %w", err)
				}

				return os.WriteFile(coverageFile, data, 0o644)
			},
			expectedRange: []float64{95.0, 99.0},
		},
		{
			name:    "Test with sample coverage file",
			command: func() string { return "" }(), // Will be set in setupFunc
			setupFunc: func(t *testing.T, tempDir string) error {
				t.Helper()

				coverageFile := filepath.Join(tempDir, "coverage.out")

				// Copy from testdata
				sourceFile := filepath.Join(originalWD, "testdata", "coverage-sample.out")
				data, err := os.ReadFile(sourceFile)
				if err != nil {
					return fmt.Errorf("failed to read testdata coverage file: %w", err)
				}

				return os.WriteFile(coverageFile, data, 0o644)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()

			if tt.setupFunc != nil {
				if err := tt.setupFunc(t, tempDir); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			command := tt.command
			if command == "" {
				switch {
				case strings.Contains(tt.name, "coverage file extraction"):
					command = "echo 'tests passed' -coverprofile=" + filepath.Join(tempDir, "custom.out")
				case strings.Contains(tt.name, "Integration test"):
					command = "echo 'tests completed' -coverprofile=" + filepath.Join(tempDir, "integration-coverage.out")
				case strings.Contains(tt.name, "sample coverage file"):
					command = "echo 'sample test run' -coverprofile=" + filepath.Join(tempDir, "coverage.out")
				default:
					command = "echo 'test' -coverprofile=" + filepath.Join(tempDir, "coverage.out")
				}
			}

			a := app{config: config{TestCommand: command, AutoClean: tt.autoClean}}

			coverage, err := a.runTestsAndGetCoverage()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expectedRange != nil {
				if coverage < tt.expectedRange[0] || coverage > tt.expectedRange[1] {
					t.Errorf("Coverage = %.1f, want between %.1f and %.1f",
						coverage, tt.expectedRange[0], tt.expectedRange[1])
				}
			}

			if tt.autoClean && !tt.expectError {
				coverageFile := filepath.Join(tempDir, "coverage.out")
				if _, err := os.Stat(coverageFile); err == nil {
					t.Error("Expected coverage file to be cleaned up, but it still exists")
				}
			}
		})
	}
}

func TestParseCoverageFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		fileContent string
		expected    float64
		shouldError bool
	}{
		{
			name:        "Empty coverage file (mode only)",
			fileContent: `mode: set`,
			expected:    0.0,
		},
		{
			name:        "Invalid file format - no mode line",
			fileContent: "invalid content without mode line",
			shouldError: true,
		},
		{
			name: "Invalid file format - malformed lines",
			fileContent: `mode: set
invalid line format
malformed data`,
			shouldError: true,
		},
		{
			name:        "Non-existent file",
			fileContent: "", // This will be handled by creating a temp file
			shouldError: true,
		},
		{
			name: "Coverage file with missing total line",
			fileContent: `mode: set
github.com/alexaandru/stampli/main.go:55:68,55:73 1 0
github.com/alexaandru/stampli/main.go:76:2,77:16 1 1`,
			shouldError: true,
		},
		{
			name: "Coverage file with malformed total line - wrong field count",
			fileContent: `mode: set
github.com/alexaandru/stampli/main.go:55:68,55:73 1 0
github.com/alexaandru/stampli/main.go:76:2,77:16 1 1
total: statements`,
			shouldError: true,
		},
		{
			name: "Coverage file with invalid percentage format",
			fileContent: `mode: set
github.com/alexaandru/stampli/main.go:55:68,55:73 1 0
github.com/alexaandru/stampli/main.go:76:2,77:16 1 1
total:						(statements)		invalid%`,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				filename string
				err      error
			)

			if tt.name == "Non-existent file" {
				filename = "nonexistent-file.out"
			} else {
				tempDir := t.TempDir()
				filename = filepath.Join(tempDir, "coverage.out")

				err = os.WriteFile(filename, []byte(tt.fileContent), 0o644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			coverage, err := parseCoverageFile(filename)

			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !tt.shouldError {
				if abs(coverage-tt.expected) > 0.1 {
					t.Errorf("Coverage = %.1f, want %.1f", coverage, tt.expected)
				}
			}
		})
	}
}

func TestGenerateBadge(t *testing.T) {
	t.Parallel()

	simpleTemplate := `<svg><text>{{.Coverage}}%</text><rect fill="{{.Color}}"/></svg>`

	tests := []struct {
		name        string
		coverage    float64
		config      *config
		shouldError bool
		contains    []string
	}{
		{
			name:     "High coverage with green color",
			coverage: 85.5,
			config: &config{
				Template: simpleTemplate,
				Levels:   Levels{70.0: "#44cc11", 40.0: "#dfb317", 0.0: "#ff0001"},
			},
			contains: []string{"85.5", "#44cc11"},
		},
		{
			name:     "Medium coverage with yellow color",
			coverage: 55.2,
			config: &config{
				Template: simpleTemplate,
				Levels:   Levels{70.0: "#44cc11", 40.0: "#dfb317", 0.0: "#ff0001"},
			},
			contains: []string{"55.2", "#dfb317"},
		},
		{
			name:     "Low coverage with red color",
			coverage: 25.0,
			config: &config{
				Template: simpleTemplate,
				Levels:   Levels{70.0: "#44cc11", 40.0: "#dfb317", 0.0: "#ff0001"},
			},
			contains: []string{"25.0", "#ff0001"},
		},
		{
			name:     "Zero coverage",
			coverage: 0.0,
			config: &config{
				Template: simpleTemplate,
				Levels:   Levels{70.0: "#44cc11", 40.0: "#dfb317", 0.0: "#ff0001"},
			},
			contains: []string{"0.0", "#ff0001"},
		},
		{
			name:     "Perfect coverage",
			coverage: 100.0,
			config: &config{
				Template: simpleTemplate,
				Levels:   Levels{70.0: "#44cc11", 40.0: "#dfb317", 0.0: "#ff0001"},
			},
			contains: []string{"100.0", "#44cc11"},
		},
		{
			name:     "Custom template with all variables",
			coverage: 75.0,
			config: &config{
				Template: `<svg><text fill="{{.TextColor}}">{{.Coverage}}%</text><rect fill="{{.Color}}"/></svg>`,
				Levels:   Levels{70.0: "#44cc11", 40.0: "#dfb317", 0.0: "#ff0001"},
			},
			contains: []string{"75.0", "#44cc11"},
		},
		{
			name:     "Default template test",
			coverage: 80.0,
			config: &config{
				Template: defaultTemplate,
				Levels:   Levels{70.0: "#44cc11", 40.0: "#dfb317", 0.0: "#ff0001"},
			},
			contains: []string{"80.0"},
		},
		{
			name:     "Template with invalid syntax",
			coverage: 50.0,
			config: &config{
				Template: `<svg><text>{{.Coverage}%</text></svg>`, // Missing closing brace
				Levels:   Levels{70.0: "#44cc11"},
			},
			shouldError: true,
		},
		{
			name:     "Template with undefined variable",
			coverage: 60.0,
			config: &config{
				Template: `<svg><text>{{.UndefinedVar}}%</text></svg>`,
				Levels:   Levels{70.0: "#44cc11"},
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := app{config: *tt.config}
			a.CoveragePC = &tt.coverage

			result, err := a.generateBadge()
			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !tt.shouldError {
				for _, expected := range tt.contains {
					if !strings.Contains(result, expected) {
						t.Errorf("Result should contain %q, got: %s", expected, result)
					}
				}
			}
		})
	}
}

func TestWriteBadgeFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		setupFunc    func(t *testing.T, tempDir string) (string, string)
		expectError  bool
		validateFunc func(t *testing.T, filename, content string)
	}{
		{
			name: "Valid file write",
			setupFunc: func(t *testing.T, tempDir string) (string, string) {
				t.Helper()

				filename := filepath.Join(tempDir, "test-badge.svg")
				content := `<svg><text>75.5%</text></svg>`

				return filename, content
			},
			validateFunc: func(t *testing.T, filename, content string) {
				t.Helper()

				data, err := os.ReadFile(filename)
				if err != nil {
					t.Errorf("Failed to read written file: %v", err)
					return
				}

				if string(data) != content {
					t.Errorf("File content = %q, want %q", string(data), content)
				}

				// Check file permissions
				info, err := os.Stat(filename)
				if err != nil {
					t.Errorf("Failed to stat file: %v", err)
					return
				}

				expectedMode := os.FileMode(0o640)
				if info.Mode().Perm() != expectedMode {
					t.Errorf("File permissions = %v, want %v", info.Mode().Perm(), expectedMode)
				}
			},
		},
		{
			name: "Empty content",
			setupFunc: func(t *testing.T, tempDir string) (string, string) {
				t.Helper()

				filename := filepath.Join(tempDir, "empty-badge.svg")
				content := ""

				return filename, content
			},
			validateFunc: func(t *testing.T, filename, content string) {
				t.Helper()

				data, err := os.ReadFile(filename)
				if err != nil {
					t.Errorf("Failed to read written file: %v", err)
					return
				}

				if string(data) != content {
					t.Errorf("File content = %q, want %q", string(data), content)
				}
			},
		},
		{
			name: "Large content",
			setupFunc: func(t *testing.T, tempDir string) (string, string) {
				t.Helper()

				filename := filepath.Join(tempDir, "large-badge.svg")
				content := strings.Repeat("<svg>test</svg>", 1000)

				return filename, content
			},
			validateFunc: func(t *testing.T, filename, content string) {
				t.Helper()

				data, err := os.ReadFile(filename)
				if err != nil {
					t.Errorf("Failed to read written file: %v", err)
					return
				}

				if string(data) != content {
					t.Error("Large file content doesn't match expected")
				}
			},
		},
		{
			name: "Write to non-existent directory",
			setupFunc: func(t *testing.T, tempDir string) (string, string) {
				t.Helper()

				filename := filepath.Join(tempDir, "nonexistent", "badge.svg")
				content := `<svg><text>test</text></svg>`

				return filename, content
			},
			expectError: true,
		},
		{
			name: "Write to directory without write permissions",
			setupFunc: func(t *testing.T, tempDir string) (string, string) {
				t.Helper()

				if os.Geteuid() == 0 {
					t.Skip("Skipping permission test when running as root")
				}

				readOnlyDir := filepath.Join(tempDir, "readonly")
				err := os.Mkdir(readOnlyDir, 0o755)
				if err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}

				err = os.Chmod(readOnlyDir, 0o555) // Read and execute only
				if err != nil {
					t.Fatalf("Failed to set directory permissions: %v", err)
				}

				// Restore permissions for cleanup
				t.Cleanup(func() {
					os.Chmod(readOnlyDir, 0o755)
				})

				filename := filepath.Join(readOnlyDir, "badge.svg")
				content := `<svg><text>test</text></svg>`

				return filename, content
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			filename, content := tt.setupFunc(t, tempDir)
			a := app{config: config{OutputFile: filename}}

			err := a.writeBadgeFile(content)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, filename, content)
			}
		})
	}
}

// Helper function for floating point comparison
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}

	return x
}

func (l *Levels) eq(other Levels) bool {
	if len(*l) != len(other) {
		return false
	}

	for level, color := range *l {
		if other[level] != color {
			return false
		}
	}

	return true
}

func TestNewApp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		expectError bool
		validate    func(t *testing.T, app app)
	}{
		{
			name: "Valid args with defaults",
			args: []string{},
			validate: func(t *testing.T, app app) {
				t.Helper()

				if app.defaultConfig == "" {
					t.Error("defaultConfig should not be empty")
				}

				if app.defaultConfigFile != defaultConfigFile {
					t.Errorf("defaultConfigFile = %q, want %q", app.defaultConfigFile, defaultConfigFile)
				}

				if app.dumpSink == nil {
					t.Error("dumpSink should not be nil")
				}
			},
		},
		{
			name: "Valid args with flags",
			args: []string{"-quiet", "-output", "test.svg"},
			validate: func(t *testing.T, app app) {
				t.Helper()

				if !app.Quiet {
					t.Error("Quiet should be true")
				}

				if app.OutputFile != "test.svg" {
					t.Errorf("OutputFile = %q, want %q", app.OutputFile, "test.svg")
				}
			},
		},
		{
			name: "Valid coverage flag",
			args: []string{"-coverage", "85.5"},
			validate: func(t *testing.T, app app) {
				t.Helper()

				if app.CoveragePC == nil {
					t.Error("CoveragePC should not be nil")
				} else if *app.CoveragePC != 85.5 {
					t.Errorf("CoveragePC = %f, want %f", *app.CoveragePC, 85.5)
				}
			},
		},
		{
			name:        "Invalid flag",
			args:        []string{"-invalid-flag"},
			expectError: true,
		},
		{
			name:        "Help flag",
			args:        []string{"-help"},
			expectError: true, // flag.Parse will return an error for -help
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			app, err := newApp(fs, tt.args)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.validate != nil && !tt.expectError {
				tt.validate(t, app)
			}
		})
	}
}

func TestAppDumpOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   config
		contains string
	}{
		{
			name: "Dump template",
			config: config{
				DumpTemplate: true,
			},
			contains: "<svg",
		},
		{
			name: "Dump config",
			config: config{
				DumpConfig: true,
			},
			contains: "testCommand",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var output strings.Builder

			a := app{
				config:   tt.config,
				dumpSink: &output,
			}

			err := a.run()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			result := output.String()
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Output should contain %q, got: %q", tt.contains, result)
			}
		})
	}
}
