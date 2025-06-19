package y_crdt

// YXmlEvent An Event that describes changes on a YXml Element or Yxml Fragment
type YXmlEvent struct {
	YEvent
	ChildListChanged  bool // Whether the children changed.
	AttributesChanged Set  // Set of all changed attributes.
}

func NewYXmlEvent(target IAbstractType, subs Set, trans *Transaction) *YXmlEvent {
	e := &YXmlEvent{
		YEvent:            *NewYEvent(target, trans),
		ChildListChanged:  false,
		AttributesChanged: NewSet(),
	}

	subs.Range(func(element interface{}) {
		if element == nil {
			e.ChildListChanged = true
		} else {
			e.AttributesChanged.Add(element)
		}
	})

	return e
}
