package util

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

// FullName returns the full name of a message.
func FullName(message *protogen.Message) string {
	return string(message.Desc.FullName())
}

// IsRequestMessage returns whether the message name ends with Request or not.
func IsRequestMessage(message *protogen.Message) bool {
	return strings.HasSuffix(FullName(message), "Request")
}

// IsResponseMessage returns whether the message name ends with Response or not.
func IsResponseMessage(message *protogen.Message) bool {
	return strings.HasSuffix(FullName(message), "Response")
}
