package y_crdt

import (
	"strings"
)

/**
 * Define the elements to which a set of CSS queries apply.
 * {@link https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Selectors|CSS_Selectors}
 *
 * @example
 *   query = '.classSelector'
 *   query = 'nodeSelector'
 *   query = '#idSelector'
 *
 * @typedef {string} CSS_Selector
 */

/**
 * Dom filter function.
 *
 * @callback domFilter
 * @param {string} nodeName The nodeName of the element
 * @param {Map} attributes The map of attributes.
 * @return {boolean} Whether to include the Dom node in the YXmlElement.
 */

/**
 * Represents a subset of the nodes of a YXmlElement / YXmlFragment and a
 * position within them.
 *
 * Can be created with {@link YXmlFragment#createTreeWalker}
 *
 * @public
 * @implements {Iterable<YXmlElement|YXmlText|YXmlElement|YXmlHook>}
 */

type YXmlTreeWalker struct {
	Filter      func() bool
	Root        interface{}
	CurrentNode *Item
	FirstCall   bool
}

type IXmlType interface {
	ToString() string
}

type YXmlFragment struct {
	AbstractType
	PrelimContent ArrayAny
}

func (y *YXmlFragment) GetFirstChild() interface{} {
	first := y.First()
	if first != nil && len(first.Content.GetContent()) > 0 {
		return first.Content.GetContent()[0]
	}

	return nil
}

// Integrate this type into the Yjs instance.
//
// Save this struct in the os
// This type is sent to other client
// Observer functions are fired
func (y *YXmlFragment) Integrate(doc *Doc, item *Item) {
	y.AbstractType.Integrate(doc, item)
	y.Insert(0, y.PrelimContent)
	y.PrelimContent = nil
}

func (y *YXmlFragment) Copy() IAbstractType {
	return NewYXmlFragment()
}

func (y *YXmlFragment) Clone() IAbstractType {
	el := NewYXmlFragment()

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

func (y *YXmlFragment) GetLength() Number {
	if y.PrelimContent == nil {
		return y.Length
	}

	return len(y.PrelimContent)
}

func (y *YXmlFragment) CreateTreeWalker(filter func(abstractType IAbstractType) bool) *YXmlTreeWalker {
	return NewYXmlTreeWalker(y, filter)
}

// 暂不支持
func (y *YXmlFragment) QuerySelector(query interface{}) {

}

// 暂不支持
func (y *YXmlFragment) QuerySelectorAll(query interface{}) {

}

// Creates YXmlEvent and calls observers.
func (y *YXmlFragment) CallObserver(trans *Transaction, parentSubs Set) {
	CallTypeObservers(y, trans, NewYXmlEvent(y, parentSubs, trans))
}

// Get the string representation of all the children of this YXmlFragment.
func (y *YXmlFragment) ToString() string {
	elements := TypeListMap(y, func(c interface{}, i Number, _ IAbstractType) interface{} {
		xml, ok := c.(IXmlType)
		if ok {
			return xml.ToString()
		}

		return ""
	})

	var data []string
	for _, element := range elements {
		str, ok := element.(string)
		if ok && str != "" {
			data = append(data, str)
		}
	}

	return strings.Join(data, "")
}

func (y *YXmlFragment) ToJson() interface{} {
	return y.ToString()
}

// 暂不支持
func (y *YXmlFragment) ToDOM() {

}

// Insert Inserts new content at an index.
//
// @example
//
//	// Insert character 'a' at position 0
//
// xml.insert(0, [new Y.XmlText('text')])
func (y *YXmlFragment) Insert(index Number, content ArrayAny) {
	if y.Doc != nil {
		Transact(y.Doc, func(trans *Transaction) {
			TypeListInsertGenerics(trans, y, index, content)
		}, nil, true)
	} else {
		SpliceArray(&y.PrelimContent, index, 0, content)
	}
}

// Inserts new content at an index.
//
// @example
//
//	// Insert character 'a' at position 0
//	xml.insert(0, [new Y.XmlText('text')])
func (y *YXmlFragment) InsertAfter(ref interface{}, content ArrayAny) {
	if y.Doc != nil {
		Transact(y.Doc, func(trans *Transaction) {
			var refItem *Item

			a, ok := ref.(IAbstractType)
			if ok {
				refItem = a.GetItem()
			} else {
				refItem, _ = ref.(*Item)
			}

			TypeListInsertGenericsAfter(trans, y, refItem, content)
		}, nil, true)
	} else {
		pc := y.PrelimContent
		index := 0

		if ref != nil {
			index = FindIndex(pc, func(e interface{}) bool {
				return e == ref
			}) + 1
		}

		if index == 0 && ref != nil {
			Logf("reference item not found")
			return
		}

		SpliceArray(&pc, index, 0, content)
	}
}

// Deletes elements starting from an index.
// Default: length = 1
func (y *YXmlFragment) Delete(index, length Number) {
	if y.Doc != nil {
		Transact(y.Doc, func(trans *Transaction) {
			TypeListDelete(trans, y, index, length)
		}, nil, true)
	} else {
		// @ts-ignore _prelimContent is defined because this is not yet integrated
		SpliceArray(&y.PrelimContent, index, length, nil)
	}
}

// Transforms this YArray to a JavaScript Array.
func (y *YXmlFragment) ToArray() ArrayAny {
	return TypeListToArray(y)
}

// Appends content to this YArray.
func (y *YXmlFragment) Push(content ArrayAny) {
	y.Insert(y.Length, content)
}

// Preppends content to this YArray.
func (y *YXmlFragment) Unshift(content ArrayAny) {
	y.Insert(0, content)
}

// Returns the i-th element from a YArray.
func (y *YXmlFragment) Get(index Number) interface{} {
	return TypeListGet(y, index)
}

// Transforms this YArray to a JavaScript Array.
// Default: start = 0
func (y *YXmlFragment) Slice(start, end Number) ArrayAny {
	return TypeListSlice(y, start, end)
}

// Transform the properties of this type to binary and write it to an
// BinaryEncoder.
//
// This is called when this Item is sent to a remote peer.
//
// @param {UpdateEncoderV1 | UpdateEncoderV2} encoder The encoder to write data to.
func (y *YXmlFragment) Write(encoder *UpdateEncoderV1) {
	encoder.WriteTypeRef(YXmlFragmentRefID)
}

func NewYXmlFragment() *YXmlFragment {
	return &YXmlFragment{}
}

func NewYXmlFragmentType() IAbstractType {
	return NewYXmlFragment()
}

// 暂不支持
func NewYXmlTreeWalker(root interface{}, f func(abstractType IAbstractType) bool) *YXmlTreeWalker {

	return nil
}
