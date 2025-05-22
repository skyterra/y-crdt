# y-crdt
Implement the [Yjs](https://github.com/yjs/yjs) algorithms in Go.

# Encoding & Decoding

## Encoding
### Basic Types
- **WriteByte**: Write `uint8` to buffer.
- **WriteVarUint**: Variable-length `uint64` encoding.
- **WriteVarInt**: Variable-length `int` encoding with sign handling.
- **WriteFloat32/64**: Big-endian float encoding.
- **WriteInt64**: Big-endian `int64` encoding.

### Composite Types
- **WriteString**: Write length + string data.
- **WriteObject**: Write key-value count, then key-value pairs.
- **WriteArray**: Write element count, then elements.
- **WriteVarUint8Array/WriteUint8Array**: Write byte arrays with/without length prefix.

### Universal
- **WriteAny**: Type-specific encoding with flag bytes.

## Decoding
### Basic Types
- **readVarUint**: Read variable-length `uint64`.
- **ReadUint8**: Read `uint8`.
- **ReadVarInt**: Restore `int` from variable-length bytes.
- **ReadFloat32/64**: Parse big-endian floats.
- **ReadBigInt64**: Parse big-endian `int64`.

### Composite Types
- **ReadString**: Read length, then string data.
- **ReadObject**: Read count, then key-value pairs.
- **ReadArray**: Read count, then elements.
- **ReadVarUint8Array**: Read length, then byte array.

### Universal
- **ReadAny**: Decode based on type flag byte.

Encoding and decoding are symmetric, using variable-length and big-endian formats for efficiency.

# Yjs Data Structures
## YMap
- **YMap**: A key-value store with efficient updates.
- **YMapItem**: Represents a key-value pair in the map.
- **YMapItemMap**: Maps keys to YMapItem pointers.
## YArray
- **YArray**: A dynamic array with efficient updates.
- **YArrayItem**: Represents an element in the array.
## YText
- **YText**: A text document with efficient updates.
- **YTextItem**: Represents a character in the text.
- **YTextItemMap**: Maps positions to YTextItem pointers.
## YXmlFragment
- **YXmlFragment**: A fragment of an XML document.   
- **YXmlFragmentItem**: Represents an XML node.
- **YXmlFragmentItemMap**: Maps positions to YXmlFragmentItem pointers.

