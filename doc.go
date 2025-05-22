package y_crdt

import (
	"fmt"
)

type Doc struct {
	*Observable
	Guid     string
	ClientID Number

	GC           bool
	GCFilter     func(item *Item) bool
	Share        map[string]IAbstractType
	Store        *StructStore
	Trans        *Transaction
	TransCleanup []*Transaction
	SubDocs      Set
	Item         *Item // If this document is a subdocument - a document integrated into another document - then _item is defined.
	ShouldLoad   bool
	AutoLoad     bool
	Meta         interface{}
}

// Notify the parent document that you request to load data into this subdocument (if it is a subdocument).
//
//	`load()` might be used in the future to request any provider to load the most current data.
//	It is safe to call `load()` multiple times.
func (doc *Doc) Load() {
	item := doc.Item
	if item != nil && !doc.ShouldLoad {
		parent := item.Parent.(IAbstractType)
		Transact(parent.GetDoc(), func(trans *Transaction) {
			trans.SubdocsLoaded.Add(doc)
		}, nil, true)
	}
	doc.ShouldLoad = true
}

func (doc *Doc) GetSubdocs() Set {
	return doc.SubDocs
}

func (doc *Doc) GetSubdocGuids() Set {
	s := NewSet()
	for k := range doc.SubDocs {
		guid := k.(string)
		s.Add(guid)
	}
	return s
}

// Changes that happen inside of a transaction are bundled. This means that
// the observer fires _after_ the transaction is finished and that all changes
// that happened inside of the transaction are sent as one message to the
// other peers.
func (doc *Doc) Transact(f func(trans *Transaction), origin interface{}) {
	Transact(doc, f, origin, true)
}

// Define a shared data type.
//
// Multiple calls of `y.get(name, TypeConstructor)` yield the same result
// and do not overwrite each other. I.e.
// `y.define(name, Y.Array) === y.define(name, Y.Array)`
//
// After this method is called, the type is also available on `y.share.get(name)`.
//
// Best Practices:
// Define all types right after the Yjs instance is created and store them in a separate object.
// Also use the typed methods `getText(name)`, `getArray(name)`, ..
//
// example
//
//	const y = new Y(..)
//	const appState = {
//	  document: y.getText('document')
//	  comments: y.getArray('comments')
//	}
func (doc *Doc) Get(name string, typeConstructor TypeConstructor) (IAbstractType, error) {
	_, exist := doc.Share[name]
	if !exist {
		t := typeConstructor()
		t.Integrate(doc, nil)
		doc.Share[name] = t
	}

	constr := doc.Share[name]
	if !IsSameType(typeConstructor(), NewAbstractType()) && !IsSameType(constr, typeConstructor()) {
		if IsSameType(constr, NewAbstractType()) {
			t := typeConstructor()
			t.SetMap(constr.GetMap())
			for _, n := range t.GetMap() {
				for ; n != nil; n = n.Left {
					n.Parent = t
				}
			}

			t.SetStartItem(constr.StartItem())
			for n := t.StartItem(); n != nil; n = n.Right {
				n.Parent = t
			}

			t.SetLength(constr.GetLength())
			t.Integrate(doc, nil)
			doc.Share[name] = t
			return t, nil
		} else {
			return nil, fmt.Errorf("Type with the name %s has already been defined with a different constructor ", name)
		}
	}

	return constr, nil
}

func (doc *Doc) GetArray(name string) *YArray {
	arr, err := doc.Get(name, NewYArrayType)
	if err != nil {
		return nil
	}

	a, ok := arr.(*YArray)
	if ok {
		return a
	}

	return nil
}

func (doc *Doc) GetText(name string) *YText {
	text, err := doc.Get(name, NewYTextType)
	if err != nil {
		return nil
	}

	a, ok := text.(*YText)
	if ok {
		return a
	}

	return nil
}

func (doc *Doc) GetMap(name string) IAbstractType {
	m, err := doc.Get(name, NewYMapType)
	if err != nil {
		return nil
	}
	return m
}

func (doc *Doc) GetXmlFragment(name string) IAbstractType {
	xml, err := doc.Get(name, NewYXmlFragmentType)
	if err != nil {
		return nil
	}
	return xml
}

// Converts the entire document into a js object, recursively traversing each yjs type
// Doesn't log types that have not been defined (using ydoc.getType(..)).
//
// Do not use this method and rather call toJSON directly on the shared types.
func (doc *Doc) ToJson() Object {
	object := NewObject()
	for key, value := range doc.Share {
		object[key] = value.ToJson()
	}
	return object
}

// Emit `destroy` event and unregister all event handlers.
func (doc *Doc) Destroy() {
	for k := range doc.SubDocs {
		subDoc := k.(*Doc)
		subDoc.Destroy()
	}

	item := doc.Item
	if item != nil {
		doc.Item = nil
		content := item.Content.(*ContentDoc)
		if item.Deleted() {
			content.Doc = nil
		} else {
			content.Doc = NewDoc(doc.Guid, content.Opts[OptKeyGC].(bool), DefaultGCFilter, content.Opts[OptKeyMeta], content.Opts[OptKeyAutoLoad].(bool))
			content.Doc.Item = item
		}

		Transact(item.Parent.(IAbstractType).GetDoc(), func(trans *Transaction) {
			if !item.Deleted() {
				trans.SubdocsAdded.Add(content.Doc)
			}
		}, nil, true)
	}

	doc.Emit("destroyed", true)
	doc.Emit("destroy", doc)
	doc.Observable.Destroy()
}

func (doc *Doc) On(eventName string, handler *ObserverHandler) {
	doc.Observable.On(eventName, handler)
}

func (doc *Doc) Off(eventName string, handler *ObserverHandler) {
	doc.Observable.Off(eventName, handler)
}

func NewDoc(guid string, gc bool, gcFilter func(item *Item) bool, meta interface{}, autoLoad bool) *Doc {
	doc := &Doc{
		Observable: NewObservable(),
		ClientID:   GenerateNewClientID(),
		Guid:       guid,
		GC:         gc,
		GCFilter:   gcFilter,
		Meta:       meta,
		AutoLoad:   autoLoad,
		Store:      NewStructStore(),
		Share:      make(map[string]IAbstractType),
	}

	return doc
}
