package y_crdt

import (
	"reflect"
)

type StackItem struct {
	Insertions *DeleteSet
	Deletions  *DeleteSet
	Meta       map[interface{}]interface{} // Use this to save and restore metadata like selection range
}

// @typedef {Object} UndoManagerOptions
// @property {number} [UndoManagerOptions.captureTimeout=500]
// @property {function(Item):boolean} [UndoManagerOptions.deleteFilter=()=>true] Sometimes
// it is necessary to filter whan an Undo/Redo operation can delete. If this
// filter returns false, the type/item won't be deleted even it is in the
// undo/redo scope.
// @property {Set<any>} [UndoManagerOptions.trackedOrigins=new Set([null])]
//
//	Fires 'stack-item-added' event when a stack item was added to either the undo- or
//	the redo-stack. You may store additional stack information via the
//	metadata property on `event.stackItem.meta` (it is a `Map` of metadata properties).
//	Fires 'stack-item-popped' event when a stack item was popped from either the
//	undo- or the redo-stack. You may restore the saved stack information from `event.stackItem.meta`.
//
//	@extends {Observable<'stack-item-added'|'stack-item-popped'>}
type UndoManager struct {
	*Observable
	Scopes         []IAbstractType
	DeleteFilter   func(item *Item) bool
	TrackedOrigins Set
	UndoStack      []*StackItem
	RedoStack      []*StackItem

	// Whether the client is currently undoing (calling UndoManager.undo)
	Undoing    bool
	Redoing    bool
	LastChange Number
}

func (u *UndoManager) GetDoc() *Doc {
	return u.Scopes[0].GetDoc()
}

func (u *UndoManager) Clear() {
	u.GetDoc().Transact(func(trans *Transaction) {
		clearItem := func(stackItem *StackItem) {
			IterateDeletedStructs(trans, stackItem.Deletions, func(s IAbstractStruct) {
				item, ok := s.(*Item)
				isParent := false
				for _, scope := range u.Scopes {
					if IsParentOf(scope, item) {
						isParent = true
						break
					}
				}

				if ok && isParent {
					KeepItem(item, false)
				}
			})
		}

		for _, stack := range u.UndoStack {
			clearItem(stack)
		}

		for _, stack := range u.RedoStack {
			clearItem(stack)
		}
	}, nil)

	u.UndoStack = nil
	u.RedoStack = nil
}

// UndoManager merges Undo-StackItem if they are created within time-gap
// smaller than `options.captureTimeout`. Call `um.stopCapturing()` so that the next
// StackItem won't be merged.
//
// @example
//
//	// without stopCapturing
//	ytext.insert(0, 'a')
//	ytext.insert(1, 'b')
//	um.undo()
//	ytext.toString() // => '' (note that 'ab' was removed)
//	// with stopCapturing
//	ytext.insert(0, 'a')
//	um.stopCapturing()
//	ytext.insert(0, 'b')
//	um.undo()
//	ytext.toString() // => 'a' (note that only 'b' was removed)
func (u *UndoManager) StopCapturing() {
	u.LastChange = 0
}

// Undo last changes on type.
func (u *UndoManager) Undo() *StackItem {
	u.Undoing = true
	res := PopStackItem(u, u.UndoStack, "undo")
	u.Undoing = false
	return res
}

// Redo last undo operation.
func (u *UndoManager) Redo() *StackItem {
	u.Redoing = true
	res := PopStackItem(u, u.RedoStack, "redo")
	u.Redoing = false
	return res
}

func NewStackItem(deletes *DeleteSet, insertions *DeleteSet) *StackItem {
	return &StackItem{
		Deletions:  deletes,
		Insertions: insertions,
		Meta:       make(map[interface{}]interface{}),
	}
}

