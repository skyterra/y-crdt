package y_crdt

import (
	"bytes"
	"math"
	"testing"
)

func TestWriteByte(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	WriteByte(encoder, 128)

	if encoder.Len() != 1 {
		t.Errorf("Expected buffer length to be 1, got %d", encoder.Len())
	}

	expected := []byte{128}
	if !bytes.Equal(encoder.Bytes(), expected) {
		t.Errorf("Expected buffer to be %v, got %v", expected, encoder.Bytes())
	}
}

func TestWriteVarUnit8Array(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	buf := []uint8{1, 2, 3, 4, 5}
	WriteVarUint8Array(encoder, buf)

	if encoder.Len() != 6 {
		t.Errorf("Expected buffer length to be 5, got %d", encoder.Len())
	}

	expected := []byte{5, 1, 2, 3, 4, 5} // this first byte is the length of the array
	if !bytes.Equal(encoder.Bytes(), expected) {
		t.Errorf("Expected buffer to be %v, got %v", expected, encoder.Bytes())
	}
}

func TestWriteUint8Array(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	buf := []uint8{1, 2, 3, 4}

	WriteUint8Array(encoder, buf)
	if encoder.Len() != 4 {
		t.Errorf("Expected buffer length to be 4, got %d", encoder.Len())
	}

	expected := []byte{1, 2, 3, 4}
	if !bytes.Equal(encoder.Bytes(), expected) {
		t.Errorf("Expected buffer to be %v, got %v", expected, encoder.Bytes())
	}
}

func TestWriteVarUint(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	WriteVarUint(encoder, 128)
	if encoder.Len() != 2 {
		t.Errorf("Expected buffer length to be 2, got %d", encoder.Len())
	}

	expected := []byte{128, 1}
	if !bytes.Equal(encoder.Bytes(), expected) {
		t.Errorf("Expected buffer to be %v, got %v", expected, encoder.Bytes())
	}

	encoder.Reset()
	WriteVarUint(encoder, math.MaxUint64)
	if encoder.Len() != 10 {
		t.Errorf("Expected buffer length to be 8, got %d", encoder.Len())
	}

	expected = []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 1}
	if !bytes.Equal(encoder.Bytes(), expected) {
		t.Errorf("Expected buffer to be %v, got %v", expected, encoder.Bytes())
	}
}

func TestWriteVarInt(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	WriteVarInt(encoder, 128)
	if encoder.Len() != 2 {
		t.Errorf("Expected buffer length to be 2, got %d", encoder.Len())
	}
	expected := []byte{128, 2}
	if !bytes.Equal(encoder.Bytes(), expected) {
		t.Errorf("Expected buffer to be %v, got %v", expected, encoder.Bytes())
	}
	encoder.Reset()
	WriteVarInt(encoder, -128)
	if encoder.Len() != 2 {
		t.Errorf("Expected buffer length to be 2, got %d", encoder.Len())
	}

	expected = []byte{192, 2}
	if !bytes.Equal(encoder.Bytes(), expected) {
		t.Errorf("Expected buffer to be %v, got %v", expected, encoder.Bytes())
	}
}

func TestWriteFloat32(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	WriteFloat32(encoder, 1.0)
	if encoder.Len() != 4 {
		t.Errorf("Expected buffer length to be 4, got %d", encoder.Len())
	}
	expected := []byte{63, 128, 0, 0}
	if !bytes.Equal(encoder.Bytes(), expected) {
		t.Errorf("Expected buffer to be %v, got %v", expected, encoder.Bytes())
	}
}

func TestWriteFloat64(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	WriteFloat64(encoder, 1.0)
	if encoder.Len() != 8 {
		t.Errorf("Expected buffer length to be 8, got %d", encoder.Len())
	}
	expected := []byte{63, 240, 0, 0, 0, 0, 0, 0}
	if !bytes.Equal(encoder.Bytes(), expected) {
		t.Errorf("Expected buffer to be %v, got %v", expected, encoder.Bytes())
	}
}

func TestWriteInt64(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	WriteInt64(encoder, 1)
	if encoder.Len() != 8 {
		t.Errorf("Expected buffer length to be 8, got %d", encoder.Len())
	}
	expected := []byte{0, 0, 0, 0, 0, 0, 0, 1}
	if !bytes.Equal(encoder.Bytes(), expected) {
		t.Errorf("Expected buffer to be %v, got %v", expected, encoder.Bytes())
	}
}

func TestWriteString(t *testing.T) {
	encoder := bytes.NewBuffer(nil)
	WriteString(encoder, "hello")
	if encoder.Len() != 6 {
		t.Errorf("Expected buffer length to be 6, got %d", encoder.Len())
	}
	expected := []byte{5, 104, 101, 108, 108, 111}
	if !bytes.Equal(encoder.Bytes(), expected) {
		t.Errorf("Expected buffer to be %v, got %v", expected, encoder.Bytes())
	}
}
