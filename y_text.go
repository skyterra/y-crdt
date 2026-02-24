package y_crdt

import (
	"errors"
	"fmt"

	"github.com/mitchellh/copystructure"
)

type ItemTextListPosition struct {
	Left              *Item
	Right             *Item
	Index             Number
	CurrentAttributes Object
}

// Only call this if you know that this.right is defined
func (it *ItemTextListPosition) Forward() error {
	if it.Right == nil {
		return errors.New("unexpected case")
	}

	switch it.Right.Content.(type) {
	case *ContentEmbed, *ContentString:
		if !it.Right.Deleted() {
			it.Index += it.Right.Length
		}
		break

	case *ContentFormat:
		if !it.Right.Deleted() {
			UpdateCurrentAttributes(it.CurrentAttributes, it.Right.Content.(*ContentFormat))
		}
		break
	}

	it.Left = it.Right
	it.Right = it.Right.Right

	return nil
}

func FindNextPosition(trans *Transaction, pos *ItemTextListPosition, count Number) *ItemTextListPosition {
	for pos.Right != nil && count > 0 {
		switch pos.Right.Content.(type) {
		case *ContentEmbed, *ContentString:
			if !pos.Right.Deleted() {
				if count < pos.Right.Length {
					// split right
					GetItemCleanStart(trans, GenID(pos.Right.ID.Client, pos.Right.ID.Clock+count))
				}
				pos.Index += pos.Right.Length
				count -= pos.Right.Length
			}
			break
		case *ContentFormat:
			if !pos.Right.Deleted() {
				UpdateCurrentAttributes(pos.CurrentAttributes, pos.Right.Content.(*ContentFormat))
			}
		}

		pos.Left = pos.Right
		pos.Right = pos.Right.Right

		// pos.forward() - we don't forward because that would halve the performance because we already do the checks above
	}

	return pos
}

func FindPosition(trans *Transaction, parent IAbstractType, index Number) *ItemTextListPosition {
	currentAttributes := NewObject()
	marker := FindMarker(parent, index)

	if marker != nil {
		pos := NewItemTextListPosition(marker.P.Left, marker.P, marker.Index, currentAttributes)
		return FindNextPosition(trans, pos, index-marker.Index)
	} else {
		pos := NewItemTextListPosition(nil, parent.StartItem(), 0, currentAttributes)
		return FindNextPosition(trans, pos, index)
	}
}

// Negate applied formats
func InsertNegatedAttributes(trans *Transaction, parent IAbstractType, currPos *ItemTextListPosition, negatedAttributes Object) {
	// check if we really need to remove attributes
	for currPos.Right != nil && (currPos.Right.Deleted() ||
		(IsSameType(currPos.Right.Content, &ContentFormat{}) &&
			EqualAttrs(negatedAttributes[currPos.Right.Content.(*ContentFormat).Key], currPos.Right.Content.(*ContentFormat).Value))) {
		if !currPos.Right.Deleted() {
			delete(negatedAttributes, currPos.Right.Content.(*ContentFormat).Key)
		}
		currPos.Forward()
	}

	doc := trans.Doc
	ownClientId := doc.ClientID

	for key, value := range negatedAttributes {
		id := GenID(ownClientId, GetState(doc.Store, ownClientId))
		left := currPos.Left
		right := currPos.Right
		nextFormat := NewItem(id, left, GetItemLastID(left), right, GetItemID(right), parent, "", NewContentFormat(key, value))
		nextFormat.Integrate(trans, 0)
		currPos.Right = nextFormat
		currPos.Forward()
	}
}

func UpdateCurrentAttributes(currentAttributes Object, format *ContentFormat) {
	key, value := format.Key, format.Value
	if value == nil {
		delete(currentAttributes, key)
	} else {
		currentAttributes[key] = value
	}
}

func MinimizeAttributeChanges(currPos *ItemTextListPosition, attributes Object) {
	// go right while attributes[right.key] === right.value (or right is deleted)
	if currPos.Right == nil {
		return
	}

	isEqual := func(content IAbstractContent, attributes Object) bool {
		cf, ok := currPos.Right.Content.(*ContentFormat)
		if !ok {
			return false
		}

		return EqualAttrs(attributes[cf.Key], cf.Value)
	}

	for currPos.Right != nil && (currPos.Right.Deleted() || isEqual(currPos.Right.Content, attributes)) {
		currPos.Forward()
	}
}

