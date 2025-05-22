package y_crdt

import (
	"errors"
)

// A relative position is based on the Yjs model and is not affected by document changes.
// E.g. If you place a relative position before a certain character, it will always point to this character.
// If you place a relative position at the end of a type, it will always point to the end of the type.
//
// A numeric position is often unsuited for user selections, because it does not change when content is inserted
// before or after.
//
// ```Insert(0, 'x')('a|bc') = 'xa|bc'``` Where | is the relative position.
//
// One of the properties must be defined.
//
// @example
//   // Current cursor position is at position 10
//   const relativePosition = createRelativePositionFromIndex(yText, 10)
//   // modify yText
//   yText.insert(0, 'abc')
//   yText.delete(3, 10)
//   // Compute the cursor position
//   const absolutePosition = createAbsolutePositionFromRelativePosition(y, relativePosition)
//   absolutePosition.type === yText // => true
//   console.log('cursor location is ' + absolutePosition.index) // => cursor location is 3

type RelativePosition struct {
	Type  *ID
	Tname string
	Item  *ID

	// A relative position is associated to a specific character. By default
	// assoc >= 0, the relative position is associated to the character
	// after the meant position.
	// I.e. position 1 in 'ab' is associated to character 'b'.
	//
	// If assoc < 0, then the relative position is associated to the caharacter
	// before the meant position.
	Assoc Number
}

func RelativePositionToJSON(rpos *RelativePosition) Object {
	json := NewObject()
	if rpos.Type != nil {
		json["type"] = rpos.Type
	}

	if rpos.Tname != "" {
		json["tname"] = rpos.Tname
	}

	if rpos.Item != nil {
		json["item"] = rpos.Item
	}

	json["assoc"] = rpos.Assoc
	return json
}

func CreateRelativePositionFromJSON(json Object) *RelativePosition {
	r := &RelativePosition{}
	if v, exist := json["type"]; exist {
		id := v.(ID)
		// r.Type = GenID(id.Client, id.Clock)
		r.Type = &id
	}

	if v, exist := json["tname"]; exist {
		r.Tname = v.(string)
	}

	if v, exist := json["item"]; exist {
		id := v.(ID)
		// r.Item = GenID(id.Client, id.Clock)
		r.Item = &id
	}

	if v, exist := json["assoc"]; exist {
		assoc := v.(Number)
		r.Assoc = assoc
	}
	return r
}

type AbsolutePosition struct {
	Type  IAbstractType
	Index Number
	Assoc Number
}

func NewAbsolutePosition(t IAbstractType, index, assoc Number) *AbsolutePosition {
	return &AbsolutePosition{
		Type:  t,
		Index: index,
		Assoc: assoc,
	}
}

func NewRelativePosition(t IAbstractType, item *ID, assoc Number) *RelativePosition {
	var typeid ID
	var tname string

	if t.GetItem() == nil {
		tname = FindRootTypeKey(t)
	} else {
		typeid = GenID(t.GetItem().ID.Client, t.GetItem().ID.Clock)
	}

	return &RelativePosition{
		Type:  &typeid,
		Tname: tname,
		Item:  item,
		Assoc: assoc,
	}
}

// Create a relativePosition based on a absolute position.
func NewRelativePositionFromTypeIndex(tp IAbstractType, index, assoc Number) *RelativePosition {
	t := tp.StartItem()
	if assoc < 0 {
		// associated to the left character or the beginning of a type, increment index if possible.
		if index == 0 {
			return NewRelativePosition(tp, nil, assoc)
		}

		index--
	}

	for t != nil {
		if !t.Deleted() && t.Countable() {
			if t.Length > index {
				// case 1: found position somewhere in the linked list
				item := GenID(t.ID.Client, t.ID.Clock+index)
				return NewRelativePosition(tp, &item, assoc)
			}
			index -= t.Length
		}

		if t.Right == nil && assoc < 0 {
			// left-associated position, return last available id
			return NewRelativePosition(tp, t.LastID(), assoc)
		}

		t = t.Right
	}

	return NewRelativePosition(tp, nil, assoc)
}

