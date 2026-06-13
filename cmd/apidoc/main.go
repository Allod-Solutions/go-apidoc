package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Allod-Solutions/go-apidoc/internal/parser"
	"github.com/Allod-Solutions/go-apidoc/internal/renderer"
)

const usage = `go-apidoc — static HTML generator for OpenAPI 3.x specs

Usage:
  apidoc [flags] <spec.json|spec.yaml>

Flags:
  -o <file>   Write output to file instead of stdout

Examples:
  apidoc openapi.json > docs.html
  apidoc -o docs/index.html openapi.yaml
`

func main() {
	out := flag.String("o", "", "output file (default: stdout)")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	doc, err := parser.Load(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var w io.Writer = os.Stdout
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}

	if err := renderer.Render(w, doc); err != nil {
		fmt.Fprintf(os.Stderr, "render error: %v\n", err)
		os.Exit(1)
	}
}
