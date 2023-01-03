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
func (g *Generator) addPathsToDoc(doc *openapi3.T, services []*protogen.Service) error {
	// Default to config-defined.
	contentType := *g.config.ContentType
	host := *g.config.Host

	for _, service := range services {
		var pathPrefix string

		// Apply/Override file options.
		extFile := proto.GetExtension(service.Desc.ParentFile().Options(), oapiv1.E_File)
		if extFile != nil && extFile != oapiv1.E_File.InterfaceOf(oapiv1.E_File.Zero()) {
			fileOptions := extFile.(*oapiv1.FileOptions)
			host = fileOptions.Host
			pathPrefix = fileOptions.Prefix
		}

		tagName := string(service.Desc.Name())
		packageName := string(service.Desc.ParentFile().Package())

		serviceOptions := new(oapiv1.ServiceOptions)

		// Service options.
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
				"x-displayName": serviceOptions.XDisplayName,
			},
		}

		doc.Tags = append(doc.Tags, &openapi3.Tag{
			Name:           tagName,
			Description:    serviceDescription,
			ExtensionProps: props,
		})

		tagGroup := strings.TrimSpace(serviceOptions.XTagGroup)
		if tagGroup != "" {
			err := addTagGroup(doc, tagGroup, tagName)
			if err != nil {
				return err
			}
		}

		parameters, err := g.parseParameters(openapi3.ParameterInPath, pathPrefix, serviceOptions.PathParameter)
		if err != nil {
			return err
		}

		for _, method := range service.Methods {
			err := g.addOperation(addOperationParams{
				doc:            doc,
				service:        service,
				serviceOptions: serviceOptions,
				method:         method,
				host:           host,
				contentType:    contentType,
				tagName:        tagName,
				pathPrefix:     pathPrefix,
				packageName:    packageName,
				parameters:     parameters,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// parseParameters parses and returns openapi3 converted parameters from defined parameters.
func (g *Generator) parseParameters(in, path string, parameters []*oapiv1.Parameter) (openapi3.Parameters, error) {
	params := make(openapi3.Parameters, 0)

	for _, parameter := range parameters {
		if !strings.Contains(path, fmt.Sprintf("{%s}", parameter.Name)) {
			return nil, fmt.Errorf("parameter {%s} is missing from path %s", parameter.Name, path)
		}

		paramRef := &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:     parameter.Name,
				In:       in,
				Required: in == openapi3.ParameterInPath,
			},
		}

		var paramType string

		switch parameter.Type {
		case oapiv1.Parameter_TYPE_UNSPECIFIED, oapiv1.Parameter_TYPE_STRING:
			paramType = openapi3.TypeString
		case oapiv1.Parameter_TYPE_INTEGER:
			paramType = openapi3.TypeInteger
		case oapiv1.Parameter_TYPE_NUMBER:
			paramType = openapi3.TypeNumber
		case oapiv1.Parameter_TYPE_BOOLEAN:
			paramType = openapi3.TypeBoolean
		default:
			return nil, fmt.Errorf("invalid parameter type: %s", parameter.Type)
		}

		paramRef.Value.Schema = &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type: paramType,
			},
		}

		if strings.TrimSpace(parameter.Description) != "" {
			paramRef.Value.Description = parameter.Description
		}

		if strings.TrimSpace(parameter.Example) != "" {
			paramRef.Value.Example = parameter.Example
		}

		if parameter.Required != nil {
			paramRef.Value.Required = *parameter.Required
		}

		if parameter.Options != nil {
			err := setProperties(paramRef.Value.Schema.Value, parameter.Options)
			if err != nil {
				return nil, err
			}
		}

		params = append(params, paramRef)
	}

	return params, nil
}

type addOperationParams struct {
	doc            *openapi3.T
	service        *protogen.Service
	method         *protogen.Method
	serviceOptions *oapiv1.ServiceOptions
	host           string
	contentType    string
	tagName        string
	pathPrefix     string
	packageName    string
	parameters     openapi3.Parameters
}

