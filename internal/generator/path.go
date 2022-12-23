package generator

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	oapiv1 "github.com/technicallyjosh/protoc-gen-openapi/api/oapi/v1"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
)

func newServer(host string) (*openapi3.Server, error) {
	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	if u.Scheme == "" {
		u.Scheme = "https"
	}

	server := &openapi3.Server{
		URL: u.String(),
	}

	return server, nil
}

// addPathsToDoc adds paths from services and methods to the OAPI doc. This includes all request and
// response bodies.
// TODO: Break apart into smaller functions.
func (g *Generator) addPathsToDoc(doc *openapi3.T, services []*protogen.Service) error {
	contentType := *g.config.ContentType

	for _, service := range services {
		var host, pathPrefix string

		extFile := proto.GetExtension(service.Desc.ParentFile().Options(), oapiv1.E_File)
		if extFile != nil && extFile != oapiv1.E_File.InterfaceOf(oapiv1.E_File.Zero()) {
			fileOptions := extFile.(*oapiv1.FileOptions)
			host = fileOptions.Host
			pathPrefix = fileOptions.Prefix
		}

		tagName := string(service.Desc.Name())
		packageName := string(service.Desc.ParentFile().Package())

		serviceOptions := new(oapiv1.ServiceOptions)

		extService := proto.GetExtension(service.Desc.Options(), oapiv1.E_Service)
		if extService != nil && extService != oapiv1.E_Service.InterfaceOf(oapiv1.E_Service.Zero()) {
			serviceOptions = extService.(*oapiv1.ServiceOptions)
		}

		if serviceOptions.Host != "" {
			// Use service defined host.
			host = serviceOptions.Host
		}

		if serviceOptions.Prefix != "" {
			// Use service defined prefix.
			pathPrefix = serviceOptions.Prefix
		}

		if serviceOptions.ContentType != "" {
			// Use service defined content type.
			contentType = serviceOptions.ContentType
		}

		serviceDescription := g.parseComments(service.Comments.Leading).Description

		props := openapi3.ExtensionProps{
			Extensions: map[string]any{
				"x-displayName": serviceOptions.DisplayName,
			},
		}

		doc.Tags = append(doc.Tags, &openapi3.Tag{
			Name:           tagName,
			Description:    serviceDescription,
			ExtensionProps: props,
		})

		for _, method := range service.Methods {
			operationID := string(service.Desc.Name() + "_" + method.Desc.Name())
			description := g.parseComments(method.Comments.Leading).Description

			var methodOptions *oapiv1.MethodOptions

			extMethod := proto.GetExtension(method.Desc.Options(), oapiv1.E_Method)
			if extMethod != nil && extMethod != oapiv1.E_Method.InterfaceOf(oapiv1.E_Method.Zero()) {
				methodOptions = extMethod.(*oapiv1.MethodOptions)
			} else {
				continue
			}

			if methodOptions.Host != "" {
				// Use service defined host.
				host = methodOptions.Host
			}

			if methodOptions.ContentType != "" {
				// Use method defined content type.
				contentType = methodOptions.ContentType
			}

			// Append the defined host as a server. Duplicates are removed later.
			server, err := newServer(host)
			if err != nil {
				return err
			}
			doc.Servers = append(doc.Servers, server)

			var methodPath, methodName string

			op := &openapi3.Operation{
				Tags:        []string{tagName},
				Description: description,
				OperationID: operationID,
				Servers:     &doc.Servers,
				Deprecated:  methodOptions.Deprecated,
				Responses:   make(openapi3.Responses),
				Summary:     methodOptions.Summary,
			}

			switch m := methodOptions.Method.(type) {
			case *oapiv1.MethodOptions_Get:
				methodName = http.MethodGet
				methodPath = m.Get
			case *oapiv1.MethodOptions_Put:
				methodName = http.MethodPut
				methodPath = m.Put
			case *oapiv1.MethodOptions_Post:
				methodName = http.MethodPost
				methodPath = m.Post
			case *oapiv1.MethodOptions_Delete:
				methodName = http.MethodDelete
				methodPath = m.Delete
			case *oapiv1.MethodOptions_Patch:
				methodName = http.MethodPatch
				methodPath = m.Patch
			default:
				return fmt.Errorf("method '%s' is missing a method", method.Desc.FullName())
			}

			// If the method's path starts with a "/", don't append the prefix from the service.
			if !strings.HasPrefix(methodPath, "/") {
				methodPath = path.Join(pathPrefix, methodPath)
			}

			var defaultResponseDesc string

			// Set the default response from the method or service if defined. Otherwise, use the
			// globally set one.
			if methodOptions.DefaultResponse != "" {
				// Use the method response.
				schemaName := g.getPackageSchema(packageName, methodOptions.DefaultResponse)
				_, ok := doc.Components.Schemas[schemaName]
				if !ok {
					return fmt.Errorf("schema '%s' for method '%s' default response not found", schemaName, method.Desc.FullName())
				}

				op.Responses["default"] = &openapi3.ResponseRef{
					Value: &openapi3.Response{
						Description: &defaultResponseDesc,
						Content: openapi3.Content{
							contentType: &openapi3.MediaType{
								Schema: &openapi3.SchemaRef{
									Ref: newSchemaRef(schemaName),
								},
							},
						},
					},
				}
			} else if serviceOptions.DefaultResponse != "" {
				// Use the service response
				schemaName := g.getPackageSchema(packageName, serviceOptions.DefaultResponse)
				_, ok := doc.Components.Schemas[schemaName]
				if !ok {
					return fmt.Errorf("schema '%s' for service '%s' default response not found", schemaName, service.Desc.FullName())
				}

				op.Responses["default"] = &openapi3.ResponseRef{
					Value: &openapi3.Response{
						Description: &defaultResponseDesc,
						Content: openapi3.Content{
							contentType: &openapi3.MediaType{
								Schema: &openapi3.SchemaRef{
									Ref: newSchemaRef(schemaName),
								},
							},
						},
					},
				}
			} else {
				// Use the global default if available
				if doc.Components.Responses.Default() != nil {
					op.Responses["default"] = &openapi3.ResponseRef{
						Ref: newResponseRef("default"),
					}
				}
			}

			if methodOptions.Status == 0 {
				// Default to 200 OK.
				methodOptions.Status = http.StatusOK
			}

			requestContent := openapi3.Content{
				contentType: &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Properties: make(openapi3.Schemas),
						},
					},
				},
			}

			responseContent := openapi3.Content{
				contentType: &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{},
				},
			}

			if methodName != http.MethodGet {
				inputFullName := string(method.Input.Desc.FullName())
				message := allMessages.Get(inputFullName)

				// If another type is defined such as google.protobuf.Any, for now we'll just
				// exit.
				// TODO: support `Any` type for requests 🤔
				if message != nil {
					requestSchemaRef := &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Properties: make(openapi3.Schemas),
						},
					}

					err := g.buildSchema(doc, message, requestSchemaRef)
					if err != nil {
						return err
					}
					requestContent.Get(contentType).Schema = requestSchemaRef

					op.RequestBody = &openapi3.RequestBodyRef{
						Value: &openapi3.RequestBody{
							Content: requestContent,
						},
					}
				}
			}

			outputFullName := string(method.Output.Desc.FullName())
			message := allMessages.Get(outputFullName)
			responseSchema := &openapi3.Schema{
				Properties: make(openapi3.Schemas),
			}
			err = g.buildSchema(doc, message, responseSchema.NewRef())
			if err != nil {
				return err
			}
			responseContent.Get(contentType).Schema = responseSchema.NewRef()

			responseCode := fmt.Sprintf("%d", methodOptions.Status)
			var responseDescription string
			op.Responses[responseCode] = &openapi3.ResponseRef{
				Value: &openapi3.Response{
					Content:     responseContent,
					Description: &responseDescription,
				},
			}

			// Check for an existing path an append if it exists.
			existingPath := doc.Paths.Find(methodPath)
			if existingPath == nil {
				pathItem := new(openapi3.PathItem)
				pathItem.SetOperation(methodName, op)

				doc.Paths[methodPath] = pathItem
			} else {
				if existingPath.GetOperation(methodName) != nil {
					return fmt.Errorf("duplicate method '%s' for path '%s'", methodName, methodPath)
				}

				// append the method
				existingPath.SetOperation(methodName, op)
			}
		}
	}

	return nil
}