func InsertAttributes(trans *Transaction, parent IAbstractType, currPos *ItemTextListPosition, attributes Object) Object {
	doc := trans.Doc
	ownClientId := doc.ClientID
	negatedAttributes := NewObject()
	// insert format-start items
	for key, val := range attributes {
		currentVal := currPos.CurrentAttributes[key]

		if !EqualAttrs(currentVal, val) {
			// save negated attribute (set null if currentVal undefined)
			negatedAttributes[key] = currentVal
			left, right := currPos.Left, currPos.Right

			currPos.Right = NewItem(GenID(ownClientId, GetState(doc.Store, ownClientId)), left, GetItemLastID(left), right, GetItemID(right), parent, "", NewContentFormat(key, val))
			currPos.Right.Integrate(trans, 0)
			currPos.Forward()
		}
	}

	return negatedAttributes
}

func InsertText(trans *Transaction, parent IAbstractType, currPos *ItemTextListPosition, text interface{}, attributes Object) {
	// The following code of yjs-javascript golang no need
	// currPos.currentAttributes.forEach((val, key) => {
	// 	if (attributes[key] === undefined) {
	// 		attributes[key] = null
	// 	}
	// })

	doc := trans.Doc
	ownClientId := doc.ClientID
	MinimizeAttributeChanges(currPos, attributes)
	negatedAttributes := InsertAttributes(trans, parent, currPos, attributes)

	// insert content
	var content IAbstractContent
	_, ok := text.(string)
	if ok {
		content = NewContentString(text.(string))
	} else {
		content = NewContentEmbed(text)
	}

	left, right, index := currPos.Left, currPos.Right, currPos.Index
	if parent.GetSearchMarker() != nil {
		UpdateMarkerChanges(parent.GetSearchMarker(), currPos.Index, content.GetLength())
	}

	right = NewItem(GenID(ownClientId, GetState(doc.Store, ownClientId)), left, GetItemLastID(left), right, GetItemID(right), parent, "", content)
	right.Integrate(trans, 0)
	currPos.Right = right
	currPos.Index = index
	currPos.Forward()
	InsertNegatedAttributes(trans, parent, currPos, negatedAttributes)
}

func FormatText(trans *Transaction, parent IAbstractType, currPos *ItemTextListPosition, length Number, attributes Object) {
	doc := trans.Doc
	ownClientId := doc.ClientID
	MinimizeAttributeChanges(currPos, attributes)
	negatedAttributes := InsertAttributes(trans, parent, currPos, attributes)

	// iterate until first non-format or null is found
	// delete all formats with attributes[format.key] != null
	for length > 0 && currPos.Right != nil {
		if !currPos.Right.Deleted() {
			switch currPos.Right.Content.(type) {
			case *ContentFormat:
				cf := currPos.Right.Content.(*ContentFormat)
				key, value := cf.Key, cf.Value
				attr, exist := attributes[key]
				if exist {
					if EqualAttrs(attr, value) {
						delete(negatedAttributes, key)
					} else {
						negatedAttributes[key] = value
					}

					currPos.Right.Delete(trans)
				}
			case *ContentEmbed, *ContentString:
				if length < currPos.Right.Length {
					GetItemCleanStart(trans, GenID(currPos.Right.ID.Client, currPos.Right.ID.Clock+length))
				}
				length -= currPos.Right.Length
			}
		}

		currPos.Forward()
	}

	// Quill just assumes that the editor starts with a newline and that it always
	// ends with a newline. We only insert that newline when a new newline is
	// inserted - i.e when length is bigger than type.length
	if length > 0 {
		newlines := ""
		for ; length > 0; length-- {
			newlines = fmt.Sprintf("%s\n", newlines)
		}

		currPos.Right = NewItem(GenID(ownClientId, GetState(doc.Store, ownClientId)), currPos.Left, GetItemLastID(currPos.Left), currPos.Right, GetItemID(currPos.Right), parent, "", NewContentString(newlines))
		currPos.Right.Integrate(trans, 0)
		currPos.Forward()
	}

	InsertNegatedAttributes(trans, parent, currPos, negatedAttributes)
}

