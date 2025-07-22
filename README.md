# Stampli (aka the badge Stamper)

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Build and Test](https://github.com/alexaandru/stampli/actions/workflows/ci.yml/badge.svg)](https://github.com/alexaandru/stampli/actions/workflows/ci.yml)
![Coverage](coverage-badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexaandru/stampli)](https://goreportcard.com/report/github.com/alexaandru/stampli)
[![Go Reference](https://pkg.go.dev/badge/github.com/alexaandru/stampli.svg)](https://pkg.go.dev/github.com/alexaandru/stampli)
[![Socket.dev](https://socket.dev/api/badge/go/package/github.com/alexaandru/stampli)](https://socket.dev/go/package/github.com/alexaandru/stampli)

A lightweight, local Go test coverage badge generator that creates SVG badges similar to shields.io.

## Features

- 🚀 **Runs entirely locally** - no external API calls or Internet required
- 🧪 **Automatically runs Go tests** and extracts coverage data, if needed
- 📊 **Generates coverage badges** based on test provided coverage % if you already have it
- 🎨 **Generates shields.io-style SVG badges** with embedded template
- 🔧 **Fully configurable** test commands, thresholds, and templates
- 📦 **Single binary** with no dependencies - template is embedded
- 🛠️ **Template/Config dumping** - export the default template and config for customization

Default Color Scheme & Levels:

![Excellent](testdata/badge-excellent.svg)
![Fair](testdata/badge-fair.svg)
![Good](testdata/badge-good.svg)
![Poor](testdata/badge-poor.svg)

## Installation

```bash
go install github.com/alexaandru/stampli@latest # or, even better
go get tool github.com/alexaandru/stampli@latest
```

## Quick Start

Run with defaults (uses `go test ./... -coverprofile=coverage.out`):

```bash
./stampli # -h for help
```

This generates `coverage-badge.svg` in the current directory by running
the tests and extracting coverage data.

Alternatively, you can provide the coverage percentage directly:

```bash
./stampli -coverage 85.4
```

See help for more options, including customizing the badge SVG template,
the command used for running tests (i.e. replace it with `make test`, etc.)
the levels or the default config, etc.

### Coverage Levels System

The `Levels` system allows fine-grained control over thresholds and colors,
either via cli flag or via JSON config file:

```bash
# Format: level=color,level=color,...
./stampli -levels "95=#00cc00,85=#44cc11,70=#dfb317,50=#ff8c00,=#e05d44"
```

The levels **MUST** include a default level (i.e. `0=#...` or `=#...`).

### SVG Template Customization

Stampli uses Go's `text/template` package. Your template receives:

- `{{ .Coverage }}` - Coverage percentage as string (e.g., "85.4")
- `{{ .Color }}` - Color hex code based on coverage levels
- `{{ .TextColor }}` - Optimal text color, #ffffff or #000000 depending
  on the background.

## Integration Examples

### GitHub Actions

```yaml
- name: Generate coverage badge
  run: |
    ./stampli -output docs/coverage.svg
    git add docs/coverage.svg
    git commit -m "Update coverage badge" || exit 0
```

### Make Integration

```makefile
test:
	go test ./... -coverprofile=coverage.out

badge:
	./stampli -command "make test" -quiet

# Generate badge with custom levels
badge-strict:
	./stampli -levels "95=#00ff00,80=#ffff00,60=#ff8000,0=#ff0000"
```

### Pre-commit Hook

```bash
#!/bin/sh
./stampli
git add coverage-badge.svg
```

## License

[MIT](LICENSE)
