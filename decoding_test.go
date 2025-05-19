package y_crdt

import (
	"bytes"
	"math"
	"testing"
)

func TestHasCOntent(t *testing.T) {
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
