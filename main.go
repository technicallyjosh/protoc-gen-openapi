package main

import (
	"flag"

	"github.com/technicallyjosh/protoc-gen-openapi/internal/generator"
	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	var flags flag.FlagSet

	conf := generator.Config{
		ContentType:     flags.String("content_type", "application/json", "Default content-type for all paths."),
		DefaultResponse: flags.String("default_response", "", "Default response message to use for API responses not defined."),
		Description:     flags.String("description", "", "Description of the API."),
		Host:            flags.String("host", "", "Host to be used for all routes."),
		Ignore:          flags.String("ignore", "", "Packages to ignore."),
		JSONOutput:      flags.Bool("json_out", false, "Generate a JSON file instead of YAML."),
		Title:           flags.String("title", "", "Title of the API"),
		UseJSONNames:    flags.Bool("json_names", false, "Use JSON names instead of the proto names of fields."),
		Version:         flags.String("version", "0.0.1", "Version of the API."),
	}

	opts := protogen.Options{
		ParamFunc: flags.Set,
	}

	opts.Run(func(plugin *protogen.Plugin) error {
		return generator.New(plugin, conf).Run()
	})
}
