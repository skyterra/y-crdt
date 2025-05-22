package y_crdt

import (
	"errors"
	"fmt"
	"sort"
)

/*
 * We use the first five bits in the info flag for determining the type of the struct.
 *
 * 0: GC
 * 1: Item with Deleted content
 * 2: Item with JSON content
 * 3: Item with Binary content
 * 4: Item with String content
 * 5: Item with Embed content (for richtext content)
 * 6: Item with Format content (a formatting marker for richtext content)
 * 7: Item with Type
 */

type ClientStructRef struct {
	I    Number
	Refs []IAbstractStruct
}

type RestStructs struct {
	Missing map[Number]Number
	Update  []uint8
}

func NewRestStructs() *RestStructs {
	return &RestStructs{
		Missing: make(map[Number]Number),
	}
}

func WriteStructs(encoder *UpdateEncoderV1, structs *[]IAbstractStruct, client, clock Number) {
	// write first id
	clock = Max(clock, (*structs)[0].GetID().Clock)
	startNewStructs, _ := FindIndexSS(*structs, clock) // make sure the first id exists

	// write # encoded structs
	WriteVarUint(encoder.RestEncoder, uint64(len(*structs)-startNewStructs))
	encoder.WriteClient(client)
	WriteVarUint(encoder.RestEncoder, uint64(clock))

	firstStruct := (*structs)[startNewStructs]

	// write first struct with an offset
	firstStruct.Write(encoder, clock-firstStruct.GetID().Clock)
	for i := startNewStructs + 1; i < len(*structs); i++ {
		(*structs)[i].Write(encoder, 0)
	}
}

func WriteClientsStructs(encoder *UpdateEncoderV1, store *StructStore, _sm map[Number]Number) {
	// we filter all valid _sm entries into sm
	sm := make(map[Number]Number)

	for client, clock := range _sm {
		if GetState(store, client) > clock {
			sm[client] = clock // only write if new structs are available
		}
	}

	sv := GetStateVector(store)
	for client := range sv {
		if _, exist := _sm[client]; !exist {
			sm[client] = 0
		}
	}

	// write # states that were updated
	WriteVarUint(encoder.RestEncoder, uint64(len(sm)))

	// Write items with higher client ids first
	// This heavily improves the conflict algorithm.
	MapSortedRange(sm, false, func(client, clock Number) {
		WriteStructs(encoder, store.Clients[client], client, clock)
	})
}

