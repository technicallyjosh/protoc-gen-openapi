package util

import (
	"github.com/getkin/kin-openapi/openapi3"
)

func UniqueServers(doc *openapi3.T) {
	u := make([]*openapi3.Server, 0, len(doc.Servers))
	m := make(map[string]bool)

	for _, server := range doc.Servers {
		if _, ok := m[server.URL]; !ok {
			m[server.URL] = true
			u = append(u, server)
		}
	}

	doc.Servers = u
}

func UniqueTags(doc *openapi3.T) {
	u := make([]*openapi3.Tag, 0, len(doc.Tags))
	m := make(map[string]bool)

	for _, tag := range doc.Tags {
		if _, ok := m[tag.Name]; !ok {
			m[tag.Name] = true
			u = append(u, tag)
		}
	}

	doc.Tags = u
}
