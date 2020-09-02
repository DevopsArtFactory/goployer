/*
copyright 2020 the Goployer authors

licensed under the apache license, version 2.0 (the "license");
you may not use this file except in compliance with the license.
you may obtain a copy of the license at

    http://www.apache.org/licenses/license-2.0

unless required by applicable law or agreed to in writing, software
distributed under the license is distributed on an "as is" basis,
without warranties or conditions of any kind, either express or implied.
see the license for the specific language governing permissions and
limitations under the license.
*/

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	blackfriday "github.com/russross/blackfriday/v2"

	"github.com/DevopsArtFactory/goployer/pkg/constants"
)

const (
	defPrefix = "#/definitions/"
)

var (
	regexpDefaults = regexp.MustCompile("(.*)Defaults to `(.*)`")
	regexpExample  = regexp.MustCompile("(.*)For example: `(.*)`")
	pTags          = regexp.MustCompile("(<p>)|(</p>)")

	// patterns for enum-type values
	enumValuePattern     = "^[ \t]*`(?P<name>[^`]+)`([ \t]*\\(default\\))?: .*$"
	regexpEnumDefinition = regexp.MustCompile("(?m).*Valid [a-z]+ are((\\n" + enumValuePattern + ")*)")
	regexpEnumValues     = regexp.MustCompile("(?m)" + enumValuePattern)
)

type Generator struct {
	strict bool
}

type Schema struct {
	*Definition
	Definitions map[string]*Definition `json:"definitions,omitempty"`
}

