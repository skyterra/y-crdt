package y_crdt

import (
	"errors"
	"math"
)

type TypeConstructor = func() IAbstractType

type IAbstractType interface {
	GetLength() Number
	GetItem() *Item
	GetMap() map[string]*Item
	StartItem() *Item
	SetStartItem(item *Item)
	GetDoc() *Doc
	UpdateLength(n Number)
	SetSearchMarker(mark []*ArraySearchMarker)
	Parent() IAbstractType
	Integrate(doc *Doc, item *Item)
	Copy() IAbstractType
	Clone() IAbstractType
	Write(encoder *UpdateEncoderV1)
	First() *Item
	CallObserver(trans *Transaction, parentSubs Set)
	Observe(f func(interface{}, interface{}))
	ObserveDeep(f func(interface{}, interface{}))
	Unobserve(f func(interface{}, interface{}))
	UnobserveDeep(f func(interface{}, interface{}))
	ToJson() interface{}
	GetDEH() *EventHandler
	GetEH() *EventHandler
	SetMap(map[string]*Item)
	SetLength(number Number)
	GetSearchMarker() *[]*ArraySearchMarker
}

const maxSearchMarker = 80

var GlobalSearchMarkerTimestamp = 0

// A unique timestamp that identifies each marker.
// Time is relative,.. this is more like an ever-increasing clock.
type ArraySearchMarker struct {
	P         *Item
	Index     Number
	Timestamp Number
}

type AbstractType struct {
	Item         *Item
	Map          map[string]*Item
	Start        *Item
	Doc          *Doc
	Length       Number
	EH           *EventHandler // event handlers
	DEH          *EventHandler // deep event handlers
	SearchMarker []*ArraySearchMarker
}

func (t *AbstractType) GetLength() Number {
	return t.Length
}

func (t *AbstractType) SetLength(number Number) {
	t.Length = number
}

func (t *AbstractType) GetItem() *Item {
	return t.Item
}

func (t *AbstractType) GetMap() map[string]*Item {
	if t.Map == nil {
		t.Map = make(map[string]*Item)
	}
	return t.Map
}

func (t *AbstractType) GetSearchMarker() *[]*ArraySearchMarker {
	return &t.SearchMarker
}

func (t *AbstractType) SetMap(m map[string]*Item) {
	t.Map = m
}

func (t *AbstractType) StartItem() *Item {
	return t.Start
}

func (t *AbstractType) SetStartItem(item *Item) {
	t.Start = item
}

func (t *AbstractType) GetDoc() *Doc {
	return t.Doc
}

func (t *AbstractType) UpdateLength(n Number) {
	t.Length += n
}

func (t *AbstractType) SetSearchMarker(marker []*ArraySearchMarker) {
	t.SearchMarker = marker
}

func (t *AbstractType) Parent() IAbstractType {
	if t.Item == nil || t.Item.Parent == nil {
		return nil
	}

	return t.Item.Parent.(IAbstractType)
}

// Integrate this type into the Yjs instance.
//
// * Save this struct in the os
// * This type is sent to other client
// * Observer functions are fired
func (t *AbstractType) Integrate(y *Doc, item *Item) {
	t.Doc = y
	t.Item = item
}

func (t *AbstractType) Copy() IAbstractType {
	return nil
}

func (t *AbstractType) Clone() IAbstractType {
	return nil
}

func (t *AbstractType) Write(encoder *UpdateEncoderV1) {

}

// The first non-deleted item
func (t *AbstractType) First() *Item {
	item := t.Start
	for item != nil && item.Deleted() {
		item = item.Right
	}

	return item
}

// Creates YEvent and calls all type observers.
// Must be implemented by each type.
func (t *AbstractType) CallObserver(trans *Transaction, parentSubs Set) {
	if !trans.Local && len(t.SearchMarker) > 0 {
		t.SearchMarker = nil
	}
}

// Observe all events that are created on this type.
func (t *AbstractType) Observe(f func(interface{}, interface{})) {
	AddEventHandlerListener(t.EH, f)
}

// Observe all events that are created by this type and its children.
func (t *AbstractType) ObserveDeep(f func(interface{}, interface{})) {
	AddEventHandlerListener(t.DEH, f)
}

