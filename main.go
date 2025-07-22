package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

type config struct {
	TestCommand     string
	OutputFile      string
	Template        string
	RedThreshold    float64
	YellowThreshold float64
	DumpTemplate    bool
	Quiet           bool
	AutoClean       bool
}

//go:embed coverage-template.svg
var defaultTemplate string

//nolint:gochecknoglobals,mnd // ok
var cfg = &config{
	TestCommand:     "go test ./... -coverprofile=coverage.out",
	OutputFile:      "coverage-badge.svg",
	RedThreshold:    50.0,
	YellowThreshold: 80.0,
	AutoClean:       true,
}

func main() {
	flag.Parse()

	if cfg.DumpTemplate {
		fmt.Print(defaultTemplate) //nolint:forbidigo // ok
		return
	}

	if err := runApplication(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func runApplication(config *config) error {
	if err := loadTemplate(config); err != nil {
		return err
	}

	coverage, err := runTestsAndGetCoverage(config.TestCommand, config.AutoClean)
	if err != nil {
		return fmt.Errorf("error getting coverage: %w", err)
	}

	badge, err := generateBadge(coverage, config)
	if err != nil {
		return fmt.Errorf("error generating badge: %w", err)
	}

	if err = writeBadgeFile(config.OutputFile, badge); err != nil {
		return fmt.Errorf("error writing badge file: %w", err)
	}

	if !config.Quiet {
		fmt.Printf("Coverage badge generated: %s (%.1f%% coverage)\n", config.OutputFile, coverage) //nolint:forbidigo // ok
	}

	return nil
}

func loadTemplate(config *config) error {
	if config.Template != "" {
		data, err := os.ReadFile(config.Template)
		if err != nil {
			return fmt.Errorf("could not read template file %s: %w", config.Template, err)
		}

		config.Template = string(data)
	} else {
		config.Template = defaultTemplate
	}

	return nil
}

func writeBadgeFile(filename, content string) error {
	return os.WriteFile(filename, []byte(content), 0o640) //nolint:mnd // ok
}

func runTestsAndGetCoverage(command string, autoClean bool) (_ float64, err error) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return 0, errors.New("empty command")
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

	if autoClean {
		defer func() {
			err = errors.Join(err, os.Remove(coverageFile))
		}()
	}

	return parseCoverageFile(coverageFile)
}

func parseCoverageFile(filename string) (float64, error) {
	file, err := os.Open(filename) //nolint:gosec // it's safe, file is not eval'ed or anything
	if err != nil {
		return 0, fmt.Errorf("could not open coverage file %s: %w", filename, err)
	}
	defer file.Close() //nolint:errcheck // ok

	var totalStatements, coveredStatements int

	scanner := bufio.NewScanner(file)

	// Skip the first line (mode line).
	if scanner.Scan() && !strings.HasPrefix(scanner.Text(), "mode:") {
		return 0, errors.New("invalid coverage file format")
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 { //nolint:mnd // ok
			continue
		}

		// Last two parts are statement count and covered count.
		stmtCount, err1 := strconv.Atoi(parts[len(parts)-2])
		coveredCount, err2 := strconv.Atoi(parts[len(parts)-1])

		if err1 != nil || err2 != nil {
			continue
		}

		totalStatements += stmtCount
		if coveredCount > 0 {
			coveredStatements += stmtCount
		}
	}

	if err = scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading coverage file: %w", err)
	}

	if totalStatements == 0 {
		return 0, nil
	}

	return (float64(coveredStatements) / float64(totalStatements)) * 100, nil //nolint:mnd // ok
}

func generateBadge(coverage float64, config *config) (string, error) {
	color := getColor(coverage, config.RedThreshold, config.YellowThreshold)

	data := struct {
		Coverage string
		Color    string
	}{
		Coverage: fmt.Sprintf("%.1f", coverage),
		Color:    color,
	}

	tmpl, err := template.New("badge").Parse(config.Template)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return buf.String(), nil
}

func getColor(coverage, redThreshold, yellowThreshold float64) string {
	switch {
	case coverage < redThreshold:
		return "#e05d44"
	case coverage < yellowThreshold:
		return "#dfb317"
	default:
		return "#44cc11"
	}
}

func init() {
	flag.StringVar(&cfg.TestCommand, "command", cfg.TestCommand, "Command to run tests and generate coverage")
	flag.StringVar(&cfg.OutputFile, "output", cfg.OutputFile, "Output SVG file path")
	flag.Float64Var(&cfg.RedThreshold, "red", cfg.RedThreshold, "Red threshold (coverage below this is red)")
	flag.Float64Var(&cfg.YellowThreshold, "yellow", cfg.YellowThreshold, "Yellow threshold (coverage below this is yellow)")
	flag.StringVar(&cfg.Template, "template", cfg.Template, "Path to custom SVG template file (optional)")
	flag.BoolVar(&cfg.DumpTemplate, "dump-template", cfg.DumpTemplate, "Dump the default SVG template to stdout and exit")
	flag.BoolVar(&cfg.Quiet, "quiet", false, "Suppress output messages (only errors will be printed)")
	flag.BoolVar(&cfg.AutoClean, "auto-clean", cfg.AutoClean, "Automatically clean up coverage files after generating the badge")
}
