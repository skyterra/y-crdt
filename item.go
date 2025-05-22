package y_crdt

import (
	"errors"
)

// Abstract class that represents any content.
type Item struct {
	AbstractStruct
	Origin      *ID   // The item that was originally to the left of this item.
	Left        *Item // The item that is currently to the left of this item.
	Right       *Item // The item that is currently to the right of this item.
	RightOrigin *ID   // The item that was originally to the right of this item.

	// Is a type if integrated, is null if it is possible to copy parent from
	// left or right, is ID before integration to search for it.
	Parent interface{} // AbstractType<any> | ID

	// If the parent refers to this item with some kind of key (e.g. YMap, the
	// key is specified here. The key is then used to refer to the list in which
	// to insert this item. If `parentSub = null` type._start is the list in
	// which to insert to. Otherwise it is `parent._map`.
	ParentSub string

	// If this type's effect is reundone this type refers to the type that undid
	// this operation.
	Redone *ID

	Content IAbstractContent

	Info uint8 // BIT1, BIT2, BIT3, BIT4 - mark node as fast-search-marker
}

// This is used to mark the item as an indexed fast-search marker
func (item *Item) Marker() bool {
	return item.Info&BIT4 > 0
}

// If true, do not garbage collect this Item.
func (item *Item) Keep() bool {
	return item.Info&BIT1 > 0
}

func (item *Item) Countable() bool {
	return item.Info&BIT2 > 0
}

// Whether this item was deleted or not.
func (item *Item) Deleted() bool {
	return item.Info&BIT3 > 0
}

func (item *Item) SetMarker(marked bool) {
	item.setInfo(BIT4, marked)
}

func (item *Item) SetKeep(keep bool) {
	item.setInfo(BIT1, keep)
}

func (item *Item) SetCountable(countable bool) {
	item.setInfo(BIT2, countable)
}

func (item *Item) SetDeleted(deleted bool) {
	item.setInfo(BIT3, deleted)
}

func (item *Item) MarkDeleted() {
	item.Info |= BIT3
}

func (item *Item) setInfo(pos uint8, on bool) {
	state := item.Info&pos > 0
	if state != on {
		item.Info ^= pos
	}
}

// Return the creator clientID of the missing op or define missing items and return null.
func (item *Item) GetMissing(trans *Transaction, store *StructStore) (Number, error) {
	if item.Origin != nil && item.Origin.Client != item.ID.Client && item.Origin.Clock >= GetState(store, item.Origin.Client) {
		return item.Origin.Client, nil
	}

	if item.RightOrigin != nil && item.RightOrigin.Client != item.ID.Client && item.RightOrigin.Clock >= GetState(store, item.RightOrigin.Client) {
		return item.RightOrigin.Client, nil
	}

	if item.Parent != nil && IsIDPtr(item.Parent) && item.ID.Client != item.Parent.(*ID).Client && item.Parent.(*ID).Clock >= GetState(store, item.Parent.(*ID).Client) {
		return item.Parent.(*ID).Client, nil
	}

	// We have all missing ids, now find the items

	if item.Origin != nil {
		item.Left = GetItemCleanEnd(trans, store, *item.Origin)
		if item.Left != nil {
			item.Origin = item.Left.LastID()
		} else {
			item.Origin = nil
			item.Parent = nil
		}
	}

	if item.RightOrigin != nil {
		item.Right = GetItemCleanStart(trans, *item.RightOrigin)
		if item.Right != nil {
			item.RightOrigin = item.Right.GetID()
		} else {
			item.RightOrigin = nil
			item.Parent = nil
		}
	}

	// yjs源码中，有效；golang版中，无效，因为item.Left 和 item.Right 明确为 *Item 类型
	if (item.Left != nil && IsGCPtr(item.Left)) || (item.Right != nil && IsGCPtr(item.Right)) {
		item.Parent = nil
	}

	// only set parent if this shouldn't be garbage collected
	if item.Parent == nil {
		if item.Left != nil && IsItemPtr(item.Left) {
			item.Parent = item.Left.Parent
			item.ParentSub = item.Left.ParentSub
		}

		if item.Right != nil && IsItemPtr(item.Right) {
			item.Parent = item.Right.Parent
			item.ParentSub = item.Right.ParentSub
		}
	} else if IsIDPtr(item.Parent) {
		parentItem := GetItem(store, *item.Parent.(*ID))
		// if IsGCPtr(parentItem) {
		if !IsItemPtr(parentItem) {
			item.Parent = nil
		} else {
			contentType, ok := parentItem.(*Item).Content.(*ContentType)
			if ok {
				item.Parent = contentType.Type
			} else {
				item.Parent = nil
			}
		}
	}

	return 0, errors.New("not found creator clientID")
}