// Call this function after string content has been deleted in order to
// clean up formatting Items.
func CleanupFormattingGap(trans *Transaction, start *Item, end *Item, startAttributes Object, endAttributes Object) Number {
	for end != nil && !IsSameType(end.Content, &ContentString{}) && !IsSameType(end.Content, &ContentEmbed{}) {
		if !end.Deleted() && IsSameType(end.Content, &ContentFormat{}) {
			UpdateCurrentAttributes(endAttributes, end.Content.(*ContentFormat))
		}
		end = end.Right
	}

	cleanups := 0
	for start != end {
		if !start.Deleted() {
			content := start.Content
			switch content.(type) {
			case *ContentFormat:
				key, value := content.(*ContentFormat).Key, content.(*ContentFormat).Value
				if !EqualAttrs(endAttributes[key], value) || EqualAttrs(startAttributes[key], value) {
					// Either this format is overwritten or it is not necessary because the attribute already existed.
					start.Delete(trans)
					cleanups++
				}
				break
			}
		}

		start = start.Right
	}

	return cleanups
}

func CleanupContextlessFormattingGap(trans *Transaction, item *Item) {
	// iterate until item.right is null or content
	for item != nil && item.Right != nil && (item.Right.Deleted() || (!IsSameType(item.Right.Content, &ContentString{}) && !IsSameType(item.Right.Content, &ContentEmbed{}))) {
		item = item.Right
	}

	attrs := NewSet()

	// iterate back until a content item is found
	for item != nil && (item.Deleted() || (!IsSameType(item.Content, &ContentString{}) && !IsSameType(item.Content, &ContentEmbed{}))) {
		if !item.Deleted() && IsSameType(item.Content, &ContentFormat{}) {
			key := item.Content.(*ContentFormat).Key
			if attrs.Has(key) {
				item.Delete(trans)
			} else {
				attrs.Add(key)
			}
		}
		item = item.Left
	}
}

// This function is experimental and subject to change / be removed.
//
// Ideally, we don't need this function at all. Formatting attributes should be cleaned up
// automatically after each change. This function iterates twice over the complete YText type
// and removes unnecessary formatting attributes. This is also helpful for testing.
//
// This function won't be exported anymore as soon as there is confidence that the YText type works as intended.
func CleanupYTextFormatting(t *YText) Number {
	res := 0
	Transact(t.Doc, func(trans *Transaction) {
		start := t.Start
		end := t.Start
		startAttributes := NewObject()
		currentAttributes := NewObject()
		for end != nil {
			if !end.Deleted() {
				switch end.Content.(type) {
				case *ContentFormat:
					UpdateCurrentAttributes(currentAttributes, end.Content.(*ContentFormat))
					break
				case *ContentEmbed, *ContentString:
					res += CleanupFormattingGap(trans, start, end, startAttributes, currentAttributes)
					cpdata, _ := copystructure.Copy(currentAttributes)
					startAttributes = cpdata.(Object)
					start = end
				}
			}
			end = end.Right
		}
	}, nil, true)
	return res
}

func DeleteText(trans *Transaction, currPos *ItemTextListPosition, length Number) *ItemTextListPosition {
	startLength := length
	cpdata, _ := copystructure.Copy(currPos.CurrentAttributes)
	startAttrs := cpdata.(Object)
	start := currPos.Right

	for length > 0 && currPos.Right != nil {
		if !currPos.Right.Deleted() {
			switch currPos.Right.Content.(type) {
			case *ContentEmbed, *ContentString:
				if length < currPos.Right.Length {
					GetItemCleanStart(trans, GenID(currPos.Right.ID.Client, currPos.Right.ID.Clock+length))
				}
				length -= currPos.Right.Length
				currPos.Right.Delete(trans)
			}
		}
		currPos.Forward()
	}

	if start != nil {
		data, _ := copystructure.Copy(currPos.CurrentAttributes)
		cpAttributes := data.(Object)
		CleanupFormattingGap(trans, start, currPos.Right, startAttrs, cpAttributes)
	}

	var parent IAbstractType
	if currPos.Left != nil {
		parent = currPos.Left.Parent.(IAbstractType)
	} else {
		parent = currPos.Right.Parent.(IAbstractType)
	}

	if parent.GetSearchMarker() != nil {
		UpdateMarkerChanges(parent.GetSearchMarker(), currPos.Index, -startLength+length)
	}

	return currPos
}

/*
 * The Quill Delta format represents changes on a text document with
 * formatting information. For mor information visit {@link https://quilljs.com/docs/delta/|Quill Delta}
 *
 * @example
 *   {
 *     ops: [
 *       { insert: 'Gandalf', attributes: { bold: true } },
 *       { insert: ' the ' },
 *       { insert: 'Grey', attributes: { color: '#cccccc' } }
 *     ]
 *   }
 *
 */