type Definition struct {
	Ref                  string                 `json:"$ref,omitempty"`
	Items                *Definition            `json:"items,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Properties           map[string]*Definition `json:"properties,omitempty"`
	AdditionalProperties interface{}            `json:"additionalProperties,omitempty"`
	PreferredOrder       []string               `json:"preferredOrder,omitempty"`
	AnyOf                []*Definition          `json:"anyOf,omitempty"`
	Type                 string                 `json:"type,omitempty"`
	Description          string                 `json:"description,omitempty"`
	HTMLDescription      string                 `json:"x-intellij-html-description,omitempty"`
	Default              interface{}            `json:"default,omitempty"`
	Examples             []string               `json:"examples,omitempty"`
	Enum                 []string               `json:"enum,omitempty"`

	inlines []*Definition
	tags    string
}

func main() {
	if err := generateSchemas(".", false, "config", "schema"); err != nil {
		fmt.Println(err.Error())
	}

	if err := generateSchemas(".", false, "metric_config", "metric"); err != nil {
		fmt.Println(err.Error())
	}
}

func generateSchemas(root string, dryRun bool, inputFile, outputFile string) error {
	input := filepath.Join(root, "pkg", "schemas", inputFile+".go")
	output := filepath.Join(root, "docs", "content", "en", "schemas", outputFile+".json")

	generator := Generator{}

	buf, err := generator.Apply(input)
	if err != nil {
		return err
	}

	var current []byte
	if _, err := os.Stat(output); err == nil {
		var err error
		current, err = ioutil.ReadFile(output)
		if err != nil {
			return fmt.Errorf("unable to read existing config schema: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("unable to check that file exists %q: %w", output, err)
	}

	current = bytes.Replace(current, []byte("\r\n"), []byte("\n"), -1)

	if !dryRun {
		if err := ioutil.WriteFile(output, buf, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write schema %q: %w", output, err)
		}
	}

	same := string(current) == string(buf)
	if same {
		return nil
	}

	return nil
}

func (g Generator) Apply(input string) ([]byte, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, input, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var preferredOrder []string
	definitions := make(map[string]*Definition)

	for _, i := range node.Decls {
		declaration, ok := i.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range declaration.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			comment := declaration.Doc.Text()
			if len(comment) == 0 {
				continue
			}
			name := ts.Name.Name
			preferredOrder = append(preferredOrder, name)
			definitions[name] = g.ParseDefinition(name, ts.Type, comment)
		}
	}

	var inlines []string

	for _, k := range preferredOrder {
		def := definitions[k]
		if len(def.inlines) == 0 {
			continue
		}

		for _, inlineStruct := range def.inlines {
			ref := strings.TrimPrefix(inlineStruct.Ref, defPrefix)
			inlines = append(inlines, ref)
		}

		// First, inline definitions without `oneOf`
		inlineIndex := 0
		var defPreferredOrder []string
		for _, k := range def.PreferredOrder {
			if k != "<inline>" {
				defPreferredOrder = append(defPreferredOrder, k)
				continue
			}

			inlineStruct := def.inlines[inlineIndex]
			inlineIndex++

			ref := strings.TrimPrefix(inlineStruct.Ref, defPrefix)
			inlineStructRef := definitions[ref]
			if isOneOf(inlineStructRef) {
				continue
			}

			if def.Properties == nil {
				def.Properties = make(map[string]*Definition, len(inlineStructRef.Properties))
			}
			for k, v := range inlineStructRef.Properties {
				def.Properties[k] = v
			}

			defPreferredOrder = append(defPreferredOrder, inlineStructRef.PreferredOrder...)
			def.Required = append(def.Required, inlineStructRef.Required...)
		}
		def.PreferredOrder = defPreferredOrder

		// Then add options for `oneOf` definitions
		var options []*Definition
		for _, inlineStruct := range def.inlines {
			ref := strings.TrimPrefix(inlineStruct.Ref, defPrefix)
			inlineStructRef := definitions[ref]
			if !isOneOf(inlineStructRef) {
				continue
			}

			for _, key := range inlineStructRef.PreferredOrder {
				var preferredOrder []string
				choice := make(map[string]*Definition)

				if len(def.Properties) > 0 {
					for _, pkey := range def.PreferredOrder {
						preferredOrder = append(preferredOrder, pkey)
						choice[pkey] = def.Properties[pkey]
					}
				}

				preferredOrder = append(preferredOrder, key)
				choice[key] = inlineStructRef.Properties[key]

				options = append(options, &Definition{
					Properties:           choice,
					PreferredOrder:       preferredOrder,
					AdditionalProperties: false,
				})
			}
		}

		if len(options) == 0 {
			continue
		}

		options = append([]*Definition{{
			Properties:           def.Properties,
			PreferredOrder:       def.PreferredOrder,
			AdditionalProperties: false,
		}}, options...)

		def.Properties = nil
		def.PreferredOrder = nil
		def.AdditionalProperties = nil
		def.AnyOf = options
	}

	for _, ref := range inlines {
		delete(definitions, ref)
	}

	schema := Schema{
		Definition: &Definition{
			Type: "object",
			AnyOf: []*Definition{{
				Ref: defPrefix + preferredOrder[0],
			}},
		},
		Definitions: definitions,
	}

	return toJSON(schema)
}

func (g Generator) ParseDefinition(name string, t ast.Expr, comment string) *Definition {
	def := &Definition{}

	switch tt := t.(type) {
	case *ast.Ident:
		typeName := tt.Name
		setTypeOrRef(def, typeName)

		switch typeName {
		case constants.StringText:
			def.Default = "\"\""
		case "bool":
			def.Default = "false"
		case "int", "int64":
			def.Default = "0"
		}

	case *ast.StarExpr:
		if ident, ok := tt.X.(*ast.Ident); ok {
			typeName := ident.Name
			setTypeOrRef(def, typeName)
		}

	case *ast.ArrayType:
		def.Type = "array"
		def.Items = g.ParseDefinition("", tt.Elt, "")
		if def.Items != nil {
			if def.Items.Ref == "" {
				def.Default = "[]"
			}
		}

	case *ast.MapType:
		def.Type = "object"
		def.Default = "{}"
		def.AdditionalProperties = g.ParseDefinition("", tt.Value, "")

	case *ast.StructType:
		for _, field := range tt.Fields.List {
			if field.Tag == nil {
				continue
			}
			yamlName := yamlFieldName(field)

			if strings.Contains(field.Tag.Value, "inline") {
				def.PreferredOrder = append(def.PreferredOrder, "<inline>")
				def.inlines = append(def.inlines, &Definition{
					Ref: defPrefix + field.Type.(*ast.Ident).Name,
				})
				continue
			}

			if yamlName == "" || yamlName == "-" || yamlName == "ansible_tags" {
				continue
			}

			if strings.Contains(field.Tag.Value, "required") {
				def.Required = append(def.Required, yamlName)
			}

			if def.Properties == nil {
				def.Properties = make(map[string]*Definition)
			}

			def.PreferredOrder = append(def.PreferredOrder, yamlName)
			def.Properties[yamlName] = g.ParseDefinition(field.Names[0].Name, field.Type, field.Doc.Text())
			def.AdditionalProperties = false
		}
	}

	if g.strict && name != "" {
		if !strings.HasPrefix(comment, name+" ") {
			panic(fmt.Sprintf("comment should start with field name on field %s", name))
		}
	}

	// process enums before stripping out newlines
	if m := regexpEnumDefinition.FindStringSubmatch(comment); m != nil {
		enums := make([]string, 0)
		if n := regexpEnumValues.FindAllStringSubmatch(m[1], -1); n != nil {
			for _, matches := range n {
				enums = append(enums, matches[1])
			}
			def.Enum = enums
		}
	}

	description := strings.TrimSpace(strings.Replace(comment, "\n", " ", -1))

	// Extract default value
	if m := regexpDefaults.FindStringSubmatch(description); m != nil {
		description = strings.TrimSpace(m[1])
		def.Default = m[2]
	}

	// Extract example
	if m := regexpExample.FindStringSubmatch(description); m != nil {
		description = strings.TrimSpace(m[1])
		def.Examples = []string{m[2]}
	}

	// Remove type prefix
	description = regexp.MustCompile("^"+name+" (\\*.*\\* )?((is (the )?)|(are (the )?)|(lists ))?").ReplaceAllString(description, "$1")

	if g.strict && name != "" {
		if description == "" {
			panic(fmt.Sprintf("no description on field %s", name))
		}
		if !strings.HasSuffix(description, ".") {
			panic(fmt.Sprintf("description should end with a dot on field %s", name))
		}
	}
	def.Description = description

	// Convert to HTML
	html := string(blackfriday.Run([]byte(description), blackfriday.WithNoExtensions()))
	def.HTMLDescription = strings.TrimSpace(pTags.ReplaceAllString(html, ""))

	return def
}

//nolint:golint,goconst
func setTypeOrRef(def *Definition, typeName string) {
	switch typeName {
	// Special case for ResourceType that is an alias of string.
	// Fixes #3623
	case constants.StringText, "ResourceType":
		def.Type = constants.StringText
	case "bool":
		def.Type = "boolean"
	case "int", "int64", "int32":
		def.Type = "integer"
	default:
		def.Ref = defPrefix + typeName
	}
}

func yamlFieldName(field *ast.Field) string {
	tag := strings.Replace(field.Tag.Value, "`", "", -1)
	tags := reflect.StructTag(tag)
	yamlTag := tags.Get("yaml")

	return strings.Split(yamlTag, ",")[0]
}

// Make sure HTML description are not encoded
func toJSON(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(v); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func isOneOf(definition *Definition) bool {
	return len(definition.Properties) > 0 &&
		strings.Contains(definition.Properties[definition.PreferredOrder[0]].tags, "oneOf=")
}
