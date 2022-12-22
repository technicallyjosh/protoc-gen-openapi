package generator

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/technicallyjosh/protoc-gen-openapi/internal/generator/util"
	"google.golang.org/protobuf/compiler/protogen"
	"gopkg.in/yaml.v3"
)

// Config holds the configuration for the generator.
type Config struct {
	Version         *string
	Title           *string
	Description     *string
	Ignore          *string
	DefaultResponse *string
	ContentType     *string
	UseJSONNames    *bool
	JSONOutput      *bool
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

	outFile := g.plugin.NewGeneratedFile(filename, "")
	_, err = outFile.Write(fileBuffer.Bytes())

	return err
}

// buildDocument builds out the base of the OAPI document with some defaults.
func (g *Generator) buildDocument() (*openapi3.T, error) {
	doc := &openapi3.T{
		ExtensionProps: openapi3.ExtensionProps{},
		OpenAPI:        "3.0.3",
		Components: openapi3.Components{
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
		err = g.addPathsToDoc(doc, file.Services)
		if err != nil {
			return nil, err
		}
	}

	util.UniqueServers(doc)
	util.UniqueTags(doc)

	return doc, nil
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
