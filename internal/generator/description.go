package generator

import (
	"bufio"
	"regexp"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

// exampleRegexp is for breaking apart a comment with an example.
var exampleRegexp = regexp.MustCompile(`(?is)(.*)Example:(.*)`)

// parsedComments holds the data parsed and extracted from a comment.
type parsedComments struct {
	Description string
	Example     string
}

// parseComments returns a filtered comment string for an OAPI description.
func (g *Generator) parseComments(c protogen.Comments) *parsedComments {
	comment := strings.TrimSpace(string(c))

	withExample := exampleRegexp.FindStringSubmatch(comment)

	var rawDescription, rawExample string

	if len(withExample) > 1 {
		rawDescription = strings.TrimSpace(withExample[1])
		rawExample = strings.TrimSpace(withExample[2])
	} else {
		rawDescription = comment
	}

	// Iterate over lines to trim while preserving line breaks.
	description := strings.Builder{}
	scanner := bufio.NewScanner(strings.NewReader(rawDescription))
	for scanner.Scan() {
		description.WriteString(strings.TrimLeft(scanner.Text(), " ") + "\n")
	}

	example := strings.Builder{}
	scanner = bufio.NewScanner(strings.NewReader(rawExample))
	for scanner.Scan() {
		example.WriteString(strings.TrimSpace(scanner.Text()))
	}

	return &parsedComments{
		Description: description.String(),
		Example:     example.String(),
	}
}
