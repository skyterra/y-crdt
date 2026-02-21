package y_crdt

import (
	"fmt"
	"sort"
	"strings"
)

// YXmlText Represents text in a Dom Element. In the future this type will also handle
// simple formatting information like bold and italic.
type YXmlText struct {
	YText
}

func (y *YXmlText) GetNextSibling() IAbstractType {
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

func (y *YXmlText) GetPreSibling() IAbstractType {
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

func (y *YXmlText) Copy() IAbstractType {
	return NewYXmlText()
}

func (y *YXmlText) Clone() IAbstractType {
	text := NewYXmlText()
	text.ApplyDelta(y.ToDelta(nil, nil, nil), true)
	return text
}

// not supported yet.
func (y *YXmlText) ToDOM() {

}

func (y *YXmlText) ToString() string {
	delta := y.ToDelta(nil, nil, nil)
	var list []string
	for _, data := range delta {
		var nestedNodes []Object
		for nodeName, _ := range data.Attributes {
			var attrs []Object

			nodeAttrs, ok := data.Attributes[nodeName].(Object)
			if ok {
				for key, _ := range nodeAttrs {
					attrs = append(attrs, Object{"key": key, "value": nodeAttrs[key]})
				}

				// sort attributes to get a unique order
				sort.Slice(attrs, func(i, j int) bool {
					return attrs[i]["key"].(string) < attrs[j]["key"].(string)
				})

				nestedNodes = append(nestedNodes, Object{"nodeName": nodeName, "attrs": attrs})
			}
		}

		// sort node order to get a unique order
		sort.Slice(nestedNodes, func(i, j int) bool {
			return nestedNodes[i]["nodeName"].(string) < nestedNodes[j]["nodeName"].(string)
		})

		// now convert to dom string
		var str string
		for i := 0; i < len(nestedNodes); i++ {
			node := nestedNodes[i]
			str = fmt.Sprintf(`%s<%s`, str, node["nodeName"])

			attrs, ok := node["attrs"].([]Object)
			if ok {
				for j := 0; j < len(attrs); j++ {
					attr := attrs[j]
					value, _ := attr["value"].(string)
					str = fmt.Sprintf(`%s %s="%s"`, str, attr["key"], value)
				}

				str = fmt.Sprintf(`%s>`, str)
			}
		}

		str = fmt.Sprintf("%s%v", str, data.Insert)
		for i := len(nestedNodes) - 1; i >= 0; i-- {
			str = fmt.Sprintf(`%s</%s>`, str, nestedNodes[i]["nodeName"])
		}

		list = append(list, str)
	}

	return strings.Join(list, "")
}

func (y *YXmlText) ToJSON() string {
	return y.ToString()
}

func (y *YXmlText) Write(encoder *UpdateEncoderV1) {
	encoder.WriteTypeRef(YXmlTextRefID)
}

func NewYXmlText() *YXmlText {
	yText := YText{}
	yText.EH = NewEventHandler()
	yText.DEH = NewEventHandler()
	return &YXmlText{
		YText: yText,
	}
}
