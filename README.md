# Slider

**Render slideshow-style documents as web pages or PDFs**

Slider is a command-line tool for creating beautiful, presentation-ready status reports and slide decks. Define your content in JSON, style it with HTML templates, and render it as an interactive web page or print-ready PDF.

## Features

- 📄 **Template-based** - Separate content from presentation with Go html/templates
- 🎨 **Fully customizable** - Complete control over HTML/CSS styling
- 🔄 **Multi-page navigation** - Built-in pagination with forward/back controls
- 📊 **JSON Schema validation** - Catch errors before rendering
- 🖨️ **PDF export** - Generate print-ready PDFs with one command
- 🖼️ **Static assets** - Serve images and other resources
- 🚀 **Zero configuration** - Works out of the box with example templates

## Example

The included example template generates a modern, dark-themed status report:

```bash
slider serve --dir example
```

**Page 1: Lorem Ipsum Dolor**

Features:
- Dark gradient background with decorative pattern overlay
- Glassmorphism card design
- Accent lighting
- Horizontal workstream layout
- Status badge
- Footer with placeholder branding

Perfect for weekly status updates, program reviews, or executive briefings.

## Installation

### Download Binary

**macOS/Linux:**
```bash
# Download the binary
curl -L -o slider https://github.com/your-org/slider/releases/latest/download/slider
chmod +x slider
sudo mv slider /usr/local/bin/
```

**Windows:**
```powershell
# Download slider.exe from releases
# Add to your PATH or run directly
```

### Build from Source

Requirements:
- Go 1.23+
- [statik](https://github.com/rakyll/statik) - `go install github.com/rakyll/statik@latest`

```bash
git clone https://github.com/your-org/slider.git
cd slider
./make.bash build
```

## Quick Start

### 1. Create a New Project

```bash
slider setup myproject
cd myproject
```

This creates the following structure:
```
myproject/
├── deck/              # Your slide deck documents
│   └── default.json   # Default deck
├── template/          # HTML templates
│   ├── status.html
│   └── status.schema.json
├── img/               # Image assets
└── pdf/               # Generated PDFs (auto-created)
```

### 2. Edit Your Content

Edit `deck/default.json` to add your content:

```json
{
  "schema": "status.schema.json",
  "pages": [
    {
      "template": "status.html",
      "data": {
        "programName": "My Project",
        "weekEnding": "December 25, 2024",
        "pageNumber": 1,
        "workstreams": [
          {
            "name": "Development",
            "accomplishments": [
              "Completed feature X",
              "Fixed critical bug in module Y"
            ],
            "nextSteps": [
              "Deploy to staging",
              "Begin QA testing"
            ]
          }
        ]
      }
    }
  ]
}
```

### 3. Preview in Browser

```bash
slider serve --dir myproject
```

Opens an interactive web page at `http://localhost:<random-port>`

### 4. Generate PDF

```bash
slider topdf --dir myproject
```

Creates `myproject/pdf/default.pdf`

## Usage

### Commands

```bash
slider serve [flags]      # Start web server
slider topdf [flags]      # Generate PDF
slider setup [directory]  # Create new project
```

### Global Flags

```bash
--dir string    Data directory (default ".")
--deck string   Document to render (default: default.json, index.json, home.json, root.json)
```

### Examples

```bash
# Serve a specific deck
slider serve --dir myproject --deck quarterly-review.json

# Generate PDF with custom output path
slider topdf --dir myproject --output ~/Desktop/report.pdf

# Create project in specific directory
slider setup ~/Documents/status-reports
```

## Customization

### Templates

Templates use Go's [html/template](https://pkg.go.dev/html/template) syntax:

```html
<h1>{{.ProgramName}}</h1>
<p>Week ending: {{.WeekEnding}}</p>

{{range .Workstreams}}
  <div class="workstream">
    <h2>{{.Name}}</h2>
    <ul>
      {{range .Accomplishments}}
        <li>{{.}}</li>
      {{end}}
    </ul>
  </div>
{{end}}
```

### JSON Schema

Validate your documents with JSON Schema:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["pages"],
  "properties": {
    "pages": {
      "type": "array",
      "items": {
        "required": ["template", "data"],
        "properties": {
          "template": {"type": "string"},
          "data": {"type": "object"}
        }
      }
    }
  }
}
```

### Built-in Template Functions

- `{{add a b}}` - Addition
- `{{sub a b}}` - Subtraction  
- `{{gt a b}}` - Greater than
- `{{lt a b}}` - Less than

## AI Integration

Slider includes AI skill documentation for code assistants. After running `slider setup`, see `AGENTS.md` in your project directory for guidance on:

- Directory structure
- JSON document format
- Template syntax
- Navigation patterns
- PDF generation

Perfect for use with Claude, GitHub Copilot, or other AI coding assistants.

## Building

Install the embedded filesystem generator first:

```bash
go install github.com/rakyll/statik@latest
```

```bash
./make.bash build          # Build for the current platform
./make.bash build-release  # Build release binaries
```

`build-release` creates binaries in `.local/`:
- `slider` - macOS/Linux binary
- `slider.exe` - Windows binary

The older `./make.bash release` command remains available as an alias.

## Release Checklist

Before creating a release:

```bash
go test ./...
./make.bash build-release
```

Verify the binaries in `.local/`, then create a semantic version tag such as
`v1.0.0` and publish the release from that tag.

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome:

1. Fork the repository and create a focused branch.
2. Make the smallest change that solves the problem.
3. Run `go test ./...` and the relevant `make.bash` command locally.
4. If you change files under `example/`, regenerate the embedded filesystem with `statik -src=example -dest=src -f`.
5. Open a pull request against `main` with a concise description and validation notes.