/*
 * Attributes that can be assigned to a selection of text.
 *
 * @example
 *   {
 *     bold: true,
 *     font-size: '40px'
 *   }
 *
 * @typedef {Object} TextAttributes
 */

// Event that describes the changes on a YText type.
type YTextEvent struct {
	YEvent
	ChildListChanged bool // Whether the children changed.
	KeysChanged      Set  // Set of all changed attributes.
}

func (y *YMapEvent) GetChanges() Object {
	if y.Changes == nil || len(y.Changes) == 0 {
		changes := Object{
			"keys":    y.Keys,
			"delta":   y.GetDelta(),
			"added":   NewSet(),
			"deleted": NewSet(),
		}

		y.Changes = changes
	}

	return y.Changes
}

// Compute the changes in the delta format.
// A {@link https://quilljs.com/docs/delta/|Quill delta}) that represents the changes on the document.
func (y *YTextEvent) GetDelta() []EventOperator {
	if y.delta == nil {
		doc := y.Target.GetDoc()
		var delta []EventOperator

		Transact(doc, func(trans *Transaction) {
			currentAttributes := NewObject() // saves all current attributes for insert
			oldAttributes := NewObject()

			item := y.Target.StartItem()
			action := ""
			attributes := Object{} // counts added or removed new attributes for retain
			var insert interface{} = ""
			retain := 0
			deleteLen := 0

			addOp := func() {
				if action != "" {
					var op EventOperator

					switch action {
					case "delete":
						op = EventOperator{
							Delete:          deleteLen,
							IsDeleteDefined: true,
						}
						deleteLen = 0
					case "insert":
						op = EventOperator{
							Insert:          insert,
							IsInsertDefined: true,
						}

						if len(currentAttributes) > 0 {
							attr := Object{}
							for key, value := range currentAttributes {
								if value != nil {
									attr[key] = value
								}
							}

							op.Attributes = attr
							op.IsAttributesDefined = true
						}
						insert = ""
					case "retain":
						op = EventOperator{
							Retain:          retain,
							IsRetainDefined: true,
						}

						if len(attributes) > 0 {
							attr := Object{}
							for key, value := range attributes {
								attr[key] = value
							}
							op.Attributes = attr
							op.IsAttributesDefined = true
						}

						retain = 0
					}
					delta = append(delta, op)
					action = ""
				}
			}

			for item != nil {
				switch item.Content.(type) {
				case *ContentEmbed:
					if y.Adds(item) {
						if !y.Deletes(item) {
							addOp()
							action = "insert"
							insert = item.Content.(*ContentEmbed).Embed
							addOp()
						}
					} else if y.Deletes(item) {
						if action != "delete" {
							addOp()
							action = "delete"
						}
						deleteLen += 1
					} else if !item.Deleted() {
						if action != "retain" {
							addOp()
							action = "retain"
						}
						retain += 1
					}

				case *ContentString:
					if y.Adds(item) {
						if !y.Deletes(item) {
							if action != "insert" {
								addOp()
								action = "insert"
							}
							insert = fmt.Sprintf("%s%s", insert.(string), item.Content.(*ContentString).Str)
						}
					} else if y.Deletes(item) {
						if action != "delete" {
							addOp()
							action = "delete"
						}
						deleteLen += item.Length
					} else if !item.Deleted() {
						if action != "retain" {
							addOp()
							action = "retain"
						}
						retain += item.Length
					}
				case *ContentFormat:
					key, value := item.Content.(*ContentFormat).Key, item.Content.(*ContentFormat).Value
					if y.Adds(item) {
						if !y.Deletes(item) {
							curVal := currentAttributes[key]
							if !EqualAttrs(curVal, value) {
								if action == "retain" {
									addOp()
								}

								if EqualAttrs(value, oldAttributes[key]) {
									delete(attributes, key)
								} else {
									attributes[key] = value
								}
							} else {
								item.Delete(trans)
							}
						}
					} else if y.Deletes(item) {
						oldAttributes[key] = value
						curVal := currentAttributes[key]

						if !EqualAttrs(curVal, value) {
							if action == "retain" {
								addOp()
							}
							attributes[key] = curVal
						}
					} else if !item.Deleted() {
						oldAttributes[key] = value
						attr := attributes[key]
						if attr != nil {
							if !EqualAttrs(attr, value) {
								if action == "retain" {
									addOp()
								}

								if value == nil {
									attributes[key] = value
								} else {
									delete(attributes, key)
								}
							} else {
								item.Delete(trans)
							}
						}
					}

					if !item.Deleted() {
						if action == "insert" {
							addOp()
						}

						UpdateCurrentAttributes(currentAttributes, item.Content.(*ContentFormat))
					}
				}

				item = item.Right
			}

			addOp()
			for len(delta) > 0 {
				lastOp := delta[len(delta)-1]
				if lastOp.IsRetainDefined && !lastOp.IsAttributesDefined {
					// retain delta's if they don't assign attributes
					delta = delta[:len(delta)-1]
				} else {
					break
				}
			}
		}, nil, true)

		y.delta = delta
	}

	return y.delta
}

