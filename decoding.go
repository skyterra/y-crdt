package y_crdt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

/*
 * Encoding table:
 | Data Type           | Prefix | Encoding Method    | Comment                                                                              |
 | ------------------- | ------ | ------------------ | ------------------------------------------------------------------------------------ |
 | undefined           |    127 |                    | Functions, symbol, and everything that cannot be identified is encoded as undefined  |
 | null                |    126 |                    |                                                                                      |
 | integer             |    125 | WriteVarInt        | Only encodes 32 bit signed integers                                                  |
 | float32             |    124 | WriteFloat32       |                                                                                      |
 | float64             |    123 | WriteFloat64       |                                                                                      |
 | bigint              |    122 | writeBigInt64      |                                                                                      |
 | boolean (false)     |    121 |                    | True and false are different data types so we save the following byte                |
 | boolean (true)      |    120 |                    | - 0b01111000 so the last bit determines whether true or false                        |
 | string              |    119 | writeVarString     |                                                                                      |
 | object<string,any>  |    118 | custom             | Writes {length} then {length} key-value pairs                                        |
 | array<any>          |    117 | custom             | Writes {length} then {length} json values                                            |
 | Uint8Array          |    116 | WriteVarUint8Array | We use Uint8Array for any kind of binary data                                        |
*/

var overflow = errors.New("binary: varint overflows a 64-bit integer")

var ReadAnyLookupTable = []func(decoder *bytes.Buffer) (any, error){
	undefined,     // CASE 127: undefined
	null,          // CASE 126: null
	ReadVarInt,    // CASE 125: integer
	ReadFloat32,   // CASE 124: float32
	ReadFloat64,   // CASE 123: float64
	ReadBigInt64,  // CASE 122: bigint
	readFalse,     // CASE 121: boolean (false)
	readTrue,      // CASE 120: boolean (true)
	ReadVarString, // CASE 119: string
}

func init() {
	ReadAnyLookupTable = append(ReadAnyLookupTable, ReadObject)        // CASE 118: object<string,any>
	ReadAnyLookupTable = append(ReadAnyLookupTable, ReadArray)         // CASE 117: array<any>
	ReadAnyLookupTable = append(ReadAnyLookupTable, ReadVarUint8Array) // CASE 116: Uint8Array
}

// undefined returns the Undefined constant indicating an undefined value.
func undefined(decoder *bytes.Buffer) (any, error) {
	return Undefined, nil
}

// null returns the Null constant indicating a null value.
func null(decoder *bytes.Buffer) (any, error) {
	return Null, nil
}

// hasContent checks if the decoder has any content to read.
func hasContent(decoder *bytes.Buffer) bool {
	return decoder.Len() > 0
}

// readVarUint decodes a variable-length unsigned integer (Uvarint) from the decoder buffer.
func readVarUint(decoder *bytes.Buffer) (any, error) {
	number, err := binary.ReadUvarint(decoder)
	if err != nil {
		return uint64(0), err
	}

	return number, nil
}

// readFalse returns the boolean false value.
func readFalse(decoder *bytes.Buffer) (any, error) {
	return false, nil
}

// readTrue returns the boolean true value.
func readTrue(decoder *bytes.Buffer) (any, error) {
	return true, nil
}

// ReadUint8 reads and returns a single uint8 from the decoder buffer.
func ReadUint8(decoder *bytes.Buffer) (uint8, error) {
	data, err := decoder.ReadByte()
	if err != nil {
		return 0, err
	}

	return data, err
}

// ReadVarInt reads and returns a varint-encoded integer from the decoder buffer.
func ReadVarInt(decoder *bytes.Buffer) (any, error) {
	data, err := decoder.ReadByte()
	if err != nil {
		return nil, err
	}

	// read the low 6 bits of the byte, the low 6 bits are the number.
	number := data & BITS6

	// the first bit is the sign bit, if the first bit is 1, then the number is negative, otherwise it is positive.
	sign := 1
	if data&BIT7 > 0 {
		sign = -1
	}

	// the next_flag is the 8th bit, if the next_flag is 0, then the number is done
	if data&BIT8 == 0 {
		return sign * Number(number), nil
	}

	n := uint64(number)
	s := uint(6)
	for i := 0; i < binary.MaxVarintLen64; i++ {
		b, err := decoder.ReadByte()
		if err != nil {
			return n, err
		}

		// if the next bit is 0, then the number is done
		if b < BIT8 {
			// a 10-byte varint can represent at most a 64-bit integer,
			// where the first 9 bytes each contribute 7 bits,
			// and the 10th byte contributes 1 bit.
			if i == 9 && b > 1 {
				return n, overflow
			}

			return sign * Number(n|uint64(b)<<s), nil
		}

		n |= uint64(b&BITS7) << s
		s += 7
	}

	return sign * Number(n), overflow
}

