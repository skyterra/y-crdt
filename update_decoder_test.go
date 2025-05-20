package y_crdt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

// TestWriteReadDsClock verifies encoding/decoding of DeleteSet clock values
func TestWriteReadDsClock(t *testing.T) {
	encoder := NewUpdateEncoderV1()
	originalClock := Number(12345)
	encoder.WriteDsClock(originalClock)
	data := encoder.ToUint8Array()

	decoder := NewUpdateDecoderV1(data)
	decodedClock, err := decoder.ReadDsClock()
	if err != nil {
		t.Fatalf("ReadDsClock failed: %v", err)
	}

	if decodedClock != originalClock {
		t.Errorf("DsClock mismatch: got %d, want %d", decodedClock, originalClock)
	}
}

// TestWriteReadDsLen verifies encoding/decoding of DeleteSet length values
func TestWriteReadDsLen(t *testing.T) {
	encoder := NewUpdateEncoderV1()
	originalLen := Number(67890)
	encoder.WriteDsLen(originalLen)
	data := encoder.ToUint8Array()

	decoder := NewUpdateDecoderV1(data)
	decodedLen, err := decoder.ReadDsLen()
	if err != nil {
		t.Fatalf("ReadDsLen failed: %v", err)
	}

	if decodedLen != originalLen {
		t.Errorf("DsLen mismatch: got %d, want %d", decodedLen, originalLen)
	}
}

// TestWriteReadID verifies encoding/decoding of ID structs
func TestWriteReadID(t *testing.T) {
	encoder := NewUpdateEncoderV1()
	originalID := &ID{Client: 42, Clock: 100}
	encoder.WriteID(originalID)
	data := encoder.ToUint8Array()

	decoder := NewUpdateDecoderV1(data)
	decodedID, err := decoder.ReadID()
	if err != nil {
		t.Fatalf("ReadID failed: %v", err)
	}

	if decodedID.Client != originalID.Client || decodedID.Clock != originalID.Clock {
		t.Errorf("ID mismatch: got %+v, want %+v", decodedID, originalID)
	}
}

// TestWriteReadClient verifies encoding/decoding of client numbers
func TestWriteReadClient(t *testing.T) {
	encoder := NewUpdateEncoderV1()
	originalClient := Number(99)
	encoder.WriteClient(originalClient)
	data := encoder.ToUint8Array()

	decoder := NewUpdateDecoderV1(data)
	decodedClient, err := decoder.ReadClient()
	if err != nil {
		t.Fatalf("ReadClient failed: %v", err)
	}

	if decodedClient != originalClient {
		t.Errorf("Client mismatch: got %d, want %d", decodedClient, originalClient)
	}
}

// TestWriteReadInfo verifies encoding/decoding of info bytes
func TestWriteReadInfo(t *testing.T) {
	encoder := NewUpdateEncoderV1()
	originalInfo := uint8(0xAB)
	encoder.WriteInfo(originalInfo)
	data := encoder.ToUint8Array()

	decoder := NewUpdateDecoderV1(data)
	decodedInfo, err := decoder.ReadInfo()
	if err != nil {
		t.Fatalf("ReadInfo failed: %v", err)
	}

	if decodedInfo != originalInfo {
		t.Errorf("Info mismatch: got 0x%X, want 0x%X", decodedInfo, originalInfo)
	}
}

// TestWriteReadString verifies encoding/decoding of strings
func TestWriteReadString(t *testing.T) {
	encoder := NewUpdateEncoderV1()
	originalStr := "test string content"
	if err := encoder.WriteString(originalStr); err != nil {
		t.Fatalf("WriteString failed: %v", err)
	}
	data := encoder.ToUint8Array()

	decoder := NewUpdateDecoderV1(data)
	decodedStr, err := decoder.ReadString()
	if err != nil {
		t.Fatalf("ReadString failed: %v", err)
	}

	if decodedStr != originalStr {
		t.Errorf("String mismatch: got '%s', want '%s'", decodedStr, originalStr)
	}
}

// TestWriteReadParentInfo verifies encoding/decoding of parent info flags
func TestWriteReadParentInfo(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected bool
	}{
		{"true value", true, true},
		{"false value", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewUpdateEncoderV1()
			encoder.WriteParentInfo(tt.input)
			data := encoder.ToUint8Array()

			decoder := NewUpdateDecoderV1(data)
			decoded, err := decoder.ReadParentInfo()
			if err != nil {
				t.Fatalf("ReadParentInfo failed: %v", err)
			}

			if decoded != tt.expected {
				t.Errorf("ParentInfo mismatch: got %v, want %v", decoded, tt.expected)
			}
		})
	}
}

