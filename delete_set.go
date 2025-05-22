package y_crdt

import (
	"errors"
	"sort"

	"github.com/mitchellh/copystructure"
)

/*
 * We no longer maintain a DeleteStore. DeleteSet is a temporary object that is created when needed.
 * - When created in a transaction, it must only be accessed after sorting, and merging
 *   - This DeleteSet is send to other clients
 * - We do not create a DeleteSet when we send a sync message. The DeleteSet message is created directly from StructStore
 * - We read a DeleteSet as part of a sync/update message. In this case the DeleteSet is already sorted and merged.
 */

type DeleteItem struct {
	Clock  Number
	Length Number
}

type DeleteSet struct {
	Clients map[Number][]*DeleteItem
}

func NewDeleteItem(clock Number, length Number) *DeleteItem {
	return &DeleteItem{Clock: clock, Length: length}
}

func NewDeleteSet() *DeleteSet {
	return &DeleteSet{
		Clients: make(map[Number][]*DeleteItem),
	}
}

// Iterate over all structs that the DeleteSet gc's.
// f func(*GC|*Item)
func IterateDeletedStructs(trans *Transaction, ds *DeleteSet, f func(s IAbstractStruct)) {
	for clientId, deletes := range ds.Clients {
		ss := trans.Doc.Store.Clients[clientId]
		if len(*ss) == 0 {
			continue
		}

		for i := 0; i < len(deletes); i++ {
			del := deletes[i]
			IterateStructs(trans, ss, del.Clock, del.Length, f)
		}
	}
}

func FindIndexDS(dis []*DeleteItem, clock Number) (Number, error) {
	left := 0
	right := len(dis) - 1

	for left <= right {
		midIndex := (left + right) / 2
		mid := dis[midIndex]
		midClock := mid.Clock

		if midClock <= clock {
			if clock < midClock+mid.Length {
				return midIndex, nil
			}
			left = midIndex + 1
		} else {
			right = midIndex - 1
		}
	}

	return 0, errors.New("not found")
}

func IsDeleted(ds *DeleteSet, id *ID) bool {
	dis := ds.Clients[id.Client]
	_, err := FindIndexDS(dis, id.Clock)
	return dis != nil && err == nil
}

func SortAndMergeDeleteSet(ds *DeleteSet) {
	for client, dels := range ds.Clients {
		sort.Slice(dels, func(i, j int) bool {
			return dels[i].Clock < dels[j].Clock
		})

		// merge items without filtering or splicing the array
		// i is the current pointer
		// j refers to the current insert position for the pointed item
		// try to merge dels[i] into dels[j-1] or set dels[j]=dels[i]
		i, j := 1, 1
		for ; i < len(dels); i++ {
			left := dels[j-1]
			right := dels[i]

			if left.Clock+left.Length >= right.Clock {
				left.Length = Max(left.Length, right.Clock+right.Length-left.Clock)
			} else {
				if j < i {
					dels[j] = right
				}
				j++
			}
		}

		ds.Clients[client] = dels[:j]
	}
}

func MergeDeleteSets(dss []*DeleteSet) *DeleteSet {
	merged := NewDeleteSet()

	for dssI := 0; dssI < len(dss); dssI++ {
		for client, delsLeft := range dss[dssI].Clients {
			_, exist := merged.Clients[client]
			if !exist {
				// Write all missing keys from current ds and all following.
				// If merged already contains `client` current ds has already been added.

				cp, err := copystructure.Copy(delsLeft)
				if err != nil {
					continue
				}

				dels := cp.([]*DeleteItem)
				for i := dssI + 1; i < len(dss); i++ {
					dels = append(dels, dss[i].Clients[client]...)
				}

				merged.Clients[client] = dels
			}
		}
	}

	SortAndMergeDeleteSet(merged)
	return merged
}

func AddToDeleteSet(ds *DeleteSet, client Number, clock Number, length Number) {
	ds.Clients[client] = append(ds.Clients[client], NewDeleteItem(clock, length))
}

func NewDeleteSetFromStructStore(ss *StructStore) *DeleteSet {
	ds := NewDeleteSet()
	for client, structs := range ss.Clients {
		var disItems []*DeleteItem
		for i := 0; i < len(*structs); i++ {
			s := (*structs)[i]
			if s.Deleted() {
				clock := s.GetID().Clock
				length := s.GetLength()

				for i+1 < len(*structs) {
					next := (*structs)[i+1]
					if next.Deleted() {
						length += next.GetLength()
						i++
					} else {
						break
					}
				}

				disItems = append(disItems, NewDeleteItem(clock, length))
			}
		}

		if len(disItems) > 0 {
			ds.Clients[client] = disItems
		}
	}

	return ds
}