// Unregister an observer function.
func (t *AbstractType) Unobserve(f func(interface{}, interface{})) {
	RemoveEventHandlerListener(t.EH, f)
}

// Unregister an observer function.
func (t *AbstractType) UnobserveDeep(f func(interface{}, interface{})) {
	RemoveEventHandlerListener(t.DEH, f)
}

func (t *AbstractType) ToJson() interface{} {
	return nil
}

func (t *AbstractType) GetDEH() *EventHandler {
	return t.DEH
}

func (t *AbstractType) GetEH() *EventHandler {
	return t.EH
}

func NewArraySearchMarker(p *Item, index Number) *ArraySearchMarker {
	return &ArraySearchMarker{
		P:     p,
		Index: index,
	}
}

func RefreshMarkerTimestamp(marker *ArraySearchMarker) {
	marker.Timestamp = GlobalSearchMarkerTimestamp
	GlobalSearchMarkerTimestamp++
}

// This is rather complex so this function is the only thing that should overwrite a marker
func OverwriteMarker(marker *ArraySearchMarker, p *Item, index Number) {
	marker.P.SetMarker(false)

	p.SetMarker(true)
	marker.P = p

	marker.Index = index
	marker.Timestamp = GlobalSearchMarkerTimestamp
	GlobalSearchMarkerTimestamp++
}

func MarkPosition(searchMarker *[]*ArraySearchMarker, p *Item, index Number) *ArraySearchMarker {
	if len(*searchMarker) >= maxSearchMarker {
		// override oldest marker (we don't want to create more objects)
		marker := (*searchMarker)[0]
		for _, m := range *searchMarker {
			if m.Timestamp < marker.Timestamp {
				marker = m
			}
		}

		OverwriteMarker(marker, p, index)
		return marker
	} else {
		// create new marker
		pm := NewArraySearchMarker(p, index)
		*searchMarker = append(*searchMarker, pm)
		return pm
	}
}

// Search marker help us to find positions in the associative array faster.
//
// They speed up the process of finding a position without much bookkeeping.
//
// A maximum of `maxSearchMarker` objects are created.
//
// This function always returns a refreshed marker (updated timestamp)
func FindMarker(yarray IAbstractType, index Number) *ArraySearchMarker {
	if yarray.StartItem() == nil || index == 0 || yarray.GetSearchMarker() == nil {
		return nil
	}

	var marker *ArraySearchMarker
	if len(*yarray.GetSearchMarker()) > 0 {
		marker = (*yarray.GetSearchMarker())[0]
		for _, m := range *yarray.GetSearchMarker() {
			if math.Abs(float64(index-m.Index)) < math.Abs(float64(index-marker.Index)) {
				marker = m
			}
		}
	}

	p := yarray.StartItem()
	pindex := 0

	if marker != nil {
		p = marker.P
		pindex = marker.Index
		RefreshMarkerTimestamp(marker) // we used it, we might need to use it again
	}

	// iterate to right if possible
	for p.Right != nil && pindex < index {
		if !p.Deleted() && p.Countable() {
			if index < pindex+p.Length {
				break
			}
			pindex += p.Length
		}
		p = p.Right
	}

	// iterate to left if necessary (might be that pindex > index)
	for p.Left != nil && pindex > index {
		p = p.Left
		if !p.Deleted() && p.Countable() {
			pindex -= p.Length
		}
	}

	// we want to make sure that p can't be merged with left, because that would screw up everything
	// in that cas just return what we have (it is most likely the best marker anyway)
	// iterate to left until p can't be merged with left
	for p.Left != nil && p.Left.ID.Client == p.ID.Client && p.Left.ID.Clock+p.Left.Length == p.ID.Clock {
		p = p.Left
		if !p.Deleted() && p.Countable() {
			pindex -= p.Length
		}
	}

	if marker != nil && Number(math.Abs(float64(marker.Index-pindex))) < p.Parent.(IAbstractType).GetLength()/maxSearchMarker {
		// adjust existing marker
		OverwriteMarker(marker, p, pindex)
		return marker
	} else {
		// create new marker
		return MarkPosition(yarray.GetSearchMarker(), p, pindex)
	}
}