func NewYTextEvent(ytext *YText, trans *Transaction, subs Set) *YTextEvent {
	yTextEvent := &YTextEvent{
		YEvent:           *NewYEvent(ytext, trans),
		ChildListChanged: false,
		KeysChanged:      NewSet(),
	}

	subs.Range(func(element interface{}) {
		if element == nil {
			yTextEvent.ChildListChanged = true
		} else {
			yTextEvent.KeysChanged.Add(element)
		}
	})

	return yTextEvent
}

// Type that represents text with formatting information.
//
// This type replaces y-richtext as this implementation is able to handle
// block formats (format information on a paragraph), embeds (complex elements
// like pictures and videos), and text formats (**bold**, *italic*).
type YText struct {
	AbstractType
	Pending      []func()
	SearchMarker []ArraySearchMarker
}

func (y *YText) Length() Number {
	return y.AbstractType.Length
}

func (y *YText) Integrate(doc *Doc, item *Item) {
	y.AbstractType.Integrate(doc, item)
	for _, f := range y.Pending {
		f()
	}

	y.Pending = nil
}

func (y *YText) Copy() IAbstractType {
	return NewYText("")
}

func (y *YText) Clone() IAbstractType {
	text := NewYText("")
	text.ApplyDelta(y.ToDelta(nil, nil, nil), true)
	return text
}

// Creates YTextEvent and calls observers.
func (y *YText) CallObserver(trans *Transaction, parentSubs Set) {
	y.AbstractType.CallObserver(trans, parentSubs)
	event := NewYTextEvent(y, trans, parentSubs)
	doc := trans.Doc

	CallTypeObservers(y, trans, event)

	// If a remote change happened, we try to cleanup potential formatting duplicates.
	if !trans.Local {
		// check if another formatting item was inserted
		foundFormattingItem := false
		for client, afterClock := range trans.AfterState {
			clock := trans.BeforeState[client]
			if afterClock == clock {
				continue
			}

			IterateStructs(trans, doc.Store.Clients[client], clock, afterClock, func(s IAbstractStruct) {
				item, ok := s.(*Item)
				if ok && !item.Deleted() && IsSameType(item.Content, &ContentFormat{}) {
					foundFormattingItem = true
				}
			})

			if !foundFormattingItem {
				IterateDeletedStructs(trans, trans.DeleteSet, func(s IAbstractStruct) {
					if IsSameType(s, &GC{}) || foundFormattingItem {
						return
					}

					item, ok := s.(*Item)
					if ok && IsSameType(item.Content, &ContentFormat{}) {
						foundFormattingItem = true
					}
				})
			}

			Transact(doc, func(trans *Transaction) {
				if foundFormattingItem {
					// If a formatting item was inserted, we simply clean the whole type.
					// We need to compute currentAttributes for the current position anyway.
					CleanupYTextFormatting(y)
				} else {
					// If no formatting attribute was inserted, we can make due with contextless
					// formatting cleanups.
					// Contextless: it is not necessary to compute currentAttributes for the affected position.
					IterateDeletedStructs(trans, trans.DeleteSet, func(s IAbstractStruct) {
						if IsSameType(s, &GC{}) {
							return
						}

						item, ok := s.(*Item)
						if ok && item.Parent == y {
							CleanupContextlessFormattingGap(trans, item)
						}
					})
				}
			}, nil, true)
		}
	}
}

// Returns the unformatted string representation of this YText type.
func (y *YText) ToString() string {
	str := ""
	n := y.Start
	for n != nil {
		if !n.Deleted() && n.Countable() && IsSameType(n.Content, &ContentString{}) {
			str = fmt.Sprintf("%s%s", str, n.Content.(*ContentString).Str)
		}
		n = n.Right
	}
	return str
}