func (item *Item) Integrate(trans *Transaction, offset Number) {
	if offset > 0 {
		item.ID.Clock += offset
		item.Left = GetItemCleanEnd(trans, trans.Doc.Store, GenID(item.ID.Client, item.ID.Clock-1))
		if item.Left != nil {
			item.Origin = item.Left.LastID()
		}
		item.Content = item.Content.Splice(offset)
		item.Length -= offset
	}

	// set o to the first conflicting item
	if item.Parent != nil {
		if (item.Left == nil && (item.Right == nil || item.Right.Left != nil)) ||
			(item.Left != nil && item.Left.Right != item.Right) {
			left := item.Left

			var o *Item
			if left != nil {
				o = left.Right
			} else if item.ParentSub != "" {
				o = item.Parent.(IAbstractType).GetMap()[item.ParentSub]
				for o != nil && o.Left != nil {
					o = o.Left
				}
			} else {
				o = item.Parent.(IAbstractType).StartItem()
			}

			conflictingItems := NewSet()
			itemsBeforeOrigin := NewSet()

			// Let c in conflictingItems, b in itemsBeforeOrigin
			// ***{origin}bbbb{this}{c,b}{c,b}{o}***
			// Note that conflictingItems is a subset of itemsBeforeOrigin
			for o != nil && o != item.Right {
				itemsBeforeOrigin.Add(o)
				conflictingItems.Add(o)

				if CompareIDs(item.Origin, o.Origin) {
					// case 1
					if o.ID.Client < item.ID.Client {
						left = o
						conflictingItems = NewSet() // clear all
					} else if CompareIDs(item.RightOrigin, o.RightOrigin) {
						// this and o are conflicting and point to the same
						// integration points. The id decides which item comes first.
						// Since this is to the left of o, we can break here
						break
					}
					//} else if o.Origin != nil && itemsBeforeOrigin.Has(GetItem(trans.Doc.Store, *o.Origin)) {
				} else if o.Origin != nil {
					// else, o might be integrated before an item that this conflicts with.
					// If so, we will find it in the next iterations
					itemTmp := GetItem(trans.Doc.Store, *o.Origin)
					if !itemsBeforeOrigin.Has(itemTmp) {
						break
					}

					// case 2
					if !conflictingItems.Has(itemTmp) {
						left = o
						conflictingItems = NewSet()
					}
				} else {
					break
				}

				o = o.Right
			}
			item.Left = left
		}

		// reconnect left/right + update parent map/start if necessary
		if item.Left != nil {
			right := item.Left.Right
			item.Right = right
			item.Left.Right = item
		} else {
			var r *Item
			if item.ParentSub != "" {
				r = item.Parent.(IAbstractType).GetMap()[item.ParentSub]
				for r != nil && r.Left != nil {
					r = r.Left
				}
			} else {
				r = item.Parent.(IAbstractType).StartItem()
				item.Parent.(IAbstractType).SetStartItem(item)
			}
			item.Right = r
		}

		if item.Right != nil {
			item.Right.Left = item
		} else if item.ParentSub != "" {
			// set as current parent value if right === null and this is parentSub
			item.Parent.(IAbstractType).GetMap()[item.ParentSub] = item
			if item.Left != nil {
				// this is the current attribute value of parent. delete right
				item.Left.Delete(trans)
			}
		}

		// adjust length of parent
		if item.ParentSub == "" && item.Countable() && !item.Deleted() {
			item.Parent.(IAbstractType).UpdateLength(item.Length)
		}

		if err := AddStruct(trans.Doc.Store, item); err != nil {
			return
		}

		item.Content.Integrate(trans, item)

		// add parent to transaction.changed
		AddChangedTypeToTransaction(trans, item.Parent.(IAbstractType), item.ParentSub)
		if item.Parent.(IAbstractType).GetItem() != nil && item.Parent.(IAbstractType).GetItem().Deleted() || item.ParentSub != "" && item.Right != nil {
			// delete if parent is deleted or if this is not the current attribute value of parent
			item.Delete(trans)
		}
	} else {
		// parent is not defined. Integrate GC struct instead
		gc := NewGC(item.ID, item.Length)
		gc.Integrate(trans, 0)
	}
}

