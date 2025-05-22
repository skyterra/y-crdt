package y_crdt

// You can manage binding to a custom type with YXmlHook.
type YXmlHook struct {
	YMap
	HookName string
}

// Copy Creates an Item with the same effect as this Item (without position effect)
func (y *YXmlHook) Copy() IAbstractType {
	return NewYXmlHook(y.HookName)
}

// Clone
func (y *YXmlHook) Clone() IAbstractType {
	el := NewYXmlHook(y.HookName)
	y.ForEach(func(key string, value interface{}, yMap *YMap) {
		el.Set(key, value)
	})
	return el
}

func (y *YXmlHook) ToDOM() {

}

// Transform the properties of this type to binary and write it to an
// BinaryEncoder.
//
// This is called when this Item is sent to a remote peer.
func (y *YXmlHook) Write(encoder *UpdateEncoderV1) {
	encoder.WriteTypeRef(YXmlHookRefID)
	err := encoder.WriteKey(y.HookName)
	if err != nil {
		Logf("[crdt] %s.", err.Error())
	}
}

func NewYXmlHook(hookName string) *YXmlHook {
	h := &YXmlHook{
		YMap:     *NewYMap(nil),
		HookName: hookName,
	}
	return h
}