func ReadClientsStructRefs(decoder *UpdateDecoderV1, doc *Doc) (map[Number]*ClientStructRef, error) {
	clientRefs := make(map[Number]*ClientStructRef)
	numOfStateUpdates := ReadVarUint(decoder.RestDecoder)
	gcCnt, skipCnt, itemCnt := 0, 0, 0
	for i := uint64(0); i < numOfStateUpdates; i++ {
		numberOfStructs := ReadVarUint(decoder.RestDecoder)
		client, _ := decoder.ReadClient()

		// 防止编解码不对齐导致内存爆
		if numberOfStructs > uint64(decoder.RestDecoder.Len()) {
			errString := fmt.Sprintf("buf is not enough, numberOfStructs:%d buf left:%d", numberOfStructs, decoder.RestDecoder.Len())
			return clientRefs, errors.New(errString)
		}

		// clientStructRef := &ClientStructRef{I: 0, Refs: make([]IAbstractStruct, numberOfStructs)}
		clientStructRef := &ClientStructRef{I: 0, Refs: make([]IAbstractStruct, 0, numberOfStructs)}
		clientRefs[client] = clientStructRef

		clock := Number(ReadVarUint(decoder.RestDecoder))
		// logger.Debugf("ReadClientsStructRefs->UpdateCnt:%d UpdateIndex:%d Client:%d Clock:%d StructCnt:%d\n", numOfStateUpdates, i , client, clock, numberOfStructs)

		for i := uint64(0); i < numberOfStructs; i++ {
			info, _ := decoder.ReadInfo()
			switch BITS5 & info {
			case 0: // GC
				gcCnt++
				length, _ := decoder.ReadLen()
				if length > 0 {
					// clientStructRef.Refs[i] = NewGC(GenID(client, clock), length)
					clientStructRef.Refs = append(clientStructRef.Refs, NewGC(GenID(client, clock), length))
					clock += length
				}
				break

			case 10: // Skip Struct (nothing to apply)
				// @todo we could reduce the amount of checks by adding Skip struct to clientRefs so we know that something is missing.
				length := Number(ReadVarUint(decoder.RestDecoder))
				// clientStructRef.Refs[i] = NewSkip(GenID(client, clock), length)
				clientStructRef.Refs = append(clientStructRef.Refs, NewSkip(GenID(client, clock), length))
				clock += length
				skipCnt++
				break

			default: // Item with content
				// The optimized implementation doesn't use any variables because inlining variables is faster.
				// Below a non-optimized version is shown that implements the basic algorithm with
				// a few comments
				itemCnt++
				cantCopyParentInfo := (info & (BIT7 | BIT8)) == 0

				// If parent = null and neither left nor right are defined, then we know that `parent` is child of `y`
				// and we read the next string as parentYKey.
				// It indicates how we store/retrieve parent from `y.share`
				// @type {string|null}
				var origin *ID
				if info&BIT8 == BIT8 {
					origin, _ = decoder.ReadLeftID()
				}

				var rightOrigin *ID
				if info&BIT7 == BIT7 {
					rightOrigin, _ = decoder.ReadRightID()
				}

				var parent IAbstractType
				if cantCopyParentInfo {
					ok, _ := decoder.ReadParentInfo()
					if ok {
						name, _ := decoder.ReadString()
						parent, _ = doc.Get(name, NewAbstractType)
					} else {
						parent, _ = decoder.ReadLeftID()
					}
				}

				var parentSub string
				if cantCopyParentInfo && ((info & BIT6) == BIT6) {
					parentSub, _ = decoder.ReadString()
				}

				s := NewItem(GenID(client, clock), nil, origin, nil, rightOrigin, parent, parentSub, ReadItemContent(decoder, info))
				if s == nil {
					return clientRefs, errors.New("new item failed")
				}

				// A non-optimized implementation of the above algorithm:
				//
				//   // The item that was originally to the left of this item.
				//   const origin = (info & binary.BIT8) === binary.BIT8 ? decoder.readLeftID() : null
				//   // The item that was originally to the right of this item.
				//   const rightOrigin = (info & binary.BIT7) === binary.BIT7 ? decoder.readRightID() : null
				//   const cantCopyParentInfo = (info & (binary.BIT7 | binary.BIT8)) === 0
				//   const hasParentYKey = cantCopyParentInfo ? decoder.readParentInfo() : false
				//   // If parent = null and neither left nor right are defined, then we know that `parent` is child of `y`
				//   // and we read the next string as parentYKey.
				//   // It indicates how we store/retrieve parent from `y.share`
				//   // @type {string|null}
				//   const parentYKey = cantCopyParentInfo && hasParentYKey ? decoder.readString() : null
				//
				//   const struct = new Item(
				//     createID(client, clock),
				//     null, // leftd
				//     origin, // origin
				//     null, // right
				//     rightOrigin, // right origin
				//     cantCopyParentInfo && !hasParentYKey ? decoder.readLeftID() : (parentYKey !== null ? doc.get(parentYKey) : null), // parent
				//     cantCopyParentInfo && (info & binary.BIT6) === binary.BIT6 ? decoder.readString() : null, // parentSub
				//     readItemContent(decoder, info) // item content
				//   )

				// clientStructRef.Refs[i] = s
				clientStructRef.Refs = append(clientStructRef.Refs, s)
				clock += s.GetLength()
			}
		}
		// logger.Debugf("GC:%d Skip:%d Item:%d\n", gcCnt, skipCnt, itemCnt)
	}

	totalCnt := gcCnt + skipCnt + itemCnt
	if totalCnt > 1000000 { // 数量大于100w
		Logf("too many struct, docID:%s updateCnt:%d totalStructCnt:%d GC:%d Skip:%d Item:%d", doc.Guid, numOfStateUpdates, totalCnt, gcCnt, skipCnt, itemCnt)
	}

	return clientRefs, nil
}