// Returns the unformatted string representation of this YText type.
func (y *YText) ToJson() interface{} {
	return y.ToString()
}

// Apply a {@link delta} on this shared YText type.
// sanitize = true
func (y *YText) ApplyDelta(delta []EventOperator, sanitize bool) {
	if y.Doc != nil {
		Transact(y.Doc, func(trans *Transaction) {
			currPos := NewItemTextListPosition(nil, y.Start, 0, NewObject())
			for i := 0; i < len(delta); i++ {
				op := delta[i]
				if op.IsInsertDefined {
					// Quill assumes that the content starts with an empty paragraph.
					// Yjs/Y.Text assumes that it starts empty. We always hide that
					// there is a newline at the end of the content.
					// If we omit this step, clients will see a different number of
					// paragraphs, but nothing bad will happen.
					var ins interface{}

					strInsert, ok := op.Insert.(*YString)
					if !sanitize && ok && (i == len(delta)-1) && currPos == nil && strInsert.Str[len(strInsert.Str)-1] == '\n' {
						ins = strInsert.Str[:len(strInsert.Str)-1]
					} else {
						ins = op.Insert
					}

					strIns, ok := ins.(string)
					if !ok || len(strIns) > 0 {
						InsertText(trans, y, currPos, ins, op.Attributes)
					}
				} else if op.IsRetainDefined {
					FormatText(trans, y, currPos, op.Retain, op.Attributes)
				} else if op.IsDeleteDefined {
					DeleteText(trans, currPos, op.Delete)
				}
			}
		}, nil, true)
	} else {
		y.Pending = append(y.Pending, func() {
			y.ApplyDelta(delta, true)
		})
	}
}

// Returns the delta representation of this YText type.
func (y *YText) ToDelta(snapshot *Snapshot, prevSnapshot *Snapshot, computeYChange func(string, *ID) Object) []EventOperator {
	var ops []EventOperator
	currentAttributes := NewObject()
	doc := y.Doc
	str := ""
	n := y.Start
	packStr := func() {
		if len(str) > 0 {
			// pack str with attributes to ops
			attributes := NewObject()
			addAttributes := false
			for key, value := range currentAttributes {
				addAttributes = true
				attributes[key] = value
			}

			op := EventOperator{}
			op.Insert = str
			op.IsInsertDefined = true

			if addAttributes {
				op.Attributes = attributes
			}

			ops = append(ops, op)
			str = ""
		}
	}

	// snapshots are merged again after the transaction, so we need to keep the
	// transalive until we are done
	Transact(doc, func(trans *Transaction) {
		if snapshot != nil {
			SplitSnapshotAffectedStructs(trans, snapshot)
		}

		if prevSnapshot != nil {
			SplitSnapshotAffectedStructs(trans, prevSnapshot)
		}

		for n != nil {
			if IsVisible(n, snapshot) || (prevSnapshot != nil && IsVisible(n, prevSnapshot)) {
				switch n.Content.(type) {
				case *ContentString:
					cur, _ := currentAttributes["ychange"].(Object)
					if snapshot != nil && !IsVisible(n, snapshot) {
						if cur == nil || cur["user"] != n.ID.Client || cur["state"] != "removed" {
							packStr()
							if computeYChange != nil {
								currentAttributes["ychange"] = computeYChange("removed", &n.ID)
							} else {
								currentAttributes["ychange"] = Object{
									"type": "removed",
								}
							}
						}
					} else if prevSnapshot != nil && !IsVisible(n, prevSnapshot) {
						if cur == nil || cur["user"] != n.ID.Client || cur["state"] != "added" {
							packStr()
							if computeYChange != nil {
								currentAttributes["ychange"] = computeYChange("added", &n.ID)
							} else {
								currentAttributes["ychange"] = Object{
									"type": "added",
								}
							}
						}
					} else if cur != nil {
						packStr()
						delete(currentAttributes, "ychange")
					}

					str += n.Content.(*ContentString).Str
				case *ContentEmbed:
					packStr()
					op := EventOperator{}
					op.Insert = n.Content.(*ContentEmbed).Embed
					op.IsInsertDefined = true

					if len(currentAttributes) > 0 {
						attrs := Object{}
						op.Attributes = attrs
						for key, value := range currentAttributes {
							attrs[key] = value
						}
					}
					ops = append(ops, op)
				case *ContentFormat:
					if IsVisible(n, snapshot) {
						packStr()
						UpdateCurrentAttributes(currentAttributes, n.Content.(*ContentFormat))
					}
				}

			}
			n = n.Right
		}
		packStr()
	}, SplitSnapshotAffectedStructs, true)

	return ops
}

