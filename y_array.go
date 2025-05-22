package y_crdt

// Event that describes the changes on a YArray
type YArrayEvent struct {
	YEvent
	YTrans *Transaction
}

func NewYArrayEvent(yarray *YArray, trans *Transaction) *YArrayEvent {
	y := &YArrayEvent{
		YEvent: *NewYEvent(yarray, trans),
		YTrans: trans,
	}

	return y
}

// A shared Array implementation.
type YArray struct {
	AbstractType
	PrelimContent ArrayAny
	SearchMaker   []*ArraySearchMarker
}

// Construct a new YArray containing the specified items.
func (y *YArray) From(items ArrayAny) *YArray {
	a := NewYArray()
	a.Push(items)
	return a
}

// Integrate this type into the Yjs instance.
//
//	Save this struct in the os
//	This type is sent to other client
//	Observer functions are fired
func (y *YArray) Integrate(doc *Doc, item *Item) {
	y.AbstractType.Integrate(doc, item)
	y.Insert(0, y.PrelimContent)
	y.PrelimContent = nil
}

func (y *YArray) Copy() IAbstractType {
	return NewYArray()
}

func (y *YArray) Clone() IAbstractType {
	arr := NewYArray()

	var content []interface{}
	for _, el := range y.ToArray() {
		a, ok := el.(IAbstractType)
		if ok {
			content = append(content, a.Clone())
		} else {
			content = append(content, el)
		}
	}

	arr.Insert(0, content)
	return arr
}

func (y *YArray) GetLength() Number {
	if y.PrelimContent == nil {
		return y.Length
	}

	return len(y.PrelimContent)
}

// Creates YArrayEvent and calls observers.
func (y *YArray) CallObserver(trans *Transaction, parentSubs Set) {
	y.AbstractType.CallObserver(trans, parentSubs)
	CallTypeObservers(y, trans, NewYArrayEvent(y, trans))
}

// Inserts new content at an index.
//
// Important: This function expects an array of content. Not just a content
// object. The reason for this "weirdness" is that inserting several elements
// is very efficient when it is done as a single operation.
//
//	@example
//	 // Insert character 'a' at position 0
//	 yarray.insert(0, ['a'])
//	 // Insert numbers 1, 2 at position 1
//	 yarray.insert(1, [1, 2])
func (y *YArray) Insert(index Number, content ArrayAny) {
	if y.Doc != nil {
		Transact(y.Doc, func(trans *Transaction) {
			TypeListInsertGenerics(trans, y, index, content)
		}, nil, true)
	} else {
		SpliceArray(&y.PrelimContent, index, 0, content)
	}
}

// Appends content to this YArray.
func (y *YArray) Push(content ArrayAny) {
	y.Insert(y.Length, content)
}

// Preppends content to this YArray.
func (y *YArray) Unshift(content ArrayAny) {
	y.Insert(0, content)
}

// Deletes elements starting from an index.
func (y *YArray) Delete(index, length Number) {
	if y.Doc != nil {
		Transact(y.Doc, func(trans *Transaction) {
			TypeListDelete(trans, y, index, length)
		}, nil, true)
	} else {
		SpliceArray(&y.PrelimContent, index, length, nil)
	}
}

// Returns the i-th element from a YArray.
func (y *YArray) Get(index Number) interface{} {
	return TypeListGet(y, index)
}

// Transforms this YArray to a JavaScript Array.
func (y *YArray) ToArray() ArrayAny {
	return TypeListToArray(y)
}

// Transforms this YArray to a JavaScript Array.
func (y *YArray) Splice(start, end Number) ArrayAny {
	return TypeListSlice(y, start, end)
}

// Transforms this Shared Type to a JSON object.
func (y *YArray) ToJson() interface{} {
	v := y.Map(func(c interface{}, _ Number, _ IAbstractType) interface{} {
		a, ok := c.(IAbstractType)
		if ok {
			return a.ToJson()
		} else {
			return c
		}
	})

	if v == nil {
		return ArrayAny{}
	}

	return v
}

// Returns an Array with the result of calling a provided function on every
// element of this YArray.
func (y *YArray) Map(f func(interface{}, Number, IAbstractType) interface{}) ArrayAny {
	return TypeListMap(y, f)
}

// Executes a provided function on once on overy element of this YArray.
func (y *YArray) ForEach(f func(interface{}, Number, IAbstractType)) {
	TypeListForEach(y, f)
}

func (y *YArray) Range(f func(item *Item)) {
	n := y.Start
	for ; n != nil; n = n.Right {
		if !n.Deleted() {
			f(n)
		}
	}
}

func (y *YArray) Write(encoder *UpdateEncoderV1) {
	encoder.WriteTypeRef(YArrayRefID)
}

func NewYArray() *YArray {
	a := &YArray{}
	a.EH = NewEventHandler()
	a.DEH = NewEventHandler()
	a.AbstractType.Map = make(map[string]*Item)
	return a
}

func NewYArrayType() IAbstractType {
	return NewYArray()
}