// Resume computing structs generated by struct readers.
//
// While there is something to do, we integrate structs in this order
// 1. top element on stack, if stack is not empty
// 2. next element from current struct reader (if empty, use next struct reader)
//
// If struct causally depends on another struct (ref.missing), we put next reader of
// `ref.id.client` on top of stack.
//
// At some point we find a struct that has no causal dependencies,
// then we start emptying the stack.
//
// It is not possible to have circles: i.e. struct1 (from client1) depends on struct2 (from client2)
// depends on struct3 (from client1). Therefore the max stack size is eqaul to `structReaders.length`.
//
// This method is implemented in a way so that we can resume computation if this update
// causally depends on another update.
func IntegrateStructs(trans *Transaction, store *StructStore, clientsStructRefs map[Number]*ClientStructRef) *RestStructs {
	if len(clientsStructRefs) == 0 {
		return nil
	}

	var stack []IAbstractStruct

	// sort them so that we take the higher id first, in case of conflicts the lower id will probably not conflict with the id from the higher user.
	clientsStructRefsIds := make(NumberSlice, 0, len(clientsStructRefs))
	for k := range clientsStructRefs {
		clientsStructRefsIds = append(clientsStructRefsIds, k)
	}
	sort.Sort(clientsStructRefsIds)

	getNextStructTarget := func() *ClientStructRef {
		if len(clientsStructRefsIds) == 0 {
			return nil
		}

		nextStructsTarget := clientsStructRefs[clientsStructRefsIds[len(clientsStructRefsIds)-1]]
		for len(nextStructsTarget.Refs) == nextStructsTarget.I {
			clientsStructRefsIds = clientsStructRefsIds[:len(clientsStructRefsIds)-1]
			if len(clientsStructRefsIds) > 0 {
				nextStructsTarget = clientsStructRefs[clientsStructRefsIds[len(clientsStructRefsIds)-1]]
			} else {
				return nil
			}
		}

		return nextStructsTarget
	}

	curStructsTarget := getNextStructTarget()
	if curStructsTarget == nil && len(stack) == 0 {
		return nil
	}

	restStructs := NewStructStore()
	missingSV := make(map[Number]Number)

	updateMissingSv := func(client Number, clock Number) {
		mclock, exist := missingSV[client]
		if !exist || mclock > clock {
			missingSV[client] = clock
		}
	}

	stackHead := curStructsTarget.Refs[curStructsTarget.I]
	curStructsTarget.I++

	// caching the state because it is used very often
	state := make(map[Number]interface{})

	addStackToRestSS := func() {
		for _, item := range stack {
			client := item.GetID().Client
			unapplicableItems := clientsStructRefs[client]

			if unapplicableItems != nil {
				// decrement because we weren't able to apply previous operation
				unapplicableItems.I--

				var cpRefs []IAbstractStruct
				for i := unapplicableItems.I; i < len(unapplicableItems.Refs); i++ {
					cpRefs = append(cpRefs, unapplicableItems.Refs[i])
				}

				restStructs.Clients[client] = &cpRefs
				delete(clientsStructRefs, client)

				unapplicableItems.I = 0
				unapplicableItems.Refs = nil
			} else {
				// item was the last item on clientsStructRefs and the field was already cleared. Add item to restStructs and continue
				restStructs.Clients[client] = &[]IAbstractStruct{item}
			}

			// remove client from clientsStructRefsIds to prevent users from applying the same update again
			clientsStructRefsIds = clientsStructRefsIds.Filter(func(number Number) bool {
				return number != client
			})
		}
		stack = nil
	}

	// iterate over all struct readers until we are done
	for {
		if !IsSameType(stackHead, &Skip{}) {
			state[stackHead.GetID().Client] = GetState(store, stackHead.GetID().Client)
			localClock := state[stackHead.GetID().Client].(Number)
			offset := localClock - stackHead.GetID().Clock

			if offset < 0 {
				// update from the same client is missing
				stack = append(stack, stackHead)
				updateMissingSv(stackHead.GetID().Client, stackHead.GetID().Clock-1)

				// hid a dead wall, add all items from stack to restSS
				addStackToRestSS()
			} else {
				missing, err := stackHead.GetMissing(trans, store)
				if err == nil {
					stack = append(stack, stackHead)

					// get the struct reader that has the missing struct
					structRefs := clientsStructRefs[missing]
					if structRefs == nil {
						structRefs = &ClientStructRef{}
					}

					if len(structRefs.Refs) == structRefs.I {
						// This update message causally depends on another update message that doesn't exist yet
						updateMissingSv(missing, GetState(store, missing))
						addStackToRestSS()
					} else {
						stackHead = structRefs.Refs[structRefs.I]
						structRefs.I++
						continue
					}
				} else if offset == 0 || offset < stackHead.GetLength() {
					// all fine, apply the stackhead
					stackHead.Integrate(trans, offset)
					state[stackHead.GetID().Client] = stackHead.GetID().Clock + stackHead.GetLength()
				}
			}
		}

		// iterate to next stackHead
		if len(stack) > 0 {
			stackHead = stack[len(stack)-1]
			stack = stack[:len(stack)-1]
		} else if curStructsTarget != nil && curStructsTarget.I < len(curStructsTarget.Refs) {
			stackHead = curStructsTarget.Refs[curStructsTarget.I]
			curStructsTarget.I++
		} else {
			curStructsTarget = getNextStructTarget()
			if curStructsTarget == nil {
				// we are done
				break
			} else {
				stackHead = curStructsTarget.Refs[curStructsTarget.I]
				curStructsTarget.I++
			}
		}
	}

	if len(restStructs.Clients) > 0 {
		encoder := NewUpdateEncoderV1()
		WriteClientsStructs(encoder, restStructs, make(map[Number]Number))
		// write empty deleteset
		// writeDeleteSet(encoder, new DeleteSet())
		WriteVarUint(encoder.RestEncoder, 0) // => no need for an extra function call, just write 0 deletes
		return &RestStructs{
			Missing: missingSV,
			Update:  encoder.ToUint8Array(),
		}
	}

	return nil
}

