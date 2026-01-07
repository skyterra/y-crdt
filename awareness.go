package y_crdt

import (
	"time"
)

/*
 * The Awareness class implements a simple shared state protocol that can be used for non-persistent data like awareness information
 * (cursor, username, status, ..). Each client can update its own local state and listen to state changes of
 * remote clients. Every client may set a state of a remote peer to `null` to mark the client as offline.
 *
 * Each client is identified by a unique client id (something we borrow from `doc.clientID`). A client can override
 * its own state by propagating a message with an increasing timestamp (`clock`). If such a message is received, it is
 * applied if the known state of that client is older than the new state (`clock < newClock`). If a client thinks that
 * a remote client is offline, it may propagate a message with
 * `{ clock: currentClientClock, state: null, client: remoteClient }`. If such a
 * message is received,  and the known clock of that client equals the received clock, it will override the state with `null`.
 *
 * Before a client disconnects, it should propagate a `null` state with an updated clock.
 *
 * Awareness states must be updated every 30 seconds. Otherwise the Awareness instance will delete the client state.
 *
 * @extends {Observable<string>}
 */

const OutdatedTimeout = 30 * time.Second

type Awareness struct {
	*Observable
	Doc      *Doc
	ClientID Number
	States   map[Number]Object
	Meta     map[Number]Object
	// CheckInterval *time.Timer
}

func (a *Awareness) Destroy() {
	a.Emit("destroy", []interface{}{a})
	a.SetLocalState(nil)
	a.Observable.Destroy()
	// ClearInterval(a.CheckInterval)
}

func (a *Awareness) GetLocalState() Object {
	return a.States[a.ClientID]
}

func (a *Awareness) SetLocalState(state Object) {
	clientID := a.ClientID
	currLocalMeta, ok := a.Meta[clientID]
	var clock Number
	if !ok {
		clock = 0
	} else {
		clock = currLocalMeta["clock"].(Number) + 1
	}
	prevState := a.States[clientID]
	if state == nil {
		delete(a.States, clientID)
	} else {
		a.States[clientID] = state
	}

	a.Meta[clientID] = Object{
		"clock":       clock,
		"lastUpdated": GetUnixTime(),
	}

	var added []Number
	var updated []Number
	var filteredUpdated []Number
	var removed []Number
	if state == nil {
		removed = append(removed, clientID)
	} else if prevState == nil {
		// if state != nil {
		added = append(added, clientID)
		// }
	} else {
		updated = append(updated, clientID)
		if EqualAttrs(prevState, state) {
			filteredUpdated = append(filteredUpdated, clientID)
		}
	}

	if len(added) > 0 || len(filteredUpdated) > 0 || len(removed) > 0 {
		a.Emit("change", Object{"added": added, "updated": filteredUpdated, "removed": removed}, "local")
	}

	a.Emit("update", Object{"added": added, "updated": updated, "removed": removed}, "local")
}

func (a *Awareness) SetLocalStateField(field string, value interface{}) {
	state := a.GetLocalState()
	if state != nil {
		obj := Object{}
		for key, val := range state {
			obj[key] = val
		}

		obj[field] = value
		a.SetLocalState(obj)
	}
}

func (a *Awareness) GetStates() map[Number]Object {
	return a.States
}

func NewAwareness(doc *Doc) *Awareness {
	aw := &Awareness{
		Observable: NewObservable(),
		Doc:        doc,
		ClientID:   doc.ClientID,
		States:     make(map[Number]Object),
		Meta:       make(map[Number]Object),
	}

	// aw.CheckInterval = time.AfterFunc(OutdatedTimeout/10, func() {
	// 	now := GetUnixTime()
	//
	// 	if aw.GetLocalState() != nil {
	// 		lastUpdated, ok := aw.Meta[aw.ClientID]["lastUpdated"].(int64)
	// 		if ok && int64(OutdatedTimeout/2) <= now-lastUpdated {
	// 			aw.SetLocalState(aw.GetLocalState())
	// 		}
	// 	}
	//
	// 	var remove []Number
	// 	for clientID, meta := range aw.Meta {
	// 		lastUpdated := meta["lastUpdated"].(int64)
	// 		_, existStates := aw.States[clientID]
	// 		if clientID != aw.ClientID && int64(OutdatedTimeout) <= now-lastUpdated && existStates {
	// 			remove = append(remove, clientID)
	// 		}
	// 	}
	//
	// 	if len(remove) > 0 {
	// 		RemoveAwarenessStates(aw, remove, "timeout")
	// 	}
	// })

	doc.On("destroy", NewObserverHandler(func(v ...interface{}) {
		aw.Destroy()
	}))
	aw.SetLocalState(make(Object))
	return aw
}

// Mark (remote) clients as inactive and remove them from the list of active peers.
// This change will be propagated to remote clients.
func RemoveAwarenessStates(awareness *Awareness, clients []Number, origin interface{}) {
	var added []Number
	var updated []Number
	var removed []Number
	for i := 0; i < len(clients); i++ {
		clientID := clients[i]
		if _, exist := awareness.States[clientID]; exist {
			delete(awareness.States, clientID)
			if clientID == awareness.ClientID {
				curMeta := awareness.Meta[clientID]
				awareness.Meta[clientID] = Object{
					"clock":       curMeta["clock"].(Number) + 1,
					"lastUpdated": GetUnixTime(),
				}
			}
			removed = append(removed, clientID)
		}
	}

	if len(removed) > 0 {
		awareness.Emit("change", Object{"added": added, "updated": updated, "removed": removed}, origin)
		awareness.Emit("update", Object{"added": added, "updated": updated, "removed": removed}, origin)
	}
}

