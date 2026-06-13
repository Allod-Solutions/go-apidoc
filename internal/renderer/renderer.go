// Package renderer generates static HTML from a parsed API doc.
package renderer

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/Allod-Solutions/go-apidoc/internal/parser"
)

//go:embed templates/doc.html
var templateFS embed.FS

var docTemplate = template.Must(
	template.New("doc.html").Funcs(funcMap).ParseFS(templateFS, "templates/doc.html"),
)

var funcMap = template.FuncMap{
	"lower":             strings.ToLower,
	"slugify":           slugify,
	"anchorID":          anchorID,
	"methodColor":       methodColor,
	"methodBg":          methodBg,
	"responseCodeClass": responseCodeClass,
	"schemaType":        schemaType,
	"renderSchema":      renderSchema,
	"firstLine":         firstLine,
}

// Render writes the full HTML document for doc to w.
func Render(w io.Writer, doc *parser.Doc) error {
	return docTemplate.Execute(w, doc)
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func anchorID(method, path string) string {
	return strings.ToLower(method) + "-" + slugify(path)
}

func methodColor(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return "var(--get)"
	case "POST":
		return "var(--post)"
	case "PUT":
		return "var(--put)"
	case "PATCH":
		return "var(--patch)"
	case "DELETE":
		return "var(--delete)"
	default:
		return "var(--head)"
	}
}

func methodBg(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return "color-mix(in srgb, var(--get) 15%, transparent)"
	case "POST":
		return "color-mix(in srgb, var(--post) 15%, transparent)"
	case "PUT":
		return "color-mix(in srgb, var(--put) 15%, transparent)"
	case "PATCH":
		return "color-mix(in srgb, var(--patch) 15%, transparent)"
	case "DELETE":
		return "color-mix(in srgb, var(--delete) 15%, transparent)"
	default:
		return "color-mix(in srgb, var(--head) 15%, transparent)"
	}
}

func responseCodeClass(code string) string {
	if len(code) < 1 {
		return "code-def"
	}
	switch code[0] {
	case '2':
		return "code-2xx"
	case '3':
		return "code-3xx"
	case '4':
		return "code-4xx"
	case '5':
		return "code-5xx"
	default:
		return "code-def"
	}
}

func schemaType(s *parser.Schema) string {
	if s == nil {
		return ""
	}
	if s.Ref != "" {
		return s.Ref
	}
	if s.Type == "array" && s.Items != nil {
		return fmt.Sprintf("%s[]", schemaType(s.Items))
	}
	if s.Format != "" {
		return fmt.Sprintf("%s(%s)", s.Type, s.Format)
	}
	return s.Type
}

func renderSchema(s *parser.Schema) template.HTML {
	if s == nil {
		return ""
	}
	var b strings.Builder
	writeSchema(&b, s, 0)
	return template.HTML(b.String()) //nolint:gosec // template-controlled output
}

func writeSchema(b *strings.Builder, s *parser.Schema, depth int) {
	if s == nil || depth > 6 {
		return
	}
	b.WriteString(`<div class="schema">`)

	// Type line
	typStr := schemaType(s)
	if typStr != "" {
		if s.Ref != "" {
			fmt.Fprintf(b, `<span class="schema-ref">%s</span>`, htmlEscape(typStr))
		} else {
			fmt.Fprintf(b, `<span class="schema-type">%s</span>`, htmlEscape(typStr))
		}
	}
	if s.Description != "" {
		fmt.Fprintf(b, ` <span class="param-desc">— %s</span>`, htmlEscape(s.Description))
	}
	if len(s.Enum) > 0 {
		fmt.Fprintf(b, ` <span class="prop-enum">enum: %s</span>`, htmlEscape(strings.Join(s.Enum, " | ")))
	}

	// Array items
	if s.Type == "array" && s.Items != nil {
		b.WriteString(`<div class="schema-props">`)
		b.WriteString(`<div class="schema-prop"><span class="prop-name">items</span> `)
		writeSchema(b, s.Items, depth+1)
		b.WriteString(`</div></div>`)
	}

	// Object properties
	if len(s.Properties) > 0 {
		b.WriteString(`<div class="schema-props">`)
		for _, prop := range s.Properties {
			b.WriteString(`<div class="schema-prop">`)
			fmt.Fprintf(b, `<span class="prop-name">%s</span>`, htmlEscape(prop.Name))
			if prop.Required {
				b.WriteString(` <span class="prop-required">*</span>`)
			}
			if prop.Schema != nil {
				fmt.Fprintf(b, ` <span class="prop-type">%s</span>`, htmlEscape(schemaType(prop.Schema)))
				if prop.Schema.Description != "" {
					fmt.Fprintf(b, ` <span class="prop-desc">— %s</span>`, htmlEscape(prop.Schema.Description))
				}
				if len(prop.Schema.Enum) > 0 {
					fmt.Fprintf(b, ` <span class="prop-enum">enum: %s</span>`, htmlEscape(strings.Join(prop.Schema.Enum, " | ")))
				}
				// Recurse into nested objects/arrays.
				if len(prop.Schema.Properties) > 0 || (prop.Schema.Type == "array" && prop.Schema.Items != nil) {
					writeSchema(b, prop.Schema, depth+1)
				}
			}
			b.WriteString(`</div>`)
		}
		b.WriteString(`</div>`)
	}

	b.WriteString(`</div>`)
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&#34;")
	return s
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
