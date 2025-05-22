package y_crdt

import (
	"sort"
)

/*
 * A transaction is created for every change on the Yjs model. It is possible
 * to bundle changes on the Yjs model in a single transaction to
 * minimize the number on messages sent and the number of observer calls.
 * If possible the user of this library should bundle as many changes as
 * possible. Here is an example to illustrate the advantages of bundling:
 *
 * @example
 * ----------------------------------------------------------------------------
 *	 const map = y.define('map', YMap)
 *	 // Log content when change is triggered
 *	 map.observe(() => {
 *	   console.Log('change triggered')
 *	 })
 *	 // Each change on the map type triggers a Log message:
 * 	 map.set('a', 0) // => "change triggered"
 *	 map.set('b', 0) // => "change triggered"
 *	 // When put in a transaction, it will trigger the Log after the transaction:
 *	 y.transact(() => {
 *	   map.set('a', 1)
 *	   map.set('b', 1)
 *	 }) // => "change triggered"
 * ----------------------------------------------------------------------------
 */

type Transaction struct {
	// The yjs instance
	Doc *Doc

	// Describes the set of deleted items by ids
	DeleteSet *DeleteSet

	// Holds the state before the transaction started
	BeforeState map[Number]Number

	// Holds the state after the transaction
	AfterState map[Number]Number

	// All types that were directly modified (property added or child inserted/deleted).
	// New types are not included in this Set. Maps from type to parentSubs
	// (`item.parentSub = null` for YArray).
	Changed map[interface{}]Set

	// Stores the events for the types that observe also child elements.
	// It is mainly used by `observeDeep`.
	ChangedParentTypes map[interface{}][]IEventType

	// Stores the events for the types that observe also child elements.
	// It is mainly used by `observeDeep`.
	MergeStructs []IAbstractStruct

	Origin interface{}

	// Stores meta information on the transaction
	Meta map[interface{}]Set

	// Whether this change originates from this doc.
	Local bool

	SubdocsAdded   Set
	SubdocsRemoved Set
	SubdocsLoaded  Set
}

func NewTransaction(doc *Doc, origin interface{}, local bool) *Transaction {
	return &Transaction{
		Doc:                doc,
		Origin:             origin,
		Local:              local,
		BeforeState:        GetStateVector(doc.Store),
		AfterState:         make(map[Number]Number),
		Changed:            make(map[interface{}]Set),
		DeleteSet:          NewDeleteSet(),
		ChangedParentTypes: make(map[interface{}][]IEventType),
		Meta:               make(map[interface{}]Set),
		SubdocsAdded:       NewSet(),
		SubdocsRemoved:     NewSet(),
		SubdocsLoaded:      NewSet(),
	}
}

func WriteUpdateMessageFromTransaction(encoder *UpdateEncoderV1, trans *Transaction) bool {
	if len(trans.DeleteSet.Clients) == 0 && !MapAny(trans.AfterState, func(client, clock Number) bool {
		return trans.BeforeState[client] != clock
	}) {
		return false
	}

	SortAndMergeDeleteSet(trans.DeleteSet)
	WriteStructsFromTransaction(encoder, trans)
	WriteDeleteSet(encoder, trans.DeleteSet)
	return true
}

func NextID(trans *Transaction) ID {
	y := trans.Doc
	return GenID(y.ClientID, GetState(y.Store, y.ClientID))
}

// If `type.parent` was added in current transaction, `type` technically
// did not change, it was just added and we should not fire events for `type`.
func AddChangedTypeToTransaction(trans *Transaction, t IAbstractType, parentSub string) {
	item := t.GetItem()
	if item == nil || (item.ID.Clock < trans.BeforeState[item.ID.Client] && !item.Deleted()) {
		_, exist := trans.Changed[t]
		if !exist {
			trans.Changed[t] = NewSet()
		}

		trans.Changed[t].Add(parentSub)
	}
}

func TryToMergeWithLeft(structs *[]IAbstractStruct, pos Number) {
	left := (*structs)[pos-1]
	right := (*structs)[pos]

	if left.Deleted() == right.Deleted() && IsSameType(left, right) {
		if left.MergeWith(right) {
			SpliceStruct(structs, pos, 1, nil)
			if IsItemPtr(right) {
				r := right.(*Item)
				if r.ParentSub != "" && r.Parent.(IAbstractType).GetMap()[r.ParentSub] == right {
					r.Parent.(IAbstractType).GetMap()[r.ParentSub] = left.(*Item)
				}
			}
		}
	}
}

