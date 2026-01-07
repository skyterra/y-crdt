package y_crdt

import (
	"bytes"
	"testing"
)

func TestAwareness(t *testing.T) {
	doc1 := NewDoc("6148fcd6-9d8c-4fbd-8420-676f5931f7aa", true, DefaultGCFilter, nil, false)
	doc2 := NewDoc("6148fcd6-9d8c-4fbd-8420-676f5931f7aa", true, DefaultGCFilter, nil, false)

	doc1.ClientID = 0
	doc2.ClientID = 1

	aw1 := NewAwareness(doc1)
	aw2 := NewAwareness(doc2)

	clientID1 := aw1.ClientID
	clientID2 := aw2.ClientID

	aw1.On("update", NewObserverHandler(func(v ...interface{}) {
		clients := []Number{0}
		states := map[Number]Object{0: {"updated": "updated"}, 1: {"added": "added", "removed": "removed"}}
		enc := EncodeAwarenessUpdate(aw1, clients, states)
		ApplyAwarenessUpdate(aw2, enc, "custom")
	}))

	var lastChangeLocal interface{}
	aw1.On("change", NewObserverHandler(func(v ...interface{}) {
		lastChangeLocal = Object{"change": "change"}
	}))

	var lastChange interface{}
	aw2.On("change", NewObserverHandler(func(v ...interface{}) {
		lastChange = Object{"change": "change"}
	}))

	if lastChangeLocal != nil {
		t.Errorf("expected last change local to be nil")
	}

	if lastChange != nil {
		t.Errorf("expected last change to be nil")
	}

	aw1.SetLocalState(Object{"x": 3})
	aw1.SetLocalStateField("hello", "world")

	aw1LocalState := aw1.GetLocalState()
	aw2LocalState := aw2.GetLocalState()
	aw2State := aw2.GetStates()

	clients := []Number{0}
	states := map[Number]Object{0: Object{"updated": "updated"}, 1: Object{"added": "added", "removed": "removed"}}
	enc := EncodeAwarenessUpdate(aw1, clients, states)
	if enc == nil {
		t.Errorf("expected enc to be non-nil")
	}

	ApplyAwarenessUpdate(aw2, enc, "custom")
	RemoveAwarenessStates(aw1, clients, "timeout")

	aw1.Destroy()
	aw2.Destroy()

	if aw1LocalState == nil {
		t.Errorf("expected aw1 local to be non nil")
	}

	if aw2LocalState == nil {
		t.Errorf("expected aw2 local to be non nil")
	}

	if aw2State == nil {
		t.Errorf("expected aw2 state to be non nil")
	}

	if clientID1 != 0 {
		t.Errorf("expected clientID1 to be 0")
	}

	if clientID2 != 1 {
		t.Errorf("expected clientID2 to be 1")
	}

	if aw1 == nil {
		t.Errorf("expected aw1 to be non nil")
	}

	if aw2 == nil {
		t.Errorf("expected aw2 to be non nil")
	}
}

func TestSync(t *testing.T) {
	// ReadSyncMessage
	var mask = []byte{0x1, 0x3, 0x7, 0xf, 0x1f, 0x3f, 0x7f}
	decoder := NewUpdateDecoderV1(mask)
	encoder := NewUpdateEncoderV1()
	doc1 := NewDoc("6148fcd6-9d8c-4fbd-8420-676f5931f7aa", true, DefaultGCFilter, nil, false)

	ReadSyncMessage(decoder, encoder, doc1, "snapshot")
	if encoder == nil {
		t.Errorf("expected encoder to be non-nil")
	}

	// change mask
	mask = []byte{0x2, 0x3, 0x7, 0xf}
	decoder = NewUpdateDecoderV1(mask)
	encoder = NewUpdateEncoderV1()
	doc1 = NewDoc("6148fcd6-9d8c-4fbd-8420-676f5931f7aa", true, DefaultGCFilter, nil, false)

	ReadSyncMessage(decoder, encoder, doc1, "snapshot")
	if encoder == nil {
		t.Errorf("expected encoder to be non-nil")
	}

	// WriteSyncStep1
	encoder = NewUpdateEncoderV1()
	doc1 = NewDoc("6148fcd6-9d8c-4fbd-8420-676f5931f7aa", true, DefaultGCFilter, nil, false)

	WriteSyncStep1(encoder, doc1)
	if encoder == nil {
		t.Errorf("expected encoder to be non-nil")
	}

	// WriteSyncStep1FromUpdate
	encoder = NewUpdateEncoderV1()
	doc1 = NewDoc("6148fcd6-9d8c-4fbd-8420-676f5931f7aa", true, DefaultGCFilter, nil, false)
	WriteSyncStep1(encoder, doc1)
	if encoder == nil {
		t.Errorf("expected encoder to be non-nil")
	}

	encoder1 := NewUpdateEncoderV1()
	update := EncodeStateAsUpdate(doc1, nil)
	WriteSyncStep1FromUpdate(encoder1, update)
	if encoder == nil {
		t.Errorf("expected encoder to be non-nil")
	}

	if !bytes.Equal(encoder.RestEncoder.Bytes(), encoder1.RestEncoder.Bytes()) {
		t.Errorf("expected rest encoder to be equal")
	}

	// WriteSyncStep2FromUpdate
	encoder = NewUpdateEncoderV1()
	doc1 = NewDoc("6148fcd6-9d8c-4fbd-8420-676f5931f7aa", true, DefaultGCFilter, nil, false)
	WriteSyncStep2(encoder, doc1, nil)
	if encoder == nil {
		t.Errorf("expected encoder to be non-nil")
	}

	encoder1 = NewUpdateEncoderV1()
	update = EncodeStateAsUpdate(doc1, nil)
	WriteSyncStep2FromUpdate(encoder1, update, nil)
	if encoder == nil {
		t.Errorf("expected encoder to be non-nil")
	}

	if !bytes.Equal(encoder.RestEncoder.Bytes(), encoder1.RestEncoder.Bytes()) {
		t.Errorf("expected rest encoder to be equal")
	}

	// WriteSyncStep1
	mask = []byte{0x2, 0x3, 0x7, 0xf}
	encoder = NewUpdateEncoderV1()
	WriteUpdate(encoder, mask)
	if encoder == nil {
		t.Errorf("expected encoder to be non-nil")
	}
}