// ReadFloat32 reads a 4-byte float32 from the decoder buffer using big-endian encoding.
func ReadFloat32(decoder *bytes.Buffer) (any, error) {
	bs := make([]byte, 4)
	_, err := decoder.Read(bs)
	if err != nil {
		return float32(0.0), err
	}

	return math.Float32frombits(binary.BigEndian.Uint32(bs)), nil
}

// ReadFloat64 reads an 8-byte float64 from the decoder buffer using big-endian encoding.
func ReadFloat64(decoder *bytes.Buffer) (any, error) {
	bs := make([]byte, 8)
	_, err := decoder.Read(bs)
	if err != nil {
		return 0.0, err
	}

	return math.Float64frombits(binary.BigEndian.Uint64(bs)), nil
}

// ReadBigInt64 reads an 8-byte int64 from the decoder buffer using big-endian encoding.
func ReadBigInt64(decoder *bytes.Buffer) (any, error) {
	buf := make([]byte, 8)
	_, err := decoder.Read(buf)
	if err != nil {
		return int64(0), err
	}

	return int64(binary.BigEndian.Uint64(buf)), nil
}

// ReadVarString decodes a variable-length string from the decoder buffer.
// First reads the string length as a Uvarint, then reads the corresponding bytes.
func ReadVarString(decoder *bytes.Buffer) (any, error) {
	size, err := binary.ReadUvarint(decoder)
	if err != nil {
		return "", err
	}

	if size == 0 {
		return "", nil
	}

	if size > uint64(decoder.Len()) {
		return "", fmt.Errorf("buffer is not enough, expected %d, got %d", size, decoder.Len())
	}

	buf := make([]byte, size)
	err = binary.Read(decoder, binary.LittleEndian, buf)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

// ReadString reads a variable-length string from the decoder buffer.
func ReadString(decoder *bytes.Buffer) (string, error) {
	size, err := binary.ReadUvarint(decoder)
	if err != nil {
		return "", err
	}

	if size == 0 {
		return "", nil
	}

	// the size is too large, it is greater than the buffer size
	if size > uint64(decoder.Len()) {
		return "", errors.New("buffer is not enough")
	}

	buf := make([]byte, size)
	_, err = io.ReadFull(decoder, buf)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

// ReadObject decodes an object<string, any> from the decoder buffer.
func ReadObject(decoder *bytes.Buffer) (any, error) {
	size, err := binary.ReadUvarint(decoder)
	if err != nil {
		return nil, err
	}

	obj := NewObject()
	if size == 0 {
		return obj, nil
	}

	for i := uint64(0); i < size; i++ {
		key, err := ReadVarString(decoder)
		if err != nil {
			return obj, err
		}

		value, err := ReadAny(decoder)
		if err != nil {
			return obj, err
		}

		obj[key.(string)] = value
	}

	return obj, nil
}

// ReadArray decodes an array<any> from the decoder buffer.
func ReadArray(decoder *bytes.Buffer) (any, error) {
	array := make(ArrayAny, 0)

	size, err := binary.ReadUvarint(decoder)
	if err != nil {
		return array, err
	}

	if size == 0 {
		return array, nil
	}

	array = make(ArrayAny, size)
	for i := uint64(0); i < size; i++ {
		value, err := ReadAny(decoder)
		if err != nil {
			return array, err
		}

		array[i] = value
	}

	return array, nil
}

// ReadVarUnit8Array decodes a Uint8Array (byte slice) from the decoder buffer.
func ReadVarUint8Array(decoder *bytes.Buffer) (any, error) {
	size, err := binary.ReadUvarint(decoder)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, size)
	decoder.Read(buf)

	return buf, nil
}

// ReadVarUint is a wrapper around readVarUint that returns the decoded uint64.
func ReadVarUint(decoder *bytes.Buffer) uint64 {
	value, _ := readVarUint(decoder)
	number, _ := value.(uint64)
	return number
}

// ReadAny is the general decoding dispatcher that uses ReadAnyLookupTable.
func ReadAny(decoder *bytes.Buffer) (any, error) {
	tag, err := ReadUint8(decoder)
	if err != nil {
		return nil, err
	}

	refID := 127 - tag
	if int(refID) >= len(ReadAnyLookupTable) {
		return nil, fmt.Errorf("index out of range. tag:%d refID:%d len:%d", tag, refID, len(ReadAnyLookupTable))
	}

	return ReadAnyLookupTable[127-tag](decoder)
}