func TryGcDeleteSet(ds *DeleteSet, store *StructStore, gcFilter func(item *Item) bool) {
	for client, deleteItems := range ds.Clients {
		structs := store.Clients[client]

		for di := len(deleteItems) - 1; di >= 0; di-- {
			deleteItem := deleteItems[di]
			endDeleteItemClock := deleteItem.Clock + deleteItem.Length

			si, _ := FindIndexSS(*structs, deleteItem.Clock)
			s := (*structs)[si]

			for si < len(*structs) && s.GetID().Clock < endDeleteItemClock {
				s = (*structs)[si]
				if deleteItem.Clock+deleteItem.Length <= s.GetID().Clock {
					break
				}

				if IsItemPtr(s) && s.Deleted() && !s.(*Item).Keep() && gcFilter(s.(*Item)) {
					s.(*Item).GC(store, false)
				}

				si++
				// s = (*structs)[si]
			}
		}
	}
}

func TryMergeDeleteSet(ds *DeleteSet, store *StructStore) {
	// try to merge deleted / gc'd items
	// merge from right to left for better efficiecy and so we don't miss any merge targets
	for client, deleteItems := range ds.Clients {
		structs := store.Clients[client]
		for di := len(deleteItems) - 1; di >= 0; di-- {
			deleteItem := deleteItems[di]
			// start with merging the item next to the last deleted item
			n, _ := FindIndexSS(*structs, deleteItem.Clock+deleteItem.Length-1)
			mostRightIndexToCheck := Min(len(*structs)-1, 1+n)

			si := mostRightIndexToCheck
			s := (*structs)[si]
			for si > 0 && s.GetID().Clock >= deleteItem.Clock {
				TryToMergeWithLeft(structs, si)

				si--
				s = (*structs)[si]
			}
		}
	}
}

func TryGc(ds *DeleteSet, store *StructStore, gcFilter func(item *Item) bool) {
	TryGcDeleteSet(ds, store, gcFilter)
	TryMergeDeleteSet(ds, store)
}

