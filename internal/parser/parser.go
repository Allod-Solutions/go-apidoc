// Package parser loads an OpenAPI 3.x spec and extracts a flat representation
// suited for static HTML rendering.
package parser

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Doc is the parsed representation of an OpenAPI spec, ready for rendering.
type Doc struct {
	Title       string
	Version     string
	Description string
	Servers     []string
	Tags        []TagGroup
}

// TagGroup groups endpoints under a common tag (or "default" if untagged).
type TagGroup struct {
	Name        string
	Description string
	Endpoints   []Endpoint
}

// Endpoint is a single HTTP operation.
type Endpoint struct {
	Method      string // uppercase: GET, POST, …
	Path        string
	Summary     string
	Description string
	Deprecated  bool
	Parameters  []Parameter
	RequestBody *Body
	Responses   []Response
}

// Parameter is a path/query/header/cookie parameter.
type Parameter struct {
	Name        string
	In          string // path, query, header, cookie
	Required    bool
	Description string
	Schema      *Schema
}

// Body is a request or response body.
type Body struct {
	Description string
	Required    bool
	Content     []MediaType
}

// Response is one HTTP response definition.
type Response struct {
	Code        string
	Description string
	Content     []MediaType
}

// MediaType is one content-type entry inside a body.
type MediaType struct {
	Type   string  // e.g. "application/json"
	Schema *Schema
}

// Schema is a simplified, recursion-resolved schema node for display.
type Schema struct {
	Type        string
	Format      string
	Description string
	Enum        []string
	Properties  []Property
	Items       *Schema // for array types
	Ref         string  // original $ref name if this was a reference
}

// Property is one named field inside an object schema.
type Property struct {
	Name     string
	Required bool
	Schema   *Schema
}

// Load reads an OpenAPI 3.x document from path (JSON or YAML).
func Load(path string) (*Doc, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	spec, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("load spec: %w", err)
	}
	if err := spec.Validate(context.Background()); err != nil {
		// Non-fatal: log but continue — many real-world specs have minor issues.
		_ = err
	}
	return convert(spec), nil
}

// LoadBytes parses an OpenAPI 3.x document from in-memory bytes (JSON or YAML).
func LoadBytes(data []byte) (*Doc, error) {
	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("parse spec: %w", err)
	}
	return convert(spec), nil
}

func convert(spec *openapi3.T) *Doc {
	doc := &Doc{}
	if spec.Info != nil {
		doc.Title = spec.Info.Title
		doc.Version = spec.Info.Version
		doc.Description = spec.Info.Description
	}
	for _, s := range spec.Servers {
		doc.Servers = append(doc.Servers, s.URL)
	}

	// Build tag descriptions from the top-level tags list.
	tagDesc := make(map[string]string)
	for _, t := range spec.Tags {
		tagDesc[t.Name] = t.Description
	}

	// Collect endpoints grouped by tag, preserving path order.
	groups := make(map[string]*TagGroup)
	var order []string // insertion-order tag names

	paths := spec.Paths.InMatchingOrder()
	for _, path := range paths {
		item := spec.Paths.Find(path)
		if item == nil {
			continue
		}
		for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"} {
			op := item.GetOperation(method)
			if op == nil {
				continue
			}
			ep := convertOp(method, path, op)
			tags := op.Tags
			if len(tags) == 0 {
				tags = []string{"default"}
			}
			for _, tag := range tags {
				if _, ok := groups[tag]; !ok {
					groups[tag] = &TagGroup{Name: tag, Description: tagDesc[tag]}
					order = append(order, tag)
				}
				groups[tag].Endpoints = append(groups[tag].Endpoints, ep)
			}
		}
	}

	// If the spec defines a top-level tags array, use that order.
	// Tags not listed there keep their insertion order at the end.
	if len(spec.Tags) > 0 {
		specOrder := make(map[string]int, len(spec.Tags))
		for i, t := range spec.Tags {
			specOrder[t.Name] = i
		}
		sort.SliceStable(order, func(i, j int) bool {
			oi, oki := specOrder[order[i]]
			oj, okj := specOrder[order[j]]
			if oki && okj {
				return oi < oj
			}
			return oki
		})
	}

	for _, name := range order {
		doc.Tags = append(doc.Tags, *groups[name])
	}
	return doc
}