func WriteRelativePosition(encoder *UpdateEncoderV1, rpos *RelativePosition) error {
	t, tname, item, assoc := rpos.Type, rpos.Tname, rpos.Item, rpos.Assoc
	if item != nil {
		WriteVarUint(encoder.RestEncoder, 0)
		encoder.WriteID(item)
	} else if tname != "" {
		// case 2: found position at the end of the list and type is stored in y.share
		WriteByte(encoder.RestEncoder, 1)
		encoder.WriteString(tname)
	} else if t != nil {
		// case 3: found position at the end of the list and type is attached to an item
		WriteByte(encoder.RestEncoder, 2)
		encoder.WriteID(t)
	} else {
		return errors.New("unexpected case")
	}

	WriteVarInt(encoder.RestEncoder, assoc)
	return nil
}

func EncodeRelativePosition(rpos *RelativePosition) []uint8 {
	encoder := NewUpdateEncoderV1()
	WriteRelativePosition(encoder, rpos)
	return encoder.ToUint8Array()
}

func ReadRelativePosition(decoder *UpdateDecoderV1) *RelativePosition {
	var t *ID
	var tname string
	var itemID *ID
	var assoc Number

	n, _ := readVarUint(decoder.RestDecoder)
	switch n {
	case 0:
		// case 1: found position somewhere in the linked list
		itemID, _ = decoder.ReadID()

	case 1:
		// case 2: found position at the end of the list and type is stored in y.share
		tname, _ = decoder.ReadString()

	case 2:
		// case 3: found position at the end of the list and type is attached to an item
		t, _ = decoder.ReadID()
	}

	if hasContent(decoder.RestDecoder) {
		v, _ := ReadVarInt(decoder.RestDecoder)
		assoc = v.(Number)
	}

	return &RelativePosition{
		Type:  t,
		Tname: tname,
		Item:  itemID,
		Assoc: assoc,
	}
}

func DecodeRelativePosition(uint8Array []uint8) *RelativePosition {
	return ReadRelativePosition(NewUpdateDecoderV1(uint8Array))
}

func CreateAbsolutePositionFromRelativePosition(rpos *RelativePosition, doc *Doc) *AbsolutePosition {
	store := doc.Store
	rightID := rpos.Item
	typeID := rpos.Type
	tname := rpos.Tname
	assoc := rpos.Assoc

	var t IAbstractType
	var index Number

	if rightID != nil {
		if GetState(store, rightID.Client) <= rightID.Clock {
			return nil
		}

		item, diff := FollowRedone(store, *rightID)
		right := item
		if right == nil {
			return nil
		}

		t = right.Parent.(IAbstractType)
		if t.GetItem() == nil || !t.GetItem().Deleted() {
			// adjust position based on left association if necessary
			if right.Deleted() || !right.Countable() {
				index = 0
			} else {
				if assoc >= 0 {
					index = diff
				} else {
					index = diff + 1
				}
			}

			n := right.Left
			for n != nil {
				if !n.Deleted() && n.Countable() {
					index += n.Length
				}
				n = n.Left
			}
		}
	} else {
		if tname != "" {
			t = doc.GetMap(tname)
		} else if typeID != nil {
			if GetState(store, typeID.Client) <= typeID.Clock {
				// type does not exist yet
				return nil
			}

			item, _ := FollowRedone(store, *typeID)
			if item != nil && IsSameType(item.Content, &ContentType{}) {
				t = item.Content.(*ContentType).Type
			} else {
				// struct is garbage collected
				return nil
			}
		} else {
			Logf("[crdt] unexpected case.")
			return nil
		}

		if assoc >= 0 {
			index = t.GetLength()
		} else {
			index = 0
		}
	}

	return NewAbsolutePosition(t, index, rpos.Assoc)
}

func CompareRelativePositions(a, b *RelativePosition) bool {
	return a == b || a != nil && b != nil && a.Tname == b.Tname && CompareIDs(a.Item, b.Item) && CompareIDs(a.Type, b.Type) && a.Assoc == b.Assoc
}