// Insert text at a given index.
func (y *YText) Insert(index Number, text string, attributes Object) {
	if len(text) <= 0 {
		return
	}

	doc := y.Doc
	if doc != nil {
		Transact(doc, func(trans *Transaction) {
			pos := FindPosition(trans, y, index)
			if attributes == nil {
				attributes = NewObject()
				for key, value := range pos.CurrentAttributes {
					attributes[key] = value
				}
			}
			InsertText(trans, y, pos, text, attributes)
		}, nil, true)
	} else {
		y.Pending = append(y.Pending, func() {
			y.Insert(index, text, attributes)
		})
	}
}

// Inserts an embed at a index.
func (y *YText) InsertEmbed(index Number, embed Object, attributes Object) {
	doc := y.Doc
	if y != nil {
		Transact(doc, func(trans *Transaction) {
			pos := FindPosition(trans, y, index)
			InsertText(trans, y, pos, embed, attributes)
		}, nil, true)
	} else {
		y.Pending = append(y.Pending, func() {
			y.InsertEmbed(index, embed, attributes)
		})
	}
}

// Deletes text starting from an index.
func (y *YText) Delete(index Number, length Number) {
	if length == 0 {
		return
	}

	doc := y.Doc
	if y != nil {
		Transact(doc, func(trans *Transaction) {
			DeleteText(trans, FindPosition(trans, y, index), length)
		}, nil, true)
	} else {
		y.Pending = append(y.Pending, func() {
			y.Delete(index, length)
		})
	}
}

// Assigns properties to a range of text.
func (y *YText) Format(index Number, length Number, attributes Object) {
	if length == 0 {
		return
	}

	doc := y.Doc
	if y != nil {
		Transact(doc, func(trans *Transaction) {
			pos := FindPosition(trans, y, index)
			if pos.Right == nil {
				return
			}
			FormatText(trans, y, pos, length, attributes)

		}, nil, true)
	} else {
		y.Pending = append(y.Pending, func() {
			y.Format(index, length, attributes)
		})
	}
}

// Removes an attribute.
func (y *YText) RemoveAttribute(attributeName string) {
	if y.Doc != nil {
		Transact(y.Doc, func(trans *Transaction) {
			TypeMapDelete(trans, y, attributeName)
		}, nil, true)
	} else {
		y.Pending = append(y.Pending, func() {
			y.RemoveAttribute(attributeName)
		})
	}
}

// Sets or updates an attribute.
func (y *YText) SetAttribute(attributeName string, attributeValue interface{}) {
	if y.Doc != nil {
		Transact(y.Doc, func(trans *Transaction) {
			TypeMapSet(trans, y, attributeName, attributeValue)
		}, nil, true)
	} else {
		y.Pending = append(y.Pending, func() {
			y.SetAttribute(attributeName, attributeValue)
		})
	}
}

// Returns an attribute value that belongs to the attribute name.
func (y *YText) GetAttribute(attributeName string) interface{} {
	return TypeMapGet(y, attributeName)
}

// Returns all attribute name/value pairs in a JSON Object.
func (y *YText) GetAttributes(snapshot *Snapshot) Object {
	return TypeMapGetAll(y)
}

func (y *YText) Write(encoder *UpdateEncoderV1) {
	encoder.WriteTypeRef(YTextRefID)
}

func NewYText(text string) *YText {
	yText := &YText{}
	yText.EH = NewEventHandler()
	yText.DEH = NewEventHandler()

	if text != "" {
		yText.Pending = append(yText.Pending, func() {
			yText.Insert(0, text, nil)
		})
	}

	return yText
}

func NewDefaultYText() *YText {
	return &YText{}
}

func NewYTextType() IAbstractType {
	return NewYText("")
}

func NewItemTextListPosition(left, right *Item, index Number, currentAttributes Object) *ItemTextListPosition {
	return &ItemTextListPosition{
		Left:              left,
		Right:             right,
		Index:             index,
		CurrentAttributes: currentAttributes,
	}
}
