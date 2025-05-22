package y_crdt

import "errors"

type Snapshot struct {
	Ds *DeleteSet
	Sv map[Number]Number // state map
}

func NewSnapshot(ds *DeleteSet, sv map[Number]Number) *Snapshot {
	return &Snapshot{
		Ds: ds,
		Sv: sv,
	}
}

func EqualSnapshots(snap1, snap2 *Snapshot) bool {
	ds1 := snap1.Ds.Clients
	ds2 := snap2.Ds.Clients
	sv1 := snap1.Sv
	sv2 := snap2.Sv

	if len(sv1) != len(sv2) || len(ds1) != len(ds2) {
		return false
	}

	for key, value := range sv1 {
		if sv2[key] != value {
			return false
		}
	}

	for client, delItems1 := range ds1 {
		delItems2 := ds2[client]
		if len(delItems1) != len(delItems2) {
			return false
		}

		for i := 0; i < len(delItems1); i++ {
			delItem1 := delItems1[i]
			delItem2 := delItems2[i]

			if delItem1.Clock != delItem2.Clock || delItem1.Length != delItem2.Length {
				return false
			}
		}
	}

	return true
}

func EncodeSnapshotV2(snapshot *Snapshot, encoder *UpdateEncoderV1) []uint8 {
	WriteDeleteSet(encoder, snapshot.Ds)
	WriteStateVector(encoder, snapshot.Sv)
	return encoder.ToUint8Array()
}

func EncodeSnapshot(snapshot *Snapshot) []uint8 {
	return EncodeSnapshotV2(snapshot, NewUpdateEncoderV1())
}

func DecodeSnapshotV2(buf []uint8) *Snapshot {
	decoder := NewUpdateDecoderV1(buf)
	ds := ReadDeleteSet(decoder)
	sv := ReadStateVector(decoder)

	return NewSnapshot(ds, sv)
}

func DecodeSnapshot(buf []uint8) *Snapshot {
	return DecodeSnapshotV2(buf)
}

func EmptySnapshot() *Snapshot {
	return NewSnapshot(NewDeleteSet(), make(map[Number]Number))
}

// snapshot(doc)
func NewSnapshotByDoc(doc *Doc) *Snapshot {
	return NewSnapshot(NewDeleteSetFromStructStore(doc.Store), GetStateVector(doc.Store))
}

func IsVisible(item *Item, snapshot *Snapshot) bool {
	if snapshot == nil {
		return !item.Deleted()
	}

	state := snapshot.Sv[item.ID.Client]
	return state > item.ID.Clock && !IsDeleted(snapshot.Ds, &item.ID)
}

func SplitSnapshotAffectedStructs(trans *Transaction, snapshot *Snapshot) {
	_, exist := trans.Meta[SplitSnapshotAffectedStructs]
	if !exist {
		trans.Meta[SplitSnapshotAffectedStructs] = NewSet()
	}

	meta := trans.Meta[SplitSnapshotAffectedStructs]
	store := trans.Doc.Store

	// check if we already split for this snapshot
	if _, exist := meta[snapshot]; !exist {
		for client, clock := range snapshot.Sv {
			if clock < GetState(store, client) {
				GetItemCleanStart(trans, GenID(client, clock))
			}
		}

		IterateDeletedStructs(trans, snapshot.Ds, func(s IAbstractStruct) {})
		meta.Add(snapshot)
	}
}

func CreateDocFromSnapshot(originDoc *Doc, snapshot *Snapshot, newDoc *Doc) (*Doc, error) {
	if originDoc.GC {
		// we should not try to restore a GC-ed document, because some of the restored items might have their content deleted
		return nil, errors.New("originDoc must not be garbage collected")
	}

	ds, sv := snapshot.Ds, snapshot.Sv
	encoder := NewUpdateEncoderV1()
	originDoc.Transact(func(trans *Transaction) {
		size := uint64(0)
		for _, clock := range sv {
			if clock > 0 {
				size++
			}
		}

		WriteVarUint(encoder.RestEncoder, size)
		// splitting the structs before writing them to the encoder
		for client, clock := range sv {
			if clock == 0 {
				continue
			}

			if clock < GetState(originDoc.Store, client) {
				GetItemCleanStart(trans, GenID(client, clock))
			}

			structs := originDoc.Store.Clients[client]
			lastStructIndex, _ := FindIndexSS(*structs, clock-1)

			// write # encoded structs
			WriteVarUint(encoder.RestEncoder, uint64(lastStructIndex+1))
			encoder.WriteClient(client)

			// first clock written is 0
			WriteVarUint(encoder.RestEncoder, 0)
			for i := 0; i <= lastStructIndex; i++ {
				(*structs)[i].Write(encoder, 0)
			}
		}
		WriteDeleteSet(encoder, ds)
	}, nil)

	ApplyUpdate(newDoc, encoder.ToUint8Array(), "snapshot")
	return newDoc, nil
}
