package y_crdt

const (
	MessageSync = iota
	MessageAwareness
)

type UpdateHandler func([]byte)
type WSSharedDoc struct {
	*Doc
	Awareness *Awareness

	awarenessUpdateHandler UpdateHandler
	docUpdateHandler       UpdateHandler
}

func NewWSSharedDoc(docID string, awarenessHandler UpdateHandler, docHandler UpdateHandler) *WSSharedDoc {
	sd := &WSSharedDoc{}
	sd.Doc = NewDoc(docID, true, DefaultGCFilter, nil, false)
	sd.Awareness = NewAwareness(sd.Doc)
	sd.Awareness.SetLocalState(nil)
	sd.awarenessUpdateHandler = awarenessHandler
	sd.docUpdateHandler = docHandler

	// 意识消息广播，如鼠标同步
	sd.Awareness.On("update", NewObserverHandler(func(v ...interface{}) {
		obj := v[0].(Object)
		added := obj["added"].([]Number)
		updated := obj["updated"].([]Number)
		removed := obj["removed"].([]Number)

		changedClients := append(added, updated...)
		changedClients = append(changedClients, removed...)

		encoder := NewEncoder()
		WriteVarUint(encoder, MessageAwareness)
		WriteVarUint8Array(encoder, EncodeAwarenessUpdate(sd.Awareness, changedClients, nil))

		if sd.awarenessUpdateHandler != nil {
			sd.awarenessUpdateHandler(encoder.Bytes())
		}
	}))

	// 文档更新消息广播
	sd.Doc.On("update", NewObserverHandler(func(v ...interface{}) {
		update := v[0].([]byte)
		encoder := NewUpdateEncoderV1()
		WriteVarUint(encoder.RestEncoder, MessageSync)
		WriteUpdate(encoder, update)

		if sd.docUpdateHandler != nil {
			sd.docUpdateHandler(encoder.ToUint8Array())
		}
	}))

	return sd
}
