package y_crdt

import (
	"bytes"
	"math"
	"testing"
)

func TestHasContent(t *testing.T) {
	decoder := bytes.NewBuffer([]byte{0x7F})
	if !hasContent(decoder) {
		t.Errorf("Expected decoder to have content, but it did not")
	}
}

func TestReadVarUnit(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	WriteVarUint(encoder, 255)

	decoder := bytes.NewBuffer(encoder.Bytes())
	value, err := readVarUint(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if value.(uint64) != 255 {
		t.Errorf("Expected value to be 255, got %d", value)
	}

	encoder = bytes.NewBuffer(nil)
	WriteVarUint(encoder, 256)
	decoder = bytes.NewBuffer(encoder.Bytes())
	value = ReadVarUint(decoder)
	if value.(uint64) != 256 {
		t.Errorf("Expected value to be 256, got %d", value)
	}
}

func TestReadUnit8(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	WriteByte(encoder, 128)
	decoder := bytes.NewBuffer(encoder.Bytes())
	value, err := ReadUint8(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value != 128 {
		t.Errorf("Expected value to be 128, got %d", value)
	}
}

func TestReadVarInt(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	WriteVarInt(encoder, 128)
	decoder := bytes.NewBuffer(encoder.Bytes())
	value, err := ReadVarInt(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if value.(int) != 128 {
		t.Errorf("Expected value to be 128, got %d", value)
	}

	encoder = bytes.NewBuffer(nil)
	WriteVarInt(encoder, -128)
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadVarInt(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(int) != -128 {
		t.Errorf("Expected value to be -128, got %d", value)
	}

	encoder = bytes.NewBuffer(nil)
	WriteVarInt(encoder, math.MaxInt32)
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadVarInt(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(int) != math.MaxInt32 {
		t.Errorf("Expected value to be %d, got %d", math.MaxInt32, value)
	}

	encoder = bytes.NewBuffer(nil)
	WriteVarInt(encoder, 64)
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadVarInt(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(int) != 64 {
		t.Errorf("Expected value to be 64, got %d", value)
	}

	encoder = bytes.NewBuffer(nil)
	WriteVarInt(encoder, 63)
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadVarInt(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(int) != 63 {
		t.Errorf("Expected value to be 63, got %d", value)
	}
}

func TestReadFloat32(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	WriteFloat32(encoder, 1.0)
	decoder := bytes.NewBuffer(encoder.Bytes())
	value, err := ReadFloat32(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if value.(float32) != 1.0 {
		t.Errorf("Expected value to be 1.0, got %f", value)
	}
}

func TestReadFloat64(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	WriteFloat64(encoder, 1.0)
	decoder := bytes.NewBuffer(encoder.Bytes())
	value, err := ReadFloat64(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(float64) != 1.0 {
		t.Errorf("Expected value to be 1.0, got %f", value)
	}
}

func TestReadBigInt64(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	WriteInt64(encoder, math.MaxInt64)
	decoder := bytes.NewBuffer(encoder.Bytes())
	value, err := ReadBigInt64(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(int64) != math.MaxInt64 {
		t.Errorf("Expected value to be %d, got %d", math.MaxInt64, value)
	}
}

func TestReadVarString(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	WriteString(encoder, "hello")

	decoder := bytes.NewBuffer(encoder.Bytes())
	value, err := ReadVarString(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if value.(string) != "hello" {
		t.Errorf("Expected value to be 'hello', got '%s'", value)
	}

	encoder = bytes.NewBuffer(nil)
	WriteString(encoder, "")
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadVarString(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(string) != "" {
		t.Errorf("Expected value to be '', got '%s'", value)
	}
}

func TestReadString(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	WriteString(encoder, "hello")
	decoder := bytes.NewBuffer(encoder.Bytes())
	value, err := ReadString(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value != "hello" {
		t.Errorf("Expected value to be 'hello', got '%s'", value)
	}

	encoder = bytes.NewBuffer(nil)
	WriteString(encoder, "")
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadString(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value != "" {
		t.Errorf("Expected value to be '', got '%s'", value)
	}
}

func TestReadObject(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	obj := NewObject()
	obj["hello"] = "world"
	WriteObject(encoder, obj)
	decoder := bytes.NewBuffer(encoder.Bytes())
	value, err := ReadObject(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(Object)["hello"] != "world" {
		t.Errorf("Expected value to be 'world', got '%s'", value)
	}

	encoder = bytes.NewBuffer(nil)
	WriteObject(encoder, NewObject())
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadObject(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	obj = value.(Object)
	if len(obj) != 0 {
		t.Errorf("Expected value to be empty, got '%d'", len(obj))
	}
}

func TestReadArray(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	array := []any{"hello", "world"}
	WriteArray(encoder, array)

	decoder := bytes.NewBuffer(encoder.Bytes())
	value, err := ReadArray(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(ArrayAny)[0] != "hello" {
		t.Errorf("Expected value to be 'hello', got '%s'", value)
	}
	if value.(ArrayAny)[1] != "world" {
		t.Errorf("Expected value to be 'world', got '%s'", value)
	}

	encoder = bytes.NewBuffer(nil)
	WriteArray(encoder, []any{})
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadArray(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(value.(ArrayAny)) != 0 {
		t.Errorf("Expected value to be empty, got '%d'", len(value.(ArrayAny)))
	}
}

func TestReadVarUint8Array(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	array := []uint8{1, 2, 3}
	WriteVarUint8Array(encoder, array)
	decoder := bytes.NewBuffer(encoder.Bytes())
	value, err := ReadVarUint8Array(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.([]uint8)[0] != 1 {
		t.Errorf("Expected value to be 1, got '%d'", value)
	}
	if value.([]uint8)[1] != 2 {
		t.Errorf("Expected value to be 2, got '%d'", value)
	}
	if value.([]uint8)[2] != 3 {
		t.Errorf("Expected value to be 3, got '%d'", value)
	}

	encoder = bytes.NewBuffer(nil)
	WriteVarUint8Array(encoder, []uint8{})
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadVarUint8Array(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(value.([]uint8)) != 0 {
		t.Errorf("Expected value to be empty, got '%d'", len(value.([]uint8)))
	}
}

func TestReadAny(t *testing.T) {
	// write Undefined value.
	encoder := bytes.NewBuffer(nil)
	WriteAny(encoder, nil)
	decoder := bytes.NewBuffer(encoder.Bytes())
	value, err := ReadAny(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	_, ok := value.(UndefinedType)
	if !ok {
		t.Errorf("Expected value to be undefined, got '%v'", value)
	}

	encoder = bytes.NewBuffer(nil)
	WriteAny(encoder, Undefined)
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadAny(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	_, ok = value.(UndefinedType)
	if !ok {
		t.Errorf("Expected value to be undefined, got '%v'", value)
	}

	// write Null value.
	encoder = bytes.NewBuffer(nil)
	WriteAny(encoder, Null)
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadAny(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	_, ok = value.(NullType)
	if !ok {
		t.Errorf("Expected value to be null, got '%v'", value)
	}

	encoder = bytes.NewBuffer(nil)
	var a *int
	WriteAny(encoder, a)
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadAny(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	_, ok = value.(NullType)
	if !ok {
		t.Errorf("Expected value to be null, got '%v'", value)
	}

	// write string value.
	encoder = bytes.NewBuffer(nil)
	WriteAny(encoder, "hello")
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadAny(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(string) != "hello" {
		t.Errorf("Expected value to be 'hello', got '%s'", value)
	}

	// write int8 value.
	encoder = bytes.NewBuffer(nil)
	WriteAny(encoder, int8(math.MaxInt8))
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadAny(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(Number) != math.MaxInt8 {
		t.Errorf("Expected value to be %d, got %d", math.MaxInt8, value)
	}

	// write int16 value.
	encoder = bytes.NewBuffer(nil)
	WriteAny(encoder, int16(math.MaxInt16))
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadAny(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(Number) != math.MaxInt16 {
		t.Errorf("Expected value to be %d, got %d", math.MaxInt16, value)
	}

	// write int64 value.
	encoder = bytes.NewBuffer(nil)
	WriteAny(encoder, int64(math.MaxInt64))
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadAny(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(int64) != math.MaxInt64 {
		t.Errorf("Expected value to be %d, got %d", math.MaxInt64, value)
	}

	// write number value.
	encoder = bytes.NewBuffer(nil)
	WriteAny(encoder, math.MaxInt)
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadAny(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(Number) != math.MaxInt {
		t.Errorf("Expected value to be %d, got %d", math.MaxInt, value)
	}

	// write float32 value.
	encoder = bytes.NewBuffer(nil)
	WriteAny(encoder, float32(1.0))
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadAny(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(float32) != 1.0 {
		t.Errorf("Expected value to be 1.0, got '%f'", value)
	}

	// write float64 value.
	encoder = bytes.NewBuffer(nil)
	WriteAny(encoder, 1.0)
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadAny(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(float64) != 1.0 {
		t.Errorf("Expected value to be 1.0, got '%f'", value)
	}

	// write boolean value.
	encoder = bytes.NewBuffer(nil)
	WriteAny(encoder, false)
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadAny(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(bool) != false {
		t.Errorf("Expected value to be false, got '%t'", value)
	}

	encoder = bytes.NewBuffer(nil)
	WriteAny(encoder, true)
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadAny(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(bool) != true {
		t.Errorf("Expected value to be true, got '%t'", value)
	}

	// write uint8 array value.
	encoder = bytes.NewBuffer(nil)
	WriteAny(encoder, []uint8{1, 2, 3})
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadAny(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.([]uint8)[0] != 1 {
		t.Errorf("Expected value to be 1, got '%d'", value)
	}
	if value.([]uint8)[1] != 2 {
		t.Errorf("Expected value to be 2, got '%d'", value)
	}
	if value.([]uint8)[2] != 3 {
		t.Errorf("Expected value to be 3, got '%d'", value)
	}

	// write object array value.
	encoder = bytes.NewBuffer(nil)
	WriteAny(encoder, []any{"hello", "world"})
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadAny(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(ArrayAny)[0] != "hello" {
		t.Errorf("Expected value to be 'hello', got '%s'", value)
	}
	if value.(ArrayAny)[1] != "world" {
		t.Errorf("Expected value to be 'world', got '%s'", value)
	}

	encoder = bytes.NewBuffer(nil)
	WriteAny(encoder, map[string]any{"hello": "world"})
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadAny(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(Object)["hello"] != "world" {
		t.Errorf("Expected value to be 'world', got '%s'", value)
	}

	// write object value.
	encoder = bytes.NewBuffer(nil)
	obj := NewObject()
	obj["hello"] = "world"
	WriteAny(encoder, obj)
	decoder = bytes.NewBuffer(encoder.Bytes())
	value, err = ReadAny(decoder)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value.(Object)["hello"] != "world" {
		t.Errorf("Expected value to be 'world', got '%s'", value)
	}
}
