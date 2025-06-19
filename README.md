# y-crdt
A Golang implementation of the [Yjs](https://github.com/yjs/yjs) algorithms, designed to serve as a robust backend server for multi-terminal document collaboration. This implementation enhances real-time collaboration experiences across diverse user scenarios by efficiently merging updates from various terminals, extracting differential data, and supporting data archiving.

In the future, we plan to develop a complete y-server service that synchronizes data with client terminals via WebSocket. Stay tuned for updates!

# Compatibility test
Test cases are implemented in compatibility_test.go , focusing on validating cross-version and cross-language compatibility with Yjs.   
Note: Encoder/decoder v2 support is pending development.
```go

func TestTextInsertDelete(t *testing.T) {
	// Generated via:
	//     ```js
	//        const doc = new Y.Doc()
	//        const ytext = doc.getText('type')
	//        doc..transact_mut()(function () {
	//            ytext.insert(0, 'def')
	//            ytext.insert(0, 'abc')
	//            ytext.insert(6, 'ghi')
	//            ytext.delete(2, 5)
	//        })
	//        const update = Y.encodeStateAsUpdate(doc)
	//        ytext.toString() // => 'abhi'
	//     ```
	//
	//     This way we confirm that we can decode and apply:
	//     1. blocks without left/right origin consisting of multiple characters
	//     2. blocks with left/right origin consisting of multiple characters
	//     3. delete sets

	// construct doc by golang and check to see if the result is the same as the expected.
	doc := NewDoc("guid", false, nil, nil, false)
	ytext := doc.GetText("type")
	doc.Transact(func(trans *Transaction) {
		ytext.Insert(0, "def", nil)
		ytext.Insert(0, "abc", nil)
		ytext.Insert(6, "ghi", nil)
		ytext.Delete(2, 5)
	}, nil)

	if ytext.ToString() != "abhi" {
		t.Error("expected abhi, got ", ytext.ToString())
	}
	t.Logf("construct by golang, ytext is %s", ytext.ToString())

	// apply the update and check to see if the result is the same as the expected.
	var jsUpdate = []byte{
		1, 5, 152, 234, 173, 126, 0, 1, 1, 4, 116, 121, 112, 101, 3, 68, 152, 234, 173, 126, 0, 2,
		97, 98, 193, 152, 234, 173, 126, 4, 152, 234, 173, 126, 0, 1, 129, 152, 234, 173, 126, 2,
		1, 132, 152, 234, 173, 126, 6, 2, 104, 105, 1, 152, 234, 173, 126, 2, 0, 3, 5, 2,
	}

	doc = NewDoc("guid", false, nil, nil, false)
	doc.Transact(func(trans *Transaction) {
		ApplyUpdate(doc, jsUpdate, nil)
	}, nil)

	ytext = doc.GetText("type")
	if ytext.ToString() != "abhi" {
		t.Errorf("expected abhi, got %s", ytext.ToString())
	}
	t.Logf("after apply update, ytext is %s", ytext.ToString())
}

func TestMapSet(t *testing.T) {
	//  Generated via:
	//     ```js
	//        const doc = new Y.Doc()
	//        const x = doc.getMap('test')
	//        x.set('k1', 'v1')
	//        x.set('k2', 'v2')
	//        const payload_v1 = Y.encodeStateAsUpdate(doc)
	//        console.log(payload_v1);
	//        const payload_v2 = Y.encodeStateAsUpdateV2(doc)
	//        console.log(payload_v2);
	//     ```

	// construct doc by golang and check to see if the result is the same as the expected.
	doc := NewDoc("guid", false, nil, nil, false)
	x := doc.GetMap("test").(*YMap)
	doc.Transact(func(trans *Transaction) {
		x.Set("k1", "v1")
		x.Set("k2", "v2")
	}, nil)

	content, err := json.Marshal(x.ToJson())
	t.Logf("construct by golang, x is %s, err is %v", content, err)

	// apply the update(v1) and check to see if the result is the same as the expected.
	var payload = []byte{
		1, 2, 241, 204, 241, 209, 1, 0, 40, 1, 4, 116, 101, 115, 116, 2, 107, 49, 1, 119, 2, 118,
		49, 40, 1, 4, 116, 101, 115, 116, 2, 107, 50, 1, 119, 2, 118, 50, 0,
	}
	doc = NewDoc("guid", false, nil, nil, false)
	doc.Transact(func(trans *Transaction) {
		ApplyUpdate(doc, payload, nil)
	}, nil)

	content, err = json.Marshal(doc.GetMap("test").ToJson())
	t.Logf("after apply update, x is %s, err is %v", content, err)

	// decoder v2 not support yet.
}

```


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