// Update markers when a change happened.
// This should be called before doing a deletion!
func UpdateMarkerChanges(searchMarker *[]*ArraySearchMarker, index Number, length Number) {
	for i := len(*searchMarker) - 1; i >= 0; i-- {
		m := (*searchMarker)[i]
		if length > 0 {
			p := m.P
			p.SetMarker(false)

			// Ideally we just want to do a simple position comparison, but this will only work if
			// search markers don't point to deleted items for formats.
			// Iterate marker to prev undeleted countable position so we know what to do when updating a position
			for p != nil && (p.Deleted() || !p.Countable()) {
				p = p.Left
				if p != nil && !p.Deleted() && p.Countable() {
					// adjust position. the loop should break now
					m.Index -= p.Length
				}
			}

			if p == nil || p.Marker() {
				// remove search marker if updated position is null or if position is already marked
				*searchMarker = append((*searchMarker)[:i], (*searchMarker)[i+1:]...)
				continue
			}

			p.SetMarker(true)
			m.P = p
		}

		// a simple index <= m.index check would actually suffice
		if index < m.Index || length > 0 && index == m.Index {
			m.Index = Max(index, m.Index+length)
		}
	}
}

// Accumulate all (list) children of a type and return them as an Array.
func GetTypeChildren(t IAbstractType) []*Item {
	s := t.StartItem()
	var arr []*Item
	for s != nil {
		arr = append(arr, s)
		s = s.Right
	}
	return arr
}

// Call event listeners with an event. This will also add an event to all
// parents (for `.observeDeep` handlers).
func CallTypeObservers(t IAbstractType, trans *Transaction, event IEventType) {
	changedType := t
	changedParentTypes := trans.ChangedParentTypes

	for {
		_, exist := changedParentTypes[t]
		if !exist {
			changedParentTypes[t] = append(changedParentTypes[t], event)
		}

		if t.GetItem() == nil {
			break
		}

		t = t.GetItem().Parent.(IAbstractType)
	}
	CallEventHandlerListeners(changedType.GetEH(), event, trans)
}

func NewAbstractType() IAbstractType {
	return &AbstractType{
		Map: make(map[string]*Item),
		EH:  NewEventHandler(),
		DEH: NewEventHandler(),
	}
}

func TypeListSlice(t IAbstractType, start, end Number) ArrayAny {
	if start < 0 {
		start = t.GetLength() + start
	}

	if end < 0 {
		end = t.GetLength() + end
	}

	length := end - start
	var cs ArrayAny
	n := t.StartItem()
	for n != nil && length > 0 {
		if n.Countable() && !n.Deleted() {
			c := n.Content.GetContent()
			if len(c) <= start {
				start -= len(c)
			} else {
				for i := start; i < len(c) && length > 0; i++ {
					cs = append(cs, c[i])
					length--
				}
				start = 0
			}
		}
		n = n.Right
	}

	return cs
}

func TypeListToArray(t IAbstractType) ArrayAny {
	var cs ArrayAny
	n := t.StartItem()
	for n != nil {
		if n.Countable() && !n.Deleted() {
			c := n.Content.GetContent()
			for i := 0; i < len(c); i++ {
				cs = append(cs, c[i])
			}
		}
		n = n.Right
	}

	return cs
}

func TypeListToArraySnapshot(t IAbstractType, snapshot *Snapshot) ArrayAny {
	var cs ArrayAny
	n := t.StartItem()
	for n != nil {
		if n.Countable() && IsVisible(n, snapshot) {
			c := n.Content.GetContent()
			for i := 0; i < len(c); i++ {
				cs = append(cs, c[i])
			}
		}

		n = n.Right
	}

	return cs
}

// Executes a provided function on once on overy element of this YArray.
func TypeListForEach(t IAbstractType, f func(interface{}, Number, IAbstractType)) {
	index := 0
	n := t.StartItem()
	for n != nil {
		if n.Countable() && !n.Deleted() {
			c := n.Content.GetContent()
			for i := 0; i < len(c); i++ {
				f(c[i], index, t)
				index++
			}
		}
		n = n.Right
	}
}

