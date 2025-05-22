package y_crdt

import (
	"reflect"
)

type YMapIter struct {
	Key  string
	Item *Item
}

// Event that describes the changes on a YMap.
type YMapEvent struct {
	YEvent
	KeysChanged Set
}

func NewYMapEvent(ymap *YMap, trans *Transaction, subs Set) *YMapEvent {
	return &YMapEvent{
		YEvent:      *NewYEvent(ymap, trans),
		KeysChanged: subs,
	}
}

// A shared Map implementation.
type YMap struct {
	AbstractType
	PrelimContent map[string]interface{}
}

// Integrate this type into the Yjs instance.
//
//	Save this struct in the os
//	This type is sent to other client
//	Observer functions are fired
func (y *YMap) Integrate(doc *Doc, item *Item) {
	y.AbstractType.Integrate(doc, item)
	for key, value := range y.PrelimContent {
		y.Set(key, value)
	}

	y.PrelimContent = make(map[string]interface{})
}

func (y *YMap) Copy() IAbstractType {
	return NewYMap(nil)
}

func (y *YMap) Clone() IAbstractType {
	m := NewYMap(nil)

	y.ForEach(func(key string, value interface{}, yMap *YMap) {
		v, ok := value.(IAbstractType)
		if ok {
			m.Set(key, v.Clone())
		} else {
			m.Set(key, value)
		}
	})

	return m
}

// Creates YMapEvent and calls observers.
func (y *YMap) CallObserver(trans *Transaction, parentSubs Set) {
	CallTypeObservers(y, trans, NewYMapEvent(y, trans, parentSubs))
}

// Transforms this Shared Type to a JSON object.
func (y *YMap) ToJson() interface{} {
	m := NewObject()
	for key, item := range y.Map {
		if !item.Deleted() {
			v := item.Content.GetContent()[item.Length-1]
			t, ok := v.(IAbstractType)
			if ok {
				m[key] = t.ToJson()
			} else {
				if reflect.TypeOf(v) != reflect.TypeOf(UndefinedType{}) {
					m[key] = v
				}
			}
		}
	}
	return m
}

// Returns the size of the YMap (count of key/value pairs)
func (y *YMap) GetSize() Number {
	return len(createMapIterator(y.Map))
}

// Returns the keys for each element in the YMap Type.
func (y *YMap) Keys() []string {
	its := createMapIterator(y.Map)

	keys := make([]string, 0, len(its))
	for _, it := range its {
		keys = append(keys, it.Key)
	}

	return keys
}

// Returns the values for each element in the YMap Type.
func (y *YMap) Values() []interface{} {
	its := createMapIterator(y.Map)

	values := make([]interface{}, 0, len(its))
	for _, it := range its {
		values = append(values, it.Item.Content.GetContent()[it.Item.Length-1])
	}
	return values
}

// Returns an Iterator of [key, value] pairs
func (y *YMap) Entries() map[string]interface{} {
	m := make(map[string]interface{})
	its := createMapIterator(y.Map)

	for _, it := range its {
		m[it.Key] = it.Item.Content.GetContent()[it.Item.Length-1]
	}

	return m
}

// Executes a provided function on once on every key-value pair.
func (y *YMap) ForEach(f func(string, interface{}, *YMap)) Object {
	m := NewObject()
	for key, item := range y.Map {
		if !item.Deleted() {
			f(key, item.Content.GetContent()[item.Length-1], y)
		}
	}
	return m
}

func (y *YMap) Range(f func(key string, val interface{})) {
	entries := y.Entries()
	for key, value := range entries {
		f(key, value)
	}
}

// Remove a specified element from this YMap.
func (y *YMap) Delete(key string) {
	if y.Doc != nil {
		Transact(y.Doc, func(trans *Transaction) {
			TypeMapDelete(trans, y, key)
		}, nil, true)
	} else {
		delete(y.PrelimContent, key)
	}
}

// Adds or updates an element with a specified key and value.
func (y *YMap) Set(key string, value interface{}) interface{} {
	if y.Doc != nil {
		Transact(y.Doc, func(trans *Transaction) {
			TypeMapSet(trans, y, key, value)
		}, nil, true)
	} else {
		y.PrelimContent[key] = value
	}

	return value
}

// Returns a specified element from this YMap.
func (y *YMap) Get(key string) interface{} {
	return TypeMapGet(y, key)
}

func (y *YMap) Has(key string) bool {
	return TypeMapHas(y, key)
}

// Removes all elements from this YMap.
func (y *YMap) Clear() {
	if y.Doc != nil {
		Transact(y.Doc, func(trans *Transaction) {
			y.Range(func(key string, val interface{}) {
				TypeMapDelete(trans, y, key)
			})

		}, nil, true)
	} else {
		y.PrelimContent = make(map[string]interface{})
	}
}

func (y *YMap) Write(encoder *UpdateEncoderV1) {
	encoder.WriteTypeRef(YMapRefID)
}

func NewYMap(entries map[string]interface{}) *YMap {
	ymap := &YMap{
		AbstractType: AbstractType{
			Map: make(map[string]*Item),
			EH:  NewEventHandler(),
			DEH: NewEventHandler(),
		},
	}

	if entries == nil {
		ymap.PrelimContent = make(map[string]interface{})
	} else {
		ymap.PrelimContent = entries
	}

	return ymap
}

func NewYMapType() IAbstractType {
	return NewYMap(nil)
}

func createMapIterator(m map[string]*Item) []YMapIter {
	var its []YMapIter

	for key, item := range m {
		if !item.Deleted() {
			its = append(its, YMapIter{
				Key:  key,
				Item: item,
			})
		}
	}

	return its
}
