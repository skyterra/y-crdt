package y_crdt

import (
	"bytes"
)

const (
	MessagePermissionDenied = 0
)

func WritePermissionDenied(encoder *bytes.Buffer, reason string) {
	WriteVarUint(encoder, MessagePermissionDenied)
	_ = WriteString(encoder, reason)
}

func ReadAuthMessage(decoder *bytes.Buffer, doc *Doc, permissionDeniedHandler func(doc *Doc, reason string)) {
	switch ReadVarUint(decoder) {
	case MessagePermissionDenied:
		reason, _ := ReadString(decoder)
		permissionDeniedHandler(doc, reason)
	}
}
