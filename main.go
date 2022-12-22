package main

import (
	"flag"

	"github.com/technicallyjosh/protoc-gen-openapi/internal/generator"
	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	var flags flag.FlagSet

	conf := generator.Config{
		Version:         flags.String("version", "0.0.1", "Version of the API."),
		Title:           flags.String("title", "", "Title of the API"),
		Description:     flags.String("description", "", "Description of the API."),
		Ignore:          flags.String("ignore", "", "Packages to ignore."),
		DefaultResponse: flags.String("default_response", "", "Default response message to use for API responses not defined."),
		ContentType:     flags.String("content_type", "application/json", "Default content-type for all paths."),
		UseJSONNames:    flags.Bool("json_names", false, "Use JSON names instead of the proto names of fields."),
		JSONOutput:      flags.Bool("json_out", false, "Generate a JSON file instead of YAML."),
	}

	opts := protogen.Options{
		ParamFunc: flags.Set,
	}

	opts.Run(func(plugin *protogen.Plugin) error {
		// Enable "optional" keyword in front of type. (e.g. optional string label = 1;)
		// plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)

		return generator.New(plugin, conf).Run()
	})
}