func WriteDeleteSet(encoder *UpdateEncoderV1, ds *DeleteSet) {
	WriteVarUint(encoder.RestEncoder, uint64(len(ds.Clients)))

	for client, dsItems := range ds.Clients {
		encoder.ResetDsCurVal()
		WriteVarUint(encoder.RestEncoder, uint64(client))

		length := len(dsItems)
		WriteVarUint(encoder.RestEncoder, uint64(length))

		for i := 0; i < length; i++ {
			item := dsItems[i]
			encoder.WriteDsClock(item.Clock)
			encoder.WriteDsLen(item.Length)
		}
	}
}

func WriteDeleteSetV2(encoder *UpdateEncoderV2, ds *DeleteSet) {
	WriteVarUint(encoder.RestEncoder, uint64(len(ds.Clients)))

	for client, dsItems := range ds.Clients {
		encoder.ResetDsCurVal()
		WriteVarUint(encoder.RestEncoder, uint64(client))

		length := len(dsItems)
		WriteVarUint(encoder.RestEncoder, uint64(length))

		for i := 0; i < length; i++ {
			item := dsItems[i]
			encoder.WriteDsClock(item.Clock)
			encoder.WriteDsLen(item.Length)
		}
	}
}

func ReadDeleteSet(decoder *UpdateDecoderV1) *DeleteSet {
	ds := NewDeleteSet()

	n, err := readVarUint(decoder.RestDecoder)
	if err != nil {
		return nil
	}

	numClients := n.(uint64)
	for i := uint64(0); i < numClients; i++ {
		decoder.ResetDsCurVal()

		n, err = readVarUint(decoder.RestDecoder)
		if err != nil {
			return nil
		}

		client := Number(n.(uint64))

		n, err = readVarUint(decoder.RestDecoder)
		if err != nil {
			return nil
		}

		numberOfDeletes := n.(uint64)

		for j := uint64(0); j < numberOfDeletes; j++ {
			dsClock, err := decoder.ReadDsClock()
			if err != nil {
				return nil
			}

			dsLength, err := decoder.ReadDsLen()
			if err != nil {
				return nil
			}

			ds.Clients[client] = append(ds.Clients[client], NewDeleteItem(dsClock, dsLength))
		}
	}

	return ds
}

func ReadAndApplyDeleteSet(decoder *UpdateDecoderV1, trans *Transaction, store *StructStore) []uint8 {
	n, err := readVarUint(decoder.RestDecoder)
	if err != nil {
		return nil
	}

	unappliedDS := NewDeleteSet()
	numClients := n.(uint64)

	for i := uint64(0); i < numClients; i++ {
		decoder.ResetDsCurVal()

		n, err = readVarUint(decoder.RestDecoder)
		if err != nil {
			return nil
		}
		client := Number(n.(uint64))

		n, err = readVarUint(decoder.RestDecoder)
		if err != nil {
			return nil
		}
		numberOfDeletes := n.(uint64)

		structs := store.Clients[client]
		state := GetState(store, client)
		for j := uint64(0); j < numberOfDeletes; j++ {
			clock, err := decoder.ReadDsClock()
			if err != nil {
				return nil
			}

			length, err := decoder.ReadLen()
			if err != nil {
				return nil
			}

			clockEnd := clock + length

			if clock < state {
				if state < clockEnd {
					AddToDeleteSet(unappliedDS, client, state, clockEnd-state)
				}

				index, err := FindIndexSS(*structs, clock)
				if err != nil {
					return nil
				}

				// We can ignore the case of GC and Delete structs, because we are going to skip them
				s := (*structs)[index]

				// split the first item if necessary
				if !s.Deleted() && s.GetID().Clock < clock {
					items := []IAbstractStruct{SplitItem(trans, s.(*Item), clock-s.GetID().Clock)}
					SpliceStruct(structs, index+1, 0, items)
					index++ // increase we now want to use the next struct
				}

				for index < len(*structs) {
					st := (*structs)[index].(IAbstractStruct)
					index++

					if st.GetID().Clock < clockEnd {
						if !st.Deleted() {
							item, _ := st.(*Item)
							if item != nil {
								if clockEnd < item.GetID().Clock+item.GetLength() {
									items := []IAbstractStruct{SplitItem(trans, item, clockEnd-st.GetID().Clock)}
									SpliceStruct(structs, index, 0, items)
								}
								item.Delete(trans)
							}
						}
					} else {
						break
					}
				}
			} else {
				AddToDeleteSet(unappliedDS, client, clock, clockEnd-clock)
			}
		}
	}

	if len(unappliedDS.Clients) > 0 {
		ds := NewUpdateEncoderV2()
		WriteVarUint(ds.RestEncoder, 0)
		WriteDeleteSetV2(ds, unappliedDS)
		return ds.ToUint8Array()
	}

	return nil
}
