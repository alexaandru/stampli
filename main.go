package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

//nolint:govet,recvcheck // ok
type app struct {
	config

	defaultConfig     string
	defaultConfigFile string
	dumpSink          io.Writer
}

type config struct {
	Levels       Levels   `json:"levels,omitzero"`
	CoveragePC   *float64 `json:"-"`
	TestCommand  string   `json:"testCommand"`
	OutputFile   string   `json:"outputFile"`
	ConfigFile   string   `json:"-"`
	Template     string   `json:"template"`
	DumpTemplate bool     `json:"dumpTemplate"`
	DumpConfig   bool     `json:"dumpConfig"`
	Quiet        bool     `json:"quiet"`
	AutoClean    bool     `json:"autoClean"`
}

const defaultConfigFile = "stampli.json"

//go:embed coverage-badge.tmpl
var defaultTemplate string

//go:embed stampli.json
var defaultConfig string

var (
	errEmptyCommand      = errors.New("empty command")
	errInvalidFileFormat = errors.New("invalid coverage file format")
)

func main() {
	a, err := newApp(flag.CommandLine, os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if err = a.run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func newApp(fs *flag.FlagSet, args []string) (a app, err error) {
	a.defaultConfig = defaultConfig
	a.defaultConfigFile = defaultConfigFile
	a.dumpSink = os.Stdout
	err = a.loadConfig(fs, args)

	return
}

func (c *config) merge(other *config) (err error) {
	js, err := json.Marshal(other)
	if err != nil {
		return fmt.Errorf("failed to marshal other config: %w", err)
	}

	if err = json.Unmarshal(js, c); err != nil {
		return fmt.Errorf("failed to unmarshal into current config: %w", err)
	}

	if other.CoveragePC != nil {
		c.CoveragePC = other.CoveragePC
	}

	return
}

func (c *config) loadTemplate() error {
	if c.Template != "" {
		data, err := os.ReadFile(c.Template)
		if err != nil {
			return fmt.Errorf("could not read template file %s: %w", c.Template, err)
		}

		c.Template = string(data)
	} else {
		c.Template = defaultTemplate
	}

	return nil
}

func (a *app) loadConfig(fs *flag.FlagSet, args []string) error {
	cfg := &a.config

	// Start with embedded defaults.
	if err := json.Unmarshal([]byte(a.defaultConfig), cfg); err != nil {
		return fmt.Errorf("failed to load embedded defaults: %w", err)
	}

	cfg2 := &config{}

	fs.StringVar(&cfg2.TestCommand, "command", cfg.TestCommand, "Command to run tests and generate coverage")
	fs.StringVar(&cfg2.OutputFile, "output", cfg.OutputFile, "Output SVG file path")
	fs.StringVar(&cfg.ConfigFile, "config", a.defaultConfigFile, "Path to JSON configuration file")
	fs.StringVar(&cfg2.Template, "template", cfg.Template, "Path to custom SVG template file (optional)")
	fs.Var(&cfg2.Levels, "levels", fmt.Sprintf("Coverage levels and colors (default %q)", cfg.Levels.String()))
	fs.BoolVar(&cfg2.DumpTemplate, "dump-template", cfg.DumpTemplate, "Dump the default SVG template to stdout and exit")
	fs.BoolVar(&cfg2.DumpConfig, "dump-config", cfg.DumpConfig, "Dump the default configuration to stdout and exit")
	fs.BoolVar(&cfg2.Quiet, "quiet", cfg.Quiet, "Suppress output messages (only errors will be printed)")
	fs.BoolVar(&cfg2.AutoClean, "auto-clean", cfg.AutoClean, "Automatically clean up coverage files after generating the badge")

	var (
		coverageFlag float64
		coverageSet  bool
	)

	fs.Func("coverage", "Coverage percentage to use directly (skips running tests)", func(s string) error {
		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err //nolint:wrapcheck // ok
		}

		coverageFlag = val
		coverageSet = true

		return nil
	})

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	if coverageSet {
		cfg2.CoveragePC = &coverageFlag
	}

	if cfg.ConfigFile != "" {
		if data, err := os.ReadFile(cfg.ConfigFile); err == nil {
			if err = json.Unmarshal(data, cfg); err != nil {
				return fmt.Errorf("failed to parse config file %s: %w", cfg.ConfigFile, err)
			}
		} else if cfg.ConfigFile != a.defaultConfigFile {
			// Otherwise, if a non-default config file was requested, it must exist.
			return fmt.Errorf("failed to load config file %s: %w", cfg.ConfigFile, err)
		}
	}

	return cfg.merge(cfg2)
}

func (a app) run() (err error) {
	if a.DumpTemplate {
		fmt.Fprint(a.dumpSink, defaultTemplate) //nolint:errcheck // ok
		return
	}

	if a.DumpConfig {
		fmt.Fprint(a.dumpSink, defaultConfig) //nolint:errcheck // ok
		return
	}

	if err = a.loadTemplate(); err != nil {
		return
	}

	if a.CoveragePC == nil {
		var coverage float64

		coverage, err = a.runTestsAndGetCoverage()
		if err != nil {
			return fmt.Errorf("error getting coverage: %w", err)
		}

		a.CoveragePC = &coverage
	}

	badge, err := a.generateBadge()
	if err != nil {
		return fmt.Errorf("error generating badge: %w", err)
	}

	if err = a.writeBadgeFile(badge); err != nil {
		return fmt.Errorf("error writing badge file: %w", err)
	}

	if !a.Quiet {
		fmt.Printf("Coverage badge generated: %s (%.1f%% coverage)\n", a.OutputFile, *a.CoveragePC) //nolint:forbidigo // ok
	}

	return
}

func (a app) runTestsAndGetCoverage() (_ float64, err error) {
	command := a.TestCommand

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return 0, errEmptyCommand
	}

	cmd := exec.Command(parts[0], parts[1:]...) //nolint:noctx,gosec // yes, we actually do want end users to be able to drive this

	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("test command failed: %w\nOutput: %s", err, output)
	}

	coverageFile := "coverage.out"

	if strings.Contains(command, "-coverprofile=") {
		re := regexp.MustCompile(`-coverprofile=(\S+)`)
		if matches := re.FindStringSubmatch(command); len(matches) > 1 {
			coverageFile = matches[1]
		}
	}

	if a.AutoClean {
		defer func() {
			err = errors.Join(err, os.Remove(coverageFile))
		}()
	}

	return parseCoverageFile(coverageFile)
}