func TypeListMap(t IAbstractType, f func(c interface{}, i Number, _ IAbstractType) interface{}) ArrayAny {
	var result ArrayAny
	TypeListForEach(t, func(c interface{}, i Number, _ IAbstractType) {
		result = append(result, f(c, i, t))
	})
	return result
}

// func TypeListCreateIterator(t IAbstractType) {
// }

// Executes a provided function on once on overy element of this YArray.
// Operates on a snapshotted state of the document.
func TypeListForEachSnapshot(t IAbstractType, f func(interface{}, Number, IAbstractType), snapshot *Snapshot) {
	index := 0
	n := t.StartItem()
	for n != nil {
		if n.Countable() && IsVisible(n, snapshot) {
			c := n.Content.GetContent()
			for i := 0; i < len(c); i++ {
				f(c[i], index, t)
				index++
			}
		}
		n = n.Right
	}
}

func TypeListGet(t IAbstractType, index Number) interface{} {
	marker := FindMarker(t, index)
	n := t.StartItem()
	if marker != nil {
		n = marker.P
		index -= marker.Index
	}

	for ; n != nil; n = n.Right {
		if !n.Deleted() && n.Countable() {
			if index < n.Length {
				return n.Content.GetContent()[index]
			}
			index -= n.Length
		}
	}

	return nil
}

func TypeListInsertGenericsAfter(trans *Transaction, parent IAbstractType, referenceItem *Item, content ArrayAny) error {
	left := referenceItem
	doc := trans.Doc
	ownClientId := doc.ClientID
	store := doc.Store

	var right *Item
	if referenceItem == nil {
		right = parent.StartItem()
	} else {
		right = referenceItem.Right
	}

	jsonContent := ArrayAny{}
	packJsonContent := func() {
		if len(jsonContent) > 0 {
			left = NewItem(GenID(ownClientId, GetState(store, ownClientId)), left, GetItemLastID(left), right, GetItemID(right), parent, "", NewContentAny(jsonContent))
			left.Integrate(trans, 0)
			jsonContent = nil
		}
	}

	for _, c := range content {
		switch c.(type) {
		case Number, Object, bool, ArrayAny, string:
			jsonContent = append(jsonContent, c)
		default:
			packJsonContent()
			switch c.(type) {
			case []uint8, string:
				left = NewItem(GenID(ownClientId, GetState(store, ownClientId)), left, GetItemLastID(left), right, GetItemID(right), parent, "", NewContentBinary(c.([]uint8)))
				left.Integrate(trans, 0)
			case *Doc:
				left = NewItem(GenID(ownClientId, GetState(store, ownClientId)), left, GetItemLastID(left), right, GetItemID(right), parent, "", NewContentDoc(c.(*Doc)))
				left.Integrate(trans, 0)
			default:
				if IsIAbstractType(c) {
					left = NewItem(GenID(ownClientId, GetState(store, ownClientId)), left, GetItemLastID(left), right, GetItemID(right), parent, "", NewContentType(c.(IAbstractType)))
					left.Integrate(trans, 0)
				} else {
					return errors.New("unexpected content type in insert operation")
				}
			}
		}
	}

	packJsonContent()
	return nil
}

func TypeListInsertGenerics(trans *Transaction, parent IAbstractType, index Number, content ArrayAny) error {
	if index > parent.GetLength() {
		return errors.New("[crdt] length exceeded")
	}

	if index == 0 {
		if parent.GetSearchMarker() != nil {
			UpdateMarkerChanges(parent.GetSearchMarker(), index, len(content))
		}
		return TypeListInsertGenericsAfter(trans, parent, nil, content)
	}

	startIndex := index
	marker := FindMarker(parent, index)
	n := parent.StartItem()
	if marker != nil {
		n = marker.P
		index -= marker.Index
		// we need to iterate one to the left so that the algorithm works
		if index == 0 {
			// @todo refactor this as it actually doesn't consider formats
			n = n.Prev() // important! get the left undeleted item so that we can actually decrease index
			if n != nil && n.Countable() && !n.Deleted() {
				index += n.Length
			}
		}
	}

	for ; n != nil; n = n.Right {
		if !n.Deleted() && n.Countable() {
			if index <= n.Length {
				if index < n.Length {
					// insert in-between
					GetItemCleanStart(trans, GenID(n.ID.Client, n.ID.Clock+index))
				}
				break
			}
			index -= n.Length
		}
	}

	if parent.GetSearchMarker() != nil {
		UpdateMarkerChanges(parent.GetSearchMarker(), startIndex, len(content))
	}

	return TypeListInsertGenericsAfter(trans, parent, n, content)
}