// Returns the next non-deleted item
func (item *Item) Next() *Item {
	nextItem := item.Right
	for nextItem != nil && nextItem.Deleted() {
		nextItem = nextItem.Right
	}

	return nextItem
}

// Returns the previous non-deleted item
func (item *Item) Prev() *Item {
	prevItem := item.Left
	for prevItem != nil && prevItem.Deleted() {
		prevItem = prevItem.Left
	}

	return prevItem
}

// Computes the last content address of this Item
func (item *Item) LastID() *ID {
	// allocating ids is pretty costly because of the amount of ids created, so we try to reuse whenever possible
	if item.Length == 1 {
		return &item.ID
	}

	id := GenID(item.ID.Client, item.ID.Clock+item.Length-1)
	return &id
}

// Try to merge two items
func (item *Item) MergeWith(right IAbstractStruct) bool {
	r, ok := right.(*Item)
	if ok &&
		CompareIDs(r.Origin, item.LastID()) &&
		item.Right == r &&
		CompareIDs(item.RightOrigin, r.RightOrigin) &&
		item.ID.Client == r.ID.Client &&
		item.ID.Clock+item.Length == r.ID.Clock &&
		item.Deleted() == r.Deleted() &&
		item.Redone == nil &&
		r.Redone == nil &&
		IsSameType(item.Content, r.Content) &&
		item.Content.MergeWith(r.Content) {

		parent, ok := item.Parent.(IAbstractType)
		if ok {
			searchMarker := parent.GetSearchMarker()
			if searchMarker != nil {
				for _, marker := range *searchMarker {
					if marker.P == right {
						// right is going to be "forgotten" so we need to update the marker
						marker.P = item

						// adjust marker index
						if !item.Deleted() && item.Countable() {
							marker.Index -= item.Length
						}
					}
				}
			}
		}

		if r.Keep() {
			item.SetKeep(true)
		}

		item.Right = r.Right
		if item.Right != nil {
			item.Right.Left = item
		}

		item.Length += r.Length
		return true
	}

	return false
}

// Mark this Item as deleted.
func (item *Item) Delete(trans *Transaction) {
	if !item.Deleted() {
		parent := item.Parent

		// adjust the length of parent
		if item.Countable() && item.ParentSub == "" {
			parent.(IAbstractType).UpdateLength(-item.Length)
		}
		item.MarkDeleted()
		AddToDeleteSet(trans.DeleteSet, item.ID.Client, item.ID.Clock, item.Length)
		AddChangedTypeToTransaction(trans, parent.(IAbstractType), item.ParentSub)
		item.Content.Delete(trans)
	}
}

func (item *Item) GC(store *StructStore, parentGCd bool) {
	if !item.Deleted() {
		return
	}

	item.Content.GC(store)
	if parentGCd {
		ReplaceStruct(store, item, NewGC(item.ID, item.Length))
	} else {
		item.Content = NewContentDeleted(item.Length)
	}
}

