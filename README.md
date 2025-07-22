# Stampli (aka the badge Stamper)

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Build and Test](https://github.com/alexaandru/stampli/actions/workflows/ci.yml/badge.svg)](https://github.com/alexaandru/stampli/actions/workflows/ci.yml)
![Coverage](coverage-badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexaandru/stampli)](https://goreportcard.com/report/github.com/alexaandru/stampli)
[![Go Reference](https://pkg.go.dev/badge/github.com/alexaandru/stampli.svg)](https://pkg.go.dev/github.com/alexaandru/stampli)
[![Socket.dev](https://socket.dev/api/badge/go/package/github.com/alexaandru/stampli)](https://socket.dev/go/package/github.com/alexaandru/stampli)

A lightweight, local Go test coverage badge generator that creates SVG badges similar to shields.io.

## Features

- 🚀 **Runs entirely locally** - no external API calls or internet required
- 🧪 **Automatically runs Go tests** and extracts coverage data
- 🎨 **Generates shields.io-style SVG badges** with embedded template
- 🔧 **Fully configurable** test commands, thresholds, and templates
- 📦 **Single binary** with no dependencies - template is embedded
- 🛠️ **Template dumping** - export the default template for customization

## Installation

```bash
go install github.com/alexaandru/stampli@latest # or, even better
go get tool github.com/alexaandru/stampli@latest
```

## Quick Start

Run with defaults (uses `go test ./... -coverprofile=coverage.out`):

```bash
./stampli
```

This generates `coverage-badge.svg` in the current directory.

### Options

- `-command string` - Test command to run (default: `"go test ./... -coverprofile=coverage.out"`)
- `-output string` - Output SVG file path (default: `"coverage-badge.svg"`)
- `-red float` - Red threshold - coverage below this is red (default: `50.0`)
- `-yellow float` - Yellow threshold - coverage below this is yellow (default: `80.0`)
- `-template string` - Path to custom SVG template file (optional)
- `-dump-template` - Dump the default SVG template to stdout and exit

## Examples

### Basic Usage

```bash
./stampli
# Generates coverage-badge.svg with default settings
```

### Custom Test Command

Use `make test` instead of the default Go command:

```bash
./stampli -command "make test"
```

### Custom Output Location

Save badge to a specific directory:

```bash
./stampli -output "docs/coverage-badge.svg"
```

### Strict Coverage Thresholds

Set higher standards (red < 70%, yellow < 90%):

```bash
./stampli -red 70 -yellow 90
```

### Working with Templates

**Export the default template for customization:**

```bash
./stampli -dump-template > my-template.svg
```

**Edit the template and use it:**

```bash
# Edit my-template.svg with your favorite editor
./stampli -template my-template.svg
```

**Save template to a specific location:**

```bash
./stampli -dump-template > templates/coverage-badge.svg
./stampli -template templates/coverage-badge.svg
```

## Template Customization

Stampli uses Go's `text/template` package. Your template receives:

- `{{.Coverage}}` - Coverage percentage as string (e.g., "85.4")
- `{{.Color}}` - Color hex code based on thresholds (#e05d44, #dfb317, or #4c1)

### Customization Workflow

1. **Export the default template:**

   ```bash
   ./stampli -dump-template > custom-badge.svg
   ```

2. **Edit the template:**

   ```bash
   # Modify custom-badge.svg with your preferred editor
   # Change colors, fonts, dimensions, layout, etc.
   ```

3. **Use your custom template:**
   ```bash
   ./stampli -template custom-badge.svg
   ```

### Example Custom Template

Here's a minimal custom template:

```svg
<svg xmlns="http://www.w3.org/2000/svg" width="120" height="24">
  <rect width="70" height="24" fill="#555"/>
  <rect x="70" width="50" height="24" fill="{{.Color}}"/>
  <text x="35" y="17" fill="white" text-anchor="middle" font-family="Arial" font-size="12">tests</text>
  <text x="95" y="17" fill="white" text-anchor="middle" font-family="Arial" font-size="12">{{.Coverage}}%</text>
</svg>
```

## Color Scheme

- 🔴 **Red** (#e05d44): Coverage below red threshold
- 🟡 **Yellow** (#dfb317): Coverage between red and yellow thresholds
- 🟢 **Green** (#4c1): Coverage above yellow threshold

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
coverage-badge:
	./stampli -command "make test-coverage"

test-coverage:
	go test ./... -coverprofile=coverage.out

# Export template for customization
template:
	./stampli -dump-template > badge-template.svg
```

### Pre-commit Hook

```bash
#!/bin/sh
./stampli
git add coverage-badge.svg
```

## License

MIT License - see LICENSE file for details.
