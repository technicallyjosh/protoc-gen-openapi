# protoc-gen-openapi

**Yes**, this is _another_ protoc generator for OpenAPI. I created this for a
couple
reasons...

- I wanted to learn protoc generation with a real-world problem.
- The official google one sticks to gRPC and envoy standards. My team and I use
  Twirp and other REST frameworks. _Sometimes you just want to define models and
  an API for docs._
- Others try to do too much per the spec and fail to do the most common things
  well.

_**DISCLAIMER: This will be a limited subset of the OAPI3 specification. Not
everything will make it in here. Why? Read the last bullet point above. :)**_

_Some patterns were heavily inspired
by [gnostic](https://github.com/google/gnostic)._

## Installation

```terminal
go install github.com/technicallyjosh/protoc-gen-openapi@latest
```

## Options

| Option             | Description                                                                       | Default          |
|--------------------|-----------------------------------------------------------------------------------|------------------|
| `version`          | The version of the API.                                                           | 0.0.1            |
| `title`            | The title of the API.                                                             |                  |
| `description`      | A description of the API.                                                         |                  |
| `ignore`           | A list of package names to ignore delimited with a pipe.                          |                  |
| `default_response` | The default response to be used.<sup>1</sup>                                      |                  |
| `content_type`     | The content type to be associated with all operations.<sup>1</sup>                | application/json |
| `json_names`       | Use the JSON names that Protobuf provides. Otherwise, proto field names are used. | false            |
| `json_out`         | Create a JSON file instead of the default YAML.                                   | false            |

<sup>1</sup> _Can be overridden on a service or method._

## Using Buf

Yup, I've only actually used this in `buf` so far. I'm sure it works with the
standard protoc calls, but why would you do that to yourself 😂?

**buf.yaml**

```yaml
# ... other things
deps:
  - buf.build/technicallyjosh/protoc-gen-openapi
```

**buf.gen.yaml**

```yaml
plugins:
  - name: go
    out: api
    opt:
      - paths=source_relative
  - name: openapi
    strategy: all # important so all files are ran in the same generation.
    out: api
    opt:
      - title=My Awesome API
      - description=Look how awesome my API is!
      - ignore=module.v1|module.v2
      - default_response=SomeErrorObject
```

## Basic Usage Example

```protobuf
syntax = "proto3";

import "oapi/v1/field.proto";
import "oapi/v1/file.proto";
import "oapi/v1/method.proto";
import "oapi/v1/service.proto";

option (oapi.v1.file) = {
  host: "myawesomeapi.com"
};

service MyService {
  option (oapi.v1.service) = {
    prefix: "/v1"
    display_name: "My Service"
  };

  rpc CreateSomething (CreateSomethingRequest) returns (GetSomethingResponse) {
    option (oapi.v1.method) = {
      post: "create-something"
      summary: "Create Something"
    };
  }
}

message CreateSomethingRequest {
  // The name of something
  // Example: something-awesome
  string name = 1 [(oapi.v1.required) = true];
}

message CreateSomethingResponse {
  string id = 1;
  string name = 2;
}
```

## Features

> **Note**
>
> Defining features is a work in progress. I aim to explain all that's possible
> the best I can.

### Host definitions

You can define hosts at the file, service, or method level. Each one overrides
the previous. This allows for more advanced composition.

**Example:**

```protobuf
syntax = "proto3";

import "oapi/v1/file.proto";
import "oapi/v1/service.proto";
import "oapi/v1/method.proto";

option (oapi.v1.file) = {
  host: "myawesomeapi.com" // file-defined for all services and methods
};

service MyService {
  option (oapi.v1.service) = {
    host: "myawesomeapi2.com" // overrides file-defined
  };

  rpc CreateSomething (google.protobuf.Empty) returns (google.protobuf.Empty) {
    option (oapi.v1.method) = {
      host: "myaweseomeapi3.com" // overrides service-defined
    };
  }
}
```

## Contributing

Coming... Right now I prefer that it's just me until I get a solid hold on
generator patterns and the package is stable. I'm fully open to any suggestions
though!