func WriteStructsFromTransaction(encoder *UpdateEncoderV1, trans *Transaction) {
	WriteClientsStructs(encoder, trans.Doc.Store, trans.BeforeState)
}

// Read and apply a document update.
// This function has the same effect as `applyUpdate` but accepts an decoder.
func ReadUpdateV2(decoder *UpdateDecoderV1, ydoc *Doc, transactionOrigin interface{}, structDecoder *UpdateDecoderV1) {
	Transact(ydoc, func(trans *Transaction) {
		// force that transaction.local is set to non-local
		trans.Local = false
		retry := false
		doc := trans.Doc
		store := doc.Store
		ss, err := ReadClientsStructRefs(structDecoder, doc)
		if err != nil {
			Logf("ReadClientsStructRefs failed. docID:%s err:%s", ydoc.Guid, err.Error())
			return
		}

		/*
			totalCnt := 0
			totalLength := 0
			totalConCnt := 0
			for k,v := range ss{
				length := 0
				clock := v.Refs[0].GetID().Clock
				gcCnt := 0
				skipCnt := 0
				itemCnt := 0
				continueCnt := 0
				for _, v1 := range v.Refs{
					length += v1.GetLength()
					continueCnt++
					if IsSameType(v1, &GC{}) {
						gcCnt++
					}else if IsSameType(v1, &Skip{}) {
						skipCnt++
						break
					}else{
						itemCnt++
					}
				}
				totalLength += length
				totalConCnt += continueCnt
				fmt.Printf("k:%d, len:%d itemLength:%d continueCnt:%d clock:%d gcCnt:%d skipCnt:%d itemCnt:%d\n", k, len(v.Refs), length, continueCnt,  clock, gcCnt, skipCnt, itemCnt)
				totalCnt += len(v.Refs)
			}
			fmt.Printf("clientCnt:%d itemCnt:%d totalConCnt:%d totalLength:%d\n", len(ss), totalCnt, totalConCnt, totalLength)
		*/

		restStructs := IntegrateStructs(trans, store, ss)
		pending := store.PendingStructs

		if pending != nil {
			// check if we can apply something
			for client, clock := range pending.Missing {
				if clock < GetState(store, client) {
					retry = true
					break
				}
			}

			if restStructs != nil {
				// merge restStructs into store.pending
				for client, clock := range restStructs.Missing {
					mclock, exist := pending.Missing[client]
					if !exist || mclock > clock {
						pending.Missing[client] = clock
					}
				}
				pending.Update = MergeUpdatesV2([][]uint8{pending.Update, restStructs.Update}, NewUpdateDecoderV1, NewUpdateEncoderV1, false)
			}
		} else {
			store.PendingStructs = restStructs
		}

		dsRest := ReadAndApplyDeleteSet(structDecoder, trans, store)
		if store.PendingDs != nil {
			// todo we could make a lower-bound state-vector check as we do above
			pendingDSUpdate := NewUpdateDecoderV1(store.PendingDs)
			readVarUint(pendingDSUpdate.RestDecoder) // read 0 structs, because we only encode deletes in pendingdsupdate
			dsRest2 := ReadAndApplyDeleteSet(pendingDSUpdate, trans, store)

			if dsRest != nil && dsRest2 != nil {
				// case 1: ds1 != null && ds2 != null
				store.PendingDs = MergeUpdatesV2([][]uint8{dsRest, dsRest2}, NewUpdateDecoderV1, NewUpdateEncoderV1, false)
			} else {
				// case 2: ds1 != null
				// case 3: ds2 != null
				// case 4: ds1 == null && ds2 == null
				if dsRest != nil {
					store.PendingDs = dsRest
				} else {
					store.PendingDs = dsRest2
				}
			}
		} else {
			// Either dsRest == null && pendingDs == null OR dsRest != null
			store.PendingDs = dsRest
		}

		if retry {
			update := store.PendingStructs.Update
			store.PendingStructs = nil
			ApplyUpdateV2(trans.Doc, update, nil, NewUpdateDecoderV1(update))
		}
	}, transactionOrigin, false)
}

