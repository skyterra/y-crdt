package y_crdt

import (
	"bytes"
	"encoding/json"
)

type DSEncoderV1 struct {
	RestEncoder *bytes.Buffer
}

type UpdateEncoderV1 struct {
	DSEncoderV1
}

// ToUint8Array returns the encoded bytes.
func (v1 *DSEncoderV1) ToUint8Array() []uint8 {
	return v1.RestEncoder.Bytes()
}

// ResetDsCurVal resets the current value of DeleteSet.
func (v1 *DSEncoderV1) ResetDsCurVal() {
	// nop
}

// WriteDsClock writes the clock value of DeleteSet.
func (v1 *DSEncoderV1) WriteDsClock(clock Number) {
	WriteVarUint(v1.RestEncoder, uint64(clock))
}

// WriteDsLen writes the length of DeleteSet.
func (v1 *DSEncoderV1) WriteDsLen(length Number) {
	WriteVarUint(v1.RestEncoder, uint64(length))
}

// WriteID writes the ID of Item.
func (v1 *UpdateEncoderV1) WriteID(id *ID) {
	WriteVarUint(v1.RestEncoder, uint64(id.Client))
	WriteVarUint(v1.RestEncoder, uint64(id.Clock))
}

// WriteLeftID writes the left ID of Item.
func (v1 *UpdateEncoderV1) WriteLeftID(id *ID) {
	v1.WriteID(id)
}

// WriteRightID writes the right ID of Item.
func (v1 *UpdateEncoderV1) WriteRightID(id *ID) {
	v1.WriteID(id)
}

// WriteClient writes the client of Item.
func (v1 *UpdateEncoderV1) WriteClient(client Number) {
	WriteVarUint(v1.RestEncoder, uint64(client))
}

// WriteInfo writes the info of Item.
func (v1 *UpdateEncoderV1) WriteInfo(info uint8) {
	WriteByte(v1.RestEncoder, info)
}

// WriteString writes the string of Item.
func (v1 *UpdateEncoderV1) WriteString(str string) error {
	return WriteString(v1.RestEncoder, str)
}

// WriteParentInfo writes the parent info of Item.
func (v1 *UpdateEncoderV1) WriteParentInfo(isYKey bool) {
	code := uint64(0)
	if isYKey {
		code = 1
	}

	WriteVarUint(v1.RestEncoder, code)
}

// WriteTypeRef writes the type ref of Item.
func (v1 *UpdateEncoderV1) WriteTypeRef(info uint8) {
	WriteVarUint(v1.RestEncoder, uint64(info))
}

// WriteLen write len of a struct - well suited for Opt RLE encoder.
func (v1 *UpdateEncoderV1) WriteLen(length Number) {
	WriteVarUint(v1.RestEncoder, uint64(length))
}

// WriteAny writes the any of Item.
func (v1 *UpdateEncoderV1) WriteAny(any any) {
	WriteAny(v1.RestEncoder, any)
}

// WriteBuf writes the buf of Item.
func (v1 *UpdateEncoderV1) WriteBuf(buf []uint8) {
	WriteVarUint8Array(v1.RestEncoder, buf)
}

// WriteJson writes the json of Item.
func (v1 *UpdateEncoderV1) WriteJson(embed interface{}) error {
	data, err := json.Marshal(embed)
	if err != nil {
		return err
	}

	return WriteString(v1.RestEncoder, string(data))
}

// WriteKey writes the key of Item.
func (v1 *UpdateEncoderV1) WriteKey(key string) error {
	return WriteString(v1.RestEncoder, key)
}

// NewUpdateEncoderV1 creates a new UpdateEncoderV1 instance.
func NewUpdateEncoderV1() *UpdateEncoderV1 {
	return &UpdateEncoderV1{
		DSEncoderV1{
			RestEncoder: new(bytes.Buffer),
		},
	}
}

// NewEncoder creates a new UpdateEncoderV1 instance.
func NewEncoder() *bytes.Buffer {
	return new(bytes.Buffer)
}
