# Slider AI Skill

## Overview
Slider is a web server for rendering slideshow-style HTML documents from JSON data files. It supports templates, schemas, multi-page navigation, and PDF export.

## Directory Structure
A slider data directory should contain:

```
data-dir/
├── template/           # HTML templates and JSON schemas
│   ├── *.html         # Go html/template files
│   └── *.schema.json  # JSON Schema files for validation
├── deck/              # Document JSON files
│   ├── default.json   # Default document (or index.json, home.json, root.json)
│   └── *.json         # Other documents
├── img/               # Image assets referenced by templates
│   └── *.png|jpg|svg
└── pdf/               # Generated PDF output (auto-created)
    └── *.pdf
```

## Document JSON Structure
Documents reference a schema and contain an array of pages:

```json
{
  "schema": "status.schema.json",
  "pages": [
    {
      "template": "status.html",
      "data": {
        "programName": "Project Name",
        "weekEnding": "August 8, 2026",
        "pageNumber": 1,
        "workstreams": [
          {
            "name": "Workstream Name",
            "accomplishments": ["Item 1", "Item 2"],
            "nextSteps": ["Next 1", "Next 2"]
          }
        ]
      }
    }
  ]
}
```

## JSON Schema
Schemas validate document structure. Place them in `template/` directory:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Document Title",
  "type": "object",
  "required": ["pages"],
  "properties": {
    "pages": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["template", "data"],
        "properties": {
          "template": {"type": "string"},
          "data": {
            "type": "object",
            "required": ["field1", "field2"],
            "properties": {
              "field1": {"type": "string"},
              "field2": {"type": "integer"}
            }
          }
        }
      }
    }
  }
}
```

## HTML Templates
Templates use Go html/template syntax. Key features:

- `{{.FieldName}}` - Access struct fields (capitalized)
- `{{range .Items}} ... {{end}}` - Iterate over arrays
- `{{if condition}} ... {{end}}` - Conditionals
- Available functions: `add`, `sub`, `gt`, `lt`

Templates automatically receive:
- `TotalPages` - Total number of pages in document
- `PageNumber` - Current page number (1-based)

Navigation example:
```html
{{if gt .PageNumber 1}}
<a href="/page/{{sub .PageNumber 1}}">←</a>
{{end}}
<span>Page {{.PageNumber}}</span>
{{if lt .PageNumber .TotalPages}}
<a href="/page/{{add .PageNumber 1}}">→</a>
{{end}}
```

## Usage
```bash
slider [command]

Commands:
  serve            Start web server to render documents
  topdf            Render document to PDF file
  setup            Create example slider project structure

Global Flags:
  --dir string     Data directory (default ".")
  --deck string     Document to render (default: default.json, index.json, home.json, root.json)

Serve Flags:
  (uses global flags only)

Topdf Flags:
  --output string  Output PDF path (defaults to pdf/ folder)
```

Examples:
```bash
# Start web server with default document
slider serve --dir mydata

# Start web server with specific document
slider serve --dir mydata --deck myreport.json

# Generate PDF in pdf/ folder (uses same document defaults)
slider topdf --dir mydata

# Generate PDF of specific document
slider topdf --dir mydata --deck myreport.json

# Generate PDF with custom path
slider topdf --dir mydata --output /path/to/output.pdf

# Create new project
slider setup myproject
```

## Routes
- `/` - First page of document
- `/page/:num` - Specific page (1-based)

## Validation
On startup, slider validates the document against the specified schema. Validation errors show:
- Field path
- Expected vs actual type/value
- Missing required fields

## Example Workflow
1. Run `slider setup myproject` to create a new project structure
2. Write HTML template in `template/`
3. Create JSON schema matching template data requirements
4. Create document JSON in `deck/` referencing the schema
5. Run `slider serve --dir mydata` or `slider topdf --dir mydata`
6. For web mode, navigate to printed URL; for PDF mode, check `pdf/` folder

## Tips
- Use landscape layout (11in x 8.5in) for presentation-style documents
- Keep templates self-contained with embedded CSS
- Use `@media print` for print-specific styles
- Validate JSON structure early with `--deck` flag
- Page numbers should be sequential starting from 1
