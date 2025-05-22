package y_crdt

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
)

type UpdateDecoderV1 struct {
	RestDecoder *bytes.Buffer
}

// ResetDsCurVal resets the current value of DeleteSet.
func (v1 *UpdateDecoderV1) ResetDsCurVal() {
	// nop
}

// ReadDsClock reads the clock value of DeleteSet.
func (v1 *UpdateDecoderV1) ReadDsClock() (Number, error) {
	number, err := binary.ReadUvarint(v1.RestDecoder)
	if err != nil {
		return 0, err
	}

	return Number(number), nil
}

// ReadDsLen reads the length of DeleteSet.
func (v1 *UpdateDecoderV1) ReadDsLen() (Number, error) {
	number, err := binary.ReadUvarint(v1.RestDecoder)
	if err != nil {
		return 0, err
	}

	return Number(number), nil
}

// ReadID reads the ID of Item.
func (v1 *UpdateDecoderV1) ReadID() (*ID, error) {
	client, err := binary.ReadUvarint(v1.RestDecoder)
	if err != nil {
		return nil, err
	}

	clock, err := binary.ReadUvarint(v1.RestDecoder)
	if err != nil {
		return nil, err
	}

	return &ID{Client: Number(client), Clock: Number(clock)}, nil
}

// ReadLeftID reads the left ID of Item.
func (v1 *UpdateDecoderV1) ReadLeftID() (*ID, error) {
	return v1.ReadID()
}

// ReadRightID reads the right ID of Item.
func (v1 *UpdateDecoderV1) ReadRightID() (*ID, error) {
	return v1.ReadID()
}

// ReadClient reads the client of Item.
func (v1 *UpdateDecoderV1) ReadClient() (Number, error) {
	number, err := binary.ReadUvarint(v1.RestDecoder)
	if err != nil {
		return 0, err
	}

	return Number(number), err
}

// ReadInfo reads the info of Item.
func (v1 *UpdateDecoderV1) ReadInfo() (uint8, error) {
	buf := make([]byte, 1)
	_, err := io.ReadFull(v1.RestDecoder, buf)
	if err != nil {
		return 0, err
	}

	return buf[0], nil
}

// ReadString reads the string of Item.
func (v1 *UpdateDecoderV1) ReadString() (string, error) {
	return ReadString(v1.RestDecoder)
}

// ReadParentInfo reads the parent info of Item.
func (v1 *UpdateDecoderV1) ReadParentInfo() (bool, error) {
	info, err := binary.ReadUvarint(v1.RestDecoder)
	if err != nil {
		return false, err
	}

	return info == 1, nil
}

// ReadTypeRef reads the type ref of Item.
func (v1 *UpdateDecoderV1) ReadTypeRef() (uint8, error) {
	ref, err := binary.ReadUvarint(v1.RestDecoder)
	if err != nil {
		return 0, err
	}

	return uint8(ref), nil
}

// ReadLen reads the length of Item.
func (v1 *UpdateDecoderV1) ReadLen() (Number, error) {
	length, err := binary.ReadUvarint(v1.RestDecoder)
	if err != nil {
		return 0, err
	}

	return Number(length), nil
}

// ReadAny reads the any of Item.
func (v1 *UpdateDecoderV1) ReadAny() (any, error) {
	return ReadAny(v1.RestDecoder)
}

// ReadBuf reads the buf of Item.
func (v1 *UpdateDecoderV1) ReadBuf() ([]uint8, error) {
	data, err := ReadVarUint8Array(v1.RestDecoder)
	if err != nil {
		return nil, err
	}

	return data.([]uint8), nil
}

// ReadJson reads the json of Item.
func (v1 *UpdateDecoderV1) ReadJson() (interface{}, error) {
	data, err := v1.ReadString()
	if err != nil {
		return nil, err
	}

	var obj any
	err = json.Unmarshal([]byte(data), &obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

// ReadKey reads the key of Item.
func (v1 *UpdateDecoderV1) ReadKey() (string, error) {
	return v1.ReadString()
}

// NewUpdateDecoderV1 creates a new UpdateDecoderV1.
func NewUpdateDecoderV1(buf []byte) *UpdateDecoderV1 {
	return &UpdateDecoderV1{
		RestDecoder: bytes.NewBuffer(buf),
	}
}

// NewDecoder creates a new decoder.
func NewDecoder(buf []byte) *bytes.Buffer {
	return bytes.NewBuffer(buf)
}
