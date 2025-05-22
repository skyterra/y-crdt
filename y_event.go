package y_crdt

const (
	ActionAdd    = "add"
	ActionDelete = "delete"
	ActionUpdate = "update"
)

type EventAction struct {
	Action   string
	OldValue interface{}
	NewValue interface{}
}

type EventOperator struct {
	Insert     interface{} // string | Array<any>
	Retain     Number
	Delete     Number
	Attributes Object

	IsInsertDefined     bool
	IsRetainDefined     bool
	IsDeleteDefined     bool
	IsAttributesDefined bool
}

type IEventType interface {
	GetTarget() IAbstractType
	GetCurrentTarget() IAbstractType
	SetCurrentTarget(t IAbstractType)
	Path() []interface{}
}

// YEvent describes the changes on a YType.
type YEvent struct {
	Target        IAbstractType // The type on which this event was created on.
	CurrentTarget IAbstractType // The current target on which the observe callback is called.
	Trans         *Transaction  // The transaction that triggered this event.
	Changes       Object
	Keys          map[string]EventAction // Map<string, { action: 'add' | 'update' | 'delete', oldValue: any, newValue: any }>}
	delta         []EventOperator
}

func (y *YEvent) GetTarget() IAbstractType {
	return y.Target
}

func (y *YEvent) GetCurrentTarget() IAbstractType {
	return y.CurrentTarget
}

func (y *YEvent) SetCurrentTarget(t IAbstractType) {
	y.CurrentTarget = t
}

// Computes the path from `y` to the changed type.
//
// @todo v14 should standardize on path: Array<{parent, index}> because that is easier to work with.
//
// The following property holds:
// @example
// ----------------------------------------------------------------------------
//
//	let type = y
//	event.path.forEach(dir => {
//	  type = type.get(dir)
//	})
//	type === event.target // => true
//
// ----------------------------------------------------------------------------
func (y *YEvent) Path() []interface{} {
	return GetPathTo(y.CurrentTarget, y.Target)
}

// Check if a struct is deleted by this event.
// In contrast to change.deleted, this method also returns true if the struct was added and then deleted.
func (y *YEvent) Deletes(s IAbstractStruct) bool {
	return IsDeleted(y.Trans.DeleteSet, s.GetID())
}

func (y *YEvent) GetKeys() map[string]EventAction {
	if y.Keys == nil {
		keys := make(map[string]EventAction)
		target := y.Target
		changed := y.Trans.Changed[target]
		for key := range changed {
			if key != nil {
				strKey, ok := key.(string)
				if !ok {
					Log("[crdt] key is not string.")
					continue
				}

				item := target.GetMap()[strKey]

				var action string
				var oldValue interface{}
				var err error

				if y.Adds(item) {
					prev := item.Left
					for prev != nil && y.Adds(prev) {
						prev = prev.Left
					}

					if y.Deletes(item) {
						if prev != nil && y.Deletes(prev) {
							action = ActionDelete
							oldValue, err = ArrayLast(prev.Content.GetContent())
							if err != nil {
								Log("[crdt] %s.", err.Error())
								return nil
							}
						} else {
							return nil
						}
					} else {
						if prev != nil && y.Deletes(prev) {
							action = ActionUpdate
							oldValue, err = ArrayLast(prev.Content.GetContent())
							if err != nil {
								Log("[crdt] %s.", err.Error())
								return nil
							}
						} else {
							action = ActionAdd
							oldValue = nil
						}
					}
				} else {
					if y.Deletes(item) {
						action = ActionDelete
						oldValue, err = ArrayLast(item.Content.GetContent())
						if err != nil {
							Log("[crdt] %s.", err.Error())
							return nil
						}
					} else {
						return nil
					}
				}

				keys[strKey] = EventAction{
					Action:   action,
					OldValue: oldValue,
				}
			}
		}

		y.Keys = keys
	}

	return y.Keys
}

func (y *YEvent) GetDelta() []EventOperator {
	return y.GetChanges()["delta"].([]EventOperator)
}

// Check if a struct is added by this event.
// In contrast to change.deleted, this method also returns true if the struct was added and then deleted.
func (y *YEvent) Adds(s IAbstractStruct) bool {
	return s.GetID().Clock >= y.Trans.BeforeState[s.GetID().Client]
}

func (y *YEvent) GetChanges() Object {
	changes := y.Changes
	if changes == nil || len(changes) == 0 {
		target := y.Target
		added := NewSet()
		deleted := NewSet()
		var delta []EventOperator

		changes = NewObject()
		changes["added"] = added
		changes["deleted"] = deleted
		changes["keys"] = y.Keys

		changed := y.Trans.Changed[target]

		_, existNil := changed[nil]
		_, existEmpty := changed[""]
		if existNil || existEmpty {
			var lastOp *EventOperator
			packOp := func() {
				if lastOp != nil {
					delta = append(delta, *lastOp)
				}
			}

			for item := target.StartItem(); item != nil; item = item.Right {
				if item.Deleted() {
					if y.Deletes(item) && !y.Adds(item) {
						if lastOp == nil || !lastOp.IsDeleteDefined {
							packOp()
							lastOp = &EventOperator{}
						}
						lastOp.Delete += item.Length
						lastOp.IsDeleteDefined = true
						deleted.Add(item)
					} // else nop
				} else {
					if y.Adds(item) {
						if lastOp == nil || !lastOp.IsInsertDefined {
							packOp()

							lastOp = &EventOperator{
								Insert:          ArrayAny{},
								IsInsertDefined: true,
							}
						}

						lastOp.Insert = append(lastOp.Insert.(ArrayAny), item.Content.GetContent())
						lastOp.IsInsertDefined = true
						added.Add(item)
					} else {
						if lastOp == nil || !lastOp.IsRetainDefined {
							packOp()

							lastOp = &EventOperator{}
						}
						lastOp.Retain += item.Length
						lastOp.IsRetainDefined = true
					}
				}
			}

			if lastOp != nil && !lastOp.IsRetainDefined {
				packOp()
			}
		}

		changes["delta"] = delta
		y.Changes = changes
	}

	return changes
}

func NewYEvent(target IAbstractType, trans *Transaction) *YEvent {
	return &YEvent{
		Target:        target,
		CurrentTarget: target,
		Trans:         trans,
		Changes:       NewObject(),
		Keys:          make(map[string]EventAction),
	}
}

func NewDefaultYEvent() *YEvent {
	return &YEvent{}
}

// Compute the path from this type to the specified target.
//
// @example
// ----------------------------------------------------------------------------
//
//	// `child` should be accessible via `type.get(path[0]).get(path[1])..`
//	const path = type.getPathTo(child)
//	// assuming `type instanceof YArray`
//	console.Log(path) // might look like => [2, 'key1']
//	child === type.get(path[0]).get(path[1])
//
// ----------------------------------------------------------------------------
func GetPathTo(parent IAbstractType, child IAbstractType) []interface{} {
	var path []interface{}

	for child.GetItem() != nil && child != parent {
		if child.GetItem().ParentSub != "" {
			// parent is map-ish
			path = Unshift(path, child.GetItem().ParentSub)
		} else {
			// parent is array-ish
			i := 0
			c := child.GetItem().Parent.(IAbstractType).StartItem()
			for c != child.GetItem() && c != nil {
				if !c.Deleted() {
					i++
				}
				c = c.Right
			}
			path = Unshift(path, i)
		}

		child = child.GetItem().Parent.(IAbstractType)
	}

	return path
}