func convertOp(method, path string, op *openapi3.Operation) Endpoint {
	ep := Endpoint{
		Method:      method,
		Path:        path,
		Summary:     op.Summary,
		Description: op.Description,
		Deprecated:  op.Deprecated,
	}
	for _, ref := range op.Parameters {
		if ref.Value == nil {
			continue
		}
		p := ref.Value
		param := Parameter{
			Name:        p.Name,
			In:          p.In,
			Required:    p.Required,
			Description: p.Description,
		}
		if p.Schema != nil {
			param.Schema = convertSchema(p.Schema, 0)
		}
		ep.Parameters = append(ep.Parameters, param)
	}
	sort.Slice(ep.Parameters, func(i, j int) bool {
		if ep.Parameters[i].In != ep.Parameters[j].In {
			order := map[string]int{"path": 0, "query": 1, "header": 2, "cookie": 3}
			return order[ep.Parameters[i].In] < order[ep.Parameters[j].In]
		}
		return ep.Parameters[i].Name < ep.Parameters[j].Name
	})
	if op.RequestBody != nil && op.RequestBody.Value != nil {
		ep.RequestBody = convertBody(op.RequestBody.Value)
	}
	for code, ref := range op.Responses.Map() {
		if ref.Value == nil {
			continue
		}
		r := Response{Code: code}
		if ref.Value.Description != nil {
			r.Description = *ref.Value.Description
		}
		for ct, mt := range ref.Value.Content {
			m := MediaType{Type: ct}
			if mt.Schema != nil {
				m.Schema = convertSchema(mt.Schema, 0)
			}
			r.Content = append(r.Content, m)
		}
		sort.Slice(r.Content, func(i, j int) bool { return r.Content[i].Type < r.Content[j].Type })
		ep.Responses = append(ep.Responses, r)
	}
	sort.Slice(ep.Responses, func(i, j int) bool { return ep.Responses[i].Code < ep.Responses[j].Code })
	return ep
}

func convertBody(b *openapi3.RequestBody) *Body {
	body := &Body{Description: b.Description, Required: b.Required}
	for ct, mt := range b.Content {
		m := MediaType{Type: ct}
		if mt.Schema != nil {
			m.Schema = convertSchema(mt.Schema, 0)
		}
		body.Content = append(body.Content, m)
	}
	sort.Slice(body.Content, func(i, j int) bool { return body.Content[i].Type < body.Content[j].Type })
	return body
}

const maxDepth = 5 // prevent infinite recursion on circular refs

func convertSchema(ref *openapi3.SchemaRef, depth int) *Schema {
	if ref == nil || depth > maxDepth {
		return nil
	}
	s := &Schema{}
	if ref.Ref != "" {
		// Extract the component name from the $ref path.
		parts := strings.Split(ref.Ref, "/")
		s.Ref = parts[len(parts)-1]
	}
	if ref.Value == nil {
		return s
	}
	v := ref.Value
	if v.Type != nil {
		s.Type = strings.Join(v.Type.Slice(), "|")
	}
	if s.Type == "" && len(v.Properties) > 0 {
		s.Type = "object"
	}
	s.Format = v.Format
	s.Description = v.Description
	for _, e := range v.Enum {
		s.Enum = append(s.Enum, fmt.Sprintf("%v", e))
	}
	// Required set for properties.
	required := make(map[string]bool)
	for _, r := range v.Required {
		required[r] = true
	}
	// Properties (object).
	propNames := make([]string, 0, len(v.Properties))
	for name := range v.Properties {
		propNames = append(propNames, name)
	}
	sort.Strings(propNames)
	for _, name := range propNames {
		propRef := v.Properties[name]
		s.Properties = append(s.Properties, Property{
			Name:     name,
			Required: required[name],
			Schema:   convertSchema(propRef, depth+1),
		})
	}
	// Items (array).
	if v.Items != nil {
		s.Items = convertSchema(v.Items, depth+1)
	}
	return s
}