func parseCoverageFile(filename string) (cov float64, err error) {
	cmd := exec.Command("go", "tool", "cover", "-func="+filename) //nolint:gosec,noctx // safe

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errInvalidFileFormat, err)
	}

	var lastLine string

	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		lastLine = scanner.Text()
	}

	if err = scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading coverage output: %w", err)
	}

	if !strings.HasPrefix(lastLine, "total:") {
		return 0, fmt.Errorf("%w: total line missing", errInvalidFileFormat)
	}

	// Extract percentage from line like "total:						(statements)		87.4%".
	parts := strings.Fields(lastLine)
	if x := len(parts); x != 3 { //nolint:mnd // ok
		return 0, fmt.Errorf("%w: last line parts count != %d", errInvalidFileFormat, x)
	}

	covStr := strings.TrimSuffix(parts[2], "%")

	cov, err = strconv.ParseFloat(covStr, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse coverage percentage: %w", err)
	}

	return
}

func (a app) generateBadge() (string, error) {
	color := a.Levels.GetColorForCoverage(*a.CoveragePC)
	textColor := getOptimalTextColor(color)

	data := struct {
		Coverage  string
		Color     string
		TextColor string
	}{
		Coverage:  fmt.Sprintf("%.1f", *a.CoveragePC),
		Color:     color,
		TextColor: textColor,
	}

	tmpl, err := template.New("badge").Parse(a.Template)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return buf.String(), nil
}

func (a app) writeBadgeFile(content string) error {
	return os.WriteFile(a.OutputFile, []byte(content), 0o640) //nolint:wrapcheck,mnd // ok
}