// addOperation creates an operation for a path and adds it.
// TODO: Break into smaller bits.
func (g *Generator) addOperation(p addOperationParams) error {
	host := p.host
	contentType := p.contentType

	operationID := string(p.service.Desc.Name() + "_" + p.method.Desc.Name())
	description := g.parseComments(p.method.Comments.Leading).Description

	var methodOptions *oapiv1.MethodOptions

	// Method options. If not present, continue to next.
	extMethod := proto.GetExtension(p.method.Desc.Options(), oapiv1.E_Method)
	if extMethod != nil && extMethod != oapiv1.E_Method.InterfaceOf(oapiv1.E_Method.Zero()) {
		methodOptions = extMethod.(*oapiv1.MethodOptions)
	} else {
		return nil
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
	p.doc.Servers = append(p.doc.Servers, server)

	var methodPath, methodName string

	op := &openapi3.Operation{
		Tags:        []string{p.tagName},
		Description: description,
		OperationID: operationID,
		Servers:     &openapi3.Servers{server},
		Deprecated:  methodOptions.Deprecated,
		Responses:   make(openapi3.Responses),
		Summary:     methodOptions.Summary,
		Parameters:  make(openapi3.Parameters, 0),
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
		return fmt.Errorf("method '%s' is missing a method", p.method.Desc.FullName())
	}

	// If the method's path starts with a "/", don't append the prefix from the service.
	if !strings.HasPrefix(methodPath, "/") {
		methodPath = path.Join(p.pathPrefix, methodPath)
	}

	var defaultResponseDesc string

	// Set the default response from the method or service if defined. Otherwise, use the
	// globally set one.
	if methodOptions.DefaultResponse != "" {
		// Use the method default response.
		schemaName := g.getPackageSchema(p.packageName, methodOptions.DefaultResponse)
		_, ok := p.doc.Components.Schemas[schemaName]
		if !ok {
			return fmt.Errorf("schema '%s' for method '%s' default response not found", schemaName, p.method.Desc.FullName())
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
	} else if p.serviceOptions.DefaultResponse != "" {
		// Use the service default response
		schemaName := g.getPackageSchema(p.packageName, p.serviceOptions.DefaultResponse)
		_, ok := p.doc.Components.Schemas[schemaName]
		if !ok {
			return fmt.Errorf("schema '%s' for service '%s' default response not found", schemaName, p.service.Desc.FullName())
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
		// Use the global default response if available.
		if p.doc.Components.Responses.Default() != nil {
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
					Description: "",
					Properties:  make(openapi3.Schemas),
					Title:       "something",
				},
			},
		},
	}

	responseContent := openapi3.Content{
		contentType: &openapi3.MediaType{
			Schema: &openapi3.SchemaRef{},
		},
	}

	if len(methodOptions.PathParameter) > 0 {
		parameters, err := g.parseParameters(openapi3.ParameterInPath, methodPath, methodOptions.PathParameter)
		if err != nil {
			return err
		}

		// Check to see if the method-defined parameter already exists and
		// error.
		for _, p1 := range parameters {
			for _, p2 := range p.parameters {
				if p2.Value.Name == p1.Value.Name {
					return fmt.Errorf(
						"parameter '%s' is already defined in service definition",
						p1.Value.Name,
					)
				}
			}
		}

		p.parameters = append(p.parameters, parameters...)
	}

	op.Parameters = p.parameters

	if methodName != http.MethodGet {
		inputFullName := string(p.method.Input.Desc.FullName())
		message := allMessages.Get(inputFullName)

		// If another type is defined such as google.protobuf.Any or
		// google.protobuf.Empty, for now we'll just exit.
		// TODO: support `Any` type for requests 🤔
		if message != nil {
			requestSchemaRef := &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Properties: make(openapi3.Schemas),
				},
			}

			err := g.buildSchema(p.doc, message, requestSchemaRef)
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

	outputFullName := string(p.method.Output.Desc.FullName())
	message := allMessages.Get(outputFullName)
	responseSchema := &openapi3.Schema{
		Properties: make(openapi3.Schemas),
	}
	err = g.buildSchema(p.doc, message, responseSchema.NewRef())
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
	existingPath := p.doc.Paths.Find(methodPath)
	if existingPath == nil {
		pathItem := new(openapi3.PathItem)
		pathItem.SetOperation(methodName, op)

		p.doc.Paths[methodPath] = pathItem
	} else {
		if existingPath.GetOperation(methodName) != nil {
			return fmt.Errorf("duplicate method '%s' for path '%s'", methodName, methodPath)
		}

		// append the method
		existingPath.SetOperation(methodName, op)
	}

	return nil
}