func EncodeAwarenessUpdate(awareness *Awareness, clients []Number, states map[Number]Object) []byte {
	if states == nil {
		states = awareness.States
	}
	length := len(clients)
	encoder := NewEncoder()
	WriteVarUint(encoder, uint64(length))
	for i := 0; i < length; i++ {
		clientID := clients[i]
		state := states[clientID]
		clock := awareness.Meta[clientID]["clock"].(Number)
		WriteVarUint(encoder, uint64(clientID))
		WriteVarUint(encoder, uint64(clock))
		WriteString(encoder, JsonString(state))

	}
	return encoder.Bytes()
}

// Modify the content of an awareness update before re-encoding it to an awareness update.
//
// This might be useful when you have a central server that wants to ensure that clients
// cant hijack somebody elses identity.
func ModifyAwarenessUpdate(update []byte, modify func(interface{}) interface{}) []byte {
	decoder := NewDecoder(update)
	encoder := NewEncoder()
	length := ReadVarUint(decoder)
	WriteVarUint(encoder, length)
	for i := uint64(0); i < length; i++ {
		clientID := ReadVarUint(decoder)
		clock := ReadVarUint(decoder)

		data, _ := ReadString(decoder)
		state := JsonObject(data)
		modifiedState := modify(state)

		WriteVarUint(encoder, clientID)
		WriteVarUint(encoder, clock)
		WriteString(encoder, JsonString(modifiedState))
	}
	return encoder.Bytes()
}

func ApplyAwarenessUpdate(awareness *Awareness, update []byte, origin interface{}) {
	decoder := NewDecoder(update)
	timestamp := GetUnixTime()
	var added []Number
	var updated []Number
	var filteredUpdated []Number
	var removed []Number
	length := ReadVarUint(decoder)

	for i := uint64(0); i < length; i++ {
		clientID := Number(ReadVarUint(decoder))
		clock := Number(ReadVarUint(decoder))

		data, _ := ReadString(decoder)
		state := JsonObject(data)

		clientMeta := awareness.Meta[clientID]
		prevState := awareness.States[clientID]

		currClock := 0
		if clientMeta != nil {
			currClock = clientMeta["clock"].(Number)
		}

		_, exist := awareness.States[clientID]
		if currClock < clock || (currClock == clock && state == nil && exist) {
			if state == nil {
				// never let a remote client remove this local state
				if clientID == awareness.ClientID && awareness.GetLocalState() != nil {
					// remote client removed the local state. Do not remote state. Broadcast a message indicating
					// that this client still exists by increasing the clock
					clock++
				} else {
					delete(awareness.States, clientID)
				}
			} else {
				awareness.States[clientID] = state.(Object)
			}

			awareness.Meta[clientID] = Object{
				"clock":       clock,
				"lastUpdated": timestamp,
			}

			if clientMeta == nil && state != nil {
				added = append(added, clientID)
			} else if clientMeta != nil && state == nil {
				removed = append(removed, clientID)
			} else if state != nil {
				if !EqualAttrs(state, prevState) {
					filteredUpdated = append(filteredUpdated, clientID)
				}
				updated = append(updated, clientID)
			}
		}
	}

	if len(added) > 0 || len(filteredUpdated) > 0 || len(removed) > 0 {
		awareness.Emit("change", Object{"added": added, "updated": filteredUpdated, "removed": removed}, origin)
	}

	if len(added) > 0 || len(updated) > 0 || len(removed) > 0 {
		awareness.Emit("update", Object{"added": added, "updated": updated, "removed": removed}, origin)
	}
}

// VenusApplyAwarenessUpdate this method is belong golang venus library. Apply awareness'
// update without emit 'update' and 'change' event.
func VenusApplyAwarenessUpdate(awareness *Awareness, update []byte) {
	decoder := NewDecoder(update)
	timestamp := GetUnixTime()
	var added []Number
	var updated []Number
	var filteredUpdated []Number
	var removed []Number
	length := ReadVarUint(decoder)

	for i := uint64(0); i < length; i++ {
		clientID := Number(ReadVarUint(decoder))
		clock := Number(ReadVarUint(decoder))

		data, _ := ReadString(decoder)
		state := JsonObject(data)

		clientMeta := awareness.Meta[clientID]
		prevState := awareness.States[clientID]

		currClock := 0
		if clientMeta != nil {
			currClock = clientMeta["clock"].(Number)
		}

		_, exist := awareness.States[clientID]
		if currClock < clock || (currClock == clock && state == nil && exist) {
			if state == nil {
				// never let a remote client remove this local state
				if clientID == awareness.ClientID && awareness.GetLocalState() != nil {
					// remote client removed the local state. Do not remote state. Broadcast a message indicating
					// that this client still exists by increasing the clock
					clock++
				} else {
					delete(awareness.States, clientID)
				}
			} else {
				awareness.States[clientID] = state.(Object)
			}

			awareness.Meta[clientID] = Object{
				"clock":       clock,
				"lastUpdated": timestamp,
			}

			if clientMeta == nil && state != nil {
				added = append(added, clientID)
			} else if clientMeta != nil && state == nil {
				removed = append(removed, clientID)
			} else if state != nil {
				if !EqualAttrs(state, prevState) {
					filteredUpdated = append(filteredUpdated, clientID)
				}
				updated = append(updated, clientID)
			}
		}
	}
}
