package generator

import (
	"github.com/technicallyjosh/protoc-gen-openapi/internal/generator/util"
	"google.golang.org/protobuf/compiler/protogen"
)

var (
	// allMessages holds all messages with their full paths for reference whenever we need to look
	// for a message to build out.
	allMessages = make(messageMap)
)

type messageMap map[string]*protogen.Message

// Set adds the specified message to the map.
func (m messageMap) Set(val *protogen.Message) {
	m[util.FullName(val)] = val
}

// Get returns the message by key or nil.
func (m messageMap) Get(key string) *protogen.Message {
	val, ok := m[key]
	if !ok {
		return nil
	}

	return val
}

// buildMessageMap recursively adds messages by full path to a map for usage later.
func (g *Generator) buildMessageMap(messages []*protogen.Message) {
	for _, message := range messages {
		allMessages.Set(message)

		if len(message.Messages) > 0 {
			g.buildMessageMap(message.Messages)
		}
	}
}
