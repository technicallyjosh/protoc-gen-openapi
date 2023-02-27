package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/objx"
	oapiv1 "github.com/technicallyjosh/protoc-gen-openapi/api/oapi/v1"
	"github.com/technicallyjosh/protoc-gen-openapi/internal/generator/util"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

// Config holds the configuration for the generator.
type Config struct {
	ContentType     *string
	DefaultResponse *string
	Description     *string
	Host            *string
	Ignore          *string
	JSONOutput      *bool
	Title           *string
	UseJSONNames    *bool
	Version         *string
}

// Generator is an instance that parses the given folder and its Protobuf files into OAPI.
type Generator struct {
	config   Config
	plugin   *protogen.Plugin
	packages []string
}

// New creates and returns a new Generator instance.
func New(plugin *protogen.Plugin, conf Config) *Generator {
	return &Generator{
		config:   conf,
		plugin:   plugin,
		packages: make([]string, 0),
	}
}

// Run is the entrypoint method to generate OAPI from all Protobuf files. It builds the document and
// then generates the OAPI file.
func (g *Generator) Run() error {
	useJSON := *g.config.JSONOutput

	doc, err := g.buildDocument()
	if err != nil {
		return err
	}

	filename := "openapi.yaml"

	fileBuffer := bytes.Buffer{}
	jsonBytes, err := doc.MarshalJSON()
	if err != nil {
		return err
	}

	if useJSON {
		filename = "openapi.json"
		fileBuffer.Write(jsonBytes)
	} else {
		// Extra hops to get JSON to YAML.
		var i any
		err = json.Unmarshal(jsonBytes, &i)
		if err != nil {
			return err
		}

		encoder := yaml.NewEncoder(&fileBuffer)
		encoder.SetIndent(2)

		err = encoder.Encode(i)
		if err != nil {
			return err
		}
	}

	fileBytes := fileBuffer.Bytes()

	outFile := g.plugin.NewGeneratedFile(filename, "")

	patchedBytes, err := g.patchEmptySchemas(fileBytes)
	if err != nil {
		return err
	}

	_, err = outFile.Write(patchedBytes)
	return err
}

// patchEmptySchemas finds any schemas that are empty and updates them to have
// an empty `Properties` node.
func (g *Generator) patchEmptySchemas(fileBytes []byte) ([]byte, error) {
	type M = map[string]any
	data := make(M)

	useJSON := *g.config.JSONOutput

	var err error
	if useJSON {
		err = json.Unmarshal(fileBytes, &data)
	} else {
		err = yaml.Unmarshal(fileBytes, &data)
	}
	if err != nil {
		return nil, err
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	m, err := objx.FromJSON(string(jsonBytes))
	if err != nil {
		return nil, err
	}

	for pathKey := range m.Get("paths").ObjxMap() {
		pathPath := "paths." + pathKey

		for methodKey := range m.Get(pathPath).ObjxMap() {
			methodPath := pathPath + "." + methodKey

			schemaKey := fmt.Sprintf("%s.requestBody.content.application/json.schema", methodPath)
			schema := m.Get(schemaKey)

			if schema.IsObjxMap() && len(schema.ObjxMap()) == 0 {
				schema.ObjxMap().Set("properties", M{})
			}

			for resKey := range m.Get(methodPath + ".responses").ObjxMap() {
				schemaKey := fmt.Sprintf("%s.responses.%s.content.application/json.schema", methodPath, resKey)
				schema := m.Get(schemaKey)
				if schema.IsObjxMap() && len(schema.ObjxMap()) == 0 {
					schema.ObjxMap().Set("properties", M{})
				}
			}
		}
	}

	if useJSON {
		return json.Marshal(m)
	}

	buffer := bytes.Buffer{}
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)

	err = encoder.Encode(m)
	return buffer.Bytes(), err
}

// buildDocument builds out the base of the OAPI document with some defaults.
func (g *Generator) buildDocument() (*openapi3.T, error) {
	doc := &openapi3.T{
		Extensions: make(map[string]any),
		OpenAPI:    "3.1.0",
		Components: &openapi3.Components{
			SecuritySchemes: make(openapi3.SecuritySchemes),
			Schemas:         make(openapi3.Schemas),
			RequestBodies:   make(openapi3.RequestBodies),
			Responses:       openapi3.NewResponses(),
		},
		Info: &openapi3.Info{
			Title:       *g.config.Title,
			Description: *g.config.Description,
			Version:     *g.config.Version,
		},
		Paths:    make(openapi3.Paths),
		Security: make(openapi3.SecurityRequirements, 0),
		Servers:  make(openapi3.Servers, 0),
		Tags:     make(openapi3.Tags, 0),
	}

	ignored := strings.Split(*g.config.Ignore, "|")
	files := filterFiles(g.plugin.Files, ignored)

	for _, file := range files {
		g.buildMessageMap(file.Messages)

		// We use the package name for fully qualified schema names.
		err := g.addSchemasToDoc(doc, file.Messages)
		if err != nil {
			return nil, err
		}

		// Capture all package names for later use.
		g.packages = append(g.packages, file.Proto.GetPackage())
	}

	err := g.addDefaultResponse(doc)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		// Add servers even if there isn't a service. (File-based)
		err = addFileServersToDoc(doc, file)
		if err != nil {
			return nil, err
		}

		err = g.addPathsToDoc(doc, file.Services)
		if err != nil {
			return nil, err
		}
	}

	util.UniqueServers(doc)
	util.UniqueTags(doc)

	return doc, nil
}

func addFileServersToDoc(doc *openapi3.T, file *protogen.File) error {
	extFile := proto.GetExtension(file.Desc.Options(), oapiv1.E_File)
	if extFile != nil && extFile != oapiv1.E_File.InterfaceOf(oapiv1.E_File.Zero()) {
		fileOptions := extFile.(*oapiv1.FileOptions)

		if fileOptions.Host != "" {
			server, err := NewServer(fileOptions.Host)
			if err != nil {
				return err
			}
			doc.Servers = append(doc.Servers, server)
		}
	}

	return nil
}

func filterFiles(allFiles []*protogen.File, ignored []string) []*protogen.File {
	files := make([]*protogen.File, 0)

Files:
	for _, file := range allFiles {
		if !file.Generate {
			continue
		}

		for _, ignoredPackage := range ignored {
			if file.Proto.GetPackage() == ignoredPackage {
				continue Files
			}
		}

		files = append(files, file)
	}

	return files
}