func TypeListDelete(trans *Transaction, parent IAbstractType, index Number, length Number) error {
	if length == 0 {
		return nil
	}

	startIndex := index
	startLength := length
	marker := FindMarker(parent, index)
	n := parent.StartItem()

	if marker != nil {
		n = marker.P
		index -= marker.Index
	}

	// compute the first item to be deleted
	for ; n != nil && index > 0; n = n.Right {
		if !n.Deleted() && n.Countable() {
			if index < n.Length {
				GetItemCleanStart(trans, GenID(n.ID.Client, n.ID.Clock+index))
			}
			index -= n.Length
		}
	}

	// delete all items until done
	for length > 0 && n != nil {
		if !n.Deleted() {
			if length < n.Length {
				GetItemCleanStart(trans, GenID(n.ID.Client, n.ID.Clock+length))
			}
			n.Delete(trans)
			length -= n.Length
		}
		n = n.Right
	}

	if length > 0 {
		return errors.New("length exceeded")
	}

	if parent.GetSearchMarker() != nil {
		UpdateMarkerChanges(parent.GetSearchMarker(), startIndex, -startLength+length) // in case we remove the above exception
	}

	return nil
}

func TypeMapDelete(trans *Transaction, parent IAbstractType, key string) {
	c, exist := parent.GetMap()[key]
	if exist {
		c.Delete(trans)
	}
}

func TypeMapSet(trans *Transaction, parent IAbstractType, key string, value interface{}) error {
	left := parent.GetMap()[key]
	doc := trans.Doc
	ownClientId := doc.ClientID
	var content IAbstractContent

	if value == nil {
		content = NewContentAny(ArrayAny{value})
	} else {
		switch value.(type) {
		case Number, Object, bool, ArrayAny, string:
			content = NewContentAny(ArrayAny{value})
			break
		case []uint8:
			content = NewContentBinary(value.([]uint8))
		case *Doc:
			content = NewContentDoc(value.(*Doc))
		default:
			if IsIAbstractType(value) {
				content = NewContentType(value.(IAbstractType))
			} else {
				return errors.New("unexpected content type")
			}
		}
	}

	item := NewItem(GenID(ownClientId, GetState(doc.Store, ownClientId)), left, GetItemLastID(left), nil, nil, parent, key, content)
	item.Integrate(trans, 0)
	return nil
}

func TypeMapGet(parent IAbstractType, key string) interface{} {
	val, exist := parent.GetMap()[key]
	if !exist || val.Deleted() {
		return nil
	}

	return val.Content.GetContent()[val.Length-1]
}

func TypeMapGetAll(parent IAbstractType) Object {
	res := NewObject()
	for key, value := range parent.GetMap() {
		if !value.Deleted() {
			res[key] = value.Content.GetContent()[value.Length-1]
		}
	}
	return res
}

func TypeMapHas(parent IAbstractType, key string) bool {
	val, exist := parent.GetMap()[key]
	return exist && !val.Deleted()
}

func TypeMapGetSnapshot(parent IAbstractType, key string, snapshot *Snapshot) interface{} {
	v := parent.GetMap()[key]
	hasClient := func(client Number) bool {
		_, exist := snapshot.Sv[client]
		return exist
	}
	for v != nil && (!hasClient(v.ID.Client) || v.ID.Clock >= snapshot.Sv[v.ID.Client]) {
		v = v.Left
	}

	if v != nil && IsVisible(v, snapshot) {
		return v.Content.GetContent()[v.Length-1]
	}

	return nil
}
