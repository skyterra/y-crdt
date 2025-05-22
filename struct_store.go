package y_crdt

import (
	"errors"
)

type StructStore struct {
	Clients        map[Number]*[]IAbstractStruct
	PendingStructs *RestStructs
	PendingDs      []uint8
}

func (ss *StructStore) GetStructs(client Number) []IAbstractStruct {
	return *ss.Clients[client]
}

func NewStructStore() *StructStore {
	return &StructStore{
		Clients: make(map[Number]*[]IAbstractStruct),
	}
}

// Return the states as a Map<client,clock>.
// Note that clock refers to the next expected clock id.
func GetStateVector(store *StructStore) map[Number]Number {
	sm := make(map[Number]Number)

	for client, structs := range store.Clients {
		s := (*structs)[len(*structs)-1]
		sm[client] = s.GetID().Clock + s.GetLength()
	}

	return sm
}

func GetState(store *StructStore, client Number) Number {
	structs, exist := store.Clients[client]
	if !exist {
		return 0
	}

	lastStruct := (*structs)[len(*structs)-1]
	return lastStruct.GetID().Clock + lastStruct.GetLength()
}

func IntegretyCheck(store *StructStore) error {
	for _, structs := range store.Clients {
		for i := 1; i < len(*structs); i++ {
			l := (*structs)[i-1]
			r := (*structs)[i]

			if l.GetID().Clock+l.GetLength() != r.GetID().Clock {
				return errors.New("StructStore failed integrety check")
			}
		}
	}

	return nil
}

func AddStruct(store *StructStore, st IAbstractStruct) error {
	client := st.GetID().Client
	ss, exist := store.Clients[client]

	if !exist {
		store.Clients[client] = &[]IAbstractStruct{st}
	} else {
		lastStruct := (*ss)[len(*ss)-1]
		if lastStruct.GetID().Clock+lastStruct.GetLength() != st.GetID().Clock {
			return errors.New("unexpected case")
		}

		*(store.Clients[client]) = append(*(store.Clients[client]), st)
	}

	return nil
}

func FindIndexSS(ss []IAbstractStruct, clock Number) (Number, error) {
	index, err := BinarySearch(ss, clock, 0, len(ss)-1)
	if err != nil {
		return 0, err
	}

	return index, nil
}

func BinarySearch(ss []IAbstractStruct, clock Number, begin, end Number) (Number, error) {
	if begin > end {
		return 0, errors.New("not found")
	}

	mid := (begin + end) / 2
	if ss[mid].GetID().Clock <= clock && clock < ss[mid].GetID().Clock+ss[mid].GetLength() {
		return mid, nil
	}

	if ss[mid].GetID().Clock <= clock {
		begin = mid + 1
	} else {
		end = mid - 1
	}

	return BinarySearch(ss, clock, begin, end)
}

func Find(store *StructStore, id ID) (IAbstractStruct, error) {
	ss := store.Clients[id.Client]
	index, err := FindIndexSS(*ss, id.Clock)
	if err != nil {
		return nil, err
	}

	return (*ss)[index], nil
}

func GetItem(store *StructStore, id ID) IAbstractStruct {
	item, err := Find(store, id)
	if err != nil {
		Logf("[crdt] %s.", err.Error())
	}
	return item
}

// ss可能会被切割，所以需要按指针传递
func FindIndexCleanStart(trans *Transaction, ss *[]IAbstractStruct, clock Number) (Number, error) {
	index, err := FindIndexSS(*ss, clock)
	if err != nil {
		return index, err
	}

	s, ok := (*ss)[index].(*Item)
	if ok && s.GetID().Clock < clock {
		items := []IAbstractStruct{SplitItem(trans, s, clock-s.GetID().Clock)}
		SpliceStruct(ss, index+1, 0, items)
		return index + 1, nil
	}

	return index, nil
}

// Expects that id is actually in store. This function throws or is an infinite loop otherwise.
func GetItemCleanStart(trans *Transaction, id ID) *Item {
	ss, exist := trans.Doc.Store.Clients[id.Client]
	if !exist {
		return nil
	}

	index, err := FindIndexCleanStart(trans, ss, id.Clock)
	if err != nil {
		return nil
	}

	item, _ := (*ss)[index].(*Item)
	return item
}

// Expects that id is actually in store. This function throws or is an infinite loop otherwise.
func GetItemCleanEnd(trans *Transaction, store *StructStore, id ID) *Item {
	ss, exist := store.Clients[id.Client]
	if !exist {
		return nil
	}

	index, err := FindIndexSS(*ss, id.Clock)
	if err != nil {
		return nil
	}

	s, ok := (*ss)[index].(*Item)
	if !ok {
		return nil
	}

	if id.Clock != s.GetID().Clock+s.GetLength()-1 {
		rightItem := SplitItem(trans, s, id.Clock-s.GetID().Clock+1)
		SpliceStruct(ss, index+1, 0, []IAbstractStruct{rightItem})
	}

	return s
}

// Replace item(*GC|*Item) with newItem(*GC|*Item) in store
func ReplaceStruct(store *StructStore, item IAbstractStruct, newItem IAbstractStruct) error {
	if item.GetID().Client != newItem.GetID().Client {
		return errors.New("cannot replace struct when tow items' client are different")
	}

	ss, exist := store.Clients[item.GetID().Client]
	if !exist {
		return errors.New("not exist client")
	}

	index, err := FindIndexSS(*ss, item.GetID().Clock)
	if err != nil {
		return err
	}

	(*ss)[index] = newItem
	return nil
}

// Iterate over a range of structs
func IterateStructs(trans *Transaction, ss *[]IAbstractStruct, clockStart Number, length Number, f func(s IAbstractStruct)) {
	if length == 0 {
		return
	}

	clockEnd := clockStart + length
	index, err := FindIndexCleanStart(trans, ss, clockStart)
	if err != nil {
		return
	}

	for {
		s := (*ss)[index]
		index++

		if clockEnd < s.GetID().Clock+s.GetLength() {
			_, err := FindIndexCleanStart(trans, ss, clockEnd)
			if err != nil {
				Logf("[crdt] %s.", err.Error())
			}
		}

		f(s)

		if index >= len(*ss) || (*ss)[index].GetID().Clock >= clockEnd {
			break
		}
	}
}