// Transform the properties of this type to binary and write it to an
// BinaryEncoder.
//
// This is called when this Item is sent to a remote peer.
func (item *Item) Write(encoder *UpdateEncoderV1, offset Number) {
	origin := item.Origin
	if offset > 0 {
		id := GenID(item.ID.Client, item.ID.Clock+offset-1)
		origin = &id
	}

	rightOrigin := item.RightOrigin
	parentSub := item.ParentSub
	info := item.Content.GetRef()&BITS5 |
		Conditional(origin == nil, uint8(0), uint8(BIT8)).(uint8) | // origin is defined
		Conditional(rightOrigin == nil, uint8(0), uint8(BIT7)).(uint8) | // right origin is defined
		Conditional(parentSub == "", uint8(0), uint8(BIT6)).(uint8)
	encoder.WriteInfo(info)
	if origin != nil {
		encoder.WriteLeftID(origin)
	}

	if rightOrigin != nil {
		encoder.WriteRightID(rightOrigin)
	}

	if origin == nil && rightOrigin == nil {
		parent := item.Parent

		if IsIAbstractType(parent) && !IsYString(parent) && !IsIDPtr(parent) {
			parentItem := parent.(IAbstractType).GetItem()
			if parentItem == nil {
				// parent type on y._map
				// find the correct key
				ykey := FindRootTypeKey(parent.(IAbstractType))
				encoder.WriteParentInfo(true)
				encoder.WriteString(ykey)
			} else {
				encoder.WriteParentInfo(false)
				encoder.WriteLeftID(&parentItem.ID)
			}
		} else if IsYString(parent) {
			encoder.WriteParentInfo(true)
			encoder.WriteString(parent.(*YString).Str)
		} else if IsIDPtr(parent) {
			encoder.WriteParentInfo(false)
			encoder.WriteLeftID(parent.(*ID))
		} else {
		}

		if parentSub != "" {
			encoder.WriteString(parentSub)
		}
	}

	item.Content.Write(encoder, offset)
}

func NewItem(id ID, left *Item, origin *ID, right *Item, rightOrigin *ID,
	parent IAbstractType, parentSub string, content IAbstractContent) *Item {

	if content == nil {
		return nil
	}

	info := uint8(0)
	if content.IsCountable() {
		info = BIT2
	}

	return &Item{
		AbstractStruct: AbstractStruct{
			ID:     id,
			Length: content.GetLength(),
		},
		Left:        left,
		Origin:      origin,
		Right:       right,
		RightOrigin: rightOrigin,
		Parent:      parent,
		ParentSub:   parentSub,
		Content:     content,
		Info:        info,
	}
}

// This should return several items
func FollowRedone(store *StructStore, id ID) (*Item, Number) {
	nextID := &id
	diff := 0
	var item *Item
	for {
		if diff > 0 {
			newID := GenID(nextID.Client, nextID.Clock+diff)
			nextID = &newID
		}

		it, ok := GetItem(store, *nextID).(*Item)
		if !ok {
			break
		}

		item = it
		diff = nextID.Clock - item.ID.Clock
		nextID = item.Redone

		if nextID == nil {
			break
		}
	}

	return item, diff
}

// Make sure that neither item nor any of its parents is ever deleted.
//
// This property does not persist when storing it into a database or when
// sending it to other peers
func KeepItem(item *Item, keep bool) {
	for item != nil && item.Keep() != keep {
		item.SetKeep(keep)
		item = item.Parent.(IAbstractType).GetItem()
	}
}

// Split leftItem into two items.
func SplitItem(trans *Transaction, leftItem *Item, diff Number) *Item {
	client, clock := leftItem.ID.Client, leftItem.ID.Clock

	var parent IAbstractType
	if leftItem.Parent == nil {
		parent = nil
	} else {
		parent = leftItem.Parent.(IAbstractType)
	}

	originID := GenID(client, clock+diff-1)
	rightItem := NewItem(GenID(client, clock+diff), leftItem, &originID,
		leftItem.Right, leftItem.RightOrigin, parent, leftItem.ParentSub, leftItem.Content.Splice(diff))

	if leftItem.Deleted() {
		rightItem.MarkDeleted()
	}

	if leftItem.Keep() {
		rightItem.SetKeep(true)
	}

	if leftItem.Redone != nil {
		id := GenID(leftItem.Redone.Client, leftItem.Redone.Clock+diff)
		rightItem.Redone = &id
	}

	// update left (do not set leftItem.rightOrigin as it will lead to problems when syncing)
	leftItem.Right = rightItem

	// update right
	if rightItem.Right != nil {
		rightItem.Right.Left = rightItem
	}

	// right is more specific.
	trans.MergeStructs = append(trans.MergeStructs, rightItem)

	// update parent._map
	if rightItem.ParentSub != "" && rightItem.Right == nil {
		rightItem.Parent.(IAbstractType).GetMap()[rightItem.ParentSub] = rightItem
	}

	leftItem.Length = diff
	return rightItem
}

