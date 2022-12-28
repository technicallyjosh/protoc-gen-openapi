package generator

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

func (g *Generator) addDefaultResponse(doc *openapi3.T) error {
	name := *g.config.DefaultResponse
	if name == "" {
		return nil
	}

	// TODO: allow for custom description?
	description := ""

	_, ok := doc.Components.Schemas[name]
	if !ok {
		return fmt.Errorf("schema '%s' for default response not found", name)
	}

	doc.Components.Responses["default"] = &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: &description,
			Content: openapi3.Content{
				*g.config.ContentType: &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Ref: newSchemaRef(name),
					},
				},
			},
		},
	}

	return nil
}

// newResponseRef returns a convenient response reference.
func newResponseRef(name string) string {
	return "#/components/responses/" + name
}
