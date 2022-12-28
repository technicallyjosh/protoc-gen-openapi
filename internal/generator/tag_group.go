package generator

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

type XTagGroups []*XTagGroup

func (g *XTagGroups) find(name string) *XTagGroup {
	for _, group := range *g {
		if group.Name == name {
			return group
		}
	}

	return nil
}

// XTagGroup represents a tag group from the x-tagGroups extension.
type XTagGroup struct {
	Name string   `json:"name" yaml:"name"`
	Tags []string `json:"tags" yaml:"tags"`
}

func (g *XTagGroup) tagExists(name string) bool {
	for _, tag := range g.Tags {
		if tag == name {
			return true
		}
	}

	return false
}

// addTagGroup adds the specified tag to the tag group extension. If the group doesn't exist, it is
// created. If not, it's appended to the tag list.
func addTagGroup(doc *openapi3.T, name, tag string) error {
	key := "x-tagGroups"
	var groups *XTagGroups

	// Ensure the extension exists first.
	foundGroups, ok := doc.ExtensionProps.Extensions[key]
	if !ok {
		groups = new(XTagGroups)
		doc.ExtensionProps.Extensions[key] = groups
	} else {
		// If it's found, set the pointer
		groups, ok = foundGroups.(*XTagGroups)
		if !ok {
			return fmt.Errorf("x-tagGroups is not a valid format: %#v", foundGroups)
		}
	}

	group := groups.find(name)
	if group == nil {
		*groups = append(*groups, &XTagGroup{
			Name: name,
			Tags: []string{tag},
		})
	} else {
		// Check if tag exists on group
		if !group.tagExists(tag) {
			group.Tags = append(group.Tags, tag)
		}
	}

	return nil
}
