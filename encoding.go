package y_crdt

import (
	"bytes"
	"encoding/binary"
	"math"
)

// WriteByte writes a single uint8 number to the encoder buffer.
func WriteByte(encoder *bytes.Buffer, number uint8) {
	buf := make([]byte, 1)
	buf[0] = number
	encoder.Write(buf)
}

// WriteUint8Array writes a byte array to the encoder buffer.
// The first byte is the length of the array, and the following bytes are the array elements.
func WriteVarUint8Array(encoder *bytes.Buffer, buf []uint8) {
	WriteVarUint(encoder, uint64(len(buf)))
	encoder.Write(buf)
}

// WriteUint8Array writes a byte array to the encoder buffer.
// The first byte is the length of the array, and the following bytes are the array elements.
func WriteUint8Array(encoder *bytes.Buffer, buf []uint8) {
	encoder.Write(buf)
}

// WriteVarUint writes a variable-length uint64 number to the encoder buffer.
func WriteVarUint(encoder *bytes.Buffer, number uint64) {
	buf := make([]byte, binary.MaxVarintLen64)
	size := binary.PutUvarint(buf, number)
	encoder.Write(buf[:size])
}

// WriteVarInt writes a variable-length int64 number to the encoder buffer.
func WriteVarInt(encoder *bytes.Buffer, number int) {
	bitSign := 0 // sign bit
	if number < 0 {
		number = -number
		bitSign = BIT7
	}

	// bitNext indicates whether there are more bytes to read after this one.
	// If the highest bit is 1, there are more bytes to read.
	bitNext := 0
	if number > BITS6 {
		bitNext = BIT8
	}

	buf := make([]byte, 1)
	buf[0] = uint8(bitNext|bitSign) | uint8(BITS6&number) // [next_flag sign_flag low_6_bits_data]
	encoder.Write(buf)

	number >>= 6

	for number > 0 {
		bitNext := uint8(0)
		if number > BITS7 {
			bitNext = BIT8
		}

		buf := make([]byte, 1)
		buf[0] = bitNext | uint8(BITS7&number) // [next_flag low_7_bits_data]
		encoder.Write(buf)
		number >>= 7
	}
}

// WriteFloat32 writes a 4-byte float32 to the encoder buffer using big-endian encoding.
func WriteFloat32(encoder *bytes.Buffer, f float32) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, math.Float32bits(f))
	encoder.Write(buf)
}

// WriteFloat64 writes an 8-byte float64 to the encoder buffer using big-endian encoding.
func WriteFloat64(encoder *bytes.Buffer, f float64) {
	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, math.Float64bits(f))
	encoder.Write(bs)
}

// WriteInt64 writes an 8-byte int64 to the encoder buffer using big-endian encoding.
func WriteInt64(encoder *bytes.Buffer, n int64) {
	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(n))
	encoder.Write(bs)
}

// WriteString writes a variable-length string to the encoder buffer.
func WriteString(encoder *bytes.Buffer, str string) error {
	data := []byte(str)
	size := uint64(len(data))

	WriteVarUint(encoder, size)
	return binary.Write(encoder, binary.LittleEndian, data)
}

// WriteObject writes an object to the encoder buffer.
func WriteObject(encoder *bytes.Buffer, obj Object) error {
	// wirte the object size.
	WriteVarUint(encoder, uint64(len(obj)))

	// write the object key-value pairs.
	for key, value := range obj {
		err := WriteString(encoder, key)
		if err != nil {
			return err
		}

		WriteAny(encoder, value)
	}

	return nil
}

// WriteArray writes an array(any) to the encoder buffer.
func WriteArray(encoder *bytes.Buffer, array []any) error {
	// write the array size.
	WriteVarUint(encoder, uint64(len(array)))

	// write the array elements.
	for _, value := range array {
		if err := WriteAny(encoder, value); err != nil {
			return err
		}
	}

	return nil
}

// WriteAny writes any type to the encoder buffer.
func WriteAny(encoder *bytes.Buffer, any any) error {
	if IsUndefined(any) {
		WriteByte(encoder, 127)
		return nil
	}

	if IsNull(any) {
		WriteByte(encoder, 126)
		return nil
	}

	switch v := any.(type) {
	case string:
		WriteByte(encoder, 119)
		if err := WriteString(encoder, v); err != nil {
			return err
		}
	case int8:
		WriteByte(encoder, 125)
		WriteVarInt(encoder, Number(v))
	case int16:
		WriteByte(encoder, 125)
		WriteVarInt(encoder, Number(v))
	case Number:
		WriteByte(encoder, 125)
		WriteVarInt(encoder, v)
	case int64:
		WriteByte(encoder, 122)
		WriteInt64(encoder, v)
	case float32:
		WriteByte(encoder, 124)
		WriteFloat32(encoder, v)
	case float64:
		WriteByte(encoder, 123)
		WriteFloat64(encoder, v)
	case bool:
		if v {
			WriteByte(encoder, 120)
		} else {
			WriteByte(encoder, 121)
		}
	case []uint8:
		WriteByte(encoder, 116)
		WriteVarUint8Array(encoder, v)
	case ArrayAny:
		WriteVarUint(encoder, 117)
		WriteArray(encoder, v)
	case Object:
		WriteByte(encoder, 118)
		if err := WriteObject(encoder, v); err != nil {
			return err
		}
	default:
		WriteByte(encoder, 126)
	}

	return nil
}
