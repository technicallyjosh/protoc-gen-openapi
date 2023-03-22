package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/suite"
)

type TestOptions struct {
	defaultResponse string
	description     string
	testProto       string
	title           string
	version         string
}

type TestSuite struct {
	suite.Suite
	rawDoc  []byte
	options TestOptions
}

func (s *TestSuite) SetupSuite() {
	s.options = TestOptions{
		defaultResponse: "test.api.Error",
		description:     "test description",
		testProto:       "basic_test.proto",
		title:           "test title",
		version:         "1.1.0",
	}
}

func (s *TestSuite) BeforeTest(suite, name string) {
	var filename string

	switch name {
	case "TestBasic":
		filename = "basic_test.proto"
	case "TestFile":
		filename = "file_test.proto"
	case "TestService":
		filename = "service_test.proto"
	case "TestMethod":
		filename = "method_test.proto"
	case "TestField":
		filename = "field_test.proto"
	default:
		s.FailNow("invalid test name")
	}

	err := exec.Command("rm", "-f", "test/openapi.yaml").Run()
	if err != nil {
		s.FailNow(err.Error())
	}

	out, err := exec.Command("protoc",
		"-I=api",
		"-I=test",
		"--openapi_out=test",
		"--openapi_opt=version="+s.options.version,
		"--openapi_opt=title="+s.options.title,
		"--openapi_opt=description="+s.options.description,
		"--openapi_opt=default_response="+s.options.defaultResponse,
		"test/"+filename,
	).CombinedOutput()
	if err != nil {
		s.FailNow(string(out))
	}

	s.rawDoc, err = os.ReadFile("test/openapi.yaml")
	if err != nil {
		s.FailNow(err.Error())
	}
}

func (s *TestSuite) TestBasic() {
	s.YAMLEq(readFile("basic_test_openapi.yaml"), string(s.rawDoc))
}

func (s *TestSuite) TestFile() {
	s.YAMLEq(readFile("file_test_openapi.yaml"), string(s.rawDoc))
}

func (s *TestSuite) TestService() {
	s.YAMLEq(readFile("service_test_openapi.yaml"), string(s.rawDoc))
}

func (s *TestSuite) TestMethod() {
	s.YAMLEq(readFile("method_test_openapi.yaml"), string(s.rawDoc))
}

func (s *TestSuite) TestField() {
	s.YAMLEq(readFile("field_test_openapi.yaml"), string(s.rawDoc))
}

func TestMain(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func readFile(name string) string {
	data, _ := os.ReadFile("test/" + name)
	return string(data)
}