func PopStackItem(undoManager *UndoManager, stack []*StackItem, eventType string) *StackItem {
	// Whether a change happened
	var result *StackItem

	// Keep a reference to the transaction so we can fire the event with the changedParentTypes
	var tr *Transaction
	doc := undoManager.GetDoc()
	scopes := undoManager.Scopes
	Transact(doc, func(trans *Transaction) {
		for len(stack) > 0 && result == nil {
			store := doc.Store
			stackItem := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			itemsToRedo := NewSet()
			var itemsToDelete []*Item
			performedChange := false

			IterateDeletedStructs(trans, stackItem.Insertions, func(s IAbstractStruct) {
				it, ok := s.(*Item)
				if ok {
					if it.Redone != nil {
						item, diff := FollowRedone(store, it.ID)
						if diff > 0 {
							item = GetItemCleanStart(trans, GenID(item.ID.Client, item.ID.Clock+diff))
						}
						it = item
					}

					isParent := false
					for _, scope := range scopes {
						if IsParentOf(scope, it) {
							isParent = true
							break
						}
					}

					if !it.Deleted() && isParent {
						itemsToDelete = append(itemsToDelete, it)
					}
				}
			})

			IterateDeletedStructs(trans, stackItem.Deletions, func(s IAbstractStruct) {
				it, ok := s.(*Item)
				isParent := false
				for _, scope := range scopes {
					if IsParentOf(scope, it) {
						isParent = true
						break
					}
				}

				if ok && isParent && !IsDeleted(stackItem.Insertions, s.GetID()) {
					// Never redo structs in stackItem.insertions because they were created and deleted in the same capture interval.
					itemsToRedo.Add(it)
				}
			})

			for s := range itemsToRedo {
				performedChange = RedoItem(trans, s.(*Item), itemsToRedo) != nil || performedChange
			}

			// We want to delete in reverse order so that children are deleted before
			// parents, so we have more information available when items are filtered.
			for i := len(itemsToDelete) - 1; i >= 0; i-- {
				item := itemsToDelete[i]
				if undoManager.DeleteFilter(item) {
					item.Delete(trans)
					performedChange = true
				}
			}

			if performedChange {
				result = stackItem
			} else {
				result = nil
			}
		}

		for t, subProps := range trans.Changed {
			// destroy search marker if necessary
			if subProps.Has(nil) && t.(IAbstractType).GetSearchMarker() != nil {
				// destroy search marker if necessary
				t.(IAbstractType).SetSearchMarker(nil)
			}
		}

		tr = trans
	}, undoManager, true)

	if result != nil {
		changedParentTypes := tr.ChangedParentTypes
		obj := NewObject()
		obj["stackItem"] = result
		obj["type"] = eventType
		obj["changedParentTypes"] = changedParentTypes
		undoManager.Emit("stack-item-popped", obj, undoManager)
	}

	return result
}

func NewUndoManager(typeScope IAbstractType, captureTimeout Number, deleteFilter func(item *Item) bool, trackedOrigins Set) *UndoManager {
	u := &UndoManager{}
	u.Observable = NewObservable()
	u.Scopes = append(u.Scopes, typeScope)
	u.DeleteFilter = deleteFilter
	trackedOrigins.Add(u)
	u.TrackedOrigins = trackedOrigins

	u.Undoing = false
	u.Redoing = false

	doc := u.GetDoc()
	u.LastChange = 0
	doc.On("afterTransaction", NewObserverHandler(func(v ...interface{}) {
		// Only track certain transactions
		if len(v) == 0 {
			return
		}

		trans := v[0].(*Transaction)
		for _, t := range u.Scopes {
			if _, exist := trans.ChangedParentTypes[t]; !exist {
				return
			}
		}

		if trans.Origin != nil {
			if !u.TrackedOrigins.Has(trans.Origin) && (trans.Origin != nil || !u.TrackedOrigins.Has(reflect.TypeOf(trans.Origin).String())) {
				return
			}
		}
		undoing := u.Undoing
		redoing := u.Redoing

		var stack *[]*StackItem
		if undoing {
			stack = &u.RedoStack
		} else {
			stack = &u.UndoStack
		}

		if undoing {
			// next undo should not be appended to last stack item
			u.StopCapturing()
		} else if !redoing {
			// neither undoing nor redoing: delete redoStack
			u.RedoStack = nil
		}

		insertions := NewDeleteSet()
		for client, endClock := range trans.AfterState {
			startClock := trans.BeforeState[client]
			length := endClock - startClock
			if length > 0 {
				AddToDeleteSet(insertions, client, startClock, length)
			}
		}

		now := GetUnixTime() // ms
		if Number(now)-u.LastChange < captureTimeout && len(*stack) > 0 && !undoing && !redoing {
			// append change to last stack op
			lastOp := (*stack)[len(*stack)-1]
			lastOp.Deletions = MergeDeleteSets([]*DeleteSet{lastOp.Deletions, trans.DeleteSet})
			lastOp.Insertions = MergeDeleteSets([]*DeleteSet{lastOp.Insertions, insertions})
		} else {
			// create a new stack op
			*stack = append(*stack, NewStackItem(trans.DeleteSet, insertions))
		}

		if !undoing && !redoing {
			u.LastChange = Number(now)
		}

		// make sure that deleted structs are not gc'd
		IterateDeletedStructs(trans, trans.DeleteSet, func(item IAbstractStruct) {
			keep := false
			if IsSameType(item, &Item{}) {
				it := item.(*Item)
				for _, t := range u.Scopes {
					if IsParentOf(t, it) {
						keep = true
						break
					}
				}
			}

			if keep {
				KeepItem(item.(*Item), true)
			}
		})

		obj := NewObject()
		obj["stackItem"] = (*stack)[len(*stack)-1]
		obj["origin"] = trans.Origin
		if undoing {
			obj["type"] = "redo"
		} else {
			obj["type"] = "undo"
		}
		obj["changedParentTypes"] = trans.ChangedParentTypes
		u.Emit("stack-item-added", obj, u)
	}))

	return u
}