// TestWriteReadTypeRef verifies encoding/decoding of type references
func TestWriteReadTypeRef(t *testing.T) {
	encoder := NewUpdateEncoderV1()
	originalRef := uint8(0x12)
	encoder.WriteTypeRef(originalRef)
	data := encoder.ToUint8Array()

	decoder := NewUpdateDecoderV1(data)
	decodedRef, err := decoder.ReadTypeRef()
	if err != nil {
		t.Fatalf("ReadTypeRef failed: %v", err)
	}

	if decodedRef != originalRef {
		t.Errorf("TypeRef mismatch: got 0x%X, want 0x%X", decodedRef, originalRef)
	}
}

// TestWriteReadLen verifies encoding/decoding of length values
func TestWriteReadLen(t *testing.T) {
	encoder := NewUpdateEncoderV1()
	originalLen := Number(1024)
	encoder.WriteLen(originalLen)
	data := encoder.ToUint8Array()

	decoder := NewUpdateDecoderV1(data)
	decodedLen, err := decoder.ReadLen()
	if err != nil {
		t.Fatalf("ReadLen failed: %v", err)
	}

	if decodedLen != originalLen {
		t.Errorf("Len mismatch: got %d, want %d", decodedLen, originalLen)
	}
}

// TestWriteReadAny verifies encoding/decoding of arbitrary data types
func TestWriteReadAny(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{"string type", "test any string", "test any string"},
		{"integer type", Number(42), Number(42)},
		{"boolean true", true, true},
		{"boolean false", false, false},
		{"byte array", []uint8{0x01, 0x02, 0x03}, []uint8{0x01, 0x02, 0x03}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewUpdateEncoderV1()
			encoder.WriteAny(tt.input)
			data := encoder.ToUint8Array()

			decoder := NewUpdateDecoderV1(data)
			decoded, err := decoder.ReadAny()
			if err != nil {
				t.Fatalf("ReadAny failed: %v", err)
			}

			if fmt.Sprintf("%v", decoded) != fmt.Sprintf("%v", tt.expected) {
				t.Errorf("Any mismatch: got %v, want %v", decoded, tt.expected)
			}
		})
	}
}

// TestWriteReadBuf verifies encoding/decoding of byte buffers
func TestWriteReadBuf(t *testing.T) {
	encoder := NewUpdateEncoderV1()
	originalBuf := []uint8{0x01, 0x02, 0x03, 0x04}
	encoder.WriteBuf(originalBuf)
	data := encoder.ToUint8Array()

	decoder := NewUpdateDecoderV1(data)
	decodedBuf, err := decoder.ReadBuf()
	if err != nil {
		t.Fatalf("ReadBuf failed: %v", err)
	}

	if !bytes.Equal(decodedBuf, originalBuf) {
		t.Errorf("Buf mismatch: got %v, want %v", decodedBuf, originalBuf)
	}
}

// TestWriteReadJson verifies encoding/decoding of JSON objects
func TestWriteReadJson(t *testing.T) {
	type TestStruct struct {
		Key   string
		Value int
	}
	originalObj := TestStruct{Key: "test", Value: 123}

	encoder := NewUpdateEncoderV1()
	if err := encoder.WriteJson(originalObj); err != nil {
		t.Fatalf("WriteJson failed: %v", err)
	}
	data := encoder.ToUint8Array()

	decoder := NewUpdateDecoderV1(data)
	decodedObj, err := decoder.ReadJson()
	if err != nil {
		t.Fatalf("ReadJson failed: %v", err)
	}

	// Convert both to JSON for comparison
	originalJSON, _ := json.Marshal(originalObj)
	decodedJSON, _ := json.Marshal(decodedObj)
	if string(originalJSON) != string(decodedJSON) {
		t.Errorf("Json mismatch: got %s, want %s", decodedJSON, originalJSON)
	}
}

// TestWriteReadKey verifies encoding/decoding of key strings
func TestWriteReadKey(t *testing.T) {
	encoder := NewUpdateEncoderV1()
	originalKey := "test_key_123"
	if err := encoder.WriteKey(originalKey); err != nil {
		t.Fatalf("WriteKey failed: %v", err)
	}
	data := encoder.ToUint8Array()

	decoder := NewUpdateDecoderV1(data)
	decodedKey, err := decoder.ReadKey()
	if err != nil {
		t.Fatalf("ReadKey failed: %v", err)
	}

	if decodedKey != originalKey {
		t.Errorf("Key mismatch: got '%s', want '%s'", decodedKey, originalKey)
	}
}
