# go-apidoc

Static HTML generator for OpenAPI 3.x specs. No Node, no npm, no build pipeline — a single Go binary that turns a JSON or YAML spec into a fully crawlable, self-contained HTML page.

```
apidoc openapi.json > docs.html
apidoc -o docs/index.html openapi.yaml
```

## Why

Most API doc tools render client-side with JavaScript. That means search engines may not index your docs, and users without JS get nothing. go-apidoc generates real HTML: every endpoint is a crawlable `<h3>`, parameters are tables, schemas are nested lists.

The output is a single `.html` file with all CSS inlined. Drop it anywhere — S3, GitHub Pages, nginx, your Go server's static dir.

## Install

```bash
go install github.com/Allod-Solutions/go-apidoc/cmd/apidoc@latest
```

Or download a binary from [Releases](https://github.com/Allod-Solutions/go-apidoc/releases).

## Usage

```
apidoc [flags] <spec.json|spec.yaml>

Flags:
  -o <file>   Write output to file (default: stdout)
```

### As a library

```go
import (
    "os"
    "github.com/Allod-Solutions/go-apidoc/internal/parser"
    "github.com/Allod-Solutions/go-apidoc/internal/renderer"
)

doc, err := parser.Load("openapi.json")
if err != nil { ... }

err = renderer.Render(os.Stdout, doc)
```

## Features

- **OpenAPI 3.x** — JSON and YAML input
- **Single file output** — all CSS inlined, no external resources
- **Crawlable HTML** — semantic heading hierarchy, no JS required to read content
- **Dark/light mode** — respects `prefers-color-scheme`
- **Fast** — a 12 MB spec renders in under 2 seconds
- **Zero npm** — pure Go, single binary

## Limitations

- OpenAPI 2.x (Swagger) not supported
- `allOf` / `oneOf` / `anyOf` are not fully expanded (shown as-is)
- No "try it out" / live request runner

## License

MIT