func CleanupTransactions(transactionCleanups []*Transaction, i Number) {
	if i < len(transactionCleanups) {
		trans := transactionCleanups[i]
		doc := trans.Doc
		store := doc.Store
		ds := trans.DeleteSet
		mergeStructs := trans.MergeStructs

		SortAndMergeDeleteSet(ds)
		trans.AfterState = GetStateVector(trans.Doc.Store)
		doc.Trans = nil
		doc.Emit("beforeObserverCalls", trans, doc)

		// An array of event callbacks.
		var fs []func(...interface{})
		for itemType, subs := range trans.Changed {
			fs = append(fs, func(...interface{}) {
				if itemType.(IAbstractType).GetItem() == nil || !itemType.(IAbstractType).GetItem().Deleted() {
					itemType.(IAbstractType).CallObserver(trans, subs)
				}
			})
		}

		fs = append(fs, func(...interface{}) {
			// deep observe events
			for t, events := range trans.ChangedParentTypes {
				fs = append(fs, func(i ...interface{}) {
					// We need to think about the possibility that the user transforms the
					// Y.Doc in the event.
					if t.(IAbstractType).GetItem() == nil || !t.(IAbstractType).GetItem().Deleted() {
						events = ArrayFilter(events, func(e IEventType) bool {
							return e.GetTarget().GetItem() == nil || !e.GetTarget().GetItem().Deleted()
						})

						for _, e := range events {
							e.SetCurrentTarget(t.(IAbstractType))
						}

						// sort events by path length so that top-level events are fired first.
						sort.Slice(events, func(i, j int) bool {
							return len(events[i].Path()) < len(events[j].Path())
						})

						// We don't need to check for events.length
						// because we know it has at least one element
						CallEventHandlerListeners(t.(IAbstractType).GetDEH(), events, trans)
					}
				})
			}

			fs = append(fs, func(...interface{}) {
				doc.Emit("afterTransaction", trans, doc)
			})
		})

		CallAll(&fs, nil, 0)

		// Replace deleted items with ItemDeleted / GC.
		// This is where content is actually remove from the Yjs Doc.
		if doc.GC {
			TryGcDeleteSet(ds, store, doc.GCFilter)
		}
		TryMergeDeleteSet(ds, store)

		// on all affected store.clients props, try to merge
		for client, clock := range trans.AfterState {
			beforeClock := trans.BeforeState[client]
			if beforeClock != clock {
				structs := store.Clients[client]

				// we iterate from right to left so we can safely remove entries
				index, _ := FindIndexSS(*structs, beforeClock)
				firstChangePos := Max(index, 1)
				for i := len(*structs) - 1; i >= firstChangePos; i-- {
					TryToMergeWithLeft(structs, i)
				}
			}
		}

		// try to merge mergeStructs
		// @todo: it makes more sense to transform mergeStructs to a DS, sort it, and merge from right to left
		//        but at the moment DS does not handle duplicates
		for i := 0; i < len(mergeStructs); i++ {
			id := mergeStructs[i].GetID()
			structs := store.Clients[id.Client]
			replacedStructPos, _ := FindIndexSS(*structs, id.Clock)
			if replacedStructPos+1 < len(*structs) {
				TryToMergeWithLeft(structs, replacedStructPos+1)
			}

			if replacedStructPos > 0 {
				TryToMergeWithLeft(structs, replacedStructPos)
			}
		}

		if !trans.Local && trans.AfterState[doc.ClientID] != trans.BeforeState[doc.ClientID] {
			doc.ClientID = GenerateNewClientID()
			Logf("[crdt] Changed the client-id because another client seems to be using it.")
		}

		// @todo Merge all the transactions into one and provide send the data as a single update message
		doc.Emit("afterTransactionCleanup", trans, doc)
		if _, exist := doc.Observers["update"]; exist {
			encoder := NewUpdateEncoderV1()
			hasContent := WriteUpdateMessageFromTransaction(encoder, trans)
			if hasContent {
				doc.Emit("update", encoder.ToUint8Array(), trans.Origin, doc, trans)
			}
		}

		if _, exist := doc.Observers["updateV2"]; exist {
			encoderV1 := NewUpdateEncoderV1()

			hasContent := WriteUpdateMessageFromTransaction(encoderV1, trans)
			if hasContent {
				encoderV2 := NewUpdateEncoderV2()
				encoderV2.RestEncoder = encoderV1.RestEncoder

				doc.Emit("updateV2", encoderV2.ToUint8Array(), trans.Origin, doc, trans)
			}
		}

		for subdoc := range trans.SubdocsAdded {
			doc.SubDocs.Add(subdoc)
		}

		for subdoc := range trans.SubdocsRemoved {
			doc.SubDocs.Delete(subdoc)
		}

		doc.Emit("subdocs", Object{
			"loaded":  trans.SubdocsLoaded,
			"added":   trans.SubdocsAdded,
			"removed": trans.SubdocsRemoved})

		for subdoc := range trans.SubdocsRemoved {
			subdoc.(*Doc).Destroy()
		}

		if len(transactionCleanups) <= i+1 {
			doc.TransCleanup = nil
			doc.Emit("afterAllTransactions", doc, transactionCleanups)
		} else {
			CleanupTransactions(transactionCleanups, i+1)
		}
	}
}

// Implements the functionality of `y.transact(()=>{..})`
//
// default parameters: origin = nil, local = true
func Transact(doc *Doc, f func(trans *Transaction), origin interface{}, local bool) {
	transactionCleanups := doc.TransCleanup
	initialCall := false

	if doc.Trans == nil {
		initialCall = true
		doc.Trans = NewTransaction(doc, origin, local)
		transactionCleanups = append(transactionCleanups, doc.Trans)
		if len(transactionCleanups) == 1 {
			doc.Emit("beforeAllTransactions", doc)
		}

		doc.Emit("beforeTransaction", doc.Trans, doc)
	}

	f(doc.Trans)

	if initialCall && transactionCleanups[0] == doc.Trans {
		// The first transaction ended, now process observer calls.
		// Observer call may create new transactions for which we need to call the observers and do cleanup.
		// We don't want to nest these calls, so we execute these calls one after
		// another.
		// Also we need to ensure that all cleanups are called, even if the
		// observes throw errors.
		// This file is full of hacky try {} finally {} blocks to ensure that an
		// event can throw errors and also that the cleanup is called.
		CleanupTransactions(transactionCleanups, 0)
	}
}