// Redoes the effect of this operation.
func RedoItem(trans *Transaction, item *Item, redoItems Set) *Item {
	doc := trans.Doc
	store := doc.Store
	ownClientID := doc.ClientID
	redone := item.Redone
	if redone != nil {
		return GetItemCleanStart(trans, *redone)
	}

	parentItem := item.Parent.(IAbstractType).GetItem()
	var left *Item
	var right *Item

	if item.ParentSub == "" {
		// Is an array item. Insert at the old position
		left = item.Left
		right = item
	} else {
		// Is a map item. Insert as current value
		left = item
		for left.Right != nil {
			left = left.Right
			if left.ID.Client != ownClientID {
				// It is not possible to redo this item because it conflicts with a
				// change from another client
				return nil
			}
		}

		if left.Right != nil {
			left = item.Parent.(IAbstractType).GetMap()[item.ParentSub]
		}
		right = nil
	}

	// make sure that parent is redone
	if parentItem != nil && parentItem.Deleted() && parentItem.Redone == nil {
		// try to undo parent if it will be undone anyway
		if !redoItems.Has(parentItem) || RedoItem(trans, parentItem, redoItems) == nil {
			return nil
		}
	}

	if parentItem != nil && parentItem.Redone != nil {
		for parentItem.Redone != nil {
			parentItem = GetItemCleanStart(trans, *parentItem.Redone)
		}

		// find next cloned_redo items
		for left != nil {
			leftTrace := left

			// trace redone until parent matches
			for leftTrace != nil && leftTrace.Parent.(IAbstractType).GetItem() != parentItem {
				if leftTrace.Redone == nil {
					leftTrace = nil
				} else {
					leftTrace = GetItemCleanStart(trans, *leftTrace.Redone)
				}
			}

			if leftTrace != nil && leftTrace.Parent.(IAbstractType).GetItem() == parentItem {
				left = leftTrace
				break
			}

			left = left.Left
		}

		for right != nil {
			rightTrace := right

			// trace redone until parent matches
			for rightTrace != nil && rightTrace.Parent.(IAbstractType).GetItem() != parentItem {
				if rightTrace.Redone == nil {
					rightTrace = nil
				} else {
					rightTrace = GetItemCleanStart(trans, *rightTrace.Redone)
				}
			}

			if rightTrace != nil && rightTrace.Parent.(IAbstractType).GetItem() == parentItem {
				right = rightTrace
				break
			}

			right = right.Right
		}
	}

	nextClock := GetState(store, ownClientID)
	nextID := GenID(ownClientID, nextClock)

	var parent IAbstractType
	if parentItem == nil {
		if item.Parent != nil {
			parent = item.Parent.(IAbstractType)
		}
	} else {
		content, ok := parentItem.Content.(*ContentType)
		if ok {
			parent = content.Type
		}
	}

	redoneItem := NewItem(nextID, left, GetItemLastID(left), right, GetItemID(right), parent, item.ParentSub, item.Content.Copy())
	item.Redone = &redoneItem.ID
	KeepItem(redoneItem, true)
	redoneItem.Integrate(trans, 0)
	return redoneItem
}

func GetItemID(item *Item) *ID {
	if item != nil {
		return &item.ID
	}

	return nil
}

func GetItemLastID(item *Item) *ID {
	if item != nil {
		return item.LastID()
	}

	return nil
}