// Read and apply a document update.
// This function has the same effect as `applyUpdate` but accepts an decoder.
func ReadUpdate(decoder *UpdateDecoderV1, ydoc *Doc, transactionOrigin interface{}) {
	ReadUpdateV2(decoder, ydoc, transactionOrigin, NewUpdateDecoderV1(decoder.RestDecoder.Bytes()))
}

// Apply a document update created by, for example, `y.on('update', update => ..)` or `update = encodeStateAsUpdate()`.
//
// This function has the same effect as `readUpdate` but accepts an Uint8Array instead of a Decoder.
func ApplyUpdateV2(ydoc *Doc, update []uint8, transactionOrigin interface{}, YDecoder *UpdateDecoderV1) {
	decoder := NewUpdateDecoderV1(update)
	ReadUpdateV2(decoder, ydoc, transactionOrigin, YDecoder)
}

// Apply a document update created by, for example, `y.on('update', update => ..)` or `update = encodeStateAsUpdate()`.
//
// This function has the same effect as `readUpdate` but accepts an Uint8Array instead of a Decoder.
func ApplyUpdate(ydoc *Doc, update []uint8, transactionOrigin interface{}) {
	ApplyUpdateV2(ydoc, update, transactionOrigin, NewUpdateDecoderV1(update))
}

