package y_crdt

import (
	"fmt"
	"sort"
	"strings"
)

const DefaultNodeName = "UNDEFINED"

type YXmlElement struct {
	YXmlFragment

	PrelimAttrs map[string]interface{}
	NodeName    string
}

// GetNextSibling return {YXmlElement|YXmlText|nil}
func (y *YXmlElement) GetNextSibling() IAbstractType {
	var n *Item
	if y.Item != nil {
		n = y.Item.Next()
	}

	if n != nil {
		t, ok := n.Content.(*ContentType)
		if ok {
			return t.Type
		}
	}

	return nil
}

// GetPrevSibling return {YXmlElement|YXmlText|nil}
func (y *YXmlElement) GetPrevSibling() IAbstractType {
	var n *Item
	if y.Item != nil {
		n = y.Item.Prev()
	}

	if n != nil {
		t, ok := n.Content.(*ContentType)
		if ok {
			return t.Type
		}
	}

	return nil
}

// Integrate this type into the Yjs instance.
//
//	Save this struct in the os
//	This type is sent to other client
//	Observer functions are fired
func (y *YXmlElement) Integrate(doc *Doc, item *Item) {
	y.AbstractType.Integrate(doc, item)

	for key, value := range y.PrelimAttrs {
		y.SetAttribute(key, value)
	}

	y.PrelimAttrs = nil
}

// Copy Creates an Item with the same effect as this Item (without position effect)
func (y *YXmlElement) Copy() IAbstractType {
	return NewYXmlElement(y.NodeName)
}

func (y *YXmlElement) Clone() IAbstractType {
	el := NewYXmlElement(y.NodeName)
	attrs := y.GetAttributes()

	for key, value := range attrs {
		el.SetAttribute(key, value)
	}

	var data []interface{}
	for _, element := range y.ToArray() {
		item, ok := element.(IAbstractType)
		if ok {
			data = append(data, item.Clone())
		} else {
			data = append(data, element)
		}
	}

	el.Insert(0, data)
	return el
}

// Returns the XML serialization of this YXmlElement.
// The attributes are ordered by attribute-name, so you can easily use this
// method to compare YXmlElements
//
// @return {string} The string representation of this type.
func (y *YXmlElement) ToString() string {
	attrs := y.GetAttributes()
	var stringBuilder []string

	keys := make([]string, 0, len(attrs))
	for key, _ := range attrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		str := fmt.Sprintf("%s=\"%v\"", key, attrs[key])
		stringBuilder = append(stringBuilder, str)
	}

	nodeName := strings.ToLower(y.NodeName)

	var attrsString string
	if len(stringBuilder) > 0 {
		attrsString = fmt.Sprintf(" %s", strings.Join(stringBuilder, " "))
	}

	return fmt.Sprintf(`<%s%s>%s</%s>`, nodeName, attrsString, y.YXmlFragment.ToString(), nodeName)
}

// RemoveAttribute Removes an attribute from this YXmlElement.
func (y *YXmlElement) RemoveAttribute(attributeName string) {
	if y.Doc != nil {
		Transact(y.Doc, func(trans *Transaction) {
			TypeMapDelete(trans, y, attributeName)
		}, nil, true)
	} else {
		delete(y.PrelimAttrs, attributeName)
	}
}

// SetAttribute Sets or updates an attribute.
func (y *YXmlElement) SetAttribute(attributeName string, attributeValue interface{}) {
	if y.Doc != nil {
		Transact(y.Doc, func(trans *Transaction) {
			TypeMapSet(trans, y, attributeName, attributeValue)
		}, nil, true)
	} else {
		y.PrelimAttrs[attributeName] = attributeValue
	}
}

// GetAttribute Returns an attribute value that belongs to the attribute name.
func (y *YXmlElement) GetAttribute(attributeName string) interface{} {
	return TypeMapGet(y, attributeName)
}

// HasAttribute Returns whether an attribute exists
func (y *YXmlElement) HasAttribute(attributeName string) bool {
	return TypeMapHas(y, attributeName)
}

// GetAttributes Returns an attribute value that belongs to the attribute name.
func (y *YXmlElement) GetAttributes() Object {
	return TypeMapGetAll(y)
}

// ToDOM Creates a Dom Element that mirrors this YXmlElement.
func (y *YXmlElement) ToDOM() {

}

func (y *YXmlElement) Write(encoder *UpdateEncoderV1) {
	encoder.WriteTypeRef(YXmlElementRefID)
	err := encoder.WriteKey(y.NodeName)
	if err != nil {
		Logf("[crdt] %s.", err.Error())
	}
}

func NewYXmlElement(nodeName string) *YXmlElement {
	el := &YXmlElement{NodeName: nodeName}
	el.PrelimAttrs = make(map[string]interface{})
	return el
}