// Write all the document as a single update message. If you specify the state of the remote client (`targetStateVector`) it will
// only write the operations that are missing.
func WriteStateAsUpdate(encoder *UpdateEncoderV1, doc *Doc, targetStateVector map[Number]Number) {
	WriteClientsStructs(encoder, doc.Store, targetStateVector)
	WriteDeleteSet(encoder, NewDeleteSetFromStructStore(doc.Store))
}

// Write all the document as a single update message that can be applied on the remote document. If you specify the state of the remote client (`targetState`) it will
// only write the operations that are missing.
// Use `writeStateAsUpdate` instead if you are working with lib0/encoding.js#Encoder
func EncodeStateAsUpdateV2(doc *Doc, encodedTargetStateVector []uint8, encoder *UpdateEncoderV1) []uint8 {
	if len(encodedTargetStateVector) == 0 {
		encodedTargetStateVector = []byte{0}
	}

	targetStateVector := DecodeStateVector(encodedTargetStateVector)
	WriteStateAsUpdate(encoder, doc, targetStateVector)

	// also add the pending updates (if there are any)
	updates := [][]byte{encoder.ToUint8Array()}
	if len(doc.Store.PendingDs) > 0 {
		updates = append(updates, doc.Store.PendingDs)
	}

	if doc.Store.PendingStructs != nil {
		updates = append(updates, DiffUpdate(doc.Store.PendingStructs.Update, encodedTargetStateVector))
	}

	if len(updates) > 1 {
		// if IsSameType(encoder, UpdateEncoderV1{}) {
		return MergeUpdates(updates, NewUpdateDecoderV1, NewUpdateEncoderV1, false)
		// } else if IsSameType(encoder, UpdateEncoderV2{}) {
		// 	return MergeUpdatesV2(updates, NewUpdateDecoderV1, NewUpdateEncoderV1)
		// }
	}

	return updates[0]
}

func EncodeStateAsUpdate(doc *Doc, encodedTargetStateVector []uint8) []uint8 {
	return EncodeStateAsUpdateV2(doc, encodedTargetStateVector, NewUpdateEncoderV1())
}

// Read state vector from Decoder and return as Map
func ReadStateVector(decoder *UpdateDecoderV1) map[Number]Number {
	ss := make(map[Number]Number)
	v, _ := readVarUint(decoder.RestDecoder)
	ssLength := Number(v.(uint64))

	for i := 0; i < ssLength; i++ {
		v, _ = readVarUint(decoder.RestDecoder)
		client := Number(v.(uint64))

		v, _ = readVarUint(decoder.RestDecoder)
		clock := Number(v.(uint64))

		ss[client] = clock
	}

	return ss
}

// Read decodedState and return State as Map.
func DecodeStateVector(decodedState []uint8) map[Number]Number {
	return ReadStateVector(NewUpdateDecoderV1(decodedState))
}

func WriteStateVector(encoder *UpdateEncoderV1, sv map[Number]Number) *UpdateEncoderV1 {
	WriteVarUint(encoder.RestEncoder, uint64(len(sv)))
	for client, clock := range sv {
		WriteVarUint(encoder.RestEncoder, uint64(client)) // @todo use a special client decoder that is based on mapping
		WriteVarUint(encoder.RestEncoder, uint64(clock))
	}
	return encoder
}

func WriteDocumentStateVector(encoder *UpdateEncoderV1, doc *Doc) {
	WriteStateVector(encoder, GetStateVector(doc.Store))
}

func EncodeStateVectorV2(doc *Doc, m map[Number]Number, encoder *UpdateEncoderV1) []uint8 {
	if m != nil {
		WriteStateVector(encoder, m)
	} else {
		WriteDocumentStateVector(encoder, doc)
	}

	return encoder.ToUint8Array()
}

func EncodeStateVector(doc *Doc, m map[Number]Number, encoder *UpdateEncoderV1) []uint8 {
	return EncodeStateVectorV2(doc, m, encoder)
}
